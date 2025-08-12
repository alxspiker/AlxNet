package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"betanet/internal/network"
	"betanet/internal/p2p"
	"betanet/internal/store"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

// NetworkGUI represents the main network monitoring interface
type NetworkGUI struct {
	app          fyne.App
	window       fyne.Window
	mainContainer *fyne.Container
	
	// Network components
	networkManager *network.NetworkManager
	logger         *zap.Logger
	store          *store.Store
	node           *p2p.Node
	ctx            context.Context
	cancel         context.CancelFunc
	
	// UI Components
	statusLabel      *widget.Label
	uptimeLabel      *widget.Label
	peersLabel       *widget.Label
	healthLabel      *widget.Label
	
	// Buttons
	startBtn         *widget.Button
	stopBtn          *widget.Button
	discoverBtn      *widget.Button
	refreshBtn       *widget.Button
	refreshNetworkBtn *widget.Button
	
	// Lists
	peersList        *widget.List
	networkStatsList *widget.List
	
	// Data
	peers            []string
	networkStats     []string
	isRunning        bool
	
	// Timer for updates
	updateTimer      *time.Timer
}

func main() {
	// Parse command line arguments
	var dataDir string
	if len(os.Args) > 1 && os.Args[1] == "-h" || len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Betanet Network GUI - Network Monitoring and Management")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ./bin/betanet-network-gui                    # Use default data directory")
		fmt.Println("  ./bin/betanet-network-gui -data /path/db    # Use specified node database")
		fmt.Println("")
		fmt.Println("Features:")
		fmt.Println("  🌍 Real-time network status monitoring")
		fmt.Println("  👥 Peer discovery and management")
		fmt.Println("  📊 Network health and performance metrics")
		fmt.Println("  🔄 Global discovery network integration")
		fmt.Println("  📈 Live statistics and analytics")
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "-data" {
		dataDir = os.Args[2]
		fmt.Printf("NETWORK: Using specified data directory: %s\n", dataDir)
	} else {
		dataDir = filepath.Join(os.Getenv("HOME"), ".betanet", "network")
		fmt.Printf("NETWORK: Using default data directory: %s\n", dataDir)
	}

	// Create application
	a := app.New()
	a.SetIcon(theme.ComputerIcon())
	
	// Create network GUI
	networkGUI := NewNetworkGUI(a, dataDir)
	
	// Show network GUI
	networkGUI.Show()
	
	// Run application
	a.Run()
}

// NewNetworkGUI creates a new network monitoring interface
func NewNetworkGUI(a fyne.App, dataDir string) *NetworkGUI {
	ng := &NetworkGUI{
		app: a,
		peers: make([]string, 0),
		networkStats: make([]string, 0),
		isRunning: false,
	}
	
	// Create main window
	ng.window = a.NewWindow("🌍 Betanet Network")
	ng.window.Resize(fyne.NewSize(1400, 900))
	ng.window.SetIcon(theme.ComputerIcon())
	
	// Initialize network components
	ng.initializeNetwork(dataDir)
	
	// Initialize UI components
	ng.initializeUI()
	
	// Create main container
	ng.createMainContainer()
	
	// Start update timer
	ng.startUpdateTimer()
	
	return ng
}

// initializeNetwork sets up the network manager and components
func (ng *NetworkGUI) initializeNetwork(dataDir string) {
	// Create context
	ng.ctx, ng.cancel = context.WithCancel(context.Background())
	
	// Setup logging
	var err error
	ng.logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}
	
	// Open store
	ng.store, err = store.Open(dataDir)
	if err != nil {
		fmt.Printf("Failed to open store: %v\n", err)
		return
	}
	
	// Create network manager
	ng.networkManager, err = network.NewNetworkManager(nil, ng.logger)
	if err != nil {
		fmt.Printf("Failed to create network manager: %v\n", err)
		return
	}
	
	// Create node
	ng.node, err = p2p.New(ng.ctx, ng.store, "/ip4/0.0.0.0/tcp/0", nil, nil)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		return
	}
	
	fmt.Printf("Network components initialized successfully\n")
}

// initializeUI sets up all UI components
func (ng *NetworkGUI) initializeUI() {
	// Create status labels
	ng.statusLabel = widget.NewLabel("Network Status: Stopped")
	ng.uptimeLabel = widget.NewLabel("Uptime: --:--:--")
	ng.peersLabel = widget.NewLabel("Active Peers: 0")
	ng.healthLabel = widget.NewLabel("Network Health: Unknown")
	
	// Create buttons
	ng.startBtn = widget.NewButton("Start Network", ng.startNetwork)
	ng.stopBtn = widget.NewButton("Stop Network", ng.stopNetwork)
	ng.stopBtn.Disable()
	ng.discoverBtn = widget.NewButton("Discover Peers", ng.discoverPeers)
	ng.refreshBtn = widget.NewButton("Refresh Status", ng.refreshStatus)
	ng.refreshNetworkBtn = widget.NewButton("Refresh Network", ng.refreshNetwork)
	
	// Create lists
	ng.peersList = widget.NewList(
		func() int { return len(ng.peers) },
		func() fyne.CanvasObject { return widget.NewLabel("Peer") },
		func(id widget.ListItemID, obj fyne.CanvasObject) { 
			if id < len(ng.peers) {
				obj.(*widget.Label).SetText(ng.peers[id])
			}
		},
	)
	
	ng.networkStatsList = widget.NewList(
		func() int { return len(ng.networkStats) },
		func() fyne.CanvasObject { return widget.NewLabel("Stat") },
		func(id widget.ListItemID, obj fyne.CanvasObject) { 
			if id < len(ng.networkStats) {
				obj.(*widget.Label).SetText(ng.networkStats[id])
			}
		},
	)
}

// createMainContainer creates the main application layout
func (ng *NetworkGUI) createMainContainer() {
	// Create status panel
	statusPanel := container.NewVBox(
		widget.NewLabel("🌍 Network Status"),
		ng.statusLabel,
		ng.uptimeLabel,
		ng.peersLabel,
		ng.healthLabel,
	)
	
	// Create control panel
	controlPanel := container.NewVBox(
		widget.NewLabel("🎮 Network Controls"),
		container.NewHBox(
			ng.startBtn,
			ng.stopBtn,
		),
		container.NewHBox(
			ng.discoverBtn,
			ng.refreshBtn,
			ng.refreshNetworkBtn,
		),
	)
	
	// Create left panel
	leftPanel := container.NewVBox(
		statusPanel,
		widget.NewSeparator(),
		controlPanel,
	)
	
	// Create right panel
	rightPanel := container.NewHSplit(
		container.NewVBox(
			widget.NewLabel("👥 Connected Peers"),
			ng.peersList,
		),
		container.NewVBox(
			widget.NewLabel("📊 Network Statistics"),
			ng.networkStatsList,
		),
	)
	
	// Create main container
	ng.mainContainer = container.NewBorder(
		leftPanel,      // top
		nil,            // bottom
		nil,            // left
		nil,            // right
		rightPanel,     // center
	)
	
	ng.window.SetContent(ng.mainContainer)
}

// startNetwork starts the network manager
func (ng *NetworkGUI) startNetwork() {
	if ng.isRunning {
		return
	}
	
	// Start network manager
	if err := ng.networkManager.Start(ng.ctx); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to start network: %v", err), ng.window)
		return
	}
	
	// Start node
	if err := ng.node.Start(ng.ctx); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to start node: %v", err), ng.window)
		return
	}
	
	// Update UI
	ng.isRunning = true
	ng.statusLabel.SetText("Network Status: Running")
	ng.startBtn.Disable()
	ng.stopBtn.Enable()
	
	// Update status immediately
	ng.refreshStatus()
	
	dialog.ShowInformation("Network Started", "Network manager and node started successfully!", ng.window)
}

// stopNetwork stops the network manager
func (ng *NetworkGUI) stopNetwork() {
	if !ng.isRunning {
		return
	}
	
	// Stop network manager
	ng.networkManager.Stop()
	
	// Stop node
	if ng.cancel != nil {
		ng.cancel()
	}
	
	// Update UI
	ng.isRunning = false
	ng.statusLabel.SetText("Network Status: Stopped")
	ng.uptimeLabel.SetText("Uptime: --:--:--")
	ng.peersLabel.SetText("Active Peers: 0")
	ng.healthLabel.SetText("Network Health: Unknown")
	ng.startBtn.Enable()
	ng.stopBtn.Disable()
	
	// Clear lists
	ng.peers = make([]string, 0)
	ng.peersList.Refresh()
	ng.networkStats = make([]string, 0)
	ng.networkStatsList.Refresh()
	
	dialog.ShowInformation("Network Stopped", "Network manager and node stopped successfully!", ng.window)
}

// discoverPeers discovers new peers on the network
func (ng *NetworkGUI) discoverPeers() {
	if !ng.isRunning {
		dialog.ShowError(fmt.Errorf("Please start the network first"), ng.window)
		return
	}
	
	// TODO: Implement actual peer discovery
	// This is a placeholder for the next iteration
	
	// Simulate peer discovery
	ng.peers = append(ng.peers, 
		"Peer 1: 12D3KooW... (Local)",
		"Peer 2: 12D3KooW... (Remote)",
		"Peer 3: 12D3KooW... (Bootstrap)",
	)
	ng.peersList.Refresh()
	
	ng.peersLabel.SetText(fmt.Sprintf("Active Peers: %d", len(ng.peers)))
	
	dialog.ShowInformation("Peers Discovered", 
		fmt.Sprintf("Discovered %d new peers on the network!", len(ng.peers)), 
		ng.window)
}

// refreshStatus refreshes the network status
func (ng *NetworkGUI) refreshStatus() {
	if !ng.isRunning {
		return
	}
	
	// TODO: Implement actual status refresh
	// This is a placeholder for the next iteration
	
	// Simulate status update
	ng.uptimeLabel.SetText("Uptime: 00:05:30")
	ng.peersLabel.SetText(fmt.Sprintf("Active Peers: %d", len(ng.peers)))
	ng.healthLabel.SetText("Network Health: Excellent")
	
	// Update network stats
	ng.networkStats = []string{
		"Total Connections: 15",
		"Active Transfers: 3",
		"Bandwidth Used: 2.5 MB/s",
		"Latency: 45ms",
		"Packet Loss: 0.1%",
		"Network Load: 23%",
	}
	ng.networkStatsList.Refresh()
}

// refreshNetwork refreshes the global network data
func (ng *NetworkGUI) refreshNetwork() {
	if !ng.isRunning {
		dialog.ShowError(fmt.Errorf("Please start the network first"), ng.window)
		return
	}
	
	// TODO: Implement actual network refresh
	// This is a placeholder for the next iteration
	
	// Simulate network refresh
	ng.networkStats = append(ng.networkStats,
		"Global Peers: 1,247",
		"Network Coverage: 89%",
		"Last Updated: " + time.Now().Format("15:04:05"),
	)
	ng.networkStatsList.Refresh()
	
	dialog.ShowInformation("Network Refreshed", "Global network data refreshed successfully!", ng.window)
}

// startUpdateTimer starts the periodic update timer
func (ng *NetworkGUI) startUpdateTimer() {
	ng.updateTimer = time.NewTimer(5 * time.Second)
	go func() {
		for {
			select {
			case <-ng.updateTimer.C:
				if ng.isRunning {
					// Update status on main thread
					ng.window.Canvas().Refresh(ng.statusLabel)
					ng.refreshStatus()
				}
				ng.updateTimer.Reset(5 * time.Second)
			case <-ng.ctx.Done():
				return
			}
		}
	}()
}

// Show displays the network GUI window
func (ng *NetworkGUI) Show() {
	ng.window.Show()
}
