package webserver

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"betanet/internal/core"
	"betanet/internal/p2p"
	"betanet/internal/store"
	"betanet/internal/wallet"

	"go.uber.org/zap"
)

// NewWalletServer creates a new wallet management web server
func NewWalletServer(store *store.Store, node *p2p.Node, logger *zap.Logger, port int) *WebServer {
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
	
	// Wallet management endpoints
	mux.HandleFunc("/", ws.handleWalletHomepage)
	mux.HandleFunc("/api/wallet/new", ws.handleCreateWallet)
	mux.HandleFunc("/api/wallet/load", ws.handleLoadWallet)
	mux.HandleFunc("/api/wallet/sites", ws.handleWalletSites)
	mux.HandleFunc("/api/wallet/add-site", ws.handleAddSite)
	mux.HandleFunc("/api/wallet/publish", ws.handlePublishContent)
	mux.HandleFunc("/api/wallet/publish-website", ws.handlePublishWebsite)
	mux.HandleFunc("/api/wallet/add-file", ws.handleAddWebsiteFile)
	mux.HandleFunc("/api/wallet/export-key", ws.handleExportKey)
	mux.HandleFunc("/api/domains/register", ws.handleRegisterDomain)
	mux.HandleFunc("/api/domains/list", ws.handleListDomains)
	mux.HandleFunc("/api/domains/resolve", ws.handleResolveDomain)
	mux.HandleFunc("/api/websites/info", ws.handleGetWebsiteInfo)
	mux.HandleFunc("/api/status", ws.handleWalletStatus)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws
}

// handleWalletHomepage serves the wallet management interface
func (ws *WebServer) handleWalletHomepage(w http.ResponseWriter, r *http.Request) {
	homepage := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Betanet Wallet Management</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: white;
        }
        .container { max-width: 1200px; margin: 0 auto; padding: 2rem; }
        .header { text-align: center; margin-bottom: 3rem; }
        .header h1 { font-size: 3rem; margin-bottom: 1rem; }
        .header p { font-size: 1.2rem; opacity: 0.9; }
        .section {
            background: rgba(255,255,255,0.1);
            padding: 2rem;
            border-radius: 10px;
            margin-bottom: 2rem;
            backdrop-filter: blur(10px);
        }
        .section h2 { margin-bottom: 1rem; }
        .form-group { margin-bottom: 1rem; }
        .form-group label { display: block; margin-bottom: 0.5rem; font-weight: bold; }
        .form-group input, .form-group textarea, .form-group select {
            width: 100%;
            padding: 0.8rem;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
        }
        .form-group button {
            padding: 1rem 2rem;
            background: #4c51bf;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
            margin-right: 1rem;
        }
        .form-group button:hover { background: #3730a3; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 2rem; }
        .result {
            background: rgba(0,0,0,0.3);
            padding: 1rem;
            border-radius: 5px;
            margin-top: 1rem;
            white-space: pre-wrap;
            font-family: monospace;
        }
        .hidden { display: none; }
        .success { background: rgba(34, 197, 94, 0.3); }
        .error { background: rgba(239, 68, 68, 0.3); }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💰 Betanet Wallet</h1>
            <p>Manage wallets, sites, and decentralized content</p>
        </div>
        
        <div class="grid">
            <!-- Wallet Management -->
            <div class="section">
                <h2>Wallet Management</h2>
                <div class="form-group">
                    <button onclick="createWallet()">Create New Wallet</button>
                    <button onclick="document.getElementById('loadWalletSection').classList.toggle('hidden')">Load Wallet</button>
                </div>
                
                <div id="loadWalletSection" class="hidden">
                    <div class="form-group">
                        <label>Wallet File (JSON):</label>
                        <input type="file" id="walletFile" accept=".json">
                    </div>
                    <div class="form-group">
                        <label>Mnemonic:</label>
                        <input type="password" id="mnemonic" placeholder="Enter your mnemonic phrase">
                    </div>
                    <div class="form-group">
                        <button onclick="loadWallet()">Load Wallet</button>
                    </div>
                </div>
                
                <div id="walletResult" class="result hidden"></div>
            </div>
            
            <!-- Site Management -->
            <div class="section">
                <h2>Site Management</h2>
                <div class="form-group">
                    <label>Site Label:</label>
                    <input type="text" id="siteLabel" placeholder="mysite">
                </div>
                <div class="form-group">
                    <button onclick="addSite()">Add New Site</button>
                    <button onclick="listSites()">List Sites</button>
                </div>
                <div id="siteResult" class="result hidden"></div>
            </div>
            
            <!-- Content Publishing -->
            <div class="section">
                <h2>Content Publishing</h2>
                <div class="form-group">
                    <label>Site Label:</label>
                    <input type="text" id="publishSiteLabel" placeholder="mysite">
                </div>
                <div class="form-group">
                    <label>Content Type:</label>
                    <select id="contentType">
                        <option value="text">Text Content</option>
                        <option value="file">File Upload</option>
                        <option value="website">Website Directory</option>
                    </select>
                </div>
                <div id="textContentDiv" class="form-group">
                    <label>Content:</label>
                    <textarea id="textContent" rows="5" placeholder="Enter your content here..."></textarea>
                </div>
                <div id="fileContentDiv" class="form-group hidden">
                    <label>File:</label>
                    <input type="file" id="contentFile">
                </div>
                <div id="websiteContentDiv" class="form-group hidden">
                    <label>Website Files:</label>
                    <input type="file" id="websiteFiles" multiple webkitdirectory>
                    <small>Select a directory containing your website files</small>
                </div>
                <div class="form-group">
                    <button onclick="publishContent()">Publish Content</button>
                </div>
                <div id="publishResult" class="result hidden"></div>
            </div>
            
            <!-- Domain Management -->
            <div class="section">
                <h2>Domain Management</h2>
                <div class="form-group">
                    <label>Domain:</label>
                    <input type="text" id="domainName" placeholder="mysite.bn">
                </div>
                <div class="form-group">
                    <label>Site Label:</label>
                    <input type="text" id="domainSiteLabel" placeholder="mysite">
                </div>
                <div class="form-group">
                    <button onclick="registerDomain()">Register Domain</button>
                    <button onclick="listDomains()">List Domains</button>
                </div>
                <div id="domainResult" class="result hidden"></div>
            </div>
        </div>
    </div>
    
    <script>
        let currentWallet = null;
        let currentMnemonic = null;
        
        // Utility functions
        function showResult(elementId, content, isError = false) {
            const element = document.getElementById(elementId);
            element.textContent = content;
            element.className = 'result ' + (isError ? 'error' : 'success');
            element.classList.remove('hidden');
        }
        
        function hideResult(elementId) {
            document.getElementById(elementId).classList.add('hidden');
        }
        
        async function apiCall(endpoint, method = 'GET', data = null) {
            try {
                const options = {
                    method,
                    headers: { 'Content-Type': 'application/json' }
                };
                if (data) {
                    options.body = JSON.stringify(data);
                }
                
                const response = await fetch(endpoint, options);
                const result = await response.json();
                
                if (!response.ok) {
                    throw new Error(result.error || 'API call failed');
                }
                
                return result;
            } catch (error) {
                throw new Error('Network error: ' + error.message);
            }
        }
        
        // Wallet functions
        async function createWallet() {
            try {
                const result = await apiCall('/api/wallet/new', 'POST');
                currentWallet = result.wallet;
                currentMnemonic = result.mnemonic;
                
                showResult('walletResult', 
                    'Wallet created successfully!\n\n' +
                    'IMPORTANT: Save this mnemonic phrase safely:\n' +
                    result.mnemonic + '\n\n' +
                    'Download link: ' + result.download_url
                );
            } catch (error) {
                showResult('walletResult', 'Error: ' + error.message, true);
            }
        }
        
        async function loadWallet() {
            const fileInput = document.getElementById('walletFile');
            const mnemonicInput = document.getElementById('mnemonic');
            
            if (!fileInput.files[0] || !mnemonicInput.value) {
                showResult('walletResult', 'Please select a wallet file and enter mnemonic', true);
                return;
            }
            
            try {
                const walletData = await fileInput.files[0].text();
                const result = await apiCall('/api/wallet/load', 'POST', {
                    wallet_data: walletData,
                    mnemonic: mnemonicInput.value
                });
                
                currentWallet = result.wallet;
                currentMnemonic = mnemonicInput.value;
                
                showResult('walletResult', 'Wallet loaded successfully!\nSites: ' + result.sites.length);
            } catch (error) {
                showResult('walletResult', 'Error: ' + error.message, true);
            }
        }
        
        // Site functions
        async function addSite() {
            if (!currentWallet || !currentMnemonic) {
                showResult('siteResult', 'Please load a wallet first', true);
                return;
            }
            
            const label = document.getElementById('siteLabel').value;
            if (!label) {
                showResult('siteResult', 'Please enter a site label', true);
                return;
            }
            
            try {
                const result = await apiCall('/api/wallet/add-site', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    label: label
                });
                
                currentWallet = result.wallet;
                showResult('siteResult', 'Site added successfully!\nSite ID: ' + result.site_id);
            } catch (error) {
                showResult('siteResult', 'Error: ' + error.message, true);
            }
        }
        
        async function listSites() {
            if (!currentWallet || !currentMnemonic) {
                showResult('siteResult', 'Please load a wallet first', true);
                return;
            }
            
            try {
                const result = await apiCall('/api/wallet/sites', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic
                });
                
                let output = 'Sites in wallet:\n';
                result.sites.forEach(site => {
                    output += '  ' + site.label + ': ' + site.site_id + '\n';
                });
                
                showResult('siteResult', output);
            } catch (error) {
                showResult('siteResult', 'Error: ' + error.message, true);
            }
        }
        
        // Publishing functions
        async function publishContent() {
            if (!currentWallet || !currentMnemonic) {
                showResult('publishResult', 'Please load a wallet first', true);
                return;
            }
            
            const siteLabel = document.getElementById('publishSiteLabel').value;
            const contentType = document.getElementById('contentType').value;
            
            if (!siteLabel) {
                showResult('publishResult', 'Please enter a site label', true);
                return;
            }
            
            try {
                let result;
                
                if (contentType === 'text') {
                    const content = document.getElementById('textContent').value;
                    if (!content) {
                        showResult('publishResult', 'Please enter content', true);
                        return;
                    }
                    
                    result = await apiCall('/api/wallet/publish', 'POST', {
                        wallet_data: JSON.stringify(currentWallet),
                        mnemonic: currentMnemonic,
                        label: siteLabel,
                        content: content
                    });
                } else if (contentType === 'file') {
                    const fileInput = document.getElementById('contentFile');
                    if (!fileInput.files[0]) {
                        showResult('publishResult', 'Please select a file', true);
                        return;
                    }
                    
                    const content = await fileInput.files[0].text();
                    result = await apiCall('/api/wallet/publish', 'POST', {
                        wallet_data: JSON.stringify(currentWallet),
                        mnemonic: currentMnemonic,
                        label: siteLabel,
                        content: content
                    });
                } else if (contentType === 'website') {
                    const fileInput = document.getElementById('websiteFiles');
                    if (!fileInput.files.length) {
                        showResult('publishResult', 'Please select website files', true);
                        return;
                    }
                    
                    // Handle website publishing (simplified for now)
                    showResult('publishResult', 'Website publishing not yet implemented in UI', true);
                    return;
                }
                
                showResult('publishResult', 
                    'Content published successfully!\n' +
                    'Site ID: ' + result.site_id + '\n' +
                    'Content CID: ' + result.content_cid
                );
            } catch (error) {
                showResult('publishResult', 'Error: ' + error.message, true);
            }
        }
        
        // Domain functions
        async function registerDomain() {
            if (!currentWallet || !currentMnemonic) {
                showResult('domainResult', 'Please load a wallet first', true);
                return;
            }
            
            const domain = document.getElementById('domainName').value;
            const siteLabel = document.getElementById('domainSiteLabel').value;
            
            if (!domain || !siteLabel) {
                showResult('domainResult', 'Please enter domain and site label', true);
                return;
            }
            
            try {
                const result = await apiCall('/api/domains/register', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    domain: domain,
                    label: siteLabel
                });
                
                showResult('domainResult', 
                    'Domain registered successfully!\n' +
                    'Domain: ' + domain + '\n' +
                    'Site ID: ' + result.site_id
                );
            } catch (error) {
                showResult('domainResult', 'Error: ' + error.message, true);
            }
        }
        
        async function listDomains() {
            try {
                const result = await apiCall('/api/domains/list');
                
                let output = 'Registered domains:\n';
                Object.entries(result.domains).forEach(([domain, siteId]) => {
                    output += '  ' + domain + ' -> ' + siteId + '\n';
                });
                
                showResult('domainResult', output);
            } catch (error) {
                showResult('domainResult', 'Error: ' + error.message, true);
            }
        }
        
        // Content type change handler
        document.getElementById('contentType').addEventListener('change', function() {
            const contentType = this.value;
            document.getElementById('textContentDiv').classList.toggle('hidden', contentType !== 'text');
            document.getElementById('fileContentDiv').classList.toggle('hidden', contentType !== 'file');
            document.getElementById('websiteContentDiv').classList.toggle('hidden', contentType !== 'website');
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(homepage))
}

// API Handlers for wallet operations

func (ws *WebServer) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create new wallet
	mnemonic, err := wallet.NewMnemonic()
	if err != nil {
		http.Error(w, "Failed to generate mnemonic", http.StatusInternalServerError)
		return
	}

	walletData := wallet.New()
	encryptedWallet, err := wallet.EncryptWallet(walletData, mnemonic)
	if err != nil {
		http.Error(w, "Failed to encrypt wallet", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":      true,
		"mnemonic":     mnemonic,
		"wallet":       walletData,
		"encrypted":    base64.StdEncoding.EncodeToString(encryptedWallet),
		"download_url": "/api/wallet/download", // TODO: implement download endpoint
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleLoadWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Parse wallet data
	encryptedWallet := []byte(req.WalletData)
	walletData, err := wallet.DecryptWallet(encryptedWallet, req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"wallet":  walletData,
		"sites":   walletData.Sites,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleWalletSites(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"sites":   walletData.Sites,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleAddSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		Label      string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	// Generate master key
	master, err := wallet.MasterKeyFromMnemonic(req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to generate master key", http.StatusInternalServerError)
		return
	}

	// Add site
	meta, _, _, err := walletData.EnsureSite(master, req.Label)
	if err != nil {
		http.Error(w, "Failed to add site", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"site_id": meta.SiteID,
		"wallet":  walletData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handlePublishContent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		Label      string `json:"label"`
		Content    string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	// Generate master key
	master, err := wallet.MasterKeyFromMnemonic(req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to generate master key", http.StatusInternalServerError)
		return
	}

	// Ensure site exists
	meta, pub, _, err := walletData.EnsureSite(master, req.Label)
	if err != nil {
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	contentBytes := []byte(req.Content)

	// Create update record
	record := &core.UpdateRecord{
		Version:    "1.0",
		SitePub:    pub,
		Seq:        1,
		PrevCID:    "",
		ContentCID: core.CIDForContent(contentBytes),
		TS:         core.NowTS(),
	}

	// Generate ephemeral key for this update
	updatePub, updatePriv, err := wallet.DeriveSiteKey(master, req.Label+"-update")
	if err != nil {
		http.Error(w, "Failed to derive update key", http.StatusInternalServerError)
		return
	}
	record.UpdatePub = updatePub

	// Sign record
	recordData, err := core.CanonicalMarshalNoUpdateSig(record)
	if err != nil {
		http.Error(w, "Failed to marshal record", http.StatusInternalServerError)
		return
	}
	record.UpdateSig = ed25519.Sign(updatePriv, recordData)

	// Store content and record
	if err := ws.store.PutContent(record.ContentCID, contentBytes); err != nil {
		http.Error(w, "Failed to store content", http.StatusInternalServerError)
		return
	}

	recordCID := core.CIDForBytes(recordData)
	if err := ws.store.PutRecord(recordCID, recordData); err != nil {
		http.Error(w, "Failed to store record", http.StatusInternalServerError)
		return
	}

	// Update site head
	if err := ws.store.PutHead(meta.SiteID, record.Seq, recordCID); err != nil {
		http.Error(w, "Failed to update site head", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"site_id":     meta.SiteID,
		"content_cid": record.ContentCID,
		"record_cid":  recordCID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handlePublishWebsite(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement website publishing with file uploads
	http.Error(w, "Website publishing not yet implemented", http.StatusNotImplemented)
}

func (ws *WebServer) handleAddWebsiteFile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement individual file addition to websites
	http.Error(w, "Add website file not yet implemented", http.StatusNotImplemented)
}

func (ws *WebServer) handleExportKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		Label      string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	// Generate master key
	master, err := wallet.MasterKeyFromMnemonic(req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to generate master key", http.StatusInternalServerError)
		return
	}

	// Get site private key
	_, _, priv, err := walletData.EnsureSite(master, req.Label)
	if err != nil {
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	keyData := base64.StdEncoding.EncodeToString(priv)

	response := map[string]interface{}{
		"success":     true,
		"label":       req.Label,
		"private_key": keyData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleRegisterDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		Domain     string `json:"domain"`
		Label      string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	// Generate master key
	master, err := wallet.MasterKeyFromMnemonic(req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to generate master key", http.StatusInternalServerError)
		return
	}

	// Ensure site exists
	meta, _, _, err := walletData.EnsureSite(master, req.Label)
	if err != nil {
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	// Register domain
	if err := ws.store.PutDomain(req.Domain, meta.SiteID); err != nil {
		http.Error(w, "Failed to register domain", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"domain":  req.Domain,
		"site_id": meta.SiteID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := ws.store.ListDomains()
	if err != nil {
		http.Error(w, "Failed to list domains", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"domains": domains,
		"count":   len(domains),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleResolveDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		http.Error(w, "Domain parameter required", http.StatusBadRequest)
		return
	}

	siteID, err := ws.store.ResolveDomain(domain)
	if err != nil {
		http.Error(w, "Failed to resolve domain", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"domain":  domain,
		"site_id": siteID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) handleGetWebsiteInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		Label      string `json:"label"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Decrypt wallet
	walletData, err := wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
		return
	}

	// Generate master key
	master, err := wallet.MasterKeyFromMnemonic(req.Mnemonic)
	if err != nil {
		http.Error(w, "Failed to generate master key", http.StatusInternalServerError)
		return
	}

	// Get site
	meta, _, _, err := walletData.EnsureSite(master, req.Label)
	if err != nil {
		http.Error(w, "Failed to get site", http.StatusInternalServerError)
		return
	}

	// Check if it's a multi-file website
	if ws.store.HasWebsiteManifest(meta.SiteID) {
		info, err := ws.store.GetWebsiteInfo(meta.SiteID)
		if err != nil {
			http.Error(w, "Failed to get website info", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"info":    info,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		response := map[string]interface{}{
			"success": false,
			"error":   "Site is not a multi-file website",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (ws *WebServer) handleWalletStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"server":    "betanet-wallet",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
		"node_id":   ws.node.Host.ID().String(),
	}

	// Get node addresses
	nodeAddrs := ws.node.Host.Addrs()
	addrs := make([]string, len(nodeAddrs))
	for i, addr := range nodeAddrs {
		addrs[i] = addr.String()
	}
	status["node_addresses"] = addrs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}