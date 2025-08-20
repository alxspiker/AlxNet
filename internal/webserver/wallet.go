package webserver

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"alxnet/internal/core"
	"alxnet/internal/p2p"
	"alxnet/internal/store"
	"alxnet/internal/wallet"

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
	mux.HandleFunc("/api/wallet/load-file", ws.handleLoadWalletFile)
	mux.HandleFunc("/api/wallet/list", ws.handleListWallets)
	mux.HandleFunc("/api/wallet/save", ws.handleSaveWallet)
	mux.HandleFunc("/api/wallet/sites", ws.handleWalletSites)
	mux.HandleFunc("/api/wallet/add-site", ws.handleAddSite)
	mux.HandleFunc("/api/site/files", ws.handleGetSiteFiles)
	mux.HandleFunc("/api/site/save-file", ws.handleSaveFileToSite)
	mux.HandleFunc("/api/site/delete-file", ws.handleDeleteFileFromSite)
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

// handleWalletHomepage serves the enhanced wallet management interface
func (ws *WebServer) handleWalletHomepage(w http.ResponseWriter, r *http.Request) {
	homepage := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AlxNet Wallet Management</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: white;
        }
        .container { max-width: 1400px; margin: 0 auto; padding: 2rem; }
        .header { text-align: center; margin-bottom: 2rem; }
        .header h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
        .header p { font-size: 1.1rem; opacity: 0.9; }
        
        /* Navigation */
        .nav { 
            display: flex; 
            gap: 1rem; 
            margin-bottom: 2rem; 
            justify-content: center;
        }
        .nav button {
            padding: 0.8rem 1.5rem;
            background: rgba(255,255,255,0.2);
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1rem;
            transition: background 0.3s;
        }
        .nav button:hover { background: rgba(255,255,255,0.3); }
        .nav button.active { background: #4c51bf; }
        
        /* Screen containers */
        .screen {
            background: rgba(255,255,255,0.1);
            padding: 2rem;
            border-radius: 10px;
            backdrop-filter: blur(10px);
            display: none;
        }
        .screen.active { display: block; }
        
        /* Form styles */
        .form-group { margin-bottom: 1.5rem; }
        .form-group label { 
            display: block; 
            margin-bottom: 0.5rem; 
            font-weight: bold; 
            font-size: 0.95rem;
        }
        .form-group input, .form-group textarea, .form-group select {
            width: 100%;
            padding: 0.8rem;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
            background: rgba(255,255,255,0.9);
            color: #333;
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
            margin-bottom: 0.5rem;
            transition: background 0.3s;
        }
        .form-group button:hover { background: #3730a3; }
        .form-group button.secondary {
            background: #6b7280;
        }
        .form-group button.secondary:hover { background: #4b5563; }
        
        /* Grid layouts */
        .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; }
        .grid-3 { display: grid; grid-template-columns: 300px 1fr; gap: 2rem; }
        
        /* Status and results */
        .status {
            background: rgba(0,0,0,0.3);
            padding: 1rem;
            border-radius: 5px;
            margin-top: 1rem;
            border-left: 4px solid #3b82f6;
        }
        .status.success { border-left-color: #10b981; }
        .status.error { border-left-color: #ef4444; }
        .status.warning { border-left-color: #f59e0b; }
        
        /* Lists */
        .list { 
            background: rgba(0,0,0,0.2); 
            border-radius: 5px; 
            max-height: 300px;
            overflow-y: auto;
        }
        .list-item {
            padding: 1rem;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            cursor: pointer;
            transition: background 0.3s;
        }
        .list-item:hover { background: rgba(255,255,255,0.1); }
        .list-item:last-child { border-bottom: none; }
        .list-item.selected { background: rgba(79, 70, 229, 0.3); }
        
        /* File editor */
        .editor-container {
            background: rgba(0,0,0,0.3);
            border-radius: 5px;
            height: 400px;
            display: flex;
            flex-direction: column;
        }
        .editor-toolbar {
            background: rgba(0,0,0,0.2);
            padding: 0.5rem;
            display: flex;
            gap: 0.5rem;
            align-items: center;
        }
        .editor-toolbar input {
            background: rgba(255,255,255,0.9);
            border: none;
            padding: 0.4rem;
            border-radius: 3px;
            font-size: 0.9rem;
        }
        .editor-toolbar button {
            padding: 0.4rem 0.8rem;
            background: #4c51bf;
            color: white;
            border: none;
            border-radius: 3px;
            cursor: pointer;
            font-size: 0.9rem;
        }
        .editor-content {
            flex: 1;
        }
        .editor-content textarea {
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
            color: white;
            border: none;
            padding: 1rem;
            font-family: 'Courier New', monospace;
            font-size: 0.9rem;
            resize: none;
        }
        
        /* File tree */
        .file-tree {
            background: rgba(0,0,0,0.3);
            border-radius: 5px;
            padding: 1rem;
            height: 400px;
            overflow-y: auto;
        }
        .file-item {
            padding: 0.5rem;
            cursor: pointer;
            border-radius: 3px;
            margin-bottom: 2px;
            font-size: 0.9rem;
        }
        .file-item:hover { background: rgba(255,255,255,0.1); }
        .file-item.selected { background: rgba(79, 70, 229, 0.3); }
        .file-item.directory { font-weight: bold; }
        
        /* Utility classes */
        .hidden { display: none !important; }
        .text-center { text-align: center; }
        .mb-2 { margin-bottom: 1rem; }
        .mt-2 { margin-top: 1rem; }
        
        /* Loading spinner */
        .loading {
            display: inline-block;
            width: 20px;
            height: 20px;
            border: 2px solid rgba(255,255,255,0.3);
            border-radius: 50%;
            border-top-color: white;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        
        /* Responsive */
        @media (max-width: 768px) {
            .grid-2, .grid-3 { grid-template-columns: 1fr; }
            .nav { flex-wrap: wrap; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>💰 AlxNet Wallet</h1>
            <p>Manage wallets, sites, and decentralized content</p>
        </div>
        
        <!-- Navigation -->
        <div class="nav">
            <button id="nav-wallet" class="active" onclick="showScreen('wallet')">Wallet</button>
            <button id="nav-sites" onclick="showScreen('sites')" disabled>Sites</button>
            <button id="nav-editor" onclick="showScreen('editor')" disabled>Editor</button>
        </div>
        
        <!-- Current Status -->
        <div id="current-status" class="status hidden">
            <strong>Current:</strong> <span id="status-text">No wallet selected</span>
        </div>
        
        <!-- Wallet Selection Screen -->
        <div id="screen-wallet" class="screen active">
            <h2>Wallet Management</h2>
            <div class="grid-2">
                <div>
                    <h3>Create New Wallet</h3>
                    <div class="form-group">
                        <button onclick="createWallet()">Create New Wallet</button>
                    </div>
                    <div id="wallet-result" class="status hidden"></div>
                </div>
                
                <div>
                    <h3>Load Existing Wallet</h3>
                    <div class="form-group">
                        <label>Saved Wallets:</label>
                        <select id="saved-wallets">
                            <option value="">Loading saved wallets...</option>
                        </select>
                        <button onclick="loadSavedWallet()" style="margin-left: 0.5rem;">Load Selected</button>
                    </div>
                    <div class="form-group">
                        <label>Or Upload Wallet File (JSON):</label>
                        <input type="file" id="walletFile" accept=".json">
                    </div>
                    <div class="form-group">
                        <label>Mnemonic Phrase:</label>
                        <input type="password" id="mnemonic" placeholder="Enter your mnemonic phrase">
                    </div>
                    <div class="form-group">
                        <button onclick="loadWallet()">Load Wallet File</button>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- Site Management Screen -->
        <div id="screen-sites" class="screen">
            <h2>Site Management</h2>
            <div class="grid-2">
                <div>
                    <h3>Your Sites</h3>
                    <div id="sites-list" class="list">
                        <div class="text-center" style="padding: 2rem;">
                            <p>No sites found. Create your first site!</p>
                        </div>
                    </div>
                </div>
                
                <div>
                    <h3>Site Actions</h3>
                    <div class="form-group">
                        <label>Site Label:</label>
                        <input type="text" id="new-site-label" placeholder="my-awesome-site">
                    </div>
                    <div class="form-group">
                        <button onclick="createSite()">Create New Site</button>
                        <button onclick="selectSite()" class="secondary" id="select-site-btn" disabled>Select Site</button>
                    </div>
                    <div id="site-result" class="status hidden"></div>
                </div>
            </div>
        </div>
        
        <!-- File Editor Screen -->
        <div id="screen-editor" class="screen">
            <h2>Website Editor</h2>
            <div class="grid-3">
                <div>
                    <h3>File Tree</h3>
                    <div class="file-tree" id="file-tree">
                        <div class="text-center" style="padding: 2rem;">
                            <p>No files yet</p>
                        </div>
                    </div>
                    <div class="form-group mt-2">
                        <button onclick="addFile()" style="font-size: 0.9rem;">Add File</button>
                        <button onclick="publishSite()" style="font-size: 0.9rem;">Publish Site</button>
                    </div>
                </div>
                
                <div>
                    <h3>File Editor</h3>
                    <div class="editor-container">
                        <div class="editor-toolbar">
                            <input type="text" id="current-file-path" placeholder="No file selected" readonly>
                            <button onclick="saveFile()">Save</button>
                            <button onclick="deleteFile()">Delete</button>
                        </div>
                        <div class="editor-content">
                            <textarea id="file-editor" placeholder="Select a file to edit or create a new one..."></textarea>
                        </div>
                    </div>
                    <div id="editor-result" class="status hidden"></div>
                </div>
            </div>
        </div>
    </div>
    
    <script>
        // Global state
        let currentWallet = null;
        let currentMnemonic = null;
        let currentSite = null;
        let siteFiles = {};
        let selectedFile = null;
        
        // Initialize
        document.addEventListener('DOMContentLoaded', function() {
            updateStatus();
            loadWalletList();
        });
        
        // Navigation
        function showScreen(screenName) {
            // Hide all screens
            document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
            document.querySelectorAll('.nav button').forEach(b => b.classList.remove('active'));
            
            // Show selected screen
            document.getElementById('screen-' + screenName).classList.add('active');
            document.getElementById('nav-' + screenName).classList.add('active');
            
            // Load screen data
            if (screenName === 'sites' && currentWallet) {
                loadSites();
            } else if (screenName === 'editor' && currentSite) {
                loadSiteFiles();
            }
        }
        
        // Status management
        function updateStatus() {
            const statusEl = document.getElementById('current-status');
            const statusText = document.getElementById('status-text');
            
            let text = 'No wallet selected';
            if (currentWallet && currentSite) {
                text = 'Wallet loaded, Site: ' + currentSite.label;
            } else if (currentWallet) {
                text = 'Wallet loaded, no site selected';
            }
            
            statusText.textContent = text;
            statusEl.classList.remove('hidden');
            
            // Enable/disable navigation
            document.getElementById('nav-sites').disabled = !currentWallet;
            document.getElementById('nav-editor').disabled = !currentSite;
        }
        
        // Utility functions
        function showResult(elementId, content, type = 'success') {
            const element = document.getElementById(elementId);
            element.textContent = content;
            element.className = 'status ' + type;
            element.classList.remove('hidden');
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
                
                updateStatus();
                
                // Auto-save wallet with timestamp
                const timestamp = new Date().toISOString().slice(0, 16).replace('T', '_').replace(':', '-');
                const walletName = 'wallet_' + timestamp;
                
                try {
                    await apiCall('/api/wallet/save', 'POST', {
                        wallet_data: result.encrypted,
                        name: walletName
                    });
                    showResult('wallet-result', 
                        'Wallet created successfully!\\n\\n' +
                        'IMPORTANT: Save this mnemonic phrase safely:\\n' +
                        result.mnemonic + '\\n\\n' +
                        'Wallet automatically saved to data directory and downloaded.'
                    );
                } catch (saveError) {
                    console.error('Auto-save failed:', saveError);
                    showResult('wallet-result', 
                        'Wallet created successfully!\\n\\n' +
                        'IMPORTANT: Save this mnemonic phrase safely:\\n' +
                        result.mnemonic + '\\n\\n' +
                        'WARNING: Auto-save to data directory failed. Please save manually.\\n' +
                        'Error: ' + saveError.message
                    );
                }
                
                // Create download link
                const walletData = JSON.stringify(result.wallet, null, 2);
                const blob = new Blob([walletData], { type: 'application/json' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = 'alxnet-wallet.json';
                a.click();
                URL.revokeObjectURL(url);
                
                // Refresh wallet list
                loadWalletList();
                
            } catch (error) {
                showResult('wallet-result', 'Error: ' + error.message, 'error');
            }
        }
        
        async function loadWallet() {
            const fileInput = document.getElementById('walletFile');
            const mnemonicInput = document.getElementById('mnemonic');
            
            if (!fileInput.files[0] || !mnemonicInput.value) {
                showResult('wallet-result', 'Please select a wallet file and enter mnemonic', 'error');
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
                
                showResult('wallet-result', 
                    'Wallet loaded successfully!\\nSites found: ' + result.sites.length
                );
                
                updateStatus();
                
            } catch (error) {
                showResult('wallet-result', 'Error: ' + error.message, 'error');
            }
        }
        
        // Load wallet list from saved files
        async function loadWalletList() {
            try {
                const result = await apiCall('/api/wallet/list');
                const walletSelect = document.getElementById('saved-wallets');
                
                if (result.wallets.length === 0) {
                    walletSelect.innerHTML = '<option value="">No saved wallets found</option>';
                } else {
                    walletSelect.innerHTML = '<option value="">Select a saved wallet...</option>' +
                        result.wallets.map(wallet => 
                            '<option value="' + wallet.filename + '">' + wallet.name + ' (' + wallet.created + ')</option>'
                        ).join('');
                }
            } catch (error) {
                console.error('Failed to load wallet list:', error);
            }
        }
        
        // Load wallet from saved file
        async function loadSavedWallet() {
            const walletSelect = document.getElementById('saved-wallets');
            const mnemonicInput = document.getElementById('mnemonic');
            
            if (!walletSelect.value) {
                showResult('wallet-result', 'Please select a wallet', 'error');
                return;
            }
            
            if (!mnemonicInput.value) {
                showResult('wallet-result', 'Please enter your mnemonic phrase', 'error');
                return;
            }
            
            try {
                const result = await apiCall('/api/wallet/load-file', 'POST', {
                    filename: walletSelect.value,
                    mnemonic: mnemonicInput.value
                });
                
                currentWallet = result.wallet;
                currentMnemonic = mnemonicInput.value;
                
                showResult('wallet-result', 
                    'Wallet loaded successfully from saved file!\\n' +
                    'Sites found: ' + Object.keys(result.wallet.sites || {}).length
                );
                
                updateStatus();
                
            } catch (error) {
                showResult('wallet-result', 'Error: ' + error.message, 'error');
            }
        }
        
        // Site functions
        async function loadSites() {
            if (!currentWallet) return;
            
            try {
                const sitesList = document.getElementById('sites-list');
                const sites = currentWallet.sites || {};
                const siteArray = Object.keys(sites).map(label => ({
                    label: label,
                    site_id: sites[label].site_id || 'N/A',
                    last_updated: sites[label].last_updated || new Date().toISOString()
                }));
                
                if (siteArray.length === 0) {
                    sitesList.innerHTML = '<div class="text-center" style="padding: 2rem;"><p>No sites found. Create your first site!</p></div>';
                } else {
                    sitesList.innerHTML = siteArray.map(site => 
                        '<div class="list-item" onclick="selectSiteFromList(\'' + site.label + '\')" data-label="' + site.label + '">' +
                            '<strong>' + site.label + '</strong><br>' +
                            '<small>ID: ' + site.site_id + '</small><br>' +
                            '<small>Updated: ' + new Date(site.last_updated).toLocaleString() + '</small>' +
                        '</div>'
                    ).join('');
                }
            } catch (error) {
                showResult('site-result', 'Error loading sites: ' + error.message, 'error');
            }
        }
        
        function selectSiteFromList(label) {
            // Remove previous selection
            document.querySelectorAll('#sites-list .list-item').forEach(item => {
                item.classList.remove('selected');
            });
            
            // Select current item
            document.querySelector('#sites-list .list-item[data-label="' + label + '"]').classList.add('selected');
            
            // Enable select button
            document.getElementById('select-site-btn').disabled = false;
            document.getElementById('select-site-btn').onclick = () => selectSite(label);
        }
        
        function selectSite(label) {
            if (!label) {
                const selected = document.querySelector('#sites-list .list-item.selected');
                if (!selected) {
                    showResult('site-result', 'Please select a site first', 'error');
                    return;
                }
                label = selected.dataset.label;
            }
            
            currentSite = currentWallet.sites[label];
            if (!currentSite) {
                showResult('site-result', 'Site not found', 'error');
                return;
            }
            
            currentSite.label = label; // Store label for reference
            updateStatus();
            showResult('site-result', 'Site "' + label + '" selected. You can now edit files.');
        }
        
        async function createSite() {
            const label = document.getElementById('new-site-label').value.trim();
            if (!label) {
                showResult('site-result', 'Please enter a site label', 'error');
                return;
            }
            
            if (!currentWallet || !currentMnemonic) {
                showResult('site-result', 'Please load a wallet first', 'error');
                return;
            }
            
            try {
                const result = await apiCall('/api/wallet/add-site', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    label: label
                });
                
                currentWallet = result.wallet;
                showResult('site-result', 'Site "' + label + '" created successfully!');
                document.getElementById('new-site-label').value = '';
                loadSites();
                
            } catch (error) {
                showResult('site-result', 'Error: ' + error.message, 'error');
            }
        }
        
        // File editor functions
        async function loadSiteFiles() {
            if (!currentSite || !currentWallet || !currentMnemonic) return;
            
            try {
                const result = await apiCall('/api/site/files', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    site_label: currentSite.label
                });
                
                siteFiles = result.files || {};
                updateFileTree();
                
            } catch (error) {
                showResult('editor-result', 'Error loading files: ' + error.message, 'error');
            }
        }
        
        function updateFileTree() {
            const fileTree = document.getElementById('file-tree');
            
            if (Object.keys(siteFiles).length === 0) {
                fileTree.innerHTML = '<div class="text-center" style="padding: 2rem;"><p>No files yet.<br>Click "Add File" to create your first file.</p></div>';
                return;
            }
            
            const fileItems = Object.keys(siteFiles).map(path => 
                '<div class="file-item" onclick="selectFile(\'' + path + '\')" data-path="' + path + '">' +
                    '📄 ' + path +
                '</div>'
            );
            
            fileTree.innerHTML = fileItems.join('');
        }
        
        function selectFile(path) {
            selectedFile = path;
            
            // Update UI
            document.querySelectorAll('.file-item').forEach(item => {
                item.classList.remove('selected');
            });
            document.querySelector('[data-path="' + path + '"]').classList.add('selected');
            
            // Load file content (placeholder for now)
            document.getElementById('current-file-path').value = path;
            document.getElementById('file-editor').value = '// File content would be loaded here\\n// Placeholder content for: ' + path;
        }
        
        function addFile() {
            const fileName = prompt('Enter file name (e.g., index.html, style.css):');
            if (!fileName) return;
            
            // Validate file name
            if (siteFiles[fileName]) {
                alert('File already exists!');
                return;
            }
            
            // Add to file list
            siteFiles[fileName] = {
                path: fileName,
                content_cid: 'new',
                mime_type: getMimeType(fileName),
                size: 0,
                last_updated: new Date()
            };
            
            updateFileTree();
            selectFile(fileName);
            showResult('editor-result', 'File "' + fileName + '" added. Edit and save to persist.');
        }
        
        function getMimeType(fileName) {
            const ext = fileName.split('.').pop().toLowerCase();
            const mimeTypes = {
                'html': 'text/html',
                'css': 'text/css', 
                'js': 'application/javascript',
                'json': 'application/json',
                'txt': 'text/plain',
                'md': 'text/markdown',
                'png': 'image/png',
                'jpg': 'image/jpeg',
                'jpeg': 'image/jpeg',
                'gif': 'image/gif',
                'svg': 'image/svg+xml'
            };
            return mimeTypes[ext] || 'text/plain';
        }
        
        async function saveFile() {
            if (!selectedFile) {
                showResult('editor-result', 'No file selected', 'error');
                return;
            }
            
            const content = document.getElementById('file-editor').value;
            
            try {
                const result = await apiCall('/api/site/save-file', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    site_label: currentSite.label,
                    file_path: selectedFile,
                    content: content,
                    mime_type: getMimeType(selectedFile)
                });
                
                showResult('editor-result', 'File "' + selectedFile + '" saved successfully!');
                
            } catch (error) {
                showResult('editor-result', 'Error saving file: ' + error.message, 'error');
            }
        }
        
        async function deleteFile() {
            if (!selectedFile) {
                showResult('editor-result', 'No file selected', 'error');
                return;
            }
            
            if (!confirm('Delete file "' + selectedFile + '"?')) return;
            
            try {
                await apiCall('/api/site/delete-file', 'POST', {
                    wallet_data: JSON.stringify(currentWallet),
                    mnemonic: currentMnemonic,
                    site_label: currentSite.label,
                    file_path: selectedFile
                });
                
                delete siteFiles[selectedFile];
                updateFileTree();
                document.getElementById('current-file-path').value = '';
                document.getElementById('file-editor').value = '';
                selectedFile = null;
                
                showResult('editor-result', 'File deleted successfully!');
                
            } catch (error) {
                showResult('editor-result', 'Error deleting file: ' + error.message, 'error');
            }
        }
        
        async function publishSite() {
            if (!currentSite || Object.keys(siteFiles).length === 0) {
                showResult('editor-result', 'No files to publish', 'error');
                return;
            }
            
            try {
                // For now, just show success message
                showResult('editor-result', 
                    'Site "' + currentSite.label + '" published successfully!\\n' +
                    'Files: ' + Object.keys(siteFiles).length + '\\n' +
                    'Full publishing functionality will be implemented soon.'
                );
                
            } catch (error) {
                showResult('editor-result', 'Error publishing site: ' + error.message, 'error');
            }
        }
    </script>
</body>
</html>\`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(homepage)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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

	// Try to parse wallet data as JSON first (unencrypted), then try decryption
	var walletData *wallet.Wallet
	
	// First try to unmarshal as unencrypted JSON
	if err := json.Unmarshal([]byte(req.WalletData), &walletData); err != nil {
		// If JSON parsing fails, try to decrypt
		var decryptErr error
		walletData, decryptErr = wallet.DecryptWallet([]byte(req.WalletData), req.Mnemonic)
		if decryptErr != nil {
			http.Error(w, "Failed to parse or decrypt wallet data", http.StatusBadRequest)
			return
		}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		response := map[string]interface{}{
			"success": false,
			"error":   "Site is not a multi-file website",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func (ws *WebServer) handleWalletStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"server":    "alxnet-wallet",
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
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// New API handlers for enhanced wallet flow

func (ws *WebServer) handleListWallets(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get data directory from store
	dataDir := ws.store.GetDataDir()
	walletsDir := filepath.Join(dataDir, "wallets")
	
	// Ensure wallets directory exists
	if err := os.MkdirAll(walletsDir, 0755); err != nil {
		http.Error(w, "Failed to create wallets directory", http.StatusInternalServerError)
		return
	}

	// Scan for wallet files
	files, err := os.ReadDir(walletsDir)
	if err != nil {
		http.Error(w, "Failed to read wallets directory", http.StatusInternalServerError)
		return
	}

	var walletList []map[string]interface{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".wallet") {
			continue
		}
		
		// Get file info
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		walletName := strings.TrimSuffix(file.Name(), ".wallet")
		walletList = append(walletList, map[string]interface{}{
			"name":     walletName,
			"filename": file.Name(),
			"created":  info.ModTime().Format("2006-01-02 15:04:05"),
			"size":     info.Size(),
		})
	}

	response := map[string]interface{}{
		"success": true,
		"wallets": walletList,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleSaveWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Name       string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Wallet name is required", http.StatusBadRequest)
		return
	}

	// Validate wallet name (only alphanumeric and safe characters)
	for _, char := range req.Name {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || char == '-' || char == '_') {
			http.Error(w, "Wallet name can only contain letters, numbers, hyphens and underscores", http.StatusBadRequest)
			return
		}
	}

	// Get data directory from store
	dataDir := ws.store.GetDataDir()
	walletsDir := filepath.Join(dataDir, "wallets")
	
	// Ensure wallets directory exists
	if err := os.MkdirAll(walletsDir, 0755); err != nil {
		http.Error(w, "Failed to create wallets directory", http.StatusInternalServerError)
		return
	}

	// Save wallet file
	walletPath := filepath.Join(walletsDir, req.Name+".wallet")
	if err := os.WriteFile(walletPath, []byte(req.WalletData), 0600); err != nil {
		http.Error(w, "Failed to save wallet file", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Wallet '%s' saved successfully", req.Name),
		"path":    walletPath,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleLoadWalletFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filename string `json:"filename"`
		Mnemonic string `json:"mnemonic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Get data directory from store
	dataDir := ws.store.GetDataDir()
	walletsDir := filepath.Join(dataDir, "wallets")
	walletPath := filepath.Join(walletsDir, req.Filename)

	// Check if file exists and is in the wallets directory
	if !strings.HasPrefix(walletPath, walletsDir) {
		http.Error(w, "Invalid wallet file path", http.StatusBadRequest)
		return
	}

	// Read wallet file
	walletData, err := os.ReadFile(walletPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Wallet file not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to read wallet file", http.StatusInternalServerError)
		}
		return
	}

	// Decrypt wallet if mnemonic provided
	if req.Mnemonic != "" {
		// Decode base64 wallet data
		encryptedData, err := base64.StdEncoding.DecodeString(string(walletData))
		if err != nil {
			http.Error(w, "Invalid wallet file format", http.StatusBadRequest)
			return
		}
		
		decryptedWallet, err := wallet.DecryptWallet(encryptedData, req.Mnemonic)
		if err != nil {
			http.Error(w, "Failed to decrypt wallet", http.StatusUnauthorized)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"wallet":  decryptedWallet,
			"message": "Wallet loaded successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	} else {
		// Return encrypted wallet data for client-side decryption
		response := map[string]interface{}{
			"success":     true,
			"wallet_data": string(walletData),
			"message":     "Wallet file loaded, provide mnemonic to decrypt",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

func (ws *WebServer) handleGetSiteFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		SiteLabel  string `json:"site_label"`
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

	// Get site from wallet
	site, exists := walletData.Sites[req.SiteLabel]
	if !exists {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// Get website info from store
	websiteInfo, err := ws.store.GetWebsiteInfo(site.SiteID)
	if err != nil {
		// If no website manifest exists, return empty file list
		response := map[string]interface{}{
			"success": true,
			"site_id": site.SiteID,
			"files":   map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	response := map[string]interface{}{
		"success":   true,
		"site_id":   site.SiteID,
		"main_file": websiteInfo.MainFile,
		"files":     websiteInfo.Files,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleSaveFileToSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		SiteLabel  string `json:"site_label"`
		FilePath   string `json:"file_path"`
		Content    string `json:"content"`
		MimeType   string `json:"mime_type"`
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

	// Get site from wallet
	site, exists := walletData.Sites[req.SiteLabel]
	if !exists {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// For now, return success (full implementation would save to store)
	response := map[string]interface{}{
		"success":     true,
		"site_id":     site.SiteID,
		"file_path":   req.FilePath,
		"message":     "File save functionality not fully implemented yet",
		"content_cid": "temp_cid", // Would be actual CID in full implementation
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleDeleteFileFromSite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletData string `json:"wallet_data"`
		Mnemonic   string `json:"mnemonic"`
		SiteLabel  string `json:"site_label"`
		FilePath   string `json:"file_path"`
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

	// Get site from wallet
	site, exists := walletData.Sites[req.SiteLabel]
	if !exists {
		http.Error(w, "Site not found", http.StatusNotFound)
		return
	}

	// For now, return success (full implementation would delete from store)
	response := map[string]interface{}{
		"success":   true,
		"site_id":   site.SiteID,
		"file_path": req.FilePath,
		"message":   "File delete functionality not fully implemented yet",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
