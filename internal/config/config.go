package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the complete application configuration
type Config struct {
	// Global settings
	Environment string `yaml:"environment"` // "development", "staging", "production"
	LogLevel    string `yaml:"log_level"`   // "debug", "info", "warn", "error"

	// Network configuration
	Network NetworkConfig `yaml:"network"`

	// Security configuration
	Security SecurityConfig `yaml:"security"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`

	// Wallet configuration
	Wallet WalletConfig `yaml:"wallet"`

	// Node configuration
	Node NodeConfig `yaml:"node"`
}

// NetworkConfig holds network-related settings
type NetworkConfig struct {
	ListenAddr     string        `yaml:"listen_addr"`
	BootstrapPeers []string      `yaml:"bootstrap_peers"`
	MaxPeers       int           `yaml:"max_peers"`
	PeerTimeout    time.Duration `yaml:"peer_timeout"`
	EnableMDNS     bool          `yaml:"enable_mdns"`
	EnableNAT      bool          `yaml:"enable_nat"`
	EnableRelay    bool          `yaml:"enable_relay"`
}

// SecurityConfig holds security-related settings
type SecurityConfig struct {
	MaxContentSize          int64         `yaml:"max_content_size"`
	MaxFileCount            int           `yaml:"max_file_count"`
	MaxPathLength           int           `yaml:"max_path_length"`
	RateLimit               int           `yaml:"rate_limit"`
	BanDuration             time.Duration `yaml:"ban_duration"`
	EnablePeerValidation    bool          `yaml:"enable_peer_validation"`
	EnableRateLimiting      bool          `yaml:"enable_rate_limiting"`
	RequireStrongPassphrase bool          `yaml:"require_strong_passphrase"`
	MaxLoginAttempts        int           `yaml:"max_login_attempts"`
	SessionTimeout          time.Duration `yaml:"session_timeout"`
}

// StorageConfig holds storage-related settings
type StorageConfig struct {
	DataDir           string        `yaml:"data_dir"`
	MaxFileSize       int64         `yaml:"max_file_size"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	EnableCompression bool          `yaml:"enable_compression"`
	EnableEncryption  bool          `yaml:"enable_encryption"`
}

// WalletConfig holds wallet-related settings
type WalletConfig struct {
	DefaultPath       string        `yaml:"default_path"`
	BackupInterval    time.Duration `yaml:"backup_interval"`
	MaxSitesPerWallet int           `yaml:"max_sites_per_wallet"`
	EnableAutoBackup  bool          `yaml:"enable_auto_backup"`
	BackupRetention   int           `yaml:"backup_retention"`
}

// NodeConfig holds node-specific settings
type NodeConfig struct {
	EnableMetrics     bool  `yaml:"enable_metrics"`
	MetricsPort       int   `yaml:"metrics_port"`
	EnableProfiling   bool  `yaml:"enable_profiling"`
	ProfilingPort     int   `yaml:"profiling_port"`
	MaxMemoryUsage    int64 `yaml:"max_memory_usage"`
	GarbageCollection bool  `yaml:"garbage_collection"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Environment: "development",
		LogLevel:    "info",

		Network: NetworkConfig{
			ListenAddr:     "/ip4/0.0.0.0/tcp/4001",
			BootstrapPeers: []string{},
			MaxPeers:       100,
			PeerTimeout:    30 * time.Second,
			EnableMDNS:     true,
			EnableNAT:      true,
			EnableRelay:    false,
		},

		Security: SecurityConfig{
			MaxContentSize:          10 * 1024 * 1024, // 10MB
			MaxFileCount:            1000,
			MaxPathLength:           255,
			RateLimit:               100,
			BanDuration:             15 * time.Minute,
			EnablePeerValidation:    true,
			EnableRateLimiting:      true,
			RequireStrongPassphrase: true,
			MaxLoginAttempts:        5,
			SessionTimeout:          24 * time.Hour,
		},

		Storage: StorageConfig{
			DataDir:           "./data",
			MaxFileSize:       100 * 1024 * 1024, // 100MB
			CleanupInterval:   5 * time.Minute,
			MaxRetries:        3,
			RetryDelay:        100 * time.Millisecond,
			EnableCompression: true,
			EnableEncryption:  true,
		},

		Wallet: WalletConfig{
			DefaultPath:       "./wallets",
			BackupInterval:    24 * time.Hour,
			MaxSitesPerWallet: 1000,
			EnableAutoBackup:  true,
			BackupRetention:   30,
		},

		Node: NodeConfig{
			EnableMetrics:     true,
			MetricsPort:       9090,
			EnableProfiling:   false,
			ProfilingPort:     6060,
			MaxMemoryUsage:    100 * 1024 * 1024, // 100MB
			GarbageCollection: true,
		},
	}
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() *Config {
	config := DefaultConfig()

	// Override with environment variables
	if env := os.Getenv("BETANET_ENV"); env != "" {
		config.Environment = env
	}

	if level := os.Getenv("BETANET_LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}

	if addr := os.Getenv("BETANET_LISTEN_ADDR"); addr != "" {
		config.Network.ListenAddr = addr
	}

	if peers := os.Getenv("BETANET_BOOTSTRAP_PEERS"); peers != "" {
		config.Network.BootstrapPeers = strings.Split(peers, ",")
	}

	if maxPeers := os.Getenv("BETANET_MAX_PEERS"); maxPeers != "" {
		if val, err := strconv.Atoi(maxPeers); err == nil {
			config.Network.MaxPeers = val
		}
	}

	if maxContent := os.Getenv("BETANET_MAX_CONTENT_SIZE"); maxContent != "" {
		if val, err := strconv.ParseInt(maxContent, 10, 64); err == nil {
			config.Security.MaxContentSize = val
		}
	}

	if dataDir := os.Getenv("BETANET_DATA_DIR"); dataDir != "" {
		config.Storage.DataDir = dataDir
	}

	return config
}

// SaveToFile saves configuration to a YAML file
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	// Validate environment
	validEnvs := []string{"development", "staging", "production"}
	valid := false
	for _, env := range validEnvs {
		if c.Environment == env {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid environment: %s (must be one of: %v)", c.Environment, validEnvs)
	}

	// Validate log level
	validLevels := []string{"debug", "info", "warn", "error"}
	valid = false
	for _, level := range validLevels {
		if c.LogLevel == level {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid log level: %s (must be one of: %v)", c.LogLevel, validLevels)
	}

	// Validate network configuration
	if err := c.Network.Validate(); err != nil {
		return fmt.Errorf("network config: %w", err)
	}

	// Validate security configuration
	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security config: %w", err)
	}

	// Validate storage configuration
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage config: %w", err)
	}

	// Validate wallet configuration
	if err := c.Wallet.Validate(); err != nil {
		return fmt.Errorf("wallet config: %w", err)
	}

	// Validate node configuration
	if err := c.Node.Validate(); err != nil {
		return fmt.Errorf("node config: %w", err)
	}

	return nil
}

// Validate performs validation of NetworkConfig
func (nc *NetworkConfig) Validate() error {
	if nc.MaxPeers <= 0 {
		return errors.New("max_peers must be positive")
	}
	if nc.MaxPeers > 1000 {
		return errors.New("max_peers too high (max 1000)")
	}
	if nc.PeerTimeout <= 0 {
		return errors.New("peer_timeout must be positive")
	}
	if nc.PeerTimeout > 5*time.Minute {
		return errors.New("peer_timeout too high (max 5 minutes)")
	}
	return nil
}

// Validate performs validation of SecurityConfig
func (sc *SecurityConfig) Validate() error {
	if sc.MaxContentSize <= 0 {
		return errors.New("max_content_size must be positive")
	}
	if sc.MaxContentSize > 1<<30 { // 1GB
		return errors.New("max_content_size too high (max 1GB)")
	}
	if sc.MaxFileCount <= 0 {
		return errors.New("max_file_count must be positive")
	}
	if sc.MaxFileCount > 10000 {
		return errors.New("max_file_count too high (max 10000)")
	}
	if sc.MaxPathLength <= 0 {
		return errors.New("max_path_length must be positive")
	}
	if sc.MaxPathLength > 1024 {
		return errors.New("max_path_length too high (max 1024)")
	}
	if sc.RateLimit <= 0 {
		return errors.New("rate_limit must be positive")
	}
	if sc.RateLimit > 10000 {
		return errors.New("rate_limit too high (max 10000)")
	}
	if sc.BanDuration <= 0 {
		return errors.New("ban_duration must be positive")
	}
	if sc.BanDuration > 24*time.Hour {
		return errors.New("ban_duration too high (max 24 hours)")
	}
	if sc.MaxLoginAttempts <= 0 {
		return errors.New("max_login_attempts must be positive")
	}
	if sc.MaxLoginAttempts > 100 {
		return errors.New("max_login_attempts too high (max 100)")
	}
	if sc.SessionTimeout <= 0 {
		return errors.New("session_timeout must be positive")
	}
	if sc.SessionTimeout > 30*24*time.Hour { // 30 days
		return errors.New("session_timeout too high (max 30 days)")
	}
	return nil
}

// Validate performs validation of StorageConfig
func (sc *StorageConfig) Validate() error {
	if sc.DataDir == "" {
		return errors.New("data_dir cannot be empty")
	}
	if sc.MaxFileSize <= 0 {
		return errors.New("max_file_size must be positive")
	}
	if sc.MaxFileSize > 1<<30 { // 1GB
		return errors.New("max_file_size too high (max 1GB)")
	}
	if sc.CleanupInterval <= 0 {
		return errors.New("cleanup_interval must be positive")
	}
	if sc.CleanupInterval > 24*time.Hour {
		return errors.New("cleanup_interval too high (max 24 hours)")
	}
	if sc.MaxRetries < 0 {
		return errors.New("max_retries cannot be negative")
	}
	if sc.MaxRetries > 100 {
		return errors.New("max_retries too high (max 100)")
	}
	if sc.RetryDelay < 0 {
		return errors.New("retry_delay cannot be negative")
	}
	if sc.RetryDelay > 10*time.Second {
		return errors.New("retry_delay too high (max 10 seconds)")
	}
	return nil
}

// Validate performs validation of WalletConfig
func (wc *WalletConfig) Validate() error {
	if wc.DefaultPath == "" {
		return errors.New("default_path cannot be empty")
	}
	if wc.BackupInterval <= 0 {
		return errors.New("backup_interval must be positive")
	}
	if wc.BackupInterval > 7*24*time.Hour { // 1 week
		return errors.New("backup_interval too high (max 1 week)")
	}
	if wc.MaxSitesPerWallet <= 0 {
		return errors.New("max_sites_per_wallet must be positive")
	}
	if wc.MaxSitesPerWallet > 10000 {
		return errors.New("max_sites_per_wallet too high (max 10000)")
	}
	if wc.BackupRetention < 0 {
		return errors.New("backup_retention cannot be negative")
	}
	if wc.BackupRetention > 365 {
		return errors.New("backup_retention too high (max 365)")
	}
	return nil
}

// Validate performs validation of NodeConfig
func (nc *NodeConfig) Validate() error {
	if nc.MetricsPort < 0 || nc.MetricsPort > 65535 {
		return errors.New("metrics_port must be between 0 and 65535")
	}
	if nc.ProfilingPort < 0 || nc.ProfilingPort > 65535 {
		return errors.New("profiling_port must be between 0 and 65535")
	}
	if nc.MaxMemoryUsage <= 0 {
		return errors.New("max_memory_usage must be positive")
	}
	if nc.MaxMemoryUsage > 1<<40 { // 1TB
		return errors.New("max_memory_usage too high (max 1TB)")
	}
	return nil
}

// GetString returns a string configuration value with fallback
func (c *Config) GetString(key string, fallback string) string {
	// This is a simplified implementation
	// In practice, you'd want to use reflection or a more sophisticated approach
	switch key {
	case "environment":
		return c.Environment
	case "log_level":
		return c.LogLevel
	case "listen_addr":
		return c.Network.ListenAddr
	case "data_dir":
		return c.Storage.DataDir
	default:
		return fallback
	}
}

// GetInt returns an integer configuration value with fallback
func (c *Config) GetInt(key string, fallback int) int {
	switch key {
	case "max_peers":
		return c.Network.MaxPeers
	case "max_content_size":
		return int(c.Security.MaxContentSize)
	case "max_file_count":
		return c.Security.MaxFileCount
	case "rate_limit":
		return c.Security.RateLimit
	case "max_retries":
		return c.Storage.MaxRetries
	case "max_sites_per_wallet":
		return c.Wallet.MaxSitesPerWallet
	case "metrics_port":
		return c.Node.MetricsPort
	case "profiling_port":
		return c.Node.ProfilingPort
	default:
		return fallback
	}
}

// GetBool returns a boolean configuration value with fallback
func (c *Config) GetBool(key string, fallback bool) bool {
	switch key {
	case "enable_mdns":
		return c.Network.EnableMDNS
	case "enable_nat":
		return c.Network.EnableNAT
	case "enable_peer_validation":
		return c.Security.EnablePeerValidation
	case "enable_rate_limiting":
		return c.Security.EnableRateLimiting
	case "enable_compression":
		return c.Storage.EnableCompression
	case "enable_encryption":
		return c.Storage.EnableEncryption
	case "enable_auto_backup":
		return c.Wallet.EnableAutoBackup
	case "enable_metrics":
		return c.Node.EnableMetrics
	case "enable_profiling":
		return c.Node.EnableProfiling
	case "garbage_collection":
		return c.Node.GarbageCollection
	default:
		return fallback
	}
}

// GetDuration returns a duration configuration value with fallback
func (c *Config) GetDuration(key string, fallback time.Duration) time.Duration {
	switch key {
	case "peer_timeout":
		return c.Network.PeerTimeout
	case "ban_duration":
		return c.Security.BanDuration
	case "session_timeout":
		return c.Security.SessionTimeout
	case "cleanup_interval":
		return c.Storage.CleanupInterval
	case "retry_delay":
		return c.Storage.RetryDelay
	case "backup_interval":
		return c.Wallet.BackupInterval
	default:
		return fallback
	}
}
