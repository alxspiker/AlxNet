# 🌐 AlxNet — Decentralized Web Platform  

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)  
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)  
[![Status](https://img.shields.io/badge/Status-Production%20Ready-green.svg)]()  
[![Security](https://img.shields.io/badge/Security-Hardened-red.svg)]()  

**AlxNet** is a production-ready decentralized web platform for creating, publishing, and hosting complete multi-file websites. It combines **enterprise-grade security**, **libp2p peer-to-peer networking**, and **integrated management tools** into a single unified system.  

---

## 🚀 Quick Start  

### Install  
```bash
git clone https://github.com/alxspiker/AlxNet.git
cd AlxNet
./build.sh
```

### Launch Entire Platform  
```bash
# Default configuration
./bin/alxnet start

# Custom ports
./bin/alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082
```

When started, AlxNet runs:  
- 🔗 **P2P Node** (4001) – Peer discovery & content distribution  
- 🌐 **Browser UI** (8080) – Access decentralized websites  
- 💰 **Wallet UI** (8081) – Site creation & publishing  
- 📊 **Node Dashboard** (8082) – Network monitoring  

---

## ✨ Core Features  

- 🌍 **Multi-File Website Hosting** — Full HTML, CSS, JS, images, media  
- 🔐 **Ed25519 Security** — Digital signatures, SHA-256 integrity checks  
- 💼 **Wallet System** — BIP-39 mnemonic-based key management  
- 🔄 **Libp2p Networking** — GossipSub content distribution, peer reputation  
- 🛡️ **Enterprise Protections** — Rate limiting, DoS mitigation, validation layers  

---

## 🎯 Web Interfaces  

- **Browser (8080)** — Explore decentralized websites by Site ID  
- **Wallet (8081)** — Manage wallets, create sites, publish content  
- **Node Dashboard (8082)** — Monitor peers, storage, and performance  

---

## 🛡️ Security Model  

Defense-in-depth across four layers:  

- **Application** — Input validation, rate limiting, safe file handling  
- **Network** — Peer scoring, ban management, geo-distribution  
- **Storage** — Size enforcement, retry logic, encryption options  
- **Cryptography** — Ed25519 signatures, SHA-256 content addressing  

---

## 🔧 Configuration  

AlxNet supports **CLI flags**, **environment variables**, and **YAML config**.  

Example `config.yaml`:  
```yaml
environment: production
log_level: warn
network:
  max_peers: 200
  enable_mdns: true
security:
  max_content_size: 20971520  # 20MB
storage:
  data_dir: "/var/lib/alxnet"
wallet:
  default_path: "/var/lib/alxnet/wallets"
```

---

## 🧪 Testing  

```bash
# Run all tests
go test ./...

# Security scans
./security-audit.sh

# Benchmarking
go test -bench=. -benchmem ./internal/core
```

---

## 📚 Documentation  

- **API Reference**: See source code interfaces in `internal/`

---

## 🤝 Contributing  

1. Fork & branch  
2. Implement & test (aim for >85% coverage)  
3. Run security scans  
4. Submit PR with documentation updates  

Security issues: Create an issue on this repo.

---

## 📄 License  

Licensed under MIT. See [LICENSE](LICENSE).  

---

**AlxNet — Powering the Next Generation of Decentralized Web**  
