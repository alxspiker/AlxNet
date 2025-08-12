package network

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// scoredPeer represents a peer with its consensus score
type scoredPeer struct {
	peer  PeerInfo
	score float64
}

// ConsensusService handles network consensus and peer selection
type ConsensusService struct {
	config    *ConsensusConfig
	logger    *zap.Logger
	discovery *DiscoveryService

	// Consensus state
	consensusData *Consensus
	peerRatings   map[string][]Rating
	networkHealth *NetworkHealth

	// Performance tracking
	peerMetrics   map[string]*NodeScore
	lastConsensus time.Time

	mu sync.RWMutex
}

// ConsensusConfig holds consensus service configuration
type ConsensusConfig struct {
	MinPeersForConsensus int                  `json:"min_peers_for_consensus"`
	ConsensusTimeout     time.Duration        `json:"consensus_timeout"`
	ScoreThreshold       float64              `json:"score_threshold"`
	GeographicPreference bool                 `json:"geographic_preference"`
	LoadBalancing        bool                 `json:"load_balancing"`
	FaultTolerance       FaultToleranceConfig `json:"fault_tolerance"`
	UpdateInterval       time.Duration        `json:"update_interval"`
}

// DefaultConsensusConfig returns default consensus configuration
func DefaultConsensusConfig() *ConsensusConfig {
	return &ConsensusConfig{
		MinPeersForConsensus: 3,
		ConsensusTimeout:     30 * time.Second,
		ScoreThreshold:       0.7,
		GeographicPreference: true,
		LoadBalancing:        true,
		FaultTolerance: FaultToleranceConfig{
			MaxFailures:     3,
			BackoffTime:     "5m",
			SwitchThreshold: 0.3,
		},
		UpdateInterval: 1 * time.Minute,
	}
}

// NewConsensusService creates a new consensus service
func NewConsensusService(config *ConsensusConfig, discovery *DiscoveryService, logger *zap.Logger) *ConsensusService {
	if config == nil {
		config = DefaultConsensusConfig()
	}

	cs := &ConsensusService{
		config:      config,
		logger:      logger,
		discovery:   discovery,
		peerRatings: make(map[string][]Rating),
		peerMetrics: make(map[string]*NodeScore),
	}

	// Start background consensus updates
	go cs.backgroundConsensus()

	return cs
}

// Start initializes the consensus service
func (ds *ConsensusService) Start(ctx context.Context) error {
	ds.logger.Info("starting consensus service")

	// Perform initial consensus
	if err := ds.UpdateConsensus(ctx); err != nil {
		ds.logger.Warn("initial consensus update failed", zap.Error(err))
	}

	ds.logger.Info("consensus service started successfully")
	return nil
}

// Stop gracefully stops the consensus service
func (ds *ConsensusService) Stop() error {
	ds.logger.Info("stopping consensus service")
	return nil
}

// UpdateConsensus updates the network consensus data
func (cs *ConsensusService) UpdateConsensus(ctx context.Context) error {
	cs.logger.Info("updating network consensus")

	// Get current master nodes
	masterNodes := cs.discovery.GetMasterNodes()
	if len(masterNodes) == 0 {
		return fmt.Errorf("no master nodes available for consensus")
	}

	// Get current network stats
	networkStats := cs.discovery.GetNetworkStats()
	if networkStats == nil {
		return fmt.Errorf("network stats not available")
	}

	// Get consensus rules
	consensusRules := cs.discovery.GetConsensusRules()
	if consensusRules == nil {
		return fmt.Errorf("consensus rules not available")
	}

	// Create new consensus data
	consensus := &Consensus{
		NetworkVersion: "1.0",
		MasterNodes:    make([]string, len(masterNodes)),
		NetworkHealth:  networkStats.NetworkHealth,
		PeerCount:      networkStats.ActiveNodes,
		Timestamp:      time.Now(),
		Signatures:     make(map[string]string),
	}

	// Extract master node IDs
	for i, mn := range masterNodes {
		consensus.MasterNodes[i] = mn.ID
	}

	cs.mu.Lock()
	cs.consensusData = consensus
	cs.lastConsensus = time.Now()
	cs.mu.Unlock()

	cs.logger.Info("consensus updated successfully",
		zap.Int("master_nodes", len(consensus.MasterNodes)),
		zap.Int("peer_count", consensus.PeerCount),
		zap.Float64("network_uptime", consensus.NetworkHealth.Uptime))

	return nil
}

// GetConsensus returns the current consensus data
func (cs *ConsensusService) GetConsensus() *Consensus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.consensusData
}

// AddPeerRating adds a rating for a peer
func (cs *ConsensusService) AddPeerRating(rating Rating) error {
	if err := cs.validateRating(rating); err != nil {
		return fmt.Errorf("invalid rating: %w", err)
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Add rating to peer's rating list
	cs.peerRatings[rating.NodeID] = append(cs.peerRatings[rating.NodeID], rating)

	// Update peer metrics
	cs.updatePeerMetrics(rating.NodeID, rating.Metrics)

	cs.logger.Info("added peer rating",
		zap.String("node_id", rating.NodeID),
		zap.String("rater_id", rating.RaterID),
		zap.Float64("score", rating.Score))

	return nil
}

// GetPeerScore calculates the consensus score for a peer
func (cs *ConsensusService) GetPeerScore(peerID string) (float64, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	ratings, exists := cs.peerRatings[peerID]
	if !exists || len(ratings) == 0 {
		return 0.0, fmt.Errorf("no ratings available for peer %s", peerID)
	}

	// Calculate weighted average score
	var totalScore, totalWeight float64

	for _, rating := range ratings {
		// Weight by recency (newer ratings have higher weight)
		age := time.Since(rating.Timestamp)
		weight := math.Exp(-age.Hours() / 24.0) // Exponential decay over 24 hours

		totalScore += rating.Score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0, fmt.Errorf("no valid ratings for peer %s", peerID)
	}

	consensusScore := totalScore / totalWeight

	// Apply geographic preference if enabled
	if cs.config.GeographicPreference {
		if metrics, exists := cs.peerMetrics[peerID]; exists {
			consensusScore *= (1.0 + metrics.Geographic*0.1) // 10% bonus for geographic preference
		}
	}

	// Ensure score is between 0 and 1
	if consensusScore < 0 {
		consensusScore = 0
	}
	if consensusScore > 1 {
		consensusScore = 1
	}

	return consensusScore, nil
}

// GetBestPeers returns the best peers based on consensus scores
func (cs *ConsensusService) GetBestPeers(limit int) ([]PeerInfo, error) {
	// Get all available peers
	allPeers := cs.discovery.GetBestPeers(100) // Get more than needed for better selection

	if len(allPeers) == 0 {
		return nil, fmt.Errorf("no peers available")
	}

	// Calculate consensus scores for all peers
	var scoredPeers []scoredPeer

	for _, peer := range allPeers {
		score, err := cs.GetPeerScore(peer.ID)
		if err != nil {
			cs.logger.Debug("failed to get score for peer",
				zap.String("peer_id", peer.ID),
				zap.Error(err))
			score = peer.Score // Fall back to discovery score
		}

		scoredPeers = append(scoredPeers, scoredPeer{
			peer:  peer,
			score: score,
		})
	}

	// Sort by consensus score (highest first)
	sort.Slice(scoredPeers, func(i, j int) bool {
		return scoredPeers[i].score > scoredPeers[j].score
	})

	// Apply load balancing if enabled
	if cs.config.LoadBalancing {
		scoredPeers = cs.applyLoadBalancing(scoredPeers)
	}

	// Return top peers
	var result []PeerInfo
	for i, sp := range scoredPeers {
		if i >= limit {
			break
		}
		result = append(result, sp.peer)
	}

	return result, nil
}

// GetNetworkHealth returns the current network health status
func (cs *ConsensusService) GetNetworkHealth() *NetworkHealth {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.networkHealth == nil {
		return &NetworkHealth{
			Uptime:       0.0,
			LatencyAvg:   0,
			BandwidthAvg: 0,
		}
	}

	return cs.networkHealth
}

// UpdateNetworkHealth updates the network health metrics
func (cs *ConsensusService) UpdateNetworkHealth(health *NetworkHealth) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.networkHealth = health
	cs.logger.Info("network health updated",
		zap.Float64("uptime", health.Uptime),
		zap.Int("latency_avg", health.LatencyAvg),
		zap.Int("bandwidth_avg", health.BandwidthAvg))
}

// GetPeerMetrics returns the performance metrics for a peer
func (cs *ConsensusService) GetPeerMetrics(peerID string) *NodeScore {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if metrics, exists := cs.peerMetrics[peerID]; exists {
		return metrics
	}

	return nil
}

// UpdatePeerMetrics updates the performance metrics for a peer
func (cs *ConsensusService) UpdatePeerMetrics(peerID string, metrics NodeScore) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.peerMetrics[peerID] = &metrics

	// Update discovery service with new score
	calculatedScore := metrics.CalculateScore(nil)
	if err := cs.discovery.UpdatePeerScore(peerID, calculatedScore, metrics); err != nil {
		cs.logger.Error("failed to update peer score in discovery service",
			zap.String("peer_id", peerID),
			zap.Error(err))
	}
}

// validateRating validates a peer rating
func (cs *ConsensusService) validateRating(rating Rating) error {
	if rating.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	if rating.RaterID == "" {
		return fmt.Errorf("rater ID is required")
	}
	if rating.Score < 0 || rating.Score > 1 {
		return fmt.Errorf("score must be between 0 and 1")
	}
	if rating.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	return nil
}

// updatePeerMetrics updates the aggregated metrics for a peer
func (cs *ConsensusService) updatePeerMetrics(peerID string, newMetrics NodeScore) {
	existing, exists := cs.peerMetrics[peerID]
	if !exists {
		cs.peerMetrics[peerID] = &newMetrics
		return
	}

	// Simple moving average for metrics
	const alpha = 0.3 // Weight for new values

	existing.Latency = existing.Latency*(1-alpha) + newMetrics.Latency*alpha
	existing.Uptime = existing.Uptime*(1-alpha) + newMetrics.Uptime*alpha
	existing.Bandwidth = existing.Bandwidth*(1-alpha) + newMetrics.Bandwidth*alpha
	existing.Storage = existing.Storage*(1-alpha) + newMetrics.Storage*alpha
	existing.Reliability = existing.Reliability*(1-alpha) + newMetrics.Reliability*alpha
	existing.Geographic = existing.Geographic*(1-alpha) + newMetrics.Geographic*alpha
	existing.Load = existing.Load*(1-alpha) + newMetrics.Load*alpha
}

// applyLoadBalancing applies load balancing to peer selection
func (cs *ConsensusService) applyLoadBalancing(scoredPeers []scoredPeer) []scoredPeer {
	if len(scoredPeers) < 2 {
		return scoredPeers
	}

	// Simple round-robin load balancing
	// This could be enhanced with more sophisticated algorithms
	result := make([]scoredPeer, len(scoredPeers))
	copy(result, scoredPeers)

	// Sort by load (lower load gets higher priority)
	sort.Slice(result, func(i, j int) bool {
		// Get current load metrics
		loadI := float64(0.5) // Default load
		loadJ := float64(0.5)

		if metrics := cs.GetPeerMetrics(result[i].peer.ID); metrics != nil {
			loadI = metrics.Load
		}
		if metrics := cs.GetPeerMetrics(result[j].peer.ID); metrics != nil {
			loadJ = metrics.Load
		}

		return loadI < loadJ
	})

	return result
}

// backgroundConsensus runs periodic consensus updates
func (cs *ConsensusService) backgroundConsensus() {
	ticker := time.NewTicker(cs.config.UpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), cs.config.ConsensusTimeout)

		if err := cs.UpdateConsensus(ctx); err != nil {
			cs.logger.Error("background consensus update failed", zap.Error(err))
		}

		cancel()
	}
}

// GetConsensusReport returns a detailed consensus report
func (cs *ConsensusService) GetConsensusReport() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	report := map[string]interface{}{
		"last_consensus": cs.lastConsensus,
		"peer_count":     len(cs.peerRatings),
		"network_health": cs.networkHealth,
	}

	if cs.consensusData != nil {
		report["consensus_data"] = cs.consensusData
	}

	// Add peer statistics
	peerStats := make(map[string]interface{})
	for peerID, ratings := range cs.peerRatings {
		peerStats[peerID] = map[string]interface{}{
			"rating_count": len(ratings),
			"avg_score":    cs.calculateAverageScore(ratings),
			"last_rating":  cs.getLastRatingTime(ratings),
		}
	}
	report["peer_statistics"] = peerStats

	return report
}

// calculateAverageScore calculates the average score from ratings
func (cs *ConsensusService) calculateAverageScore(ratings []Rating) float64 {
	if len(ratings) == 0 {
		return 0.0
	}

	var total float64
	for _, rating := range ratings {
		total += rating.Score
	}

	return total / float64(len(ratings))
}

// getLastRatingTime gets the timestamp of the most recent rating
func (cs *ConsensusService) getLastRatingTime(ratings []Rating) time.Time {
	if len(ratings) == 0 {
		return time.Time{}
	}

	latest := ratings[0].Timestamp
	for _, rating := range ratings[1:] {
		if rating.Timestamp.After(latest) {
			latest = rating.Timestamp
		}
	}

	return latest
}
