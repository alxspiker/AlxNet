package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DiscoveryService handles network discovery and peer management
type DiscoveryService struct {
	config          *DiscoveryConfig
	logger          *zap.Logger
	masterList      *MasterList
	localList       *LocalMasterList
	discoveredPeers map[string]*PeerInfo
	mu              sync.RWMutex

	// Network state
	lastUpdate     time.Time
	updateInterval time.Duration
	httpClient     *http.Client
}

// DiscoveryConfig holds discovery service configuration
type DiscoveryConfig struct {
	GitHubMasterListURL string        `json:"github_master_list_url"`
	LocalMasterListPath string        `json:"local_master_list_path"`
	UpdateInterval      time.Duration `json:"update_interval"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	Timeout             time.Duration `json:"timeout"`
}

// DefaultDiscoveryConfig returns default discovery configuration
func DefaultDiscoveryConfig() *DiscoveryConfig {
	return &DiscoveryConfig{
		GitHubMasterListURL: "https://raw.githubusercontent.com/alxspiker/alxnet/main/network/masterlist.json",
		LocalMasterListPath: "~/.alxnet/network/local-masterlist.json",
		UpdateInterval:      5 * time.Minute,
		MaxRetries:          3,
		RetryDelay:          30 * time.Second,
		Timeout:             30 * time.Second,
	}
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(config *DiscoveryConfig, logger *zap.Logger) (*DiscoveryService, error) {
	if config == nil {
		config = DefaultDiscoveryConfig()
	}

	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// Expand home directory in path
	if config.LocalMasterListPath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		config.LocalMasterListPath = filepath.Join(home, config.LocalMasterListPath[2:])
	}

	// Ensure directory exists
	dir := filepath.Dir(config.LocalMasterListPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	ds := &DiscoveryService{
		config:          config,
		logger:          logger,
		discoveredPeers: make(map[string]*PeerInfo),
		updateInterval:  config.UpdateInterval,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	// Load local master list
	if err := ds.loadLocalMasterList(); err != nil {
		logger.Warn("failed to load local master list", zap.Error(err))
	}

	// Start background updates
	go ds.backgroundUpdates()

	return ds, nil
}

// Start initializes the discovery service
func (ds *DiscoveryService) Start(ctx context.Context) error {
	ds.logger.Info("starting discovery service")

	// Perform initial discovery
	if err := ds.RefreshMasterList(ctx); err != nil {
		ds.logger.Warn("initial master list refresh failed", zap.Error(err))
	}

	ds.logger.Info("discovery service started successfully")
	return nil
}

// Stop gracefully stops the discovery service
func (ds *DiscoveryService) Stop() error {
	ds.logger.Info("stopping discovery service")
	// Save local master list before stopping
	if err := ds.saveLocalMasterList(); err != nil {
		ds.logger.Error("failed to save local master list", zap.Error(err))
	}
	return nil
}

// RefreshMasterList fetches the latest master list from GitHub or local file
func (ds *DiscoveryService) RefreshMasterList(ctx context.Context) error {
	ds.logger.Info("refreshing master list")

	// Try to load from local network directory first
	scriptDir, err := os.Getwd()
	if err == nil {
		localPath := filepath.Join(scriptDir, "network", "masterlist.json")
		if data, err := os.ReadFile(localPath); err == nil {
			var masterList MasterList
			if err := json.Unmarshal(data, &masterList); err == nil {
				if err := masterList.Validate(); err == nil {
					ds.mu.Lock()
					ds.masterList = &masterList
					ds.lastUpdate = time.Now()
					ds.mu.Unlock()

					ds.logger.Info("master list loaded from local file",
						zap.Int("master_nodes", len(masterList.MasterNodes)),
						zap.String("network_name", masterList.NetworkName))
					return nil
				}
			}
		}
	}

	// Fallback to GitHub if local file fails
	ds.logger.Info("falling back to GitHub master list")

	req, err := http.NewRequestWithContext(ctx, "GET", ds.config.GitHubMasterListURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "AlxNet-Discovery/1.0")

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch master list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var masterList MasterList
	if err := json.Unmarshal(data, &masterList); err != nil {
		return fmt.Errorf("failed to unmarshal master list: %w", err)
	}

	if err := masterList.Validate(); err != nil {
		return fmt.Errorf("invalid master list: %w", err)
	}

	ds.mu.Lock()
	ds.masterList = &masterList
	ds.lastUpdate = time.Now()
	ds.mu.Unlock()

	ds.logger.Info("master list refreshed successfully from GitHub",
		zap.Int("master_nodes", len(masterList.MasterNodes)),
		zap.String("network_name", masterList.NetworkName))

	return nil
}

// GetMasterNodes returns the current master nodes
func (ds *DiscoveryService) GetMasterNodes() []MasterNode {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.masterList == nil {
		return nil
	}

	return ds.masterList.MasterNodes
}

// GetBestPeers returns the best available peers for connection
func (ds *DiscoveryService) GetBestPeers(limit int) []PeerInfo {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var allPeers []PeerInfo

	// Add master nodes
	if ds.masterList != nil {
		for _, mn := range ds.masterList.MasterNodes {
			peer := PeerInfo{
				ID:           mn.ID,
				Address:      mn.Address,
				Score:        mn.ReliabilityScore,
				LastSeen:     mn.LastSeen,
				Capabilities: mn.Capabilities,
				Location:     mn.Location,
			}
			allPeers = append(allPeers, peer)
		}
	}

	// Add local favorites
	if ds.localList != nil {
		allPeers = append(allPeers, ds.localList.Favorites...)
	}

	// Add discovered peers
	for _, peer := range ds.discoveredPeers {
		allPeers = append(allPeers, *peer)
	}

	// Sort by score (highest first)
	sort.Slice(allPeers, func(i, j int) bool {
		return allPeers[i].Score > allPeers[j].Score
	})

	// Remove duplicates and limit results
	seen := make(map[string]bool)
	var uniquePeers []PeerInfo

	for _, peer := range allPeers {
		if !seen[peer.ID] && len(uniquePeers) < limit {
			seen[peer.ID] = true
			uniquePeers = append(uniquePeers, peer)
		}
	}

	return uniquePeers
}

// AddDiscoveredPeer adds a newly discovered peer
func (ds *DiscoveryService) AddDiscoveredPeer(peer *PeerInfo) error {
	if err := peer.Validate(); err != nil {
		return fmt.Errorf("invalid peer: %w", err)
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.discoveredPeers[peer.ID] = peer
	ds.logger.Info("added discovered peer",
		zap.String("peer_id", peer.ID),
		zap.String("address", peer.Address),
		zap.Float64("score", peer.Score))

	return nil
}

// UpdatePeerScore updates a peer's score based on performance
func (ds *DiscoveryService) UpdatePeerScore(peerID string, score float64, metrics NodeScore) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	peer, exists := ds.discoveredPeers[peerID]
	if !exists {
		return fmt.Errorf("peer %s not found", peerID)
	}

	peer.Score = score
	peer.LastSeen = time.Now()
	peer.Uptime = metrics.Uptime
	peer.Latency = int(metrics.Latency)
	peer.Bandwidth = int(metrics.Bandwidth)

	ds.logger.Info("updated peer score",
		zap.String("peer_id", peerID),
		zap.Float64("new_score", score))

	return nil
}

// AddToFavorites adds a peer to the local favorites list
func (ds *DiscoveryService) AddToFavorites(peer *PeerInfo) error {
	if err := peer.Validate(); err != nil {
		return fmt.Errorf("invalid peer: %w", err)
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Initialize local list if needed
	if ds.localList == nil {
		ds.localList = &LocalMasterList{
			Version:     "1.0",
			LastUpdated: time.Now(),
			Favorites:   []PeerInfo{},
			Discovered:  []PeerInfo{},
			Blacklist:   []string{},
			Settings: LocalDiscoverySettings{
				AutoUpdateMasterList: true,
				UpdateInterval:       "5m",
				MaxLocalPeers:        50,
				PreferLocalNetwork:   true,
			},
		}
	}

	// Check if already in favorites
	for _, fav := range ds.localList.Favorites {
		if fav.ID == peer.ID {
			return nil // Already in favorites
		}
	}

	ds.localList.Favorites = append(ds.localList.Favorites, *peer)
	ds.localList.LastUpdated = time.Now()

	// Save to disk
	if err := ds.saveLocalMasterList(); err != nil {
		ds.logger.Error("failed to save local master list", zap.Error(err))
	}

	ds.logger.Info("added peer to favorites",
		zap.String("peer_id", peer.ID),
		zap.String("address", peer.Address))

	return nil
}

// RemoveFromFavorites removes a peer from favorites
func (ds *DiscoveryService) RemoveFromFavorites(peerID string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.localList == nil {
		return nil
	}

	for i, fav := range ds.localList.Favorites {
		if fav.ID == peerID {
			ds.localList.Favorites = append(ds.localList.Favorites[:i], ds.localList.Favorites[i+1:]...)
			ds.localList.LastUpdated = time.Now()

			// Save to disk
			if err := ds.saveLocalMasterList(); err != nil {
				ds.logger.Error("failed to save local master list", zap.Error(err))
			}

			ds.logger.Info("removed peer from favorites", zap.String("peer_id", peerID))
			return nil
		}
	}

	return fmt.Errorf("peer %s not found in favorites", peerID)
}

// GetNetworkStats returns current network statistics
func (ds *DiscoveryService) GetNetworkStats() *NetworkStats {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.masterList == nil {
		return nil
	}

	return &ds.masterList.NetworkStats
}

// GetConsensusRules returns the current consensus rules
func (ds *DiscoveryService) GetConsensusRules() *ConsensusRules {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.masterList == nil {
		return nil
	}

	return &ds.masterList.ConsensusRules
}

// loadLocalMasterList loads the local master list from disk
func (ds *DiscoveryService) loadLocalMasterList() error {
	data, err := os.ReadFile(ds.config.LocalMasterListPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return fmt.Errorf("failed to read local master list: %w", err)
	}

	var localList LocalMasterList
	if err := json.Unmarshal(data, &localList); err != nil {
		return fmt.Errorf("failed to unmarshal local master list: %w", err)
	}

	ds.localList = &localList
	ds.logger.Info("loaded local master list",
		zap.Int("favorites", len(localList.Favorites)),
		zap.Int("discovered", len(localList.Discovered)))

	return nil
}

// saveLocalMasterList saves the local master list to disk
func (ds *DiscoveryService) saveLocalMasterList() error {
	if ds.localList == nil {
		return nil
	}

	data, err := json.MarshalIndent(ds.localList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal local master list: %w", err)
	}

	if err := os.WriteFile(ds.config.LocalMasterListPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write local master list: %w", err)
	}

	return nil
}

// backgroundUpdates runs periodic updates in the background
func (ds *DiscoveryService) backgroundUpdates() {
	ticker := time.NewTicker(ds.updateInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), ds.config.Timeout)

		if err := ds.RefreshMasterList(ctx); err != nil {
			ds.logger.Error("background master list refresh failed", zap.Error(err))
		}

		// Save local master list periodically
		if err := ds.saveLocalMasterList(); err != nil {
			ds.logger.Error("background local master list save failed", zap.Error(err))
		}

		cancel()
	}
}

// CleanupOldPeers removes peers that haven't been seen recently
func (ds *DiscoveryService) CleanupOldPeers(maxAge time.Duration) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var toRemove []string

	for id, peer := range ds.discoveredPeers {
		if peer.LastSeen.Before(cutoff) {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		delete(ds.discoveredPeers, id)
	}

	if len(toRemove) > 0 {
		ds.logger.Info("cleaned up old peers",
			zap.Int("removed_count", len(toRemove)),
			zap.Duration("max_age", maxAge))
	}
}
