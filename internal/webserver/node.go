package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"alxnet/internal/p2p"
	"alxnet/internal/store"

	"go.uber.org/zap"
)

// NewNodeServer creates a new node management web server
func NewNodeServer(store *store.Store, node *p2p.Node, logger *zap.Logger, port int) *WebServer {
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

	// Node management endpoints
	mux.HandleFunc("/", ws.handleNodeHomepage)
	mux.HandleFunc("/api/node/status", ws.handleNodeStatus)
	mux.HandleFunc("/api/node/peers", ws.handleNodePeers)
	mux.HandleFunc("/api/node/info", ws.handleNodeInfo)
	mux.HandleFunc("/api/storage/stats", ws.handleStorageStats)
	mux.HandleFunc("/api/storage/sites", ws.handleStorageSites)
	mux.HandleFunc("/api/storage/domains", ws.handleStorageDomains)
	mux.HandleFunc("/api/network/bootstrap", ws.handleNetworkBootstrap)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws
}

// handleNodeHomepage serves the node management interface
func (ws *WebServer) handleNodeHomepage(w http.ResponseWriter, r *http.Request) {
	homepage := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AlxNet Node Management</title>
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
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 2rem; }
        .metric {
            background: rgba(0,0,0,0.3);
            padding: 1rem;
            border-radius: 5px;
            margin-bottom: 1rem;
        }
        .metric-label { font-size: 0.9rem; opacity: 0.8; }
        .metric-value { font-size: 1.5rem; font-weight: bold; }
        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 0.5rem;
        }
        .status-online { background: #22c55e; }
        .status-offline { background: #ef4444; }
        .peer-list {
            max-height: 300px;
            overflow-y: auto;
            background: rgba(0,0,0,0.3);
            padding: 1rem;
            border-radius: 5px;
        }
        .peer-item {
            padding: 0.5rem;
            border-bottom: 1px solid rgba(255,255,255,0.1);
            font-family: monospace;
            font-size: 0.9rem;
        }
        .refresh-btn {
            padding: 0.5rem 1rem;
            background: #4c51bf;
            color: white;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            margin-bottom: 1rem;
        }
        .refresh-btn:hover { background: #3730a3; }
        .auto-refresh {
            margin-left: 1rem;
        }
        .chart {
            height: 200px;
            background: rgba(0,0,0,0.3);
            border-radius: 5px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: rgba(255,255,255,0.6);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔗 Node Management</h1>
            <p>Monitor and manage your AlxNet P2P node</p>
        </div>
        
        <div class="section">
            <h2>Node Status</h2>
            <div class="grid">
                <div>
                    <div class="metric">
                        <div class="metric-label">Status</div>
                        <div class="metric-value" id="nodeStatus">
                            <span class="status-indicator status-online"></span>Online
                        </div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Node ID</div>
                        <div class="metric-value" id="nodeId" style="font-size: 1rem; font-family: monospace;">Loading...</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Uptime</div>
                        <div class="metric-value" id="uptime">Loading...</div>
                    </div>
                </div>
                <div>
                    <div class="metric">
                        <div class="metric-label">Connected Peers</div>
                        <div class="metric-value" id="peerCount">0</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Listen Addresses</div>
                        <div class="metric-value" id="listenAddrs" style="font-size: 0.9rem; font-family: monospace;">Loading...</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Protocol Version</div>
                        <div class="metric-value" id="protocolVersion">1.0.0</div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="grid">
            <div class="section">
                <h2>Connected Peers</h2>
                <button class="refresh-btn" onclick="loadPeers()">Refresh</button>
                <label class="auto-refresh">
                    <input type="checkbox" id="autoRefreshPeers" onchange="toggleAutoRefresh()"> Auto-refresh (10s)
                </label>
                <div class="peer-list" id="peerList">
                    <div>Loading peers...</div>
                </div>
            </div>
            
            <div class="section">
                <h2>Storage Statistics</h2>
                <button class="refresh-btn" onclick="loadStorageStats()">Refresh</button>
                <div id="storageStats">
                    <div class="metric">
                        <div class="metric-label">Total Sites</div>
                        <div class="metric-value" id="totalSites">0</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Total Domains</div>
                        <div class="metric-value" id="totalDomains">0</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Storage Used</div>
                        <div class="metric-value" id="storageUsed">0 MB</div>
                    </div>
                    <div class="metric">
                        <div class="metric-label">Content Files</div>
                        <div class="metric-value" id="contentFiles">0</div>
                    </div>
                </div>
            </div>
        </div>
        
        <div class="grid">
            <div class="section">
                <h2>Recent Activity</h2>
                <div class="chart">
                    <div>Activity chart placeholder</div>
                </div>
            </div>
            
            <div class="section">
                <h2>Network Health</h2>
                <div class="chart">
                    <div>Network health chart placeholder</div>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>Recent Sites</h2>
            <button class="refresh-btn" onclick="loadRecentSites()">Refresh</button>
            <div id="recentSites">
                <div>Loading recent sites...</div>
            </div>
        </div>
    </div>
    
    <script>
        let autoRefreshInterval = null;
        
        // Load initial data
        document.addEventListener('DOMContentLoaded', function() {
            loadNodeStatus();
            loadPeers();
            loadStorageStats();
            loadRecentSites();
        });
        
        async function apiCall(endpoint) {
            try {
                const response = await fetch(endpoint);
                return await response.json();
            } catch (error) {
                console.error('API call failed:', error);
                return null;
            }
        }
        
        async function loadNodeStatus() {
            const status = await apiCall('/api/node/status');
            if (status) {
                document.getElementById('nodeId').textContent = status.node_id || 'Unknown';
                document.getElementById('uptime').textContent = formatUptime(status.uptime_seconds || 0);
                
                if (status.listen_addresses) {
                    document.getElementById('listenAddrs').innerHTML = 
                        status.listen_addresses.map(addr => '<div>' + addr + '</div>').join('');
                }
            }
        }
        
        async function loadPeers() {
            const peers = await apiCall('/api/node/peers');
            if (peers) {
                document.getElementById('peerCount').textContent = peers.count || 0;
                
                const peerList = document.getElementById('peerList');
                if (peers.peers && peers.peers.length > 0) {
                    peerList.innerHTML = peers.peers.map(peer => 
                        '<div class="peer-item">' + 
                        '<div><strong>ID:</strong> ' + peer.id + '</div>' +
                        '<div><strong>Addr:</strong> ' + (peer.address || 'Unknown') + '</div>' +
                        '<div><strong>Connected:</strong> ' + formatTime(peer.connected_at) + '</div>' +
                        '</div>'
                    ).join('');
                } else {
                    peerList.innerHTML = '<div>No peers connected</div>';
                }
            }
        }
        
        async function loadStorageStats() {
            const stats = await apiCall('/api/storage/stats');
            if (stats) {
                document.getElementById('totalSites').textContent = stats.total_sites || 0;
                document.getElementById('totalDomains').textContent = stats.total_domains || 0;
                document.getElementById('storageUsed').textContent = formatBytes(stats.storage_bytes || 0);
                document.getElementById('contentFiles').textContent = stats.content_files || 0;
            }
        }
        
        async function loadRecentSites() {
            const sites = await apiCall('/api/storage/sites');
            const recentSitesDiv = document.getElementById('recentSites');
            
            if (sites && sites.sites && sites.sites.length > 0) {
                recentSitesDiv.innerHTML = sites.sites.slice(0, 10).map(site => 
                    '<div class="peer-item">' +
                    '<div><strong>Site ID:</strong> ' + site.id + '</div>' +
                    '<div><strong>Last Updated:</strong> ' + formatTime(site.last_updated) + '</div>' +
                    '<div><strong>Files:</strong> ' + (site.file_count || 'N/A') + '</div>' +
                    '</div>'
                ).join('');
            } else {
                recentSitesDiv.innerHTML = '<div>No sites found</div>';
            }
        }
        
        function toggleAutoRefresh() {
            const checkbox = document.getElementById('autoRefreshPeers');
            if (checkbox.checked) {
                autoRefreshInterval = setInterval(() => {
                    loadPeers();
                    loadNodeStatus();
                    loadStorageStats();
                }, 10000);
            } else {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                    autoRefreshInterval = null;
                }
            }
        }
        
        function formatUptime(seconds) {
            if (seconds < 60) return seconds + 's';
            if (seconds < 3600) return Math.floor(seconds / 60) + 'm';
            if (seconds < 86400) return Math.floor(seconds / 3600) + 'h';
            return Math.floor(seconds / 86400) + 'd';
        }
        
        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
        
        function formatTime(timestamp) {
            if (!timestamp) return 'Unknown';
            return new Date(timestamp * 1000).toLocaleString();
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(homepage)); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

// API Handlers for node management

func (ws *WebServer) handleNodeStatus(w http.ResponseWriter, r *http.Request) {
	// Get node addresses
	nodeAddrs := ws.node.Host.Addrs()
	addrs := make([]string, len(nodeAddrs))
	for i, addr := range nodeAddrs {
		addrs[i] = addr.String()
	}

	status := map[string]interface{}{
		"server":           "alxnet-node-ui",
		"version":          "1.0.0",
		"timestamp":        time.Now().Unix(),
		"uptime_seconds":   time.Since(time.Now()).Seconds(), // This would need proper tracking
		"node_id":          ws.node.Host.ID().String(),
		"listen_addresses": addrs,
		"status":           "online",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleNodePeers(w http.ResponseWriter, r *http.Request) {
	peers := ws.node.Host.Network().Peers()
	peerInfos := make([]map[string]interface{}, len(peers))

	for i, peerID := range peers {
		connectedAt := time.Now().Unix() // This would need proper tracking
		peerInfo := map[string]interface{}{
			"id":           peerID.String(),
			"connected_at": connectedAt,
		}

		// Try to get peer address
		conns := ws.node.Host.Network().ConnsToPeer(peerID)
		if len(conns) > 0 {
			peerInfo["address"] = conns[0].RemoteMultiaddr().String()
		}

		peerInfos[i] = peerInfo
	}

	response := map[string]interface{}{
		"success": true,
		"peers":   peerInfos,
		"count":   len(peerInfos),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"node_id":    ws.node.Host.ID().String(),
		"agent_name": "alxnet",
		"version":    "1.0.0",
		"protocols":  []string{"/alxnet/1.0.0"},
		"public_key": ws.node.Host.ID().String(), // This would be the actual public key
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleStorageStats(w http.ResponseWriter, r *http.Request) {
	// Get domain count
	domains, err := ws.store.ListDomains()
	domainCount := 0
	if err == nil {
		domainCount = len(domains)
	}

	// Basic storage stats - in a real implementation, these would be tracked
	stats := map[string]interface{}{
		"success":       true,
		"total_sites":   0, // Would need to implement site counting
		"total_domains": domainCount,
		"storage_bytes": 0, // Would need to implement storage tracking
		"content_files": 0, // Would need to implement file counting
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleStorageSites(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would query the store for all sites
	sites := []map[string]interface{}{
		// This would be populated from actual store data
	}

	response := map[string]interface{}{
		"success": true,
		"sites":   sites,
		"count":   len(sites),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleStorageDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := ws.store.ListDomains()
	if err != nil {
		http.Error(w, "Failed to list domains", http.StatusInternalServerError)
		return
	}

	domainList := make([]map[string]interface{}, 0, len(domains))
	for domain, siteID := range domains {
		domainList = append(domainList, map[string]interface{}{
			"domain":  domain,
			"site_id": siteID,
		})
	}

	response := map[string]interface{}{
		"success": true,
		"domains": domainList,
		"count":   len(domainList),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleNetworkBootstrap(w http.ResponseWriter, r *http.Request) {
	// Return information about bootstrap nodes
	response := map[string]interface{}{
		"success":        true,
		"bootstrap_mode": false, // This would be determined by node configuration
		"known_peers":    0,     // This would be the count of known peers
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
