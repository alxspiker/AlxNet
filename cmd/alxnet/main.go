package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"alxnet/internal/p2p"
	"alxnet/internal/store"
	"alxnet/internal/webserver"

	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "start", "run":
		cmdStart()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("AlxNet - Complete Decentralized Web Platform")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  start    Start the complete AlxNet platform")
	fmt.Println("  run      Alias for start")
	fmt.Println("")
	fmt.Println("Options for start:")
	fmt.Println("  -data ./data            Data directory (default: ./data)")
	fmt.Println("  -node-port 4001         P2P node port (default: auto)")
	fmt.Println("  -browser-port 8080      Browser web interface port")
	fmt.Println("  -wallet-port 8081       Wallet management web interface port")
	fmt.Println("  -node-ui-port 8082      Node management web interface port")
	fmt.Println("  -bootstrap ADDR         Bootstrap node address")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  alxnet start                                    # Start with all defaults")
	fmt.Println("  alxnet start -node-port 4001                   # Specify P2P port")
	fmt.Println("  alxnet start -browser-port 8080 -wallet-port 8081")
	fmt.Println("")
	fmt.Println("After starting, access:")
	fmt.Println("  Browser Interface:      http://localhost:8080")
	fmt.Println("  Wallet Management:      http://localhost:8081")
	fmt.Println("  Node Management:        http://localhost:8082")
}

func cmdStart() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "data directory")
	nodePort := fs.String("node-port", "0", "P2P node port (0 = auto)")
	browserPort := fs.String("browser-port", "8080", "Browser web interface port")
	walletPort := fs.String("wallet-port", "8081", "Wallet management web interface port")
	nodeUIPort := fs.String("node-ui-port", "8082", "Node management web interface port")
	bootstrap := fs.String("bootstrap", "", "bootstrap node multiaddr")
	_ = fs.Parse(os.Args[2:])

	// Setup logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

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

	node, err := p2p.New(ctx, db, listenAddr, bootstrapPeers, nil)
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

	// Start all web servers

	// 1. Browser Interface (existing functionality)
	browserServer := webserver.NewBrowserServer(db, node, logger, mustParseInt(*browserPort))
	if err := browserServer.Start(); err != nil {
		log.Fatalf("Failed to start browser server: %v", err)
	}
	defer func() {
		if err := browserServer.Stop(); err != nil {
			logger.Error("Failed to stop browser server", zap.Error(err))
		}
	}()

	// 2. Wallet Management Interface (new)
	walletServer := webserver.NewWalletServer(db, node, logger, mustParseInt(*walletPort))
	if err := walletServer.Start(); err != nil {
		log.Fatalf("Failed to start wallet server: %v", err)
	}
	defer func() {
		if err := walletServer.Stop(); err != nil {
			logger.Error("Failed to stop wallet server", zap.Error(err))
		}
	}()

	// 3. Node Management Interface (new)
	nodeUIServer := webserver.NewNodeServer(db, node, logger, mustParseInt(*nodeUIPort))
	if err := nodeUIServer.Start(); err != nil {
		log.Fatalf("Failed to start node UI server: %v", err)
	}
	defer func() {
		if err := nodeUIServer.Stop(); err != nil {
			logger.Error("Failed to stop node UI server", zap.Error(err))
		}
	}()

	logger.Info("All AlxNet services started successfully!",
		zap.String("browser_port", *browserPort),
		zap.String("wallet_port", *walletPort),
		zap.String("node_ui_port", *nodeUIPort),
		zap.String("node_port", actualNodePort),
		zap.String("data_dir", *dataDir))

	fmt.Println("")
	fmt.Println("🚀 AlxNet Platform is running!")
	fmt.Println("=====================================")
	fmt.Printf("   🌐 Browser Interface:      http://localhost:%s\n", *browserPort)
	fmt.Printf("   💰 Wallet Management:      http://localhost:%s\n", *walletPort)
	fmt.Printf("   🔗 Node Management:        http://localhost:%s\n", *nodeUIPort)
	fmt.Printf("   📡 P2P Node Port:          %s\n", actualNodePort)
	fmt.Printf("   📂 Data Directory:         %s\n", *dataDir)
	fmt.Println("=====================================")
	fmt.Println("")
	fmt.Println("   Open your web browser and navigate to any of the URLs above")
	fmt.Println("   Press Ctrl+C to stop all services")
	fmt.Println("")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down AlxNet Platform...")
	fmt.Println("\nShutting down all services...")
}

func mustParseInt(s string) int {
	var port int
	if _, err := fmt.Sscanf(s, "%d", &port); err != nil {
		log.Fatalf("Invalid port number: %s", s)
	}
	return port
}
