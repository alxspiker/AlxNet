package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"betanet/internal/p2p"
	"betanet/internal/store"

	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "start", "run":
		startNode()
	case "status":
		statusNode()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Betanet Node - Standalone P2P Node")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start    Start the P2P node")
	fmt.Println("  run      Alias for start")
	fmt.Println("  status   Check node status")
	fmt.Println("")
	fmt.Println("Options for start:")
	fmt.Println("  -port 4001              P2P node port (default: 4001)")
	fmt.Println("  -data ./data            Data directory (default: ./data)")
	fmt.Println("  -bootstrap ADDR         Bootstrap node address")
	fmt.Println("  -verbose                Enable verbose logging")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  betanet-node start                                    # Start with defaults")
	fmt.Println("  betanet-node start -port 4001 -data ./node-data")
	fmt.Println("  betanet-node start -bootstrap /ip4/192.168.1.1/tcp/4001/p2p/...")
}

func startNode() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.String("port", "4001", "P2P node port")
	dataDir := fs.String("data", "./data", "data directory")
	bootstrap := fs.String("bootstrap", "", "bootstrap node multiaddr")
	verbose := fs.Bool("verbose", false, "enable verbose logging")
	_ = fs.Parse(os.Args[2:])

	// Parse port
	portInt, err := strconv.Atoi(*port)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	// Setup logging
	var logger *zap.Logger
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Open database
	db, err := store.Open(*dataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create listen address
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", *port)

	var bootstrapPeers []string
	if *bootstrap != "" {
		bootstrapPeers = strings.Split(*bootstrap, ",")
	}

	// Create and start P2P node
	node, err := p2p.New(ctx, db, listenAddr, bootstrapPeers, nil)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	if err := node.Start(ctx); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	defer node.Host.Close()

	logger.Info("P2P node started",
		zap.String("id", node.Host.ID().String()),
		zap.Int("port", portInt))

	fmt.Println("")
	fmt.Println("🔗 Betanet Node is running!")
	fmt.Printf("   Node ID: %s\n", node.Host.ID().String())
	fmt.Printf("   Port: %d\n", portInt)
	fmt.Printf("   Data Directory: %s\n", *dataDir)
	if len(bootstrapPeers) > 0 {
		fmt.Printf("   Bootstrap Peers: %d\n", len(bootstrapPeers))
	}
	fmt.Println("")
	fmt.Println("   This node can be used by wallet and browser applications")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println("")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down node...")
	fmt.Println("\nShutting down...")
}

func statusNode() {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "data directory")
	_ = fs.Parse(os.Args[2:])

	fmt.Println("Betanet Node Status")
	fmt.Println("==================")

	// Check if data directory exists
	if _, err := os.Stat(*dataDir); os.IsNotExist(err) {
		fmt.Printf("Data Directory: %s (not found)\n", *dataDir)
		fmt.Println("Status: No data found")
		return
	}

	fmt.Printf("Data Directory: %s ✓\n", *dataDir)

	// Try to open database
	db, err := store.Open(*dataDir)
	if err != nil {
		fmt.Printf("Database: Error - %v\n", err)
		return
	}
	defer db.Close()

	fmt.Println("Database: Connected ✓")
	fmt.Println("Status: Ready for P2P operations")
}
