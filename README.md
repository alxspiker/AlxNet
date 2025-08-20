# 🌐 AlxNet - Decentralized Web Platform

**A decentralized web platform that supports multi-file websites with HTML, CSS, JavaScript, and images - all stored on the blockchain with cryptographic security.**

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Working%20Prototype-orange.svg)]()
[![Security](https://img.shields.io/badge/Security-Hardened-red.svg)]()

## 🚀 What is AlxNet?

AlxNet is a **decentralized, censorship-resistant web platform** that replaces traditional centralized web infrastructure with peer-to-peer technology. This implementation provides:

- **🌍 Multi-File Websites** - Complete websites with HTML, CSS, JavaScript, and images
- **🔐 Cryptographic Identity** - Ed25519-based site ownership and updates
- **💼 Secure Wallet System** - BIP-39 mnemonic-based site management
- **🔄 Peer-to-Peer Networking** - libp2p-based distributed content delivery
- **🔒 Content Encryption** - Optional passphrase-based content protection
- **🎨 Unified Dashboard** - Single interface for all system operations

## 🛡️ **SECURITY & ROBUSTNESS FEATURES**

### **🔒 Enterprise-Grade Security**
- **Input Validation** - Comprehensive validation of all data structures and file paths
- **Rate Limiting** - Configurable rate limiting to prevent abuse and DoS attacks
- **Peer Validation** - Reputation-based peer management with automatic banning
- **Content Size Limits** - Configurable limits to prevent resource exhaustion
- **Path Traversal Protection** - Prevents malicious file path attacks
- **File Extension Whitelisting** - Only allows safe, web-standard file types
- **Memory Management** - Automatic cleanup and memory usage limits
- **Clock Skew Protection** - Prevents timestamp-based attacks

### **🛠️ Production-Ready Infrastructure**
- **Structured Logging** - Production-grade logging with zap logger
- **Configuration Management** - Centralized, validated configuration system
- **Retry Logic** - Automatic retry with exponential backoff for database operations
- **Resource Monitoring** - Memory usage tracking and automatic cleanup
- **Error Handling** - Comprehensive error handling with detailed error messages
- **Testing Coverage** - Extensive test suite with edge case coverage
- **Security Auditing** - Built-in security scanning and vulnerability detection

### **📊 Performance & Reliability**
- **Connection Pooling** - Efficient database connection management
- **Background Cleanup** - Automatic cleanup of old content and expired bans
- **Peer Reputation System** - Intelligent peer selection and management
- **Memory Leak Prevention** - LRU-based content cleanup and memory limits
- **Configurable Timeouts** - Adjustable timeouts for all network operations
- **Graceful Degradation** - System continues operating under adverse conditions

## ✨ Key Features

### 🌐 **Multi-File Website Support**
- **Complete websites** - HTML, CSS, JavaScript, images, and more
- **File organization** - Hierarchical file structure with proper MIME types
- **Main entry point** - Configurable index.html or main file
- **Asset management** - All files stored as separate blockchain transactions
- **Website manifests** - Cryptographic records linking all website files

### 🔐 **Cryptographic Security**
- **Ed25519 signatures** - Fast, secure digital signatures
- **Deterministic key derivation** - BIP-39 mnemonic → site keys
- **Content integrity** - SHA-256 content addressing
- **Update validation** - Cryptographic proof of site ownership
- **Link signatures** - Proof that update keys are authorized

### 🎯 **Unified User Experience**
- **Single Dashboard** - All operations accessible from one interface
- **Wallet Management** - Complete site and key management
- **Node Control** - Full network node operations
- **Network Monitoring** - Real-time peer and health monitoring
- **Web Browsing** - Browse decentralized sites directly

## 🛡️ **SECURITY ARCHITECTURE**

### **Multi-Layer Defense**
```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│  • Input validation • Rate limiting • Access control       │
├─────────────────────────────────────────────────────────────┤
│                    Network Layer                            │
│  • Peer validation • Reputation system • Connection limits │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
│  • Content validation • Size limits • Path sanitization    │
├─────────────────────────────────────────────────────────────┤
│                    Cryptographic Layer                      │
│  • Ed25519 signatures • Content hashing • Key derivation   │
└─────────────────────────────────────────────────────────────┘
```

### **Security Constants & Limits**
- **Max Content Size**: 10MB (configurable)
- **Max File Count**: 1,000 files per website
- **Max Path Length**: 255 characters
- **Rate Limit**: 100 requests per minute per peer
- **Ban Duration**: 15 minutes (configurable)
- **Memory Limit**: 100MB per node (configurable)
- **Max Peers**: 100 concurrent connections

### **Threat Mitigation**
- **Path Traversal**: Blocked through strict path validation
- **Resource Exhaustion**: Prevented through size and count limits
- **DoS Attacks**: Mitigated through rate limiting and peer banning
- **Malicious Content**: Filtered through file type whitelisting
- **Memory Attacks**: Prevented through usage limits and cleanup
- **Clock Attacks**: Blocked through timestamp validation

## 🛠️ Installation

### Prerequisites
- **Go 1.23+** - [Download from golang.org](https://golang.org/dl/)
- **Linux dependencies** (for GUI):
  ```bash
  sudo apt update
  sudo apt install -y libgl1-mesa-dev xorg-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
  ```

### Quick Install
```bash
git clone https://github.com/yourusername/alxnet.git
cd alxnet
./build.sh
```

**The enhanced build script automatically:**
- ✅ Runs comprehensive tests with race detection
- ✅ Performs security scanning (gosec, staticcheck)
- ✅ Applies security build flags (PIE, stripped binaries)
- ✅ Generates coverage reports
- ✅ Validates code quality

## 🚀 Quick Start

### 🌐 **Single Unified Command** - Complete Platform Integration

**Start the complete AlxNet platform with one command:**

```bash
# Start the complete platform (all services integrated)
./alxnet start

# Or customize ports for multiple instances
./alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082
```

**🎯 What happens when you run `alxnet start`:**
- **P2P Network Node** automatically starts on port 4001 (or specified port)
- **🌐 Browser Interface** launches at http://localhost:8080 - Browse decentralized websites
- **💰 Wallet Management** launches at http://localhost:8081 - Create/manage wallets and sites  
- **🔗 Node Dashboard** launches at http://localhost:8082 - Monitor P2P node status

### **Creating Your First Website**

1. **Start AlxNet**: `./alxnet start`
2. **Open Wallet Interface**: Navigate to http://localhost:8081
3. **Create Wallet**: Click "Create New Wallet" → Save the mnemonic phrase securely
4. **Navigate to Sites**: Click "Sites" tab → Create your first site
5. **Open Editor**: Click "Editor" tab → Add files (index.html, style.css, etc.)
6. **Publish**: Click "Publish Site" to deploy to the decentralized network
7. **Browse**: Visit http://localhost:8080 and enter your site ID to view
## 🎯 Core Components

### **🚀 alxnet** - Unified Platform Command

The **single command** that provides the complete AlxNet experience:

**🌐 Browser Interface (Port 8080)**
- Browse decentralized websites by Site ID
- Rich content rendering (HTML, CSS, JavaScript, images)
- Navigation controls and multi-site support
- Automatic P2P node integration

**💰 Wallet Management Interface (Port 8081)**  
- **Professional multi-screen workflow**: Wallet → Sites → Editor
- **Visual file management**: File tree, syntax-aware editor, add/delete files
- **Wallet operations**: Create wallets with mnemonic generation, secure storage
- **Site management**: Create sites, manage cryptographic keys
- **Multi-file publishing**: Complete website publishing with HTML/CSS/JS/images
- **Domain registration**: Human-readable `.alx` domain names

**🔗 Node Management Interface (Port 8082)**
- **Real-time P2P monitoring**: Live node status and connection metrics  
- **Peer management**: Connected peers display with auto-refresh
- **Storage statistics**: Network health and protocol information
- **Performance tracking**: Uptime monitoring and network statistics

**⚡ Integrated P2P Node** 
- **Automatic startup**: Shared P2P node across all interfaces
- **Peer discovery**: mDNS and bootstrap peer discovery  
- **Content storage**: BadgerDB-based persistent storage with multi-file website support
- **Security features**: Rate limiting, peer validation, memory management
## 🔧 Advanced Usage

### **Multi-Instance Development Setup**
```bash
# Start first instance (development)
./alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082

# Start second instance (testing) 
./alxnet start -node-port 4002 -browser-port 8090 -wallet-port 8091 -node-ui-port 8092
```

### **Configuration Management**
```bash
# Environment-based configuration
export BETANET_ENV=production
export BETANET_LOG_LEVEL=warn
export BETANET_MAX_PEERS=200
export BETANET_MAX_CONTENT_SIZE=20971520  # 20MB

# Or use configuration file
./bin/alxnet-node run -config config.yaml
```

**Example config.yaml:**
```yaml
environment: production
log_level: warn

network:
  listen_addr: "/ip4/0.0.0.0/tcp/4001"
  max_peers: 200
  peer_timeout: 60s

security:
  max_content_size: 20971520  # 20MB
  rate_limit: 200
  ban_duration: 30m

storage:
  data_dir: "/var/lib/alxnet"
  max_retries: 5
  cleanup_interval: 10m
```

### **Multi-File Website Structure**
```bash
# Example website directory structure
mywebsite/
├── index.html          # Main entry point
├── css/
│   ├── style.css      # Main stylesheet
│   └── responsive.css # Responsive design
├── js/
│   ├── app.js         # Main application logic
│   └── utils.js       # Utility functions
├── images/
│   ├── logo.png       # Website logo
│   └── favicon.ico    # Browser icon
└── assets/
    └── data.json      # Static data files
```

### **Website Publishing Workflow**
```bash
# 1. Create website directory
mkdir -p mywebsite/{css,js,images}

# 2. Add website files
echo "<!DOCTYPE html>..." > mywebsite/index.html
echo "body { ... }" > mywebsite/css/style.css
echo "console.log('...');" > mywebsite/js/app.js

# 3. Use the dashboard to publish
./launch-dashboard.sh
# Navigate to Wallet Tab → Publish Website
```

### **Domain Name System**
```bash
# All domain management is now done through the web interface:
# 1. Start AlxNet: ./alxnet start  
# 2. Open Wallet Management: http://localhost:8081
# 3. Create/load wallet → Create site → Register domain through UI

# Domain format validation
# ✅ Valid: mysite.alx, blog123.alx, news2024.alx
# ❌ Invalid: my-site.alx, site.alx, my.site.alx
```

### **Content Encryption**
```bash
# All content publishing is now done through the integrated web interface:
# 1. Open Wallet Management: http://localhost:8081
# 2. Navigate to Editor tab → Create files → Publish
# 3. Encryption options available in the publishing interface
```

### **Network Configuration**
```bash
# Start with custom node configuration
./alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082

# Connect to specific bootstrap peer (bootstrap peer management through Node UI)
./alxnet start -bootstrap /ip4/127.0.0.1/tcp/4002/p2p/12D3KooW...

# Start second instance for testing
./alxnet start -node-port 4002 -browser-port 8090 -wallet-port 8091 -node-ui-port 8092
```

## 🏗️ Architecture

### **Unified Platform Architecture**
```
┌─────────────────────────────────────────────────────────────────┐
│                        alxnet start                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Browser UI    │  │   Wallet UI     │  │   Node UI       │ │
│  │   (Port 8080)   │  │   (Port 8081)   │  │   (Port 8082)   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                │                               │
│                    ┌─────────────────┐                        │
│                    │   Shared P2P    │                        │
│                    │   Node Core     │                        │
│                    │   (Port 4001)   │                        │
│                    └─────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   BadgerDB      │
                    │  (Storage)      │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │ alxnet-dashboard│
                    │ (Unified UI)    │
                    └─────────────────┘
```

### **Multi-File Website Data Flow**
1. **Website Creation** → Wallet creates site and website manifest
2. **File Publishing** → Individual files stored with cryptographic signatures
3. **Manifest Updates** → Website manifest links all files together
4. **Domain Registration** → Wallet registers `.alx` domain
5. **Content Distribution** → GossipSub distributes to peers
6. **Content Discovery** → Browser resolves domain to site ID
7. **Website Retrieval** → Node fetches all website files and manifest

### **Security Model**
- **Site Keys** - Long-term Ed25519 keys for site ownership
- **Update Keys** - Ephemeral keys for each content update
- **Link Signatures** - Proof that update key is authorized
- **Update Signatures** - Proof that content is authentic
- **Content Integrity** - SHA-256 hashing prevents tampering
- **Website Manifests** - Cryptographic linking of all website files

## 🔍 Troubleshooting

### **Common Issues**

**"Port already in use"**
```bash
# Kill any running AlxNet instances
pkill -f alxnet

# Start with different ports
./alxnet start -node-port 4002 -browser-port 8090 -wallet-port 8091 -node-ui-port 8092
```

**"Cannot connect to peer network"**
```bash
# Check if AlxNet is running
./alxnet start

# Monitor node status through Node Management interface
# Open: http://localhost:8082

# Use bootstrap peers for testing
./alxnet start -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
```

**"Website files not loading"**
```bash
# All website management is now through the web interface:
# 1. Open Wallet Management: http://localhost:8081  
# 2. Load your wallet and navigate to Editor tab
# 3. Verify all files are present in the file tree
# 4. Use "Publish Site" to update the network
```

### **Debug Mode**
```bash
# Enable verbose logging
export BETANET_DEBUG=1
./alxnet start

# Monitor through web interfaces:
# - Node status: http://localhost:8082
# - Network health: Real-time monitoring available in Node UI
```

## 🧪 Testing

### **🤖 Automated Playwright Testing**
The unified web interface is **continuously tested** with Playwright:

- **🌐 UI Integration Testing** - Tests complete wallet → sites → editor workflow
- **🤖 Automated E2E Testing** - Validates all user interactions
- **📊 Real-Time Monitoring** - Continuous validation of web interface functionality
- **🌍 Cross-Browser Testing** - Ensures compatibility across all browsers
- **🛡️ Production Validation** - Ensures enterprise-grade UI reliability

**Run tests locally:**
```bash
npm test
```

### **Security Testing**
```bash
# Run security tests
go test -v ./internal/core -run TestSecurity

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### **Local Development Setup**
```bash
# Terminal 1: Start first AlxNet instance (development)
./alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082

# Terminal 2: Start second instance (testing)  
./alxnet start -node-port 4002 -browser-port 8090 -wallet-port 8091 -node-ui-port 8092

# Open web interfaces:
# Instance 1: http://localhost:8081 (wallet), http://localhost:8080 (browser)
# Instance 2: http://localhost:8091 (wallet), http://localhost:8090 (browser)
```

## 📚 API Reference

### **Store Interface**
```go
type Store interface {
    // Content management
    PutContent(cid string, data []byte) error
    GetContent(cid string) ([]byte, error)
    DeleteContent(cid string) error
    
    // Record management
    PutRecord(cid string, data []byte) error
    GetRecord(cid string) ([]byte, error)
    DeleteRecord(cid string) error
    
    // Site management
    SetHead(siteID string, seq uint64, headCID string) error
    GetHead(siteID string) (uint64, string, error)
    HasHead(siteID string) (bool, error)
    
    // Domain management
    RegisterDomain(domain string, siteID string, ownerPub []byte) error
    ResolveDomain(domain string) (string, error)
    ListDomains() ([]string, error)
    GetDomainOwner(domain string) ([]byte, error)
    
    // Website management
    PutWebsiteManifest(siteID string, manifest []byte) error
    GetWebsiteManifest(siteID string) ([]byte, error)
    PutFileRecord(siteID string, path string, record []byte) error
    GetFileRecord(siteID string, path string) ([]byte, error)
}
```

### **Node Interface**
```go
type Node interface {
    // Network operations
    Start(ctx context.Context) error
    Host() host.Host
    
    // Content operations
    BuildUpdate(sitePriv ed25519.PrivateKey, sitePub ed25519.PublicKey, content []byte, seq uint64, prevRecCID string) (*GossipUpdate, string, error)
    BroadcastUpdate(ctx context.Context, env GossipUpdate) error
    BroadcastDelete(ctx context.Context, del core.DeleteRecord) error
    
    // Website operations
    PublishWebsite(sitePriv ed25519.PrivateKey, sitePub ed25519.PublicKey, dir string, mainFile string) error
    AddWebsiteFile(sitePriv ed25519.PrivateKey, sitePub ed25519.PublicKey, path string, content []byte) error
    
    // Discovery
    DiscoverBestPeer(ctx context.Context, timeout time.Duration) (*peer.AddrInfo, error)
    DiscoverLocalhostPeer(ctx context.Context) (*peer.AddrInfo, error)
    
    // Browse protocol
    RequestHead(ctx context.Context, p peer.AddrInfo, siteID string) (uint64, string, string, error)
    RequestContent(ctx context.Context, p peer.AddrInfo, cid string) ([]byte, error)
}
```

## 🤝 Contributing

### **Development Setup**
```bash
git clone https://github.com/yourusername/alxnet.git
cd alxnet
go mod tidy
go test ./...
```

### **Code Style**
- **Go formatting** - Use `gofmt` and `go vet`
- **Error handling** - Return errors, don't panic
- **Documentation** - Comment exported functions
- **Testing** - Write tests for new features

### **Architecture Principles**
- **Modular design** - Clear separation of concerns
- **Interface-based** - Use interfaces for testability
- **Error handling** - Graceful degradation
- **Security first** - Cryptographic validation everywhere
- **Multi-file support** - Complete website functionality

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **libp2p** - Peer-to-peer networking library
- **BadgerDB** - Fast key-value storage
- **Fyne** - Cross-platform GUI toolkit
- **Ed25519** - Fast, secure digital signatures
- **BIP-39** - Mnemonic phrase standard
- **CBOR** - Compact binary object representation
- **zap** - Structured logging library
- **yaml.v3** - YAML configuration parsing

## 📞 Support

- **Issues** - [GitHub Issues](https://github.com/yourusername/alxnet/issues)
- **Discussions** - [GitHub Discussions](https://github.com/yourusername/alxnet/discussions)
- **Documentation** - [Wiki](https://github.com/yourusername/alxnet/wiki)

---

**🌐 AlxNet - Building the decentralized web with complete multi-file websites.**

*Now featuring a unified dashboard for complete system management - all operations accessible from one intuitive interface.*
