package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"betanet/internal/p2p"
	"betanet/internal/store"
	"betanet/internal/webserver"

	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "start", "serve":
		startBrowser()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Betanet Browser - Decentralized Web Browser")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start    Start the decentralized web browser")
	fmt.Println("  serve    Alias for start")
	fmt.Println("")
	fmt.Println("Options for start:")
	fmt.Println("  -port 8080              HTTP server port (default: 8080)")
	fmt.Println("  -data ./data            Data directory (default: ./data)")
	fmt.Println("  -node-port 4001         P2P node port (default: auto)")
	fmt.Println("  -bootstrap ADDR         Bootstrap node address")
	fmt.Println("  -connect-port 4001      Connect to existing node on port")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  betanet-browser start                           # Start with defaults")
	fmt.Println("  betanet-browser start -port 8080 -node-port 4001")
	fmt.Println("  betanet-browser start -connect-port 4002       # Connect to node on port 4002")
}

func startBrowser() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.String("port", "8080", "HTTP server port")
	dataDir := fs.String("data", "./data", "data directory")
	nodePort := fs.String("node-port", "0", "P2P node port (0 = auto)")
	bootstrap := fs.String("bootstrap", "", "bootstrap node multiaddr")
	connectPort := fs.String("connect-port", "", "connect to existing node on this port")
	_ = fs.Parse(os.Args[2:])

	// Parse ports
	httpPort, err := strconv.Atoi(*port)
	if err != nil {
		log.Fatalf("Invalid HTTP port: %v", err)
	}

	// Setup logging
	logger, err := zap.NewDevelopment()
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

	var node *p2p.Node

	// Check if we should connect to existing node or start our own
	if *connectPort != "" {
		logger.Info("Attempting to connect to existing node", zap.String("port", *connectPort))
		// TODO: Implement connection to existing node via IPC or shared state
		// For now, we'll start our own node but log the intention
		logger.Warn("Connecting to existing node not yet implemented, starting new node")
	}

	// Create and start P2P node
	var listenAddr string
	if *nodePort == "0" {
		listenAddr = "/ip4/0.0.0.0/tcp/0" // Auto-assign port
	} else {
		listenAddr = fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", *nodePort)
	}

	var bootstrapPeers []string
	if *bootstrap != "" {
		bootstrapPeers = []string{*bootstrap}
	}

	node, err = p2p.New(ctx, db, listenAddr, bootstrapPeers, nil)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	// Start P2P node
	if err := node.Start(ctx); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	defer node.Host.Close()

	// Get the actual port the node is listening on
	nodeAddrs := node.Host.Addrs()
	var actualNodePort string
	for _, addr := range nodeAddrs {
		if tcp := addr.String(); tcp != "" {
			logger.Info("P2P node listening", zap.String("address", tcp))
			// Extract port from address like /ip4/0.0.0.0/tcp/4001
			if len(addr.Protocols()) > 1 {
				actualNodePort = addr.String()[len(addr.String())-4:]
			}
			break
		}
	}

	logger.Info("P2P node started",
		zap.String("id", node.Host.ID().String()),
		zap.String("port", actualNodePort))

	// Create and start web server
	webServer := webserver.NewWebServer(db, node, logger, httpPort)
	if err := webServer.Start(); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
	defer webServer.Stop()

	logger.Info("Betanet Browser started successfully!",
		zap.Int("http_port", httpPort),
		zap.String("node_port", actualNodePort),
		zap.String("data_dir", *dataDir),
		zap.String("url", fmt.Sprintf("http://localhost:%d", httpPort)))

	fmt.Println("")
	fmt.Println("🌐 Betanet Browser is running!")
	fmt.Printf("   Browser URL: http://localhost:%d\n", httpPort)
	fmt.Printf("   Data Directory: %s\n", *dataDir)
	if actualNodePort != "" {
		fmt.Printf("   P2P Node Port: %s\n", actualNodePort)
	}
	fmt.Println("")
	fmt.Println("   Open your web browser and navigate to the URL above")
	fmt.Println("   Press Ctrl+C to stop")
	fmt.Println("")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Betanet Browser...")
	fmt.Println("\nShutting down...")
}
