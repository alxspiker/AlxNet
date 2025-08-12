package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// NetworkManager coordinates all network services
type NetworkManager struct {
	discovery *DiscoveryService
	consensus *ConsensusService
	logger    *zap.Logger
	config    *NetworkManagerConfig

	// Network state
	isRunning       bool
	startTime       time.Time
	activePeers     map[string]*PeerInfo
	peerFailures    map[string]int
	lastHealthCheck time.Time

	mu sync.RWMutex
}

// NetworkManagerConfig holds network manager configuration
type NetworkManagerConfig struct {
	DiscoveryConfig     *DiscoveryConfig `json:"discovery_config"`
	ConsensusConfig     *ConsensusConfig `json:"consensus_config"`
	HealthCheckInterval time.Duration    `json:"health_check_interval"`
	MaxPeerFailures     int              `json:"max_peer_failures"`
	PeerSwitchThreshold float64          `json:"peer_switch_threshold"`
	AutoPeerDiscovery   bool             `json:"auto_peer_discovery"`
}

// DefaultNetworkManagerConfig returns default network manager configuration
func DefaultNetworkManagerConfig() *NetworkManagerConfig {
	return &NetworkManagerConfig{
		DiscoveryConfig:     DefaultDiscoveryConfig(),
		ConsensusConfig:     DefaultConsensusConfig(),
		HealthCheckInterval: 2 * time.Minute,
		MaxPeerFailures:     3,
		PeerSwitchThreshold: 0.3,
		AutoPeerDiscovery:   true,
	}
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(config *NetworkManagerConfig, logger *zap.Logger) (*NetworkManager, error) {
	if config == nil {
		config = DefaultNetworkManagerConfig()
	}

	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// Create discovery service
	discovery, err := NewDiscoveryService(config.DiscoveryConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery service: %w", err)
	}

	// Create consensus service
	consensus := NewConsensusService(config.ConsensusConfig, discovery, logger)

	nm := &NetworkManager{
		discovery:    discovery,
		consensus:    consensus,
		logger:       logger,
		config:       config,
		activePeers:  make(map[string]*PeerInfo),
		peerFailures: make(map[string]int),
	}

	return nm, nil
}

// Start initializes and starts all network services
func (nm *NetworkManager) Start(ctx context.Context) error {
	nm.logger.Info("starting network manager")

	nm.mu.Lock()
	if nm.isRunning {
		nm.mu.Unlock()
		return fmt.Errorf("network manager is already running")
	}
	nm.isRunning = true
	nm.startTime = time.Now()
	nm.mu.Unlock()

	// Start discovery service
	if err := nm.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}

	// Start consensus service
	if err := nm.consensus.Start(ctx); err != nil {
		return fmt.Errorf("failed to start consensus service: %w", err)
	}

	// Start background health monitoring
	go nm.backgroundHealthMonitoring()

	nm.logger.Info("network manager started successfully")
	return nil
}

// Stop gracefully stops all network services
func (nm *NetworkManager) Stop() error {
	nm.logger.Info("stopping network manager")

	nm.mu.Lock()
	if !nm.isRunning {
		nm.mu.Unlock()
		return nil
	}
	nm.isRunning = false
	nm.mu.Unlock()

	// Stop consensus service
	if err := nm.consensus.Stop(); err != nil {
		nm.logger.Error("failed to stop consensus service", zap.Error(err))
	}

	// Stop discovery service
	if err := nm.discovery.Stop(); err != nil {
		nm.logger.Error("failed to stop discovery service", zap.Error(err))
	}

	nm.logger.Info("network manager stopped successfully")
	return nil
}

// GetBestPeers returns the best available peers for connection
func (nm *NetworkManager) GetBestPeers(limit int) ([]PeerInfo, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if !nm.isRunning {
		return nil, fmt.Errorf("network manager is not running")
	}

	// Get peers from consensus service
	peers, err := nm.consensus.GetBestPeers(limit)
	if err != nil {
		nm.logger.Warn("failed to get best peers from consensus", zap.Error(err))

		// Fall back to discovery service
		peers = nm.discovery.GetBestPeers(limit)
	}

	// Filter out peers with too many failures
	var filteredPeers []PeerInfo
	for _, peer := range peers {
		failures := nm.peerFailures[peer.ID]
		if failures < nm.config.MaxPeerFailures {
			filteredPeers = append(filteredPeers, peer)
		}
	}

	return filteredPeers, nil
}

// ConnectToPeer attempts to connect to a specific peer
func (nm *NetworkManager) ConnectToPeer(peerID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.isRunning {
		return fmt.Errorf("network manager is not running")
	}

	// Check if peer is already active
	if _, exists := nm.activePeers[peerID]; exists {
		return nil // Already connected
	}

	// Get peer info
	peers := nm.discovery.GetBestPeers(100)
	var targetPeer *PeerInfo
	for _, peer := range peers {
		if peer.ID == peerID {
			targetPeer = &peer
			break
		}
	}

	if targetPeer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	// Add to active peers
	nm.activePeers[peerID] = targetPeer
	nm.peerFailures[peerID] = 0

	nm.logger.Info("connected to peer",
		zap.String("peer_id", peerID),
		zap.String("address", targetPeer.Address))

	return nil
}

// DisconnectFromPeer disconnects from a specific peer
func (nm *NetworkManager) DisconnectFromPeer(peerID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.isRunning {
		return fmt.Errorf("network manager is not running")
	}

	if _, exists := nm.activePeers[peerID]; !exists {
		return fmt.Errorf("peer %s is not connected", peerID)
	}

	delete(nm.activePeers, peerID)

	nm.logger.Info("disconnected from peer", zap.String("peer_id", peerID))
	return nil
}

// ReportPeerFailure reports a failure for a specific peer
func (nm *NetworkManager) ReportPeerFailure(peerID string, failureType string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.isRunning {
		return fmt.Errorf("network manager is not running")
	}

	// Increment failure count
	nm.peerFailures[peerID]++

	// Remove from active peers if too many failures
	if nm.peerFailures[peerID] >= nm.config.MaxPeerFailures {
		delete(nm.activePeers, peerID)
		nm.logger.Warn("peer removed due to excessive failures",
			zap.String("peer_id", peerID),
			zap.Int("failures", nm.peerFailures[peerID]))
	}

	nm.logger.Info("peer failure reported",
		zap.String("peer_id", peerID),
		zap.String("failure_type", failureType),
		zap.Int("total_failures", nm.peerFailures[peerID]))

	return nil
}

// ReportPeerSuccess reports a successful operation for a peer
func (nm *NetworkManager) ReportPeerSuccess(peerID string, metrics NodeScore) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if !nm.isRunning {
		return fmt.Errorf("network manager is not running")
	}

	// Reset failure count on success
	nm.peerFailures[peerID] = 0

	// Update peer metrics in consensus service
	nm.consensus.UpdatePeerMetrics(peerID, metrics)

	// Add to favorites if performance is excellent
	if metrics.CalculateScore(nil) > 0.9 {
		if peer, exists := nm.activePeers[peerID]; exists {
			nm.discovery.AddToFavorites(peer)
		}
	}

	nm.logger.Debug("peer success reported",
		zap.String("peer_id", peerID),
		zap.Float64("score", metrics.CalculateScore(nil)))

	return nil
}

// GetNetworkStatus returns the current network status
func (nm *NetworkManager) GetNetworkStatus() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	status := map[string]interface{}{
		"is_running":        nm.isRunning,
		"start_time":        nm.startTime,
		"uptime":            time.Since(nm.startTime).String(),
		"active_peers":      len(nm.activePeers),
		"total_peers":       len(nm.peerFailures),
		"last_health_check": nm.lastHealthCheck,
	}

	// Add discovery status
	if nm.discovery != nil {
		status["discovery"] = map[string]interface{}{
			"master_nodes":  nm.discovery.GetMasterNodes(),
			"network_stats": nm.discovery.GetNetworkStats(),
		}
	}

	// Add consensus status
	if nm.consensus != nil {
		status["consensus"] = nm.consensus.GetConsensusReport()
	}

	return status
}

// RefreshNetwork refreshes the network discovery and consensus
func (nm *NetworkManager) RefreshNetwork(ctx context.Context) error {
	nm.mu.RLock()
	if !nm.isRunning {
		nm.mu.RUnlock()
		return fmt.Errorf("network manager is not running")
	}
	nm.mu.RUnlock()

	nm.logger.Info("refreshing network")

	// Refresh master list
	if err := nm.discovery.RefreshMasterList(ctx); err != nil {
		nm.logger.Error("failed to refresh master list", zap.Error(err))
	}

	// Update consensus
	if err := nm.consensus.UpdateConsensus(ctx); err != nil {
		nm.logger.Error("failed to update consensus", zap.Error(err))
	}

	nm.logger.Info("network refresh completed")
	return nil
}

// backgroundHealthMonitoring runs periodic health checks
func (nm *NetworkManager) backgroundHealthMonitoring() {
	ticker := time.NewTicker(nm.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		nm.mu.RLock()
		if !nm.isRunning {
			nm.mu.RUnlock()
			return
		}
		nm.mu.RUnlock()

		// Perform health check
		if err := nm.performHealthCheck(); err != nil {
			nm.logger.Error("health check failed", zap.Error(err))
		}

		nm.mu.Lock()
		nm.lastHealthCheck = time.Now()
		nm.mu.Unlock()
	}
}

// performHealthCheck performs a comprehensive health check
func (nm *NetworkManager) performHealthCheck() error {
	// Check discovery service health
	if nm.discovery == nil {
		return fmt.Errorf("discovery service is nil")
	}

	// Check consensus service health
	if nm.consensus == nil {
		return fmt.Errorf("consensus service is nil")
	}

	// Check active peers health
	nm.mu.RLock()
	activePeerCount := len(nm.activePeers)
	nm.mu.RUnlock()

	if activePeerCount == 0 {
		nm.logger.Warn("no active peers - network may be isolated")
	}

	// Clean up old peer failures
	nm.cleanupOldPeerFailures()

	nm.logger.Debug("health check completed successfully",
		zap.Int("active_peers", activePeerCount))

	return nil
}

// cleanupOldPeerFailures removes old peer failure records
func (nm *NetworkManager) cleanupOldPeerFailures() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Simple cleanup: remove peers with 0 failures that haven't been active
	for peerID, failures := range nm.peerFailures {
		if failures == 0 {
			if _, isActive := nm.activePeers[peerID]; !isActive {
				delete(nm.peerFailures, peerID)
			}
		}
	}
}

// GetPeerInfo returns information about a specific peer
func (nm *NetworkManager) GetPeerInfo(peerID string) *PeerInfo {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	// Check active peers first
	if peer, exists := nm.activePeers[peerID]; exists {
		return peer
	}

	// Check all available peers
	peers := nm.discovery.GetBestPeers(100)
	for _, peer := range peers {
		if peer.ID == peerID {
			return &peer
		}
	}

	return nil
}

// GetNetworkHealth returns the current network health status
func (nm *NetworkManager) GetNetworkHealth() *NetworkHealth {
	if nm.consensus == nil {
		return nil
	}

	return nm.consensus.GetNetworkHealth()
}

// UpdateNetworkHealth updates the network health metrics
func (nm *NetworkManager) UpdateNetworkHealth(health *NetworkHealth) {
	if nm.consensus == nil {
		return
	}

	nm.consensus.UpdateNetworkHealth(health)
}
