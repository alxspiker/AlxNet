package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"betanet/internal/core"
	"betanet/internal/p2p"
	"betanet/internal/store"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/fxamacker/cbor/v2"
	"golang.org/x/net/html"
)

// WebsiteFile represents a file in a multi-file website
type WebsiteFile struct {
	Path        string // e.g., "styles/main.css", "scripts/app.js", "images/logo.png"
	ContentCID  string
	Content     []byte
	MimeType    string
	LastUpdated time.Time
}

// Website represents a complete multi-file website
type Website struct {
	ID          string
	Files       map[string]*WebsiteFile // path -> file
	MainHTML    *WebsiteFile            // index.html or main entry point
	LastUpdated time.Time
}

// Tab represents a browser tab
type Tab struct {
	Title         string
	Content       *widget.RichText
	ScrollContent *container.Scroll // Scrollable wrapper for the content
	Address       string
	IsActive      bool
	Website       *Website
}

// Modern tabbed browser interface for decentralized web with multi-file website support

func main() {
	// Parse command line arguments
	var dataDir string
	if len(os.Args) > 1 && os.Args[1] == "-h" || len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Betanet Browser - Decentralized Web Browser")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  ./bin/betanet-browser                    # Use default data directory (~/.betanet/browser)")
		fmt.Println("  ./bin/betanet-browser -data /path/db    # Use specified node database")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  ./bin/betanet-browser -data /tmp/node1   # Connect to existing node database")
		fmt.Println("  ./bin/betanet-browser -data temp/demo-node # Connect to demo node")
		fmt.Println("")
		fmt.Println("The browser will start its own local network node using the specified database.")
		fmt.Println("This allows you to browse sites and domains registered in that database.")
		return
	}

	if len(os.Args) > 2 && os.Args[1] == "-data" {
		dataDir = os.Args[2]
		fmt.Printf("BROWSER: Using specified data directory: %s\n", dataDir)
	} else {
		dataDir = filepath.Join(os.Getenv("HOME"), ".betanet", "browser")
		fmt.Printf("BROWSER: Using default data directory: %s\n", dataDir)
	}

	a := app.New()
	w := a.NewWindow("Betanet Browser")
	w.Resize(fyne.NewSize(1400, 900))
	w.SetIcon(theme.ComputerIcon())

	// Hidden settings - accessed via settings button
	settingsDataDir := widget.NewEntry()
	settingsDataDir.SetPlaceHolder("/home/user/.betanet/node")
	listen := widget.NewEntry()
	listen.SetText("/ip4/0.0.0.0/tcp/0")
	bootstrap := widget.NewEntry()
	bootstrap.SetPlaceHolder("Auto-filled from discovery, or enter manually")

	// Browser status - minimal, hidden by default
	_ = ""

	// Tab management
	var tabs []*Tab
	var currentTabIndex int = 0

	// Tab container
	tabContainer := container.NewStack()

	// Tab bar
	tabBar := container.NewHBox()

	// Modern address bar with better styling
	addressBar := widget.NewEntry()
	addressBar.SetPlaceHolder("Enter site address (like example.bn or site ID)")
	addressBar.TextStyle = fyne.TextStyle{Bold: false}

	var currentNode *p2p.Node
	var currentDB *store.Store

	// Forward declarations
	var browseToSite func(string)
	var updateTabBar func()
	var closeTab func(int)
	var showSettingsTab func()
	var loadWebsite func(string) *Website
	var renderWebsite func(*Website, *Tab)

	// Function to create a new tab
	createNewTab := func(title string, address string) *Tab {
		content := widget.NewRichText()
		content.Wrapping = fyne.TextWrapWord

		// Wrap the content in a scrollable container
		scrollContent := container.NewScroll(content)
		scrollContent.SetMinSize(fyne.NewSize(800, 600)) // Ensure minimum size for scrolling

		tab := &Tab{
			Title:         title,
			Content:       content,
			ScrollContent: scrollContent,
			Address:       address,
			IsActive:      false,
			Website:       nil,
		}

		tabs = append(tabs, tab)
		return tab
	}

	// Function to switch to a tab
	switchToTab := func(index int) {
		if index >= 0 && index < len(tabs) {
			// Deactivate current tab
			if currentTabIndex < len(tabs) {
				tabs[currentTabIndex].IsActive = false
			}

			// Activate new tab
			currentTabIndex = index
			tabs[currentTabIndex].IsActive = true

			// Update tab container with scrollable content
			tabContainer.Objects = []fyne.CanvasObject{tabs[currentTabIndex].ScrollContent}
			tabContainer.Refresh()

			// Update address bar
			addressBar.SetText(tabs[currentTabIndex].Address)

			// Update tab bar styling
			updateTabBar()
		}
	}

	// Function to update tab bar styling
	updateTabBar = func() {
		tabBar.Objects = nil
		for i, tab := range tabs {
			tabIndex := i // Capture for closure
			tabBtn := widget.NewButton(tab.Title, func() {
				switchToTab(tabIndex)
			})

			if tab.IsActive {
				tabBtn.Importance = widget.HighImportance
			} else {
				tabBtn.Importance = widget.LowImportance
			}

			// Add close button for tabs (except the first tab)
			if i > 0 {
				closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
					closeTab(tabIndex)
				})
				closeBtn.Importance = widget.LowImportance

				tabRow := container.NewHBox(tabBtn, closeBtn)
				tabBar.Add(tabRow)
			} else {
				tabBar.Add(tabBtn)
			}
		}
		tabBar.Refresh()
	}

	// Function to close a tab
	closeTab = func(index int) {
		if index > 0 && index < len(tabs) { // Don't close the first tab
			// Remove tab from slice
			tabs = append(tabs[:index], tabs[index+1:]...)

			// Adjust current tab index if needed
			if currentTabIndex >= index {
				currentTabIndex--
			}
			if currentTabIndex < 0 {
				currentTabIndex = 0
			}

			// Switch to current tab
			if len(tabs) > 0 {
				switchToTab(currentTabIndex)
			}
		}
	}

	// Function to add a new tab
	addNewTab := func(title string, address string) {
		createNewTab(title, address)
		switchToTab(len(tabs) - 1)
	}

	// Function to determine MIME type from file extension
	getMimeType := func(filename string) string {
		ext := path.Ext(filename)
		switch ext {
		case ".html", ".htm":
			return "text/html"
		case ".css":
			return "text/css"
		case ".js":
			return "application/javascript"
		case ".png":
			return "image/png"
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".gif":
			return "image/gif"
		case ".svg":
			return "image/svg+xml"
		case ".ico":
			return "image/x-icon"
		case ".json":
			return "application/json"
		case ".xml":
			return "application/xml"
		case ".txt":
			return "text/plain"
		case ".md":
			return "text/markdown"
		default:
			return "application/octet-stream"
		}
	}

	// Function to extract file references from HTML
	extractFileReferences := func(htmlContent []byte) []string {
		var references []string
		doc, err := html.Parse(bytes.NewReader(htmlContent))
		if err != nil {
			return references
		}

		var traverse func(*html.Node)
		traverse = func(n *html.Node) {
			if n.Type == html.ElementNode {
				switch n.Data {
				case "link":
					for _, attr := range n.Attr {
						if attr.Key == "href" {
							references = append(references, attr.Val)
						}
					}
				case "script":
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							references = append(references, attr.Val)
						}
					}
				case "img":
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							references = append(references, attr.Val)
						}
					}
				case "video":
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							references = append(references, attr.Val)
						}
					}
				case "audio":
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							references = append(references, attr.Val)
						}
					}
				case "source":
					for _, attr := range n.Attr {
						if attr.Key == "src" {
							references = append(references, attr.Val)
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				traverse(c)
			}
		}
		traverse(doc)

		return references
	}

	// Function to load a complete website
	loadWebsite = func(siteID string) *Website {
		if currentDB == nil {
			return nil
		}

		website := &Website{
			ID:    siteID,
			Files: make(map[string]*WebsiteFile),
		}

		// First try to load as a multi-file website
		if currentDB.HasWebsiteManifest(siteID) {
			manifestBytes, err := currentDB.GetCurrentWebsiteManifest(siteID)
			if err != nil {
				fmt.Printf("BROWSER: Error loading website manifest: %v\n", err)
				return nil
			}

			var manifest core.WebsiteManifest
			dec, _ := cbor.DecOptions{}.DecMode()
			if err := dec.Unmarshal(manifestBytes, &manifest); err != nil {
				fmt.Printf("BROWSER: Error unmarshaling website manifest: %v\n", err)
				return nil
			}

			fmt.Printf("BROWSER: Loading multi-file website with %d files\n", len(manifest.Files))

			// Load the main HTML file
			mainFileRecordBytes, err := currentDB.GetFileRecordByPath(siteID, manifest.MainFile)
			if err != nil {
				fmt.Printf("BROWSER: Error loading main file record: %v\n", err)
				return nil
			}

			var mainFileRecord core.FileRecord
			if err := dec.Unmarshal(mainFileRecordBytes, &mainFileRecord); err != nil {
				fmt.Printf("BROWSER: Error unmarshaling main file record: %v\n", err)
				return nil
			}

			mainContent, err := currentDB.GetContent(mainFileRecord.ContentCID)
			if err != nil {
				fmt.Printf("BROWSER: Error loading main file content: %v\n", err)
				return nil
			}

			// Create main HTML file
			mainFile := &WebsiteFile{
				Path:        manifest.MainFile,
				ContentCID:  mainFileRecord.ContentCID,
				Content:     mainContent,
				MimeType:    mainFileRecord.MimeType,
				LastUpdated: time.Unix(mainFileRecord.TS, 0),
			}
			website.MainHTML = mainFile
			website.Files[manifest.MainFile] = mainFile

			// Load all referenced files
			for filePath, contentCID := range manifest.Files {
				if filePath == manifest.MainFile {
					continue // Already loaded
				}

				fileRecordBytes, err := currentDB.GetFileRecordByPath(siteID, filePath)
				if err != nil {
					fmt.Printf("BROWSER: Error loading file record for %s: %v\n", filePath, err)
					continue
				}

				var fileRecord core.FileRecord
				if err := dec.Unmarshal(fileRecordBytes, &fileRecord); err != nil {
					fmt.Printf("BROWSER: Error unmarshaling file record for %s: %v\n", filePath, err)
					continue
				}

				fileContent, err := currentDB.GetContent(contentCID)
				if err != nil {
					fmt.Printf("BROWSER: Error loading file content for %s: %v\n", filePath, err)
					continue
				}

				websiteFile := &WebsiteFile{
					Path:        filePath,
					ContentCID:  contentCID,
					Content:     fileContent,
					MimeType:    fileRecord.MimeType,
					LastUpdated: time.Unix(fileRecord.TS, 0),
				}
				website.Files[filePath] = websiteFile
			}

			return website
		}

		// Fall back to single-file website (legacy support)
		_, headCID, err := currentDB.GetHead(siteID)
		if err != nil {
			return nil
		}

		recBytes, err := currentDB.GetRecord(headCID)
		if err != nil {
			return nil
		}

		var rec core.UpdateRecord
		dec, _ := cbor.DecOptions{}.DecMode()
		if err := dec.Unmarshal(recBytes, &rec); err != nil {
			return nil
		}

		// Load main content
		mainContent, err := currentDB.GetContent(rec.ContentCID)
		if err != nil {
			return nil
		}

		// Determine if this is HTML content
		contentStr := string(mainContent)
		if strings.Contains(contentStr, "<html") || strings.Contains(contentStr, "<!DOCTYPE") {
			// This is an HTML file - treat as main entry point
			mainFile := &WebsiteFile{
				Path:        "index.html",
				ContentCID:  rec.ContentCID,
				Content:     mainContent,
				MimeType:    "text/html",
				LastUpdated: time.Now(),
			}
			website.MainHTML = mainFile
			website.Files["index.html"] = mainFile

			// Extract file references and load them
			references := extractFileReferences(mainContent)
			for _, ref := range references {
				// For now, we'll look for files with the same name in the database
				// In a full implementation, these would be separate blockchain transactions
				fmt.Printf("BROWSER: Found reference to file: %s\n", ref)

				// TODO: Implement loading of referenced files from blockchain
				// This would involve:
				// 1. Looking up file transactions by path/name
				// 2. Loading their content
				// 3. Adding them to the website.Files map
			}
		} else {
			// This is not HTML - treat as single file content
			mainFile := &WebsiteFile{
				Path:        "content",
				ContentCID:  rec.ContentCID,
				Content:     mainContent,
				MimeType:    getMimeType(siteID),
				LastUpdated: time.Now(),
			}
			website.Files["content"] = mainFile
		}

		return website
	}

	// Function to render a website in a tab
	renderWebsite = func(website *Website, tab *Tab) {
		if website == nil || website.MainHTML == nil {
			// Not an HTML website, show as text
			if len(website.Files) > 0 {
				for _, file := range website.Files {
					tab.Content.ParseMarkdown("# " + tab.Address + "\n\n```\n" + string(file.Content) + "\n```")
					break
				}
			}
			return
		}

		// For now, we'll show HTML content in the rich text widget
		// In a full implementation, this would use webview for proper rendering
		htmlContent := string(website.MainHTML.Content)

		// Create a formatted display of the HTML content
		displayContent := fmt.Sprintf(`# %s

## 🌐 Multi-file Website Loaded

**Files Found:** %d

### HTML Content Preview:
`+"```"+`html
%s
`+"```"+`

### Referenced Files:
%s

*Note: This is a preview. Full HTML rendering with CSS/JS support is coming soon.*`,
			tab.Address,
			len(website.Files),
			htmlContent,
			func() string {
				if len(website.Files) > 1 {
					var fileList []string
					for path := range website.Files {
						if path != "index.html" {
							fileList = append(fileList, "- "+path)
						}
					}
					return strings.Join(fileList, "\n")
				}
				return "*No additional files found*"
			}())

		tab.Content.ParseMarkdown(displayContent)
	}

	// Helper function to check if string is hex
	isHexString := func(s string) bool {
		for _, char := range s {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
		return true
	}

	// Name resolution for .bn domains using the decentralized domain system
	resolveName := func(name string) string {
		// If it's already a site ID (hex string), return as-is
		if len(name) == 64 && isHexString(name) {
			return name
		}

		// If it ends with .bn, try to resolve it
		if strings.HasSuffix(name, ".bn") {
			if currentDB != nil {
				if siteID, err := currentDB.ResolveDomain(name); err == nil {
					fmt.Printf("BROWSER: Resolved %s -> %s\n", name, siteID)
					return siteID
				} else {
					fmt.Printf("BROWSER: Domain resolution failed for %s: %v\n", name, err)
					return name // Return as-is if resolution fails
				}
			}
		}

		// Return as-is if no resolution possible
		return name
	}

	// Auto-start a local node for the browser
	initBrowser := func() {
		fmt.Println("BROWSER: Initializing decentralized network...")

		// Use specified data directory or create default
		dir := dataDir
		if dir == filepath.Join(os.Getenv("HOME"), ".betanet", "browser") {
			_ = os.MkdirAll(dir, 0o755)
		}

		db, err := store.Open(dir)
		if err != nil {
			fmt.Printf("BROWSER: Database error: %v\n", err)
			if len(tabs) > 0 {
				tabs[0].Content.ParseMarkdown("# Connection Error\n\nCould not initialize browser database.")
			}
			return
		}

		ctx := context.Background()

		// Start our own local node
		fmt.Println("BROWSER: Starting local network node...")

		node, err := p2p.New(ctx, db, "/ip4/0.0.0.0/tcp/4001", nil, nil)
		if err != nil {
			fmt.Printf("BROWSER: Local node creation failed: %v\n", err)
			if len(tabs) > 0 {
				tabs[0].Content.ParseMarkdown("# Network Error\n\nCould not create local network node.")
			}
			return
		}

		if err := node.Start(ctx); err != nil {
			fmt.Printf("BROWSER: Local node creation failed: %v\n", err)
			if len(tabs) > 0 {
				tabs[0].Content.ParseMarkdown("# Network Error\n\nCould not start local network node.")
			}
			return
		}

		currentNode = node
		currentDB = db
		fmt.Printf("BROWSER: Started local node on port 4001\n")

		// Show welcome message in first tab
		if len(tabs) > 0 {
			tabs[0].Content.ParseMarkdown(`# Welcome to Betanet Browser!

## 🌐 Decentralized Web Browser

The browser has started its own local network node on port 4001.

### How to browse:
1. Enter a site ID in the address bar above
2. Click "Go" or press Enter
3. The site content will load below

### Multi-file Website Support:
- **HTML Files**: Main entry points with HTML preview
- **CSS & JavaScript**: Styles and interactive functionality (coming soon)
- **Images & Assets**: Visual content and media files (coming soon)
- **Blockchain Storage**: Each file stored as separate transaction

### To connect to other networks:
- Start another betanet-node on a different port
- The browser will automatically discover it via mDNS
- Or manually enter the node address

*This browser works like Chrome, but for decentralized websites with multi-file support.*`)
		}
	}

	// Modern browser navigation buttons with better styling
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		fmt.Println("BROWSER: Back button clicked")
		// TODO: Implement browser history
	})
	backBtn.Importance = widget.LowImportance

	forwardBtn := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		fmt.Println("BROWSER: Forward button clicked")
		// TODO: Implement browser history
	})
	forwardBtn.Importance = widget.LowImportance

	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		fmt.Println("BROWSER: Refresh button clicked")
		// Reload current site
		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			if tabs[currentTabIndex].Address != "" {
				browseToSite(tabs[currentTabIndex].Address)
			}
		}
	})
	refreshBtn.Importance = widget.LowImportance

	goBtn := widget.NewButton("Go", func() {
		siteAddr := strings.TrimSpace(addressBar.Text)
		if siteAddr != "" {
			browseToSite(siteAddr)
		}
	})
	goBtn.Importance = widget.HighImportance

	// New tab button
	newTabBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		addNewTab("New Tab", "")
	})
	newTabBtn.Importance = widget.LowImportance

	// Function to show settings content
	showSettingsTab = func() {
		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			currentTab := tabs[currentTabIndex]
			currentTab.Content.ParseMarkdown(`# Browser Settings

## 🔧 Configuration Options

### Network Settings
- **Data Directory**: ` + settingsDataDir.Text + `
- **Listen Address**: ` + listen.Text + `
- **Bootstrap Nodes**: ` + bootstrap.Text + `

### Browser Settings
- **Current Node**: ` + fmt.Sprintf("%v", currentNode != nil) + `
- **Database**: ` + fmt.Sprintf("%v", currentDB != nil) + `

### Multi-file Website Support
- **HTML Rendering**: Basic HTML preview (webview coming soon)
- **Asset Loading**: CSS, JavaScript, images from blockchain (coming soon)
- **File References**: Automatic detection and loading

### Advanced Options
- **Node Port**: 4001
- **Discovery**: mDNS enabled

*Settings changes will take effect after restart.*`)
		}
	}

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		fmt.Println("BROWSER: Settings button clicked")
		// Open settings in a new tab
		addNewTab("Settings", "settings://internal")
		showSettingsTab()
	})
	settingsBtn.Importance = widget.LowImportance

	// Browse to site function
	browseToSite = func(siteAddr string) {
		fmt.Printf("BROWSER: Browsing to site: %s\n", siteAddr)

		if currentDB == nil || currentNode == nil {
			if len(tabs) > 0 && currentTabIndex < len(tabs) {
				tabs[currentTabIndex].Content.ParseMarkdown("# Error\n\nBrowser not connected. Please wait for initialization.")
			}
			return
		}

		// Clean site address (remove betanet:// prefix if present)
		cleanAddr := strings.TrimPrefix(siteAddr, "betanet://")
		cleanAddr = strings.TrimSpace(cleanAddr)

		// Update current tab address
		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			tabs[currentTabIndex].Address = cleanAddr
			tabs[currentTabIndex].Title = cleanAddr
			updateTabBar()
		}

		// Resolve .bn domain names to site IDs
		siteID := resolveName(cleanAddr)

		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			tabs[currentTabIndex].Content.ParseMarkdown("# Loading...\n\nSearching for site: " + siteID)
		}

		// Try to get site from local database (check for multi-file websites first)
		hasMultiFile := currentDB.HasWebsiteManifest(siteID)
		hasSingleFile, _ := currentDB.HasHead(siteID)

		if !hasMultiFile && !hasSingleFile {
			if len(tabs) > 0 && currentTabIndex < len(tabs) {
				tabs[currentTabIndex].Content.ParseMarkdown("# Site Not Found\n\nSite **" + siteID + "** is not available in the local cache.\n\nThis could mean:\n- The site doesn't exist\n- The site hasn't been published to this network\n- The site hasn't been cached locally yet\n\n*In a full implementation, this would attempt to fetch from the network.*")
			}
			return
		}

		// Load the complete website
		website := loadWebsite(siteID)
		if website == nil {
			if len(tabs) > 0 && currentTabIndex < len(tabs) {
				tabs[currentTabIndex].Content.ParseMarkdown("# Error\n\nFailed to load website: " + siteID)
			}
			return
		}

		// Store website in current tab
		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			tabs[currentTabIndex].Website = website
		}

		// Display the website
		fmt.Printf("BROWSER: Loaded website %s with %d files\n", siteID, len(website.Files))

		// Render the website
		if len(tabs) > 0 && currentTabIndex < len(tabs) {
			renderWebsite(website, tabs[currentTabIndex])
		}
	}

	// Enter key support for address bar
	addressBar.OnSubmitted = func(text string) {
		if text != "" {
			browseToSite(text)
		}
	}

	// Create modern Chrome-like layout with proper spacing
	// Left side: Navigation buttons (Back, Forward, Refresh)
	leftNav := container.NewHBox(
		backBtn,
		forwardBtn,
		refreshBtn,
	)

	// Right side: Action buttons (New Tab, Go, Settings)
	rightNav := container.NewHBox(
		newTabBtn,
		goBtn,
		settingsBtn,
	)

	// Main navigation bar: Left nav | Address bar (expanding) | Right nav
	navBar := container.NewBorder(
		nil, nil,
		leftNav, rightNav,
		addressBar, // This will expand to fill the available space
	)

	// Create initial tab
	initialTab := createNewTab("Welcome", "")
	tabs = append(tabs, initialTab)
	currentTabIndex = 0
	tabs[0].IsActive = true

	// Set initial tab content with scrollable wrapper
	tabContainer.Objects = []fyne.CanvasObject{initialTab.ScrollContent}

	// Update tab bar
	updateTabBar()

	// Main layout: Nav bar at top, tab bar below, content area filling the rest
	contentArea := container.NewBorder(nil, nil, nil, nil, tabContainer)
	mainLayout := container.NewBorder(navBar, nil, nil, nil, container.NewVBox(tabBar, contentArea))

	w.SetContent(mainLayout)

	// Auto-initialize browser when opened
	go func() {
		time.Sleep(1 * time.Second) // Brief delay to let UI render
		initBrowser()
	}()

	// Cleanup when browser closes
	w.SetOnClosed(func() {
		fmt.Println("BROWSER: Shutting down...")
		if currentNode != nil {
			currentNode.Host.Close()
		}
		if currentDB != nil {
			currentDB.Close()
		}
		fmt.Println("BROWSER: Cleanup complete")
	})

	w.ShowAndRun()
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return out
}
