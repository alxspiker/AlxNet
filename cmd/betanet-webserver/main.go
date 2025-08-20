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
	case "serve":
		serveCommand()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("betanet-webserver commands:")
	fmt.Println("  serve -data /path/db -port 8080 [-listen MA] [-bootstrap MA[,MA...]]")
}

func serveCommand() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	data := fs.String("data", "./data", "data directory")
	port := fs.String("port", "8080", "HTTP server port")
	listen := fs.String("listen", "/ip4/0.0.0.0/tcp/0", "libp2p listen multiaddr")
	bootstrap := fs.String("bootstrap", "", "comma-separated peer multiaddrs")
	_ = fs.Parse(os.Args[2:])

	// Parse port
	portInt, err := strconv.Atoi(*port)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	// Setup logging
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Open database
	db, err := store.Open(*data)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create P2P node
	var bootstrapPeers []string
	if *bootstrap != "" {
		bootstrapPeers = []string{*bootstrap}
	}

	node, err := p2p.New(ctx, db, *listen, bootstrapPeers, nil)
	if err != nil {
		log.Fatalf("Failed to create P2P node: %v", err)
	}

	// Start P2P node
	if err := node.Start(ctx); err != nil {
		log.Fatalf("Failed to start P2P node: %v", err)
	}
	defer node.Host.Close()

	logger.Info("P2P node started", zap.String("id", node.Host.ID().String()))

	// Create and start web server
	webServer := webserver.NewWebServer(db, node, logger, portInt)
	if err := webServer.Start(); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
	defer webServer.Stop()

	logger.Info("Betanet web server started",
		zap.Int("port", portInt),
		zap.String("url", fmt.Sprintf("http://localhost:%d", portInt)))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")
}
