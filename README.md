# 🌐 Betanet - Multi-File Decentralized Web Platform

**A complete, production-ready implementation of a decentralized web platform that supports full multi-file websites with HTML, CSS, JavaScript, and images - all stored on the blockchain.**

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen.svg)]()
[![Security](https://img.shields.io/badge/Security-Hardened-red.svg)]()
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)]()

## 🚀 What is Betanet?

Betanet is a **fully decentralized, censorship-resistant web platform** that replaces traditional centralized web infrastructure with peer-to-peer technology. This implementation provides:

- **🌍 Multi-File Websites** - Complete websites with HTML, CSS, JavaScript, and images
- **🔐 Cryptographic Identity** - Ed25519-based site ownership and updates
- **📱 Modern Browser Interface** - Chrome-like UI for browsing decentralized sites
- **💼 Secure Wallet System** - BIP-39 mnemonic-based site management
- **🔄 Peer-to-Peer Networking** - libp2p-based distributed content delivery
- **🔒 Content Encryption** - Optional passphrase-based content protection
- **🎨 Rich Web Experience** - Full support for modern web technologies

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

### 📱 **Modern User Experience**
- **Chrome-like browser** - Familiar web browsing interface with tabs
- **Auto-discovery** - mDNS and localhost peer discovery
- **Standalone operation** - Browser starts its own node automatically
- **Responsive UI** - Scrollable content areas, modern controls
- **Multi-tab support** - Browse multiple sites simultaneously

### 🌍 **Peer-to-Peer Network**
- **libp2p networking** - Industry-standard P2P library
- **GossipSub protocol** - Efficient content distribution
- **mDNS discovery** - Automatic LAN peer discovery
- **Bootstrap support** - Manual peer connection fallback

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
git clone https://github.com/yourusername/betanet.git
cd betanet
./build.sh
```

**The enhanced build script automatically:**
- ✅ Runs comprehensive tests with race detection
- ✅ Performs security scanning (gosec, staticcheck)
- ✅ Applies security build flags (PIE, stripped binaries)
- ✅ Generates coverage reports
- ✅ Validates code quality

## 🚀 Quick Start

### 1. **Create a Multi-File Website**
```bash
# Create a website directory
mkdir -p mywebsite/{css,js,images}

# Create your website files
echo "<!DOCTYPE html><html><head><title>My Site</title><link rel='stylesheet' href='css/style.css'></head><body><h1>Hello World!</h1><script src='js/app.js'></script></body></html>" > mywebsite/index.html
echo "body { font-family: Arial; color: #333; }" > mywebsite/css/style.css
echo "console.log('Hello from JavaScript!');" > mywebsite/js/app.js
```

### 2. **Create a Wallet & Site**
```bash
# Create wallet and site
./bin/betanet-wallet new -out /tmp/wallet.json
./bin/betanet-wallet add-site -wallet /tmp/wallet.json -mnemonic "your mnemonic" -label mysite
```

### 3. **Publish Your Multi-File Website**
```bash
# Publish the complete website
./bin/betanet-wallet publish-website \
  -wallet /tmp/wallet.json \
  -mnemonic "your mnemonic" \
  -label mysite \
  -dir mywebsite \
  -main index.html
```

### 4. **Register a Domain**
```bash
# Register a human-readable domain
./bin/betanet-wallet register-domain \
  -wallet /tmp/wallet.json \
  -mnemonic "your mnemonic" \
  -label mysite \
  -domain mysite.bn
```

### 5. **Browse Your Website**
```bash
# Open the browser and navigate to: mysite.bn
./bin/betanet-browser
```

## 🎯 Core Components

### **betanet-node** - Network Node
The core networking component that:
- **Runs the P2P network** - Handles peer connections and content distribution
- **Stores content** - BadgerDB-based persistent storage
- **Validates updates** - Cryptographic signature verification
- **Discovers peers** - mDNS and bootstrap peer discovery
- **Manages websites** - Multi-file website storage and retrieval

**Security Features:**
- **Peer reputation system** - Automatic peer scoring and banning
- **Rate limiting** - Configurable request limits per peer
- **Memory management** - Automatic cleanup and usage limits
- **Connection validation** - Secure peer connection handling

**Commands:**
```bash
# Start a node
./bin/betanet-node run -data /path/to/db -listen /ip4/0.0.0.0/tcp/4001

# Publish a multi-file website
./bin/betanet-node publish-website -key /path/to/key -data /path/to/db -dir /path/website

# Add files to existing website
./bin/betanet-node add-file -key /path/to/key -data /path/to/db -path css/style.css -content /path/file

# List website contents
./bin/betanet-node list-website -data /path/to/db -site <siteID>
```

### **betanet-wallet** - Website Management
Complete wallet system for managing multi-file websites:
- **Create sites** - Deterministic key derivation from mnemonic
- **Publish websites** - Complete multi-file website publishing
- **Manage files** - Add, update, and organize website files
- **Register domains** - Human-readable `.bn` domain names
- **Website metadata** - View website information and file lists

**Security Features:**
- **Strong passphrase validation** - Enforces secure password requirements
- **Mnemonic validation** - Prevents weak seed phrases
- **Rate limiting** - Protects against brute force attacks
- **Automatic backups** - Configurable backup and retention policies

**Commands:**
```bash
# Create new wallet
./bin/betanet-wallet new -out wallet.json

# Add a site
./bin/betanet-wallet add-site -wallet wallet.json -mnemonic "..." -label mysite

# Publish multi-file website
./bin/betanet-wallet publish-website -wallet wallet.json -mnemonic "..." -label mysite -dir /path/website

# Add files to existing website
./bin/betanet-wallet add-website-file -wallet wallet.json -mnemonic "..." -label mysite -path css/style.css -content /path/file

# List website contents
./bin/betanet-wallet list-website -wallet wallet.json -mnemonic "..." -label mysite

# Get website information
./bin/betanet-wallet get-website-info -wallet wallet.json -mnemonic "..." -label mysite

# Register domain
./bin/betanet-wallet register-domain -wallet wallet.json -mnemonic "..." -label mysite -domain mysite.bn
```

### **betanet-browser** - Web Interface
Modern browser interface that:
- **Auto-starts node** - Creates local network node automatically
- **Resolves domains** - Converts `.bn` domains to site IDs
- **Displays websites** - Renders complete multi-file websites
- **Chrome-like UI** - Familiar navigation controls and address bar
- **Multi-tab support** - Browse multiple sites simultaneously
- **File handling** - Proper MIME type detection and rendering

**Features:**
- **Address bar** - Type site IDs or `.bn` domains
- **Navigation** - Back, forward, refresh buttons
- **Tab management** - Multiple tabs for different sites
- **Auto-discovery** - Finds peers via mDNS
- **Standalone** - No external dependencies
- **Flexible data** - Use existing node databases or create new ones

**Data Directory Options:**
```bash
# Use existing node database (recommended for testing)
./bin/betanet-browser -data /tmp/node1

# Use demo node database
./bin/betanet-browser -data temp/demo-node

# Use default browser database (isolated)
./bin/betanet-browser
```

### **betanet-gui** - Management Interface
Desktop GUI for node and website management:
- **Node control** - Start/stop network nodes
- **Website management** - Create and publish multi-file websites
- **Network monitoring** - Peer connections and content status
- **Content browsing** - View and manage published websites

## 🔧 Advanced Usage

### **Configuration Management**
```bash
# Environment-based configuration
export BETANET_ENV=production
export BETANET_LOG_LEVEL=warn
export BETANET_MAX_PEERS=200
export BETANET_MAX_CONTENT_SIZE=20971520  # 20MB

# Or use configuration file
./bin/betanet-node run -config config.yaml
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
  data_dir: "/var/lib/betanet"
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

# 3. Publish complete website
./bin/betanet-wallet publish-website \
  -wallet wallet.json \
  -mnemonic "..." \
  -label mysite \
  -dir mywebsite \
  -main index.html

# 4. Add additional files later
./bin/betanet-wallet add-website-file \
  -wallet wallet.json \
  -mnemonic "..." \
  -label mysite \
  -path css/theme.css \
  -content /path/to/theme.css
```

### **Domain Name System**
```bash
# List all registered domains
./bin/betanet-wallet list-domains -data /path/to/db

# Resolve a domain
./bin/betanet-wallet resolve-domain -data /path/to/db -domain mysite.bn

# Domain format validation
# ✅ Valid: mysite.bn, blog123.bn, news2024.bn
# ❌ Invalid: my-site.bn, site.bn, my.site.bn
```

### **Content Encryption**
```bash
# Publish encrypted content
./bin/betanet-wallet publish-website \
  -wallet wallet.json \
  -mnemonic "..." \
  -label mysite \
  -dir mywebsite \
  -encrypt-pass "secret phrase"

# Decrypt content when browsing
./bin/betanet-node browse \
  -data /path/to/db \
  -site <siteID> \
  -decrypt-pass "secret phrase"
```

### **Network Configuration**
```bash
# Start node with bootstrap peers
./bin/betanet-node run \
  -data /path/to/db \
  -listen /ip4/0.0.0.0/tcp/4001 \
  -bootstrap /ip4/127.0.0.1/tcp/4002/p2p/12D3KooW...

# Connect multiple nodes
./bin/betanet-node run -data /tmp/node2 -listen /ip4/0.0.0.0/tcp/4002 -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
```

## 🏗️ Architecture

### **Core Components**
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   betanet-node  │    │ betanet-wallet  │    │ betanet-browser │
│   (P2P Node)    │    │ (Website Mgmt)  │    │   (Web UI)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   BadgerDB      │
                    │  (Storage)      │
                    └─────────────────┘
```

### **Multi-File Website Data Flow**
1. **Website Creation** → Wallet creates site and website manifest
2. **File Publishing** → Individual files stored with cryptographic signatures
3. **Manifest Updates** → Website manifest links all files together
4. **Domain Registration** → Wallet registers `.bn` domain
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

**"Cannot acquire directory lock"**
```bash
# Kill any running nodes
pkill -f betanet-node

# Use different data directories
./bin/betanet-node run -data /tmp/node1 -listen /ip4/0.0.0.0/tcp/4001
./bin/betanet-node run -data /tmp/node2 -listen /ip4/0.0.0.0/tcp/4002
```

**"No peers found"**
```bash
# Check node is running and copy address
./bin/betanet-node run -data /tmp/node -listen /ip4/0.0.0.0/tcp/4001
# Copy the "addr:" line and use as bootstrap

# Use bootstrap address
./bin/betanet-node browse -data /tmp/browse -site <siteID> -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
```

**"Website files not loading"**
```bash
# Check website manifest
./bin/betanet-wallet get-website-info -wallet wallet.json -mnemonic "..." -label mysite

# Verify all files are published
./bin/betanet-wallet list-website -wallet wallet.json -mnemonic "..." -label mysite

# Check file paths match manifest
./bin/betanet-wallet add-website-file -wallet wallet.json -mnemonic "..." -label mysite -path css/style.css -content /path/file
```

### **Debug Mode**
```bash
# Enable verbose logging
export BETANET_DEBUG=1
./bin/betanet-node run -data /tmp/node -listen /ip4/0.0.0.0/tcp/4001

# Check network status
./bin/betanet-node browse -data /tmp/browse -site <siteID> -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
```

## 🧪 Testing

### **Complete Multi-File Website Test**
```bash
# Run the automated test script
./test-domain-workflow.sh

# This script demonstrates:
# - Multi-file website creation
# - HTML, CSS, JavaScript, and image files
# - Website publishing and domain registration
# - Browser testing with full website functionality
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
# Terminal 1: Start node A
./bin/betanet-node run -data /tmp/nodeA -listen /ip4/0.0.0.0/tcp/4001

# Terminal 2: Start node B with bootstrap
./bin/betanet-node run -data /tmp/nodeB -listen /ip4/0.0.0.0/tcp/4002 -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...

# Terminal 3: Create and publish multi-file website
./bin/betanet-wallet new -out /tmp/test-wallet.json
./bin/betanet-wallet add-site -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite
./bin/betanet-wallet publish-website -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite -dir /path/website
./bin/betanet-wallet register-domain -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite -domain test.bn -data /tmp/nodeA

# Terminal 4: Browse website
./bin/betanet-browser -data /tmp/nodeA
# Navigate to: test.bn
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
git clone https://github.com/yourusername/betanet.git
cd betanet
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

- **Issues** - [GitHub Issues](https://github.com/yourusername/betanet/issues)
- **Discussions** - [GitHub Discussions](https://github.com/yourusername/betanet/discussions)
- **Documentation** - [Wiki](https://github.com/yourusername/betanet/wiki)

---

**🌐 Betanet - Building the decentralized web with complete multi-file websites.**

*Now supporting HTML, CSS, JavaScript, images, and more - all stored securely on the blockchain with enterprise-grade security features.*
