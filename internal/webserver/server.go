package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"alxnet/internal/core"
	"alxnet/internal/p2p"
	"alxnet/internal/store"

	"github.com/fxamacker/cbor/v2"
	"go.uber.org/zap"
)

// WebServer serves web interfaces for AlxNet
type WebServer struct {
	store  *store.Store
	node   *p2p.Node
	logger *zap.Logger
	server *http.Server
	port   int
	ctx    context.Context
	cancel context.CancelFunc
}

// NewBrowserServer creates a new browser web server instance
func NewBrowserServer(store *store.Store, node *p2p.Node, logger *zap.Logger, port int) *WebServer {
	ctx, cancel := context.WithCancel(context.Background())

	ws := &WebServer{
		store:  store,
		node:   node,
		logger: logger,
		port:   port,
		ctx:    ctx,
		cancel: cancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handleWebsite)
	mux.HandleFunc("/api/sites", ws.handleAPISites)
	mux.HandleFunc("/api/site/", ws.handleAPISite)
	mux.HandleFunc("/api/browse/", ws.handleAPIBrowse)
	mux.HandleFunc("/_alxnet/status", ws.handleStatus)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws
}

// Start starts the web server
func (ws *WebServer) Start() error {
	go func() {
		ws.logger.Info("starting alxnet web server", zap.Int("port", ws.port))
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ws.logger.Error("web server error", zap.Error(err))
		}
	}()

	// Wait a moment to ensure server starts
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop stops the web server
func (ws *WebServer) Stop() error {
	ws.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ws.server.Shutdown(ctx)
}

// handleWebsite serves websites by site ID or domain
func (ws *WebServer) handleWebsite(w http.ResponseWriter, r *http.Request) {
	// Parse the URL to extract site ID and file path
	urlParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(urlParts) == 0 || urlParts[0] == "" {
		ws.serveAlxNetHomepage(w, r)
		return
	}

	siteID := urlParts[0]
	filePath := "index.html"

	if len(urlParts) > 1 {
		filePath = strings.Join(urlParts[1:], "/")
	}

	// Validate site ID format
	if len(siteID) != 64 {
		http.Error(w, "Invalid site ID format", http.StatusBadRequest)
		return
	}

	ws.logger.Info("serving website content",
		zap.String("site_id", siteID),
		zap.String("file_path", filePath))

	// Try to get the website content
	content, mimeType, err := ws.getWebsiteFile(siteID, filePath)
	if err != nil {
		ws.logger.Error("failed to get website file",
			zap.String("site_id", siteID),
			zap.String("file_path", filePath),
			zap.Error(err))
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("X-AlxNet-Site-ID", siteID)
	w.Header().Set("X-AlxNet-File-Path", filePath)

	// Enable CORS for API access
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Write content
	if _, err := w.Write(content); err != nil {
		http.Error(w, "Failed to write content", http.StatusInternalServerError)
		return
	}
}

// getWebsiteFile retrieves a file from a website
func (ws *WebServer) getWebsiteFile(siteID, filePath string) ([]byte, string, error) {
	// Check if it's a multi-file website
	if ws.store.HasWebsiteManifest(siteID) {
		return ws.getWebsiteManifestFile(siteID, filePath)
	}

	// Check if it's a traditional single-file site (fallback)
	hasHead, err := ws.store.HasHead(siteID)
	if err != nil {
		return nil, "", err
	}

	if hasHead && filePath == "index.html" {
		return ws.getSingleFileContent(siteID)
	}

	return nil, "", fmt.Errorf("site not found")
}

// getWebsiteManifestFile retrieves a file from a multi-file website
func (ws *WebServer) getWebsiteManifestFile(siteID, filePath string) ([]byte, string, error) {
	// Get website manifest
	manifestData, err := ws.store.GetCurrentWebsiteManifest(siteID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get website manifest: %v", err)
	}

	// Parse manifest
	var manifest core.WebsiteManifest
	dec, _ := cbor.DecOptions{}.DecMode()
	if err := dec.Unmarshal(manifestData, &manifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse website manifest: %v", err)
	}

	// Use main file if no specific file requested or if requesting root
	if filePath == "" || filePath == "/" || filePath == "index.html" {
		filePath = manifest.MainFile
	}

	// Get content CID for the file
	contentCID, exists := manifest.Files[filePath]
	if !exists {
		return nil, "", fmt.Errorf("file not found in website: %s", filePath)
	}

	// Get file content
	content, err := ws.store.GetContent(contentCID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get file content: %v", err)
	}

	// Determine MIME type
	mimeType := ws.getMimeType(filePath)

	return content, mimeType, nil
}

// getSingleFileContent retrieves content from a traditional single-file site
func (ws *WebServer) getSingleFileContent(siteID string) ([]byte, string, error) {
	_, headCID, err := ws.store.GetHead(siteID)
	if err != nil {
		return nil, "", err
	}

	recordData, err := ws.store.GetRecord(headCID)
	if err != nil {
		return nil, "", err
	}

	var record core.UpdateRecord
	dec, _ := cbor.DecOptions{}.DecMode()
	if err := dec.Unmarshal(recordData, &record); err != nil {
		return nil, "", err
	}

	content, err := ws.store.GetContent(record.ContentCID)
	if err != nil {
		return nil, "", err
	}

	return content, "text/html", nil
}

// getMimeType determines the MIME type for a file path
func (ws *WebServer) getMimeType(filePath string) string {
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)

	if mimeType == "" {
		switch ext {
		case ".html", ".htm":
			return "text/html; charset=utf-8"
		case ".css":
			return "text/css; charset=utf-8"
		case ".js":
			return "application/javascript; charset=utf-8"
		case ".json":
			return "application/json; charset=utf-8"
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
		default:
			return "text/plain; charset=utf-8"
		}
	}

	return mimeType
}

// serveAlxNetHomepage serves the AlxNet homepage
func (ws *WebServer) serveAlxNetHomepage(w http.ResponseWriter, r *http.Request) {
	homepage := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AlxNet - Decentralized Web Browser</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: white;
        }
        .container { max-width: 800px; margin: 0 auto; padding: 2rem; }
        .header { text-align: center; margin-bottom: 3rem; }
        .header h1 { font-size: 3rem; margin-bottom: 1rem; }
        .header p { font-size: 1.2rem; opacity: 0.9; }
        .search-box { 
            background: rgba(255,255,255,0.1);
            padding: 2rem;
            border-radius: 10px;
            margin-bottom: 2rem;
            backdrop-filter: blur(10px);
        }
        .search-box input {
            width: 100%;
            padding: 1rem;
            font-size: 1.1rem;
            border: none;
            border-radius: 5px;
            margin-bottom: 1rem;
        }
        .search-box button {
            padding: 1rem 2rem;
            background: #4c51bf;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
        }
        .search-box button:hover { background: #3730a3; }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 2rem;
            margin-bottom: 3rem;
        }
        .feature {
            background: rgba(255,255,255,0.1);
            padding: 1.5rem;
            border-radius: 10px;
            backdrop-filter: blur(10px);
        }
        .feature h3 { margin-bottom: 0.5rem; }
        .api-info {
            background: rgba(255,255,255,0.1);
            padding: 1.5rem;
            border-radius: 10px;
            backdrop-filter: blur(10px);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌐 AlxNet</h1>
            <p>Decentralized Web Browser & Platform</p>
        </div>
        
        <div class="search-box">
            <h2>Browse a Decentralized Website</h2>
            <input type="text" id="siteInput" placeholder="Enter site ID (64 character hex string)" maxlength="64">
            <button onclick="browseSite()">Browse Site</button>
            <p><small>Example: b36c5d32ed19dce14f8f1f279aeede1e2c2ab397e44e8b5d31f89c9320096b33</small></p>
        </div>
        
        <div class="features">
            <div class="feature">
                <h3>🔗 P2P Network</h3>
                <p>Content distributed across peer-to-peer network without central servers</p>
            </div>
            <div class="feature">
                <h3>🌐 Full Websites</h3>
                <p>Complete websites with HTML, CSS, JavaScript, and images</p>
            </div>
            <div class="feature">
                <h3>🔒 Cryptographic Security</h3>
                <p>Content verified using cryptographic signatures</p>
            </div>
            <div class="feature">
                <h3>🚀 Modern Web Standards</h3>
                <p>Support for modern HTML5, CSS3, and JavaScript features</p>
            </div>
        </div>
        
        <div class="api-info">
            <h3>API Endpoints</h3>
            <ul>
                <li><code>/api/sites</code> - List all available sites</li>
                <li><code>/api/site/{siteID}</code> - Get site information</li>
                <li><code>/{siteID}/{filepath}</code> - Browse site content</li>
                <li><code>/_alxnet/status</code> - Server status</li>
            </ul>
        </div>
    </div>
    
    <script>
        function browseSite() {
            const siteID = document.getElementById('siteInput').value.trim();
            if (siteID.length === 64 && /^[a-fA-F0-9]+$/.test(siteID)) {
                window.location.href = '/' + siteID;
            } else {
                alert('Please enter a valid 64-character hexadecimal site ID');
            }
        }
        
        document.getElementById('siteInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                browseSite();
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(homepage)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// handleStatus serves server status information
func (ws *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"server":    "alxnet-webserver",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(time.Now()).String(), // This would need proper tracking
		"node_id":   ws.node.Host.ID().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleAPISites lists all available sites
func (ws *WebServer) handleAPISites(w http.ResponseWriter, r *http.Request) {
	// This would need implementation to list all sites from the store
	sites := []string{}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"sites": sites,
		"count": len(sites),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleAPISite provides information about a specific site
func (ws *WebServer) handleAPISite(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimPrefix(r.URL.Path, "/api/site/")

	if len(siteID) != 64 {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Get website info
	if ws.store.HasWebsiteManifest(siteID) {
		info, err := ws.store.GetWebsiteInfo(siteID)
		if err != nil {
			http.Error(w, "Failed to get site info", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(info); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Site not found", http.StatusNotFound)
	}
}

// handleAPIBrowse provides a browsing interface for sites
func (ws *WebServer) handleAPIBrowse(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimPrefix(r.URL.Path, "/api/browse/")

	if len(siteID) != 64 {
		http.Error(w, "Invalid site ID", http.StatusBadRequest)
		return
	}

	// Return information about browsing the site
	result := map[string]interface{}{
		"site_id": siteID,
		"url":     fmt.Sprintf("http://localhost:%d/%s", ws.port, siteID),
		"status":  "available",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
