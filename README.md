# 🌐 Betanet - Decentralized Web Platform

**A complete, production-ready implementation of the decentralized web platform with built-in domain names, browser, wallet, and peer-to-peer networking.**

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen.svg)]()

<img width="1297" height="350" alt="image" src="https://github.com/user-attachments/assets/3c727c41-fedc-4e17-bbd5-7d7f730ac388" />

## 🚀 What is Betanet?

Betanet is a **fully decentralized, censorship-resistant web platform** that replaces traditional centralized web infrastructure with peer-to-peer technology. This implementation provides:

- **🌍 Decentralized Domain Names** - Human-readable `.bn` domains (e.g., `mysite.bn`)
- **🔐 Cryptographic Identity** - Ed25519-based site ownership and updates
- **📱 Modern Browser Interface** - Chrome-like UI for browsing decentralized sites
- **💼 Secure Wallet System** - BIP-39 mnemonic-based site management
- **🔄 Peer-to-Peer Networking** - libp2p-based distributed content delivery
- **🔒 Content Encryption** - Optional passphrase-based content protection

## ✨ Key Features

### 🌐 **Decentralized Domain Names**
- **Unique `.bn` domains** - Only alphanumeric characters allowed
- **Global namespace** - No central authority controls domain registration
- **Cryptographic ownership** - Domains tied to wallet keys
- **Automatic resolution** - Browser resolves domains to site IDs

### 🔐 **Cryptographic Security**
- **Ed25519 signatures** - Fast, secure digital signatures
- **Deterministic key derivation** - BIP-39 mnemonic → site keys
- **Content integrity** - SHA-256 content addressing
- **Update validation** - Cryptographic proof of site ownership

### 📱 **Modern User Experience**
- **Chrome-like browser** - Familiar web browsing interface
- **Auto-discovery** - mDNS and localhost peer discovery
- **Standalone operation** - Browser starts its own node automatically
- **Responsive UI** - Scrollable content areas, modern controls

### 🌍 **Peer-to-Peer Network**
- **libp2p networking** - Industry-standard P2P library
- **GossipSub protocol** - Efficient content distribution
- **mDNS discovery** - Automatic LAN peer discovery
- **Bootstrap support** - Manual peer connection fallback

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

## 🚀 Quick Start

### 1. **Start a Network Node**
```bash
# Terminal 1: Start a node
./bin/betanet-node run -data /tmp/node1 -listen /ip4/0.0.0.0/tcp/4001
```

### 2. **Create a Wallet & Site**
```bash
# Terminal 2: Create wallet and site
./bin/betanet-wallet new -out /tmp/wallet.json
./bin/betanet-wallet add-site -wallet /tmp/wallet.json -mnemonic "your mnemonic" -label mysite
```

### 3. **Register a Domain**
```bash
# Register a human-readable domain
./bin/betanet-wallet register-domain \
  -wallet /tmp/wallet.json \
  -mnemonic "your mnemonic" \
  -label mysite \
  -domain mysite.bn \
  -data /tmp/node1
```

### 4. **Publish Content**
```bash
# Create and publish content
echo "# My Decentralized Site" > /tmp/site.md
./bin/betanet-wallet publish \
  -wallet /tmp/wallet.json \
  -mnemonic "your mnemonic" \
  -label mysite \
  -content /tmp/site.md \
  -data /tmp/node1
```

### 5. **Browse Your Site**
```bash
# Open the browser and navigate to: mysite.bn
# Use the same data directory where you registered the domain
./bin/betanet-browser -data /tmp/node1

# Or use the default browser database (won't have your /tmp/ domains)
./bin/betanet-browser
```

## 🎯 Core Components

### **betanet-node** - Network Node
The core networking component that:
- **Runs the P2P network** - Handles peer connections and content distribution
- **Stores content** - BadgerDB-based persistent storage
- **Validates updates** - Cryptographic signature verification
- **Discovers peers** - mDNS and bootstrap peer discovery

**Commands:**
```bash
# Start a node
./bin/betanet-node run -data /path/to/db -listen /ip4/0.0.0.0/tcp/4001

# Browse a site
./bin/betanet-node browse -data /path/to/db -site <siteID>

# Publish content
./bin/betanet-node publish -key /path/to/key -data /path/to/db -content /path/to/file
```

### **betanet-wallet** - Site Management
Complete wallet system for managing sites and domains:
- **Create sites** - Deterministic key derivation from mnemonic
- **Register domains** - Human-readable `.bn` domain names
- **Publish updates** - Content publishing with optional encryption
- **Manage ownership** - Site key export and management

**Commands:**
```bash
# Create new wallet
./bin/betanet-wallet new -out wallet.json

# Add a site
./bin/betanet-wallet add-site -wallet wallet.json -mnemonic "..." -label mysite

# Register domain
./bin/betanet-wallet register-domain -wallet wallet.json -mnemonic "..." -label mysite -domain mysite.bn

# Publish content
./bin/betanet-wallet publish -wallet wallet.json -mnemonic "..." -label mysite -content file.md

# List domains
./bin/betanet-wallet list-domains -data /path/to/db
```

### **betanet-browser** - Web Interface
Modern browser interface that:
- **Auto-starts node** - Creates local network node automatically
- **Resolves domains** - Converts `.bn` domains to site IDs
- **Displays content** - Renders markdown and HTML content
- **Chrome-like UI** - Familiar navigation controls and address bar

**Features:**
- **Address bar** - Type site IDs or `.bn` domains
- **Navigation** - Back, forward, refresh buttons
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
Desktop GUI for node and wallet management:
- **Node control** - Start/stop network nodes
- **Wallet management** - Create sites and publish content
- **Network monitoring** - Peer connections and content status
- **Content browsing** - View and manage published sites

## 🔧 Advanced Usage

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
./bin/betanet-wallet publish \
  -wallet wallet.json \
  -mnemonic "..." \
  -label mysite \
  -content file.md \
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
│   (P2P Node)    │    │  (Site Mgmt)    │    │   (Web UI)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   BadgerDB      │
                    │  (Storage)      │
                    └─────────────────┘
```

### **Data Flow**
1. **Content Creation** → Wallet creates site and content
2. **Domain Registration** → Wallet registers `.bn` domain
3. **Content Publishing** → Node broadcasts update to network
4. **Content Distribution** → GossipSub distributes to peers
5. **Content Discovery** → Browser resolves domain to site ID
6. **Content Retrieval** → Node fetches content from network

### **Security Model**
- **Site Keys** - Long-term Ed25519 keys for site ownership
- **Update Keys** - Ephemeral keys for each content update
- **Link Signatures** - Proof that update key is authorized
- **Update Signatures** - Proof that content is authentic
- **Content Integrity** - SHA-256 hashing prevents tampering

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

**"Domain resolution failed"**
```bash
# Check domain is registered
./bin/betanet-wallet list-domains -data /path/to/db

# Verify domain format (alphanumerical.alphanumerical)
./bin/betanet-wallet register-domain -wallet wallet.json -mnemonic "..." -label mysite -domain mysite.bn

# Browser must use same database where domain was registered
./bin/betanet-browser -data /path/to/db  # Use same path as wallet commands
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

### **Local Development Setup**
```bash
# Terminal 1: Start node A
./bin/betanet-node run -data /tmp/nodeA -listen /ip4/0.0.0.0/tcp/4001

# Terminal 2: Start node B with bootstrap
./bin/betanet-node run -data /tmp/nodeB -listen /ip4/0.0.0.0/tcp/4002 -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...

# Terminal 3: Create and publish site
./bin/betanet-wallet new -out /tmp/test-wallet.json
./bin/betanet-wallet add-site -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite
./bin/betanet-wallet register-domain -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite -domain test.bn -data /tmp/nodeA
echo "# Test Site" > /tmp/test.md
./bin/betanet-wallet publish -wallet /tmp/test-wallet.json -mnemonic "..." -label testsite -content /tmp/test.md -data /tmp/nodeA

# Terminal 4: Browse site
./bin/betanet-browser -data /tmp/nodeA
# Navigate to: test.bn

# Or use the automated test script:
./test-domain-workflow.sh
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

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **libp2p** - Peer-to-peer networking library
- **BadgerDB** - Fast key-value storage
- **Fyne** - Cross-platform GUI toolkit
- **Ed25519** - Fast, secure digital signatures
- **BIP-39** - Mnemonic phrase standard

## 📞 Support

- **Issues** - [GitHub Issues](https://github.com/yourusername/betanet/issues)
- **Discussions** - [GitHub Discussions](https://github.com/yourusername/betanet/discussions)
- **Documentation** - [Wiki](https://github.com/yourusername/betanet/wiki)

---

**🌐 Betanet - Building the decentralized web, one site at a time.**
