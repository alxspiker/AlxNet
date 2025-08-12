package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "betanet/internal/core"
    "betanet/internal/p2p"
    "betanet/internal/store"

    "github.com/fxamacker/cbor/v2"
    fyne "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

// Modern browser interface for decentralized web

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
        fmt.Println("The browser will start its own local node using the specified database.")
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
    listen := widget.NewEntry(); listen.SetText("/ip4/0.0.0.0/tcp/0")
    bootstrap := widget.NewEntry()
    bootstrap.SetPlaceHolder("Auto-filled from discovery, or enter manually")
    
    // Browser status - minimal, hidden by default
    _ = ""

    // Modern address bar with better styling
    addressBar := widget.NewEntry()
    addressBar.SetPlaceHolder("Enter site address (like example.bn or site ID)")
    addressBar.TextStyle = fyne.TextStyle{Bold: false}
    
    // Main content area with better styling
    contentArea := widget.NewRichText()
    contentArea.Wrapping = fyne.TextWrapWord
    contentScroll := container.NewScroll(contentArea)
    
    var currentNode *p2p.Node
    var currentDB *store.Store
    
    // Forward declarations
    var browseToSite func(string)
    
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
            contentArea.ParseMarkdown("# Connection Error\n\nCould not initialize browser database.")
            return
        }
        
        ctx := context.Background()
        
        // Start our own local node
        fmt.Println("BROWSER: Starting local network node...")
        
        node, err := p2p.New(ctx, db, "/ip4/0.0.0.0/tcp/4001", nil)
        if err != nil {
            fmt.Printf("BROWSER: Local node creation failed: %v\n", err)
            contentArea.ParseMarkdown("# Network Error\n\nCould not create local network node.")
            return
        }
        
        if err := node.Start(ctx); err != nil {
            fmt.Printf("BROWSER: Local node start failed: %v\n", err)
            contentArea.ParseMarkdown("# Network Error\n\nCould not start local network node.")
            return
        }
        
        currentNode = node
        currentDB = db
        fmt.Printf("BROWSER: Started local node on port 4001\n")
        
        // Show welcome message
        fyne.DoAndWait(func() {
            contentArea.ParseMarkdown(`# Welcome to Betanet Browser!

## 🌐 Decentralized Web Browser

The browser has started its own local network node on port 4001.

### How to browse:
1. Enter a site ID in the address bar above
2. Click "Go" or press Enter
3. The site content will load below

### To connect to other networks:
- Start another betanet-node on a different port
- The browser will automatically discover it via mDNS
- Or manually enter the node address

*This browser works like Chrome, but for decentralized websites.*`)
        })
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
        if addressBar.Text != "" {
            browseToSite(addressBar.Text)
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
    
    settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
        fmt.Println("BROWSER: Settings button clicked")
        // TODO: Show settings dialog
    })
    settingsBtn.Importance = widget.LowImportance
    
    // Browse to site function
    browseToSite = func(siteAddr string) {
        fmt.Printf("BROWSER: Browsing to site: %s\n", siteAddr)
        
        if currentDB == nil || currentNode == nil {
            contentArea.ParseMarkdown("# Error\n\nBrowser not connected. Please wait for initialization.")
            return
        }
        
        // Clean site address (remove betanet:// prefix if present)
        cleanAddr := strings.TrimPrefix(siteAddr, "betanet://")
        cleanAddr = strings.TrimSpace(cleanAddr)
        
        // Resolve .bn domain names to site IDs
        siteID := resolveName(cleanAddr)
        
        contentArea.ParseMarkdown("# Loading...\n\nSearching for site: " + siteID)
        
        // Try to get site from local database
        if has, _ := currentDB.HasHead(siteID); !has {
            contentArea.ParseMarkdown("# Site Not Found\n\nSite **" + siteID + "** is not available in the local cache.\n\nThis could mean:\n- The site doesn't exist\n- The site hasn't been published to this network\n- The site hasn't been cached locally yet\n\n*In a full implementation, this would attempt to fetch from the network.*")
            return
        }
        
        // Get site content
        seq, headCID, err := currentDB.GetHead(siteID)
        if err != nil {
            contentArea.ParseMarkdown("# Error\n\nFailed to load site: " + err.Error())
            return
        }
        
        recBytes, err := currentDB.GetRecord(headCID)
        if err != nil {
            contentArea.ParseMarkdown("# Error\n\nFailed to load site record: " + err.Error())
            return
        }
        
        var rec core.UpdateRecord
        dec, _ := cbor.DecOptions{}.DecMode()
        if err := dec.Unmarshal(recBytes, &rec); err != nil {
            contentArea.ParseMarkdown("# Error\n\nFailed to parse site record: " + err.Error())
            return
        }
        
        content, err := currentDB.GetContent(rec.ContentCID)
        if err != nil {
            contentArea.ParseMarkdown("# Error\n\nFailed to load site content: " + err.Error())
            return
        }
        
        // Display the content
        fmt.Printf("BROWSER: Loaded site %s (seq %d) with %d bytes\n", siteID, seq, len(content))
        
        // If it looks like HTML, show it as-is, otherwise show as markdown
        contentStr := string(content)
        if strings.Contains(contentStr, "<html>") || strings.Contains(contentStr, "<h1>") {
            // Show HTML content as markdown for now (basic rendering)
            contentArea.ParseMarkdown("# " + cleanAddr + "\n\n```html\n" + contentStr + "\n```")
        } else {
            // Show as markdown (use the original address for title)
            contentArea.ParseMarkdown("# " + cleanAddr + "\n\n" + contentStr)
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
    
    // Right side: Action buttons (Go, Settings)
    rightNav := container.NewHBox(
        goBtn,
        settingsBtn,
    )
    
    // Main navigation bar: Left nav | Address bar (expanding) | Right nav
    navBar := container.NewBorder(
        nil, nil, 
        leftNav, rightNav, 
        addressBar, // This will expand to fill the available space
    )
    
    // Main layout: Nav bar at top, content area filling the rest
    w.SetContent(container.NewBorder(navBar, nil, nil, nil, contentScroll))
    
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
    if strings.TrimSpace(s) == "" { return nil }
    parts := strings.Split(s, ",")
    out := make([]string, 0, len(parts))
    for _, p := range parts { if strings.TrimSpace(p) != "" { out = append(out, strings.TrimSpace(p)) } }
    return out
}


