package network

import (
	"encoding/json"
	"fmt"
	"time"
)

// MasterList represents the global master node list
type MasterList struct {
	Version        string         `json:"version"`
	LastUpdated    time.Time      `json:"last_updated"`
	NetworkName    string         `json:"network_name"`
	Description    string         `json:"description"`
	MasterNodes    []MasterNode   `json:"master_nodes"`
	NetworkStats   NetworkStats   `json:"network_stats"`
	ConsensusRules ConsensusRules `json:"consensus_rules"`
	UpdateFreq     string         `json:"update_frequency"`
	GitHubRepo     string         `json:"github_repo"`
}

// MasterNode represents a master node in the network
type MasterNode struct {
	ID               string    `json:"id"`
	Address          string    `json:"address"`
	Location         string    `json:"location"`
	ReliabilityScore float64   `json:"reliability_score"`
	LastSeen         time.Time `json:"last_seen"`
	Capabilities     []string  `json:"capabilities"`
	Contact          string    `json:"contact"`
	Description      string    `json:"description"`
}

// NetworkStats contains network health and statistics
type NetworkStats struct {
	TotalNodes             int            `json:"total_nodes"`
	ActiveNodes            int            `json:"active_nodes"`
	GeographicDistribution map[string]int `json:"geographic_distribution"`
	NetworkHealth          NetworkHealth  `json:"network_health"`
}

// NetworkHealth represents network performance metrics
type NetworkHealth struct {
	Uptime       float64 `json:"uptime"`
	LatencyAvg   int     `json:"latency_avg"`
	BandwidthAvg int     `json:"bandwidth_avg"`
}

// ConsensusRules defines how nodes reach consensus
type ConsensusRules struct {
	MinPeersForConsensus int                  `json:"min_peers_for_consensus"`
	ConsensusTimeout     string               `json:"consensus_timeout"`
	ScoreThreshold       float64              `json:"score_threshold"`
	GeographicPreference bool                 `json:"geographic_preference"`
	LoadBalancing        bool                 `json:"load_balancing"`
	FaultTolerance       FaultToleranceConfig `json:"fault_tolerance"`
}

// FaultToleranceConfig defines fault tolerance behavior
type FaultToleranceConfig struct {
	MaxFailures     int     `json:"max_failures"`
	BackoffTime     string  `json:"backoff_time"`
	SwitchThreshold float64 `json:"switch_threshold"`
}

// PeerInfo represents a discovered peer
type PeerInfo struct {
	ID           string    `json:"id"`
	Address      string    `json:"address"`
	Score        float64   `json:"score"`
	LastSeen     time.Time `json:"last_seen"`
	Capabilities []string  `json:"capabilities"`
	Location     string    `json:"location,omitempty"`
	Uptime       float64   `json:"uptime,omitempty"`
	Latency      int       `json:"latency,omitempty"`
	Bandwidth    int       `json:"bandwidth,omitempty"`
	Failures     int       `json:"failures,omitempty"`
	LastFailure  time.Time `json:"last_failure,omitempty"`
}

// NodeScore represents a node's performance metrics
type NodeScore struct {
	Latency     float64 `json:"latency"`     // RTT in milliseconds
	Uptime      float64 `json:"uptime"`      // 0.0 to 1.0
	Bandwidth   float64 `json:"bandwidth"`   // Mbps
	Storage     float64 `json:"storage"`     // Available storage
	Reliability float64 `json:"reliability"` // Success rate of requests
	Geographic  float64 `json:"geographic"`  // Proximity bonus
	Load        float64 `json:"load"`        // Current load (lower is better)
}

// PeerExchange represents peer discovery and rating exchange
type PeerExchange struct {
	Type      string     `json:"type"`      // "peer_list", "node_rating", "consensus"
	Peers     []PeerInfo `json:"peers"`     // New peers discovered
	Ratings   []Rating   `json:"ratings"`   // Node performance ratings
	Consensus Consensus  `json:"consensus"` // Network consensus data
	Timestamp time.Time  `json:"timestamp"`
}

// Rating represents a peer's rating of another node
type Rating struct {
	NodeID    string    `json:"node_id"`
	RaterID   string    `json:"rater_id"`
	Score     float64   `json:"score"`
	Metrics   NodeScore `json:"metrics"`
	Timestamp time.Time `json:"timestamp"`
}

// Consensus represents network consensus data
type Consensus struct {
	NetworkVersion string            `json:"network_version"`
	MasterNodes    []string          `json:"master_nodes"`
	NetworkHealth  NetworkHealth     `json:"network_health"`
	PeerCount      int               `json:"peer_count"`
	Timestamp      time.Time         `json:"timestamp"`
	Signatures     map[string]string `json:"signatures,omitempty"`
}

// LocalMasterList represents the user's local peer preferences
type LocalMasterList struct {
	Version     string                 `json:"version"`
	LastUpdated time.Time              `json:"last_updated"`
	Favorites   []PeerInfo             `json:"favorites"`
	Discovered  []PeerInfo             `json:"discovered"`
	Blacklist   []string               `json:"blacklist"`
	Settings    LocalDiscoverySettings `json:"settings"`
}

// LocalDiscoverySettings defines local discovery behavior
type LocalDiscoverySettings struct {
	AutoUpdateMasterList bool   `json:"auto_update_master_list"`
	UpdateInterval       string `json:"update_interval"`
	MaxLocalPeers        int    `json:"max_local_peers"`
	PreferLocalNetwork   bool   `json:"prefer_local_network"`
}

// DiscoveryResult represents the result of a peer discovery operation
type DiscoveryResult struct {
	Peers     []PeerInfo    `json:"peers"`
	Source    string        `json:"source"` // "github", "local", "discovered", "mdns"
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	PeerCount int           `json:"peer_count"`
}

// Validate performs validation on MasterList
func (ml *MasterList) Validate() error {
	if ml.Version == "" {
		return fmt.Errorf("version is required")
	}
	if ml.LastUpdated.IsZero() {
		return fmt.Errorf("last_updated is required")
	}
	if len(ml.MasterNodes) == 0 {
		return fmt.Errorf("at least one master node is required")
	}

	for i, node := range ml.MasterNodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("master node %d invalid: %w", i, err)
		}
	}

	return nil
}

// Validate performs validation on MasterNode
func (mn *MasterNode) Validate() error {
	if mn.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if mn.Address == "" {
		return fmt.Errorf("node address is required")
	}
	if mn.ReliabilityScore < 0 || mn.ReliabilityScore > 1 {
		return fmt.Errorf("reliability score must be between 0 and 1")
	}
	if mn.LastSeen.IsZero() {
		return fmt.Errorf("last seen timestamp is required")
	}
	return nil
}

// Validate performs validation on PeerInfo
func (pi *PeerInfo) Validate() error {
	if pi.ID == "" {
		return fmt.Errorf("peer ID is required")
	}
	if pi.Address == "" {
		return fmt.Errorf("peer address is required")
	}
	if pi.Score < 0 || pi.Score > 1 {
		return fmt.Errorf("peer score must be between 0 and 1")
	}
	return nil
}

// CalculateScore calculates the overall score for a node based on metrics
func (ns *NodeScore) CalculateScore(weights map[string]float64) float64 {
	if weights == nil {
		// Default weights if none provided
		weights = map[string]float64{
			"latency":     0.20,
			"uptime":      0.25,
			"bandwidth":   0.15,
			"storage":     0.10,
			"reliability": 0.20,
			"geographic":  0.10,
		}
	}

	score := (ns.Latency*weights["latency"] +
		ns.Uptime*weights["uptime"] +
		ns.Bandwidth*weights["bandwidth"] +
		ns.Storage*weights["storage"] +
		ns.Reliability*weights["reliability"] +
		ns.Geographic*weights["geographic"]) * (1.0 - ns.Load)

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// MarshalJSON custom marshaling for time.Time fields
func (ml *MasterList) MarshalJSON() ([]byte, error) {
	type Alias MasterList
	return json.Marshal(&struct {
		*Alias
		LastUpdated string `json:"last_updated"`
	}{
		Alias:       (*Alias)(ml),
		LastUpdated: ml.LastUpdated.Format(time.RFC3339),
	})
}

// UnmarshalJSON custom unmarshaling for time.Time fields
func (ml *MasterList) UnmarshalJSON(data []byte) error {
	type Alias MasterList
	aux := &struct {
		*Alias
		LastUpdated string `json:"last_updated"`
	}{
		Alias: (*Alias)(ml),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.LastUpdated != "" {
		parsed, err := time.Parse(time.RFC3339, aux.LastUpdated)
		if err != nil {
			return fmt.Errorf("invalid last_updated format: %w", err)
		}
		ml.LastUpdated = parsed
	}

	return nil
}
