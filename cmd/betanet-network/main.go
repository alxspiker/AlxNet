package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"betanet/internal/network"

	"go.uber.org/zap"
)

var (
	command = flag.String("command", "status", "Command: status, peers, discover, health, refresh")
	peerID  = flag.String("peer", "", "Peer ID for peer-specific commands")
	limit   = flag.Int("limit", 10, "Maximum number of peers to return")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Setup logging
	var logger *zap.Logger
	var err error
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create network manager
	nm, err := network.NewNetworkManager(nil, logger)
	if err != nil {
		logger.Fatal("Failed to create network manager", zap.Error(err))
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	// Start network manager
	if err := nm.Start(ctx); err != nil {
		logger.Fatal("Failed to start network manager", zap.Error(err))
	}
	defer nm.Stop()

	// Wait for services to initialize
	time.Sleep(2 * time.Second)

	// Execute command
	switch *command {
	case "status":
		showStatus(nm, logger)
	case "peers":
		showPeers(nm, logger, *limit)
	case "discover":
		discoverPeers(nm, logger)
	case "health":
		showHealth(nm, logger)
	case "refresh":
		refreshNetwork(nm, logger, ctx)
	default:
		logger.Fatal("Unknown command", zap.String("command", *command))
	}
}

func showStatus(nm *network.NetworkManager, logger *zap.Logger) {
	status := nm.GetNetworkStatus()

	fmt.Println("🌐 Betanet Network Status")
	fmt.Println("==========================")
	fmt.Printf("Status: %s\n", getStatusString(status["is_running"].(bool)))
	fmt.Printf("Uptime: %s\n", status["uptime"])
	fmt.Printf("Active Peers: %d\n", status["active_peers"])
	fmt.Printf("Total Peers: %d\n", status["total_peers"])
}

func showPeers(nm *network.NetworkManager, logger *zap.Logger, limit int) {
	peers, err := nm.GetBestPeers(limit)
	if err != nil {
		logger.Error("Failed to get peers", zap.Error(err))
		return
	}

	fmt.Printf("🔗 Top %d Peers\n", len(peers))
	fmt.Println("==================")

	for i, peer := range peers {
		fmt.Printf("%d. %s (Score: %.3f)\n", i+1, peer.ID[:8]+"...", peer.Score)
		fmt.Printf("   Address: %s\n", peer.Address)
		fmt.Printf("   Location: %s\n", peer.Location)
		if i < len(peers)-1 {
			fmt.Println()
		}
	}
}

func discoverPeers(nm *network.NetworkManager, logger *zap.Logger) {
	fmt.Println("🔍 Discovering Peers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := nm.RefreshNetwork(ctx); err != nil {
		logger.Error("Failed to refresh network", zap.Error(err))
		return
	}

	peers, err := nm.GetBestPeers(20)
	if err != nil {
		logger.Error("Failed to get peers after discovery", zap.Error(err))
		return
	}

	fmt.Printf("✅ Discovery completed! Found %d peers\n", len(peers))
}

func showHealth(nm *network.NetworkManager, logger *zap.Logger) {
	health := nm.GetNetworkHealth()
	if health == nil {
		fmt.Println("❌ Network health data not available")
		return
	}

	fmt.Println("💚 Network Health Report")
	fmt.Println("=========================")
	fmt.Printf("Uptime: %.1f%%\n", health.Uptime*100)
	fmt.Printf("Average Latency: %dms\n", health.LatencyAvg)
	fmt.Printf("Average Bandwidth: %d Mbps\n", health.BandwidthAvg)
}

func refreshNetwork(nm *network.NetworkManager, logger *zap.Logger, ctx context.Context) {
	fmt.Println("🔄 Refreshing Network...")

	if err := nm.RefreshNetwork(ctx); err != nil {
		logger.Error("Failed to refresh network", zap.Error(err))
		return
	}

	fmt.Println("✅ Network refresh completed successfully")
}

func getStatusString(isRunning bool) string {
	if isRunning {
		return "🟢 Running"
	}
	return "🔴 Stopped"
}
