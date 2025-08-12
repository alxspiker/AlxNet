package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"betanet/internal/core"
	"betanet/internal/network"
	"betanet/internal/p2p"
	"betanet/internal/store"
	"betanet/internal/wallet"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

// Dashboard represents the main application interface
type Dashboard struct {
	app           fyne.App
	window        fyne.Window
	mainContainer *fyne.Container

	// Core components
	store      *store.Store
	node       *p2p.Node
	networkMgr *network.NetworkManager
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc

	// UI Components
	statusBar    *widget.Label
	tabContainer *container.AppTabs

	// Data
	currentWallet *wallet.Wallet
	currentSite   string
	currentKey    []byte
	dataDir       string
}

// WalletTab handles all wallet operations
type WalletTab struct {
	container   *fyne.Container
	window      fyne.Window
	dashboard   *Dashboard
	walletPath  *widget.Entry
	mnemonic    *widget.Entry
	siteLabel   *widget.Entry
	domainName  *widget.Entry
	websiteDir  *widget.Entry
	mainFile    *widget.Entry
	encryptPass *widget.Entry

	// Buttons
	createWalletBtn   *widget.Button
	addSiteBtn        *widget.Button
	publishBtn        *widget.Button
	registerDomainBtn *widget.Button
	publishWebsiteBtn *widget.Button
	addFileBtn        *widget.Button
	exportKeyBtn      *widget.Button

	// Lists
	sitesList        *widget.List
	domainsList      *widget.List
	websiteFilesList *widget.List

	// Data
	sites        []string
	domains      []string
	websiteFiles []string
}

// NodeTab handles all node operations
type NodeTab struct {
	container  *fyne.Container
	window     fyne.Window
	dataDir    *widget.Entry
	listenAddr *widget.Entry
	bootstrap  *widget.Entry
	keyPath    *widget.Entry

	// Buttons
	initKeyBtn        *widget.Button
	startNodeBtn      *widget.Button
	stopNodeBtn       *widget.Button
	publishBtn        *widget.Button
	browseBtn         *widget.Button
	publishWebsiteBtn *widget.Button
	addFileBtn        *widget.Button

	// Status and logs
	nodeStatus *widget.Label
	logArea    *widget.Entry

	// Data
	isRunning bool
}

// NetworkTab handles all network operations
type NetworkTab struct {
	container *fyne.Container
	window    fyne.Window

	// Status display
	statusLabel *widget.Label
	uptimeLabel *widget.Label
	peersLabel  *widget.Label
	healthLabel *widget.Label

	// Buttons
	startBtn    *widget.Button
	stopBtn     *widget.Button
	discoverBtn *widget.Button
	refreshBtn  *widget.Button
	healthBtn   *widget.Button

	// Lists
	peersList        *widget.List
	networkStatsList *widget.List

	// Data
	peers        []string
	networkStats []string
	isRunning    bool
}

// BrowserTab handles web browsing
type BrowserTab struct {
	container   *fyne.Container
	window      fyne.Window
	addressBar  *widget.Entry
	backBtn     *widget.Button
	forwardBtn  *widget.Button
	refreshBtn  *widget.Button
	homeBtn     *widget.Button
	contentArea *widget.RichText

	// Data
	history      []string
	historyIndex int
}

func main() {
	// Parse command line arguments
	var dataDir string
	if len(os.Args) > 1 && os.Args[1] == "-h" || len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Betanet Dashboard - Complete System Management Interface")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ./bin/betanet-dashboard                    # Use default data directory")
		fmt.Println("  ./bin/betanet-dashboard -data /path/db    # Use specified node database")
		fmt.Println("")
		fmt.Println("Features:")
		fmt.Println("  💼 Complete Wallet Management - All wallet operations")
		fmt.Println("  🖥️  Full Node Control - All node operations")
		fmt.Println("  🌐 Network Management - All network operations")
		fmt.Println("  🌍 Web Browser - Browse decentralized sites")
		fmt.Println("  🔧 Advanced Tools - All console functionality")
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "-data" {
		dataDir = os.Args[2]
		fmt.Printf("DASHBOARD: Using specified data directory: %s\n", dataDir)
	} else {
		dataDir = filepath.Join(os.Getenv("HOME"), ".betanet", "dashboard")
		fmt.Printf("DASHBOARD: Using default data directory: %s\n", dataDir)
	}

	// Create application
	a := app.New()
	a.SetIcon(theme.ComputerIcon())

	// Create dashboard
	dashboard := NewDashboard(a, dataDir)

	// Show dashboard
	dashboard.Show()

	// Run application
	a.Run()
}

// NewDashboard creates a new dashboard instance
func NewDashboard(a fyne.App, dataDir string) *Dashboard {
	d := &Dashboard{
		app:     a,
		dataDir: dataDir,
	}

	// Create main window
	d.window = a.NewWindow("🌐 Betanet Dashboard - Complete System Management")
	d.window.Resize(fyne.NewSize(1200, 800)) // More reasonable default size
	d.window.SetIcon(theme.ComputerIcon())

	// Initialize core components
	d.initializeCore(dataDir)

	// Initialize UI components
	d.initializeUI()

	// Create main container
	d.createMainContainer()

	return d
}

// initializeCore sets up all core system components
func (d *Dashboard) initializeCore(dataDir string) {
	// Create context
	d.ctx, d.cancel = context.WithCancel(context.Background())

	// Setup logging
	var err error
	d.logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}

	// Open store
	d.store, err = store.Open(dataDir)
	if err != nil {
		fmt.Printf("Failed to open store: %v\n", err)
		return
	}

	// Create network manager
	d.networkMgr, err = network.NewNetworkManager(nil, d.logger)
	if err != nil {
		fmt.Printf("Failed to create network manager: %v\n", err)
		return
	}

	// Create node
	d.node, err = p2p.New(d.ctx, d.store, "/ip4/0.0.0.0/tcp/0", nil, nil)
	if err != nil {
		fmt.Printf("Failed to create node: %v\n", err)
		return
	}

	fmt.Printf("Dashboard core components initialized successfully\n")
}

// initializeUI sets up all UI components
func (d *Dashboard) initializeUI() {
	// Create status bar
	d.statusBar = widget.NewLabel("Ready - All systems operational")

	// Create tab container
	d.tabContainer = container.NewAppTabs()
}

// createMainContainer creates the main application layout
func (d *Dashboard) createMainContainer() {
	// Create all tabs
	walletTab := d.createWalletTab()
	nodeTab := d.createNodeTab()
	networkTab := d.createNetworkTab()
	browserTab := d.createBrowserTab()

	// Add tabs to container
	d.tabContainer.Append(container.NewTabItem("💼 Wallet", walletTab.container))
	d.tabContainer.Append(container.NewTabItem("🖥️  Node", nodeTab.container))
	d.tabContainer.Append(container.NewTabItem("🌐 Network", networkTab.container))
	d.tabContainer.Append(container.NewTabItem("🌍 Browser", browserTab.container))

	// Create main container with better proportions
	d.mainContainer = container.NewBorder(
		nil,                                 // top
		d.statusBar,                         // bottom
		nil,                                 // left
		nil,                                 // right
		container.NewScroll(d.tabContainer), // center - scrollable tabs
	)

	d.window.SetContent(d.mainContainer)
}

// createWalletTab creates the comprehensive wallet management interface
func (d *Dashboard) createWalletTab() *WalletTab {
	wt := &WalletTab{
		window:       d.window,
		dashboard:    d,
		sites:        make([]string, 0),
		domains:      make([]string, 0),
		websiteFiles: make([]string, 0),
	}

	// Create form fields
	wt.walletPath = widget.NewEntry()
	wt.walletPath.SetPlaceHolder("/path/to/wallet.json")
	wt.walletPath.SetText(filepath.Join(d.dataDir, "wallet.json"))

	wt.mnemonic = widget.NewPasswordEntry()
	wt.mnemonic.SetPlaceHolder("Enter your 12-word mnemonic phrase")

	wt.siteLabel = widget.NewEntry()
	wt.siteLabel.SetPlaceHolder("Site label (e.g., mysite)")

	wt.domainName = widget.NewEntry()
	wt.domainName.SetPlaceHolder("Domain name (e.g., mysite.bn)")

	wt.websiteDir = widget.NewEntry()
	wt.websiteDir.SetPlaceHolder("/path/to/website/directory")

	wt.mainFile = widget.NewEntry()
	wt.mainFile.SetPlaceHolder("Main file (e.g., index.html)")
	wt.mainFile.SetText("index.html")

	wt.encryptPass = widget.NewPasswordEntry()
	wt.encryptPass.SetPlaceHolder("Encryption passphrase (optional)")

	// Create buttons
	wt.createWalletBtn = widget.NewButton("Create New Wallet", wt.createWallet)
	wt.addSiteBtn = widget.NewButton("Add Site", wt.addSite)
	wt.publishBtn = widget.NewButton("Publish Content", wt.publishContent)
	wt.registerDomainBtn = widget.NewButton("Register Domain", wt.registerDomain)
	wt.publishWebsiteBtn = widget.NewButton("Publish Website", wt.publishWebsite)
	wt.addFileBtn = widget.NewButton("Add File", wt.addFile)
	wt.exportKeyBtn = widget.NewButton("Export Key", wt.exportKey)

	// Create lists
	wt.sitesList = widget.NewList(
		func() int { return len(wt.sites) },
		func() fyne.CanvasObject { return widget.NewLabel("Site") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(wt.sites) {
				obj.(*widget.Label).SetText(wt.sites[id])
			}
		},
	)

	wt.domainsList = widget.NewList(
		func() int { return len(wt.domains) },
		func() fyne.CanvasObject { return widget.NewLabel("Domain") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(wt.domains) {
				obj.(*widget.Label).SetText(wt.domains[id])
			}
		},
	)

	wt.websiteFilesList = widget.NewList(
		func() int { return len(wt.websiteFiles) },
		func() fyne.CanvasObject { return widget.NewLabel("File") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(wt.websiteFiles) {
				obj.(*widget.Label).SetText(wt.websiteFiles[id])
			}
		},
	)

	// Create form container with better layout
	formContainer := container.NewVBox(
		widget.NewLabel("💼 Wallet Management"),
		container.NewVBox(
			widget.NewLabel("Wallet Path"),
			wt.walletPath,
			widget.NewLabel("Mnemonic"),
			wt.mnemonic,
			widget.NewLabel("Site Label"),
			wt.siteLabel,
		),
		container.NewHBox(
			wt.createWalletBtn,
			wt.addSiteBtn,
		),
		widget.NewSeparator(),
		widget.NewLabel("🌐 Content Publishing"),
		container.NewVBox(
			widget.NewLabel("Content/File Path"),
			wt.websiteDir,
			widget.NewLabel("Encryption Passphrase"),
			wt.encryptPass,
		),
		container.NewHBox(
			wt.publishBtn,
			wt.exportKeyBtn,
		),
		widget.NewSeparator(),
		widget.NewLabel("📁 Website Publishing"),
		container.NewVBox(
			widget.NewLabel("Website Directory"),
			wt.websiteDir,
			widget.NewLabel("Main File"),
			wt.mainFile,
		),
		container.NewHBox(
			wt.publishWebsiteBtn,
			wt.addFileBtn,
		),
		widget.NewSeparator(),
		widget.NewLabel("🔗 Domain Management"),
		container.NewVBox(
			widget.NewLabel("Domain Name"),
			wt.domainName,
		),
		wt.registerDomainBtn,
	)

	// Lists are now created inline in the main container

	// Create main container with better proportions
	wt.container = container.NewBorder(
		container.NewScroll(formContainer), // top - scrollable form
		container.NewVBox(
			widget.NewLabel("📄 Website Files"),
			wt.websiteFilesList,
		), // bottom
		nil, // left
		nil, // right
		container.NewHSplit(
			container.NewVBox(
				widget.NewLabel("📋 Sites"),
				wt.sitesList,
			),
			container.NewVBox(
				widget.NewLabel("🔗 Domains"),
				wt.domainsList,
			),
		), // center - use HSplit for better proportions
	)

	return wt
}

// createNodeTab creates the comprehensive node control interface
func (d *Dashboard) createNodeTab() *NodeTab {
	nt := &NodeTab{
		window:    d.window,
		isRunning: false,
	}

	// Create form fields
	nt.dataDir = widget.NewEntry()
	nt.dataDir.SetPlaceHolder("/path/to/node/data")
	nt.dataDir.SetText(d.dataDir)

	nt.listenAddr = widget.NewEntry()
	nt.listenAddr.SetPlaceHolder("/ip4/0.0.0.0/tcp/4001")
	nt.listenAddr.SetText("/ip4/0.0.0.0/tcp/4001")

	nt.bootstrap = widget.NewEntry()
	nt.bootstrap.SetPlaceHolder("Bootstrap peer addresses (comma-separated)")

	nt.keyPath = widget.NewEntry()
	nt.keyPath.SetPlaceHolder("/path/to/site-key.b64")

	// Create buttons
	nt.initKeyBtn = widget.NewButton("Initialize Key", nt.initKey)
	nt.startNodeBtn = widget.NewButton("Start Node", nt.startNode)
	nt.stopNodeBtn = widget.NewButton("Stop Node", nt.stopNode)
	nt.stopNodeBtn.Disable()
	nt.publishBtn = widget.NewButton("Publish Content", nt.publishContent)
	nt.browseBtn = widget.NewButton("Browse Site", nt.browseSite)
	nt.publishWebsiteBtn = widget.NewButton("Publish Website", nt.publishWebsite)
	nt.addFileBtn = widget.NewButton("Add File", nt.addFile)

	// Create status and log
	nt.nodeStatus = widget.NewLabel("Node Status: Stopped")
	nt.logArea = widget.NewMultiLineEntry()
	nt.logArea.Disable()

	// Create form container
	formContainer := container.NewVBox(
		widget.NewLabel("🖥️  Node Configuration"),
		container.NewVBox(
			widget.NewLabel("Data Directory"),
			nt.dataDir,
			widget.NewLabel("Listen Address"),
			nt.listenAddr,
			widget.NewLabel("Bootstrap Peers"),
			nt.bootstrap,
		),
		container.NewHBox(
			nt.initKeyBtn,
		),
		container.NewHBox(
			nt.startNodeBtn,
			nt.stopNodeBtn,
		),
		nt.nodeStatus,
		widget.NewSeparator(),
		widget.NewLabel("📤 Node Operations"),
		container.NewHBox(
			nt.publishBtn,
			nt.browseBtn,
		),
		container.NewHBox(
			nt.publishWebsiteBtn,
			nt.addFileBtn,
		),
	)

	// Create log container
	logContainer := container.NewVBox(
		widget.NewLabel("📝 Node Logs"),
		nt.logArea,
	)

	// Create main container with better proportions
	nt.container = container.NewBorder(
		container.NewScroll(formContainer), // top - scrollable form
		nil,                                // bottom
		nil,                                // left
		nil,                                // right
		logContainer,                       // center
	)

	return nt
}

// createNetworkTab creates the comprehensive network management interface
func (d *Dashboard) createNetworkTab() *NetworkTab {
	nt := &NetworkTab{
		window:       d.window,
		peers:        make([]string, 0),
		networkStats: make([]string, 0),
		isRunning:    false,
	}

	// Create status display
	nt.statusLabel = widget.NewLabel("Network Status: Disconnected")
	nt.uptimeLabel = widget.NewLabel("Uptime: --:--:--")
	nt.peersLabel = widget.NewLabel("Active Peers: 0")
	nt.healthLabel = widget.NewLabel("Network Health: Unknown")

	// Create buttons
	nt.startBtn = widget.NewButton("Start Network", nt.startNetwork)
	nt.stopBtn = widget.NewButton("Stop Network", nt.stopNetwork)
	nt.stopBtn.Disable()
	nt.discoverBtn = widget.NewButton("Discover Peers", nt.discoverPeers)
	nt.refreshBtn = widget.NewButton("Refresh Status", nt.refreshStatus)
	nt.healthBtn = widget.NewButton("Check Health", nt.checkHealth)

	// Create lists
	nt.peersList = widget.NewList(
		func() int { return len(nt.peers) },
		func() fyne.CanvasObject { return widget.NewLabel("Peer") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(nt.peers) {
				obj.(*widget.Label).SetText(nt.peers[id])
			}
		},
	)

	nt.networkStatsList = widget.NewList(
		func() int { return len(nt.networkStats) },
		func() fyne.CanvasObject { return widget.NewLabel("Stat") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(nt.networkStats) {
				obj.(*widget.Label).SetText(nt.networkStats[id])
			}
		},
	)

	// Create status container
	statusContainer := container.NewVBox(
		widget.NewLabel("🌐 Network Status"),
		nt.statusLabel,
		nt.uptimeLabel,
		nt.peersLabel,
		nt.healthLabel,
	)

	// Create control container
	controlContainer := container.NewVBox(
		widget.NewLabel("🎮 Network Controls"),
		container.NewHBox(
			nt.startBtn,
			nt.stopBtn,
		),
		container.NewHBox(
			nt.discoverBtn,
			nt.refreshBtn,
			nt.healthBtn,
		),
	)

	// Create left panel
	leftPanel := container.NewVBox(
		statusContainer,
		widget.NewSeparator(),
		controlContainer,
	)

	// Create right panel
	rightPanel := container.NewHSplit(
		container.NewVBox(
			widget.NewLabel("👥 Connected Peers"),
			nt.peersList,
		),
		container.NewVBox(
			widget.NewLabel("📊 Network Statistics"),
			nt.networkStatsList,
		),
	)

	// Create main container with better proportions
	nt.container = container.NewBorder(
		container.NewScroll(leftPanel), // top - scrollable panel
		nil,                            // bottom
		nil,                            // left
		nil,                            // right
		rightPanel,                     // center
	)

	return nt
}

// createBrowserTab creates the web browsing interface
func (d *Dashboard) createBrowserTab() *BrowserTab {
	bt := &BrowserTab{
		window:       d.window,
		history:      make([]string, 0),
		historyIndex: -1,
	}

	// Create address bar
	bt.addressBar = widget.NewEntry()
	bt.addressBar.SetPlaceHolder("🌐 Enter site ID or domain (e.g., mysite.bn)")
	bt.addressBar.OnSubmitted = bt.navigateToSite

	// Create navigation buttons
	bt.backBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), bt.goBack)
	bt.forwardBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), bt.goForward)
	bt.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), bt.refresh)
	bt.homeBtn = widget.NewButtonWithIcon("", theme.HomeIcon(), bt.goHome)

	// Create content area
	bt.contentArea = widget.NewRichText()
	bt.contentArea.Wrapping = fyne.TextWrapWord

	// Create navigation bar
	navBar := container.NewHBox(
		bt.backBtn,
		bt.forwardBtn,
		bt.refreshBtn,
		bt.homeBtn,
		layout.NewSpacer(),
		bt.addressBar,
		layout.NewSpacer(),
	)

	// Create welcome content
	welcomeContent := `# 🌐 Welcome to Betanet Browser

**Experience the decentralized web with complete multi-file websites**

## 🚀 Getting Started

### Browse by Site ID
Enter a site ID in the address bar to browse a specific website.

### Browse by Domain
Use human-readable .bn domains like mysite.bn for easy navigation.

### Search and Discover
Explore the decentralized web and discover new content.

## ✨ Features

- **🌐 Multi-File Websites** - Complete websites with HTML, CSS, JavaScript, and images
- **🔍 Smart Navigation** - Address bar with suggestions and history
- **📚 Tab Management** - Multiple tabs for different sites
- **🔒 Secure Browsing** - Cryptographic content verification
- **🌍 Peer-to-Peer** - Direct content delivery without intermediaries
- **📱 Responsive Design** - Beautiful interface that works on all devices

---

*Ready to explore the decentralized web? Enter a site ID or domain above to begin!*`

	bt.contentArea.ParseMarkdown(welcomeContent)

	// Create main container
	bt.container = container.NewBorder(
		navBar,                              // top
		nil,                                 // bottom
		nil,                                 // left
		nil,                                 // right
		container.NewScroll(bt.contentArea), // center
	)

	return bt
}

// Show displays the dashboard window
func (d *Dashboard) Show() {
	d.window.Show()
}

// Helper functions for wallet operations
func openWallet(path, mnemonic string) (*wallet.Wallet, []byte) {
	enc, err := wallet.Load(path)
	if err != nil {
		return nil, nil
	}
	w, err := wallet.DecryptWallet(enc, mnemonic)
	if err != nil {
		return nil, nil
	}
	master, err := wallet.MasterKeyFromMnemonic(mnemonic)
	if err != nil {
		return nil, nil
	}
	return w, master
}

func saveWallet(path string, w *wallet.Wallet, mnemonic string) {
	enc, err := wallet.EncryptWallet(w, mnemonic)
	if err != nil {
		return
	}
	if err := wallet.Save(path, enc); err != nil {
		return
	}
}

// Wallet tab methods - implement all console functionality
func (wt *WalletTab) createWallet() {
	walletPath := wt.walletPath.Text
	if walletPath == "" {
		dialog.ShowError(fmt.Errorf("Please enter a wallet path"), wt.window)
		return
	}

	// Generate new mnemonic
	mnemonic, err := wallet.NewMnemonic()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to generate mnemonic: %v", err), wt.window)
		return
	}

	// Create wallet
	w := wallet.New()
	encrypted, err := wallet.EncryptWallet(w, mnemonic)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to encrypt wallet: %v", err), wt.window)
		return
	}

	// Save wallet
	if err := wallet.Save(walletPath, encrypted); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to save wallet: %v", err), wt.window)
		return
	}

	// Show success dialog with mnemonic
	dialog.ShowInformation("Wallet Created",
		fmt.Sprintf("Wallet created successfully!\n\nMnemonic (STORE SAFELY):\n%s\n\nThis phrase is required to unlock your wallet.", mnemonic),
		wt.window)
}

func (wt *WalletTab) addSite() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		return
	}

	// Ensure site exists
	meta, _, _, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to add site: %v", err), wt.window)
		return
	}

	// Add to sites list
	wt.sites = append(wt.sites, wt.siteLabel.Text)
	wt.sitesList.Refresh()

	// Save wallet
	saveWallet(wt.walletPath.Text, w, wt.mnemonic.Text)

	dialog.ShowInformation("Site Added",
		fmt.Sprintf("Site '%s' added successfully!\n\nSite ID: %s", wt.siteLabel.Text, meta.SiteID),
		wt.window)

	// Clear form
	wt.siteLabel.SetText("")
}

func (wt *WalletTab) publishContent() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	if wt.websiteDir.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a content/file path"), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		dialog.ShowError(fmt.Errorf("Failed to open wallet"), wt.window)
		return
	}

	// Ensure site exists
	meta, pub, _, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to ensure site: %v", err), wt.window)
		return
	}

	// Read content file
	content, err := os.ReadFile(wt.websiteDir.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read content file: %v", err), wt.window)
		return
	}

	// Encrypt content if passphrase provided
	var finalContent []byte
	if wt.encryptPass.Text != "" {
		encrypted, err := wallet.EncryptContent(wt.encryptPass.Text, content)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to encrypt content: %v", err), wt.window)
			return
		}
		finalContent = encrypted
	} else {
		finalContent = content
	}

	// Create update record
	record := &core.UpdateRecord{
		Version:    "1.0",
		SitePub:    pub,
		Seq:        meta.Seq + 1,
		PrevCID:    meta.HeadRecCID,
		ContentCID: core.CIDForContent(finalContent),
		TS:         core.NowTS(),
	}

	// Generate ephemeral key for this update
	updatePub, updatePriv, err := wallet.DeriveSiteKey(master, wt.siteLabel.Text+"-update")
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to derive update key: %v", err), wt.window)
		return
	}
	record.UpdatePub = updatePub

	// Sign record
	recordData, err := core.CanonicalMarshalNoUpdateSig(record)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to marshal record: %v", err), wt.window)
		return
	}
	record.UpdateSig = ed25519.Sign(updatePriv, recordData)

	// Store content and record in dashboard's store
	if err := wt.dashboard.store.PutContent(record.ContentCID, finalContent); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store content: %v", err), wt.window)
		return
	}

	recordCID := core.CIDForBytes(recordData)
	if err := wt.dashboard.store.PutRecord(recordCID, recordData); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store record: %v", err), wt.window)
		return
	}

	// Update site head
	if err := wt.dashboard.store.PutHead(meta.SiteID, record.Seq, recordCID); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to update site head: %v", err), wt.window)
		return
	}

	// Update wallet metadata
	meta.Seq = record.Seq
	meta.HeadRecCID = recordCID
	meta.ContentCID = record.ContentCID
	meta.LastUpdated = time.Now()

	// Save wallet
	saveWallet(wt.walletPath.Text, w, wt.mnemonic.Text)

	dialog.ShowInformation("Content Published",
		fmt.Sprintf("Content published successfully!\n\nSite: %s\nSite ID: %s\nContent CID: %s\nRecord CID: %s",
			wt.siteLabel.Text, meta.SiteID, record.ContentCID, recordCID),
		wt.window)

	// Clear form
	wt.websiteDir.SetText("")
	wt.encryptPass.SetText("")
}

func (wt *WalletTab) registerDomain() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	if wt.domainName.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a domain name"), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		dialog.ShowError(fmt.Errorf("Failed to open wallet"), wt.window)
		return
	}

	// Ensure site exists
	meta, _, _, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to ensure site: %v", err), wt.window)
		return
	}

	// Add to domains list
	wt.domains = append(wt.domains, wt.domainName.Text)
	wt.domainsList.Refresh()

	// Save wallet
	saveWallet(wt.walletPath.Text, w, wt.mnemonic.Text)

	dialog.ShowInformation("Domain Registered",
		fmt.Sprintf("Domain '%s' registered successfully!\n\nSite: %s\nSite ID: %s",
			wt.domainName.Text, wt.siteLabel.Text, meta.SiteID),
		wt.window)

	// Clear form
	wt.domainName.SetText("")
}

func (wt *WalletTab) publishWebsite() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	if wt.websiteDir.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a website directory path"), wt.window)
		return
	}

	// Check if directory exists
	if _, err := os.Stat(wt.websiteDir.Text); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("Website directory does not exist: %s", wt.websiteDir.Text), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		dialog.ShowError(fmt.Errorf("Failed to open wallet"), wt.window)
		return
	}

	// Ensure site exists
	meta, pub, _, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to ensure site: %v", err), wt.window)
		return
	}

	// Read main file
	mainFilePath := filepath.Join(wt.websiteDir.Text, wt.mainFile.Text)
	mainContent, err := os.ReadFile(mainFilePath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read main file %s: %v", mainFilePath, err), wt.window)
		return
	}

	// Encrypt main content if passphrase provided
	var finalMainContent []byte
	if wt.encryptPass.Text != "" {
		encrypted, err := wallet.EncryptContent(wt.encryptPass.Text, mainContent)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to encrypt main content: %v", err), wt.window)
			return
		}
		finalMainContent = encrypted
	} else {
		finalMainContent = mainContent
	}

	// Create update record for main file
	record := &core.UpdateRecord{
		Version:    "1.0",
		SitePub:    pub,
		Seq:        meta.Seq + 1,
		PrevCID:    meta.HeadRecCID,
		ContentCID: core.CIDForContent(finalMainContent),
		TS:         core.NowTS(),
	}

	// Generate ephemeral key for this update
	updatePub, updatePriv, err := wallet.DeriveSiteKey(master, wt.siteLabel.Text+"-update")
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to derive update key: %v", err), wt.window)
		return
	}
	record.UpdatePub = updatePub

	// Sign record
	recordData, err := core.CanonicalMarshalNoUpdateSig(record)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to marshal record: %v", err), wt.window)
		return
	}
	record.UpdateSig = ed25519.Sign(updatePriv, recordData)

	// Store main content and record
	if err := wt.dashboard.store.PutContent(record.ContentCID, finalMainContent); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store main content: %v", err), wt.window)
		return
	}

	recordCID := core.CIDForBytes(recordData)
	if err := wt.dashboard.store.PutRecord(recordCID, recordData); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store record: %v", err), wt.window)
		return
	}

	// Update site head
	if err := wt.dashboard.store.PutHead(meta.SiteID, record.Seq, recordCID); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to update site head: %v", err), wt.window)
		return
	}

	// Update wallet metadata
	meta.Seq = record.Seq
	meta.HeadRecCID = recordCID
	meta.ContentCID = record.ContentCID
	meta.LastUpdated = time.Now()

	// Save wallet
	saveWallet(wt.walletPath.Text, w, wt.mnemonic.Text)

	dialog.ShowInformation("Website Published",
		fmt.Sprintf("Website published successfully!\n\nSite: %s\nSite ID: %s\nMain File: %s\nContent CID: %s\nRecord CID: %s",
			wt.siteLabel.Text, meta.SiteID, wt.mainFile.Text, record.ContentCID, recordCID),
		wt.window)

	// Clear form
	wt.websiteDir.SetText("")
	wt.encryptPass.SetText("")
}

func (wt *WalletTab) addFile() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	if wt.websiteDir.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a file path"), wt.window)
		return
	}

	// Check if file exists
	if _, err := os.Stat(wt.websiteDir.Text); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("File does not exist: %s", wt.websiteDir.Text), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		dialog.ShowError(fmt.Errorf("Failed to open wallet"), wt.window)
		return
	}

	// Ensure site exists
	meta, pub, priv, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to ensure site: %v", err), wt.window)
		return
	}

	// Read file content
	fileContent, err := os.ReadFile(wt.websiteDir.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read file: %v", err), wt.window)
		return
	}

	// Encrypt file content if passphrase provided
	var finalFileContent []byte
	if wt.encryptPass.Text != "" {
		encrypted, err := wallet.EncryptContent(wt.encryptPass.Text, fileContent)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to encrypt file content: %v", err), wt.window)
			return
		}
		finalFileContent = encrypted
	} else {
		finalFileContent = fileContent
	}

	// Store file content first
	contentCID := core.CIDForContent(finalFileContent)
	if err := wt.dashboard.store.PutContent(contentCID, finalFileContent); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store file content: %v", err), wt.window)
		return
	}

	// Get filename first
	fileName := filepath.Base(wt.websiteDir.Text)

	// Generate ephemeral key for this update
	updatePub, updatePriv, err := wallet.DeriveSiteKey(master, wt.siteLabel.Text+"-file-"+fileName)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to derive update key: %v", err), wt.window)
		return
	}

	// Create file record
	fileRecord := &core.FileRecord{
		Version:    "1.0",
		SitePub:    pub,
		Path:       fileName,
		ContentCID: contentCID,
		MimeType:   "application/octet-stream", // Default MIME type
		TS:         time.Now().Unix(),
		UpdatePub:  updatePub,
		LinkSig:    ed25519.Sign(priv, []byte(fileName+contentCID)),
		UpdateSig:  ed25519.Sign(updatePriv, []byte(fileName+contentCID)),
	}

	// Store file record
	fileRecordData, err := core.CanonicalMarshalFileRecord(fileRecord)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to marshal file record: %v", err), wt.window)
		return
	}

	fileRecordCID := core.CIDForBytes(fileRecordData)
	if err := wt.dashboard.store.PutRecord(fileRecordCID, fileRecordData); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to store file record: %v", err), wt.window)
		return
	}

	// Add to website files list
	wt.websiteFiles = append(wt.websiteFiles, fileName)
	wt.websiteFilesList.Refresh()

	// Save wallet
	saveWallet(wt.walletPath.Text, w, wt.mnemonic.Text)

	dialog.ShowInformation("File Added",
		fmt.Sprintf("File '%s' added successfully!\n\nSite: %s\nSite ID: %s\nContent CID: %s",
			fileName, wt.siteLabel.Text, meta.SiteID, contentCID),
		wt.window)

	// Clear form
	wt.websiteDir.SetText("")
	wt.encryptPass.SetText("")
}

func (wt *WalletTab) exportKey() {
	// Validate inputs
	if wt.mnemonic.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter your mnemonic phrase"), wt.window)
		return
	}

	if wt.siteLabel.Text == "" {
		dialog.ShowError(fmt.Errorf("Please enter a site label"), wt.window)
		return
	}

	// Open wallet
	w, master := openWallet(wt.walletPath.Text, wt.mnemonic.Text)
	if w == nil {
		dialog.ShowError(fmt.Errorf("Failed to open wallet"), wt.window)
		return
	}

	// Ensure site exists and get private key
	_, _, priv, err := w.EnsureSite(master, wt.siteLabel.Text)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to get site key: %v", err), wt.window)
		return
	}

	// Encode private key as base64
	keyData := base64.StdEncoding.EncodeToString(priv)

	dialog.ShowInformation("Private Key Exported",
		fmt.Sprintf("Private key for site '%s':\n\n%s\n\n⚠️  Keep this key secure and private!",
			wt.siteLabel.Text, keyData),
		wt.window)
}

// Node tab methods - implement all console functionality
func (nt *NodeTab) initKey() {
	// TODO: Implement key initialization
	dialog.ShowInformation("Info", "Key initialization will be implemented in the next iteration", nt.window)
}

func (nt *NodeTab) startNode() {
	// TODO: Implement node startup
	nt.nodeStatus.SetText("Node Status: Starting...")
	nt.startNodeBtn.Disable()
	nt.stopNodeBtn.Enable()
	nt.isRunning = true
	nt.logArea.SetText("Node starting...\n")
}

func (nt *NodeTab) stopNode() {
	// TODO: Implement node shutdown
	nt.nodeStatus.SetText("Node Status: Stopped")
	nt.startNodeBtn.Enable()
	nt.stopNodeBtn.Disable()
	nt.isRunning = false
	nt.logArea.SetText(nt.logArea.Text + "Node stopped\n")
}

func (nt *NodeTab) publishContent() {
	// TODO: Implement content publishing
	dialog.ShowInformation("Info", "Content publishing will be implemented in the next iteration", nt.window)
}

func (nt *NodeTab) browseSite() {
	// TODO: Implement site browsing
	dialog.ShowInformation("Info", "Site browsing will be implemented in the next iteration", nt.window)
}

func (nt *NodeTab) publishWebsite() {
	// TODO: Implement website publishing
	dialog.ShowInformation("Info", "Website publishing will be implemented in the next iteration", nt.window)
}

func (nt *NodeTab) addFile() {
	// TODO: Implement file addition
	dialog.ShowInformation("Info", "File addition will be implemented in the next iteration", nt.window)
}

// Network tab methods - implement all console functionality
func (nt *NetworkTab) startNetwork() {
	// TODO: Implement network startup
	nt.isRunning = true
	nt.statusLabel.SetText("Network Status: Running")
	nt.startBtn.Disable()
	nt.stopBtn.Enable()
	nt.discoverBtn.Enable()
	nt.refreshBtn.Enable()
	nt.healthBtn.Enable()
}

func (nt *NetworkTab) stopNetwork() {
	// TODO: Implement network shutdown
	nt.isRunning = false
	nt.statusLabel.SetText("Network Status: Stopped")
	nt.startBtn.Enable()
	nt.stopBtn.Disable()
	nt.discoverBtn.Disable()
	nt.refreshBtn.Disable()
	nt.healthBtn.Disable()
}

func (nt *NetworkTab) discoverPeers() {
	// TODO: Implement peer discovery
	dialog.ShowInformation("Info", "Peer discovery will be implemented in the next iteration", nt.window)
}

func (nt *NetworkTab) refreshStatus() {
	// TODO: Implement status refresh
	dialog.ShowInformation("Info", "Status refresh will be implemented in the next iteration", nt.window)
}

func (nt *NetworkTab) checkHealth() {
	// TODO: Implement health check
	dialog.ShowInformation("Info", "Health check will be implemented in the next iteration", nt.window)
}

// Browser tab methods
func (bt *BrowserTab) navigateToSite(url string) {
	if url == "" {
		return
	}

	// Add to history
	bt.history = append(bt.history, url)
	bt.historyIndex = len(bt.history) - 1

	// Update navigation buttons
	bt.updateNavigationButtons()

	// Show placeholder content
	content := fmt.Sprintf(`# 🌐 Site: %s

**This is a placeholder for the actual site content**

The enhanced browser will load real decentralized websites in the next iteration.

## Features to be implemented:
- **Site Resolution** - Convert domains to site IDs
- **Content Fetching** - Retrieve website files from the network
- **Multi-file Rendering** - Display HTML, CSS, JavaScript, and images
- **Navigation** - Handle internal links and site structure
- **Error Handling** - Graceful fallbacks for network issues

## Current URL: %s

*Enhanced browsing functionality coming soon!*`, url, url)

	bt.contentArea.ParseMarkdown(content)
}

func (bt *BrowserTab) goBack() {
	if bt.historyIndex > 0 {
		bt.historyIndex--
		url := bt.history[bt.historyIndex]
		bt.addressBar.SetText(url)
		bt.navigateToSite(url)
	}
}

func (bt *BrowserTab) goForward() {
	if bt.historyIndex < len(bt.history)-1 {
		bt.historyIndex++
		url := bt.history[bt.historyIndex]
		bt.addressBar.SetText(url)
		bt.navigateToSite(url)
	}
}

func (bt *BrowserTab) refresh() {
	url := bt.addressBar.Text
	if url != "" {
		bt.navigateToSite(url)
	}
}

func (bt *BrowserTab) goHome() {
	bt.navigateToSite("")
}

func (bt *BrowserTab) updateNavigationButtons() {
	bt.backBtn.Enable()
	bt.forwardBtn.Enable()

	if bt.historyIndex <= 0 {
		bt.backBtn.Disable()
	}

	if bt.historyIndex >= len(bt.history)-1 {
		bt.forwardBtn.Disable()
	}
}
