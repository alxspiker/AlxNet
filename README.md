🌐 AlxNet - Decentralized Web Platform
AlxNet is a production-ready, all-in-one platform for creating, publishing, and hosting complete websites using peer-to-peer distribution and cryptographic security. It revolutionizes web hosting by removing central points of failure and control.



This project bundles a P2P node, a specialized website browser, a powerful wallet manager, and a node dashboard into a single, cohesive application. You can launch the entire stack and begin publishing decentralized content with one command.
✨ Core Features
 * Single Command Launch: Start the entire decentralized web stack (P2P Node, Browser, Wallet, Dashboard) with ./bin/alxnet start.
 * Complete Website Support: Host rich, multi-file websites with full support for HTML, CSS, JavaScript, images, videos, and fonts.
 * Cryptographic Security: Every website and update is cryptographically signed with Ed25519 keys, ensuring data integrity and authenticating ownership.
 * Professional Wallet System: A user-friendly web interface for managing sites. It features BIP-39 mnemonics for wallet recovery, a file tree editor with syntax highlighting, and a streamlined publishing workflow.
 * Robust P2P Networking: Built on libp2p, it features automatic peer discovery (mDNS), efficient content distribution via GossipSub, and a resilient storage backend.
 * Integrated Management UIs: Three distinct web interfaces provide full control over browsing, content creation, and network monitoring.
🛠️ Platform Components
When you run alxnet, you are launching four interconnected services that work together to create the decentralized web experience.
 * 🌐 Website Browser (Port 8080)
   This interface allows you to access and render decentralized websites. You navigate using a site's unique cryptographic ID. It fully supports modern web standards, executing JavaScript and rendering complex CSS.
 * 💰 Wallet & Site Manager (Port 8081)
   This is your control center for creating and managing content. The workflow guides you from wallet creation to site publishing. It includes a built-in visual editor to manage your website's file structure and content before publishing it to the network.
 * 📊 Node Dashboard (Port 8082)
   A real-time dashboard for monitoring the health and status of your P2P node. You can view connected peers, inspect storage metrics, and analyze network traffic.
 * ⚡ Core P2P Node (Port 4001)
   The engine powering the platform. This component connects to other AlxNet nodes, distributes your published content, fetches content from peers, and validates the cryptographic integrity of all data.
⚙️ Installation & Setup
Prerequisites
 * Go 1.23+ (Download)
 * A Linux, macOS, or Windows operating system
Build from Source
Clone the repository and run the enhanced build script. This script automatically handles testing, security analysis, and compilation.
# 1. Clone the repository
git clone https://github.com/alxspiker/AlxNet.git

# 2. Navigate into the directory
cd AlxNet

# 3. Run the build script
./build.sh

The compiled binary will be located at ./bin/alxnet.
🚀 Quick Start Guide
1. Launch the AlxNet Platform
Start all services with a single command from your terminal.
# Launch the platform with default port configuration
./bin/alxnet start

2. Create Your Wallet and First Site
 * Open the Wallet Manager in your browser: http://localhost:8081.
 * Click "Create New Wallet". You'll receive a 12-word mnemonic phrase.
 * IMPORTANT: Write this phrase down and store it in a secure location. It's the only way to recover your wallet and control your sites.
 * Navigate to the "Sites" tab and click "Create New Site". Give it a label, and a unique Site ID will be generated.
3. Build and Publish Your Website
 * Go to the "Editor" tab within the Wallet Manager.
 * Use the file tree on the left to create folders and files (e.g., index.html, style.css).
 * Write or paste your code into the built-in editor.
 * Once your files are ready, click the "Publish Website" button. Your site's files will be cryptographically signed and distributed across the network.
4. View Your Decentralized Site
 * Open the AlxNet Browser: http://localhost:8080.
 * Enter the Site ID generated in step 2 into the navigation bar and press Enter.
 * Your decentralized website will now load directly from the P2P network.
🏗️ System Architecture
The alxnet start command launches a unified application where the Web UIs (Browser, Wallet, Node) all communicate with a shared P2P node core. This integrated design ensures seamless operation and efficient resource management.
┌─────────────────────────────────────────────────────────────────────┐
│                         ./bin/alxnet start                          │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐      │
│ │  Browser UI     │  │  Wallet UI      │  │   Node UI       │      │
│ │  (Port 8080)    │  │  (Port 8081)    │  │  (Port 8082)    │      │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘      │
│                               │                                     │
│           ┌─────────────────────────────────────┐                   │
│           │         Shared P2P Node Core        │                   │
│           │        (LibP2P Host @ Port 4001)    │                   │
│           └─────────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌─────────────────────────────────────────────────┐
        │              BadgerDB Storage Engine            │
        │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
        │ │   Records   │ │   Content   │ │    Sites    │ │
        │ └─────────────┘ └─────────────┘ └─────────────┘ │
        └─────────────────────────────────────────────────┘

🔧 Advanced Configuration
Customize AlxNet using command-line flags, environment variables, or a config.yaml file.
Command-Line Flags
Override default ports and other common settings directly.
# Run with custom ports and connect to a specific bootstrap peer
./bin/alxnet start \
  -node-port 4002 \
  -browser-port 8090 \
  -wallet-port 8091 \
  -node-ui-port 8092 \
  -bootstrap /ip4/198.51.100.1/tcp/4001/p2p/12D3KooW...

Configuration File
For persistent, complex configurations, create a config.yaml file in the project's root directory.
Example config.yaml:
environment: production
log_level: warn

network:
  listen_addr: "/ip4/0.0.0.0/tcp/4001"
  max_peers: 200
  peer_timeout: 60s

security:
  max_content_size: 20971520  # 20MB
  rate_limit: 200 # Requests per minute per peer
  ban_duration: 30m

storage:
  data_dir: "/var/lib/alxnet"
  cleanup_interval: 10m

🛡️ Security Model
Security is a core principle of AlxNet, implemented through a defense-in-depth strategy.
 * Cryptographic Integrity: All content is addressed by its hash (SHA-256) and all publishing actions are authorized by Ed25519 digital signatures. This prevents unauthorized modification and ensures data authenticity.
 * Network Security: The platform mitigates risks like DDoS and Sybil attacks through peer validation, reputation scoring, strict rate limiting, and connection management.
 * Input Sanitization: All incoming data, file paths, and user inputs are rigorously validated to prevent attacks like path traversal.
 * Resource Management: Strict limits on file sizes, website file counts, and memory usage prevent resource exhaustion attacks.
🤝 Contributing
We welcome contributions from the community!
 * Fork the repository.
 * Create a new branch for your feature or bug fix.
 * Write your code and add comprehensive tests.
 * Ensure all tests and quality checks pass by running ./build.sh.
 * Submit a pull request with a clear description of your changes.
For major architectural changes, please open an issue first to start a discussion with the maintainers.
📄 License
This project is licensed under the MIT License. See the LICENSE file for complete details.