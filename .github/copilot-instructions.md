# AlxNet Decentralized Web Platform

AlxNet is a Go-based decentralized web platform that enables creation, publishing, and hosting of complete websites with cryptographic security and peer-to-peer distribution. The platform provides a unified command-line interface and multiple web interfaces for management.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap, Build, and Test Repository
- `go mod tidy` -- downloads dependencies, takes ~12 seconds on first run. NEVER CANCEL.
- `./build.sh` -- comprehensive build with security scanning, takes 3-4 minutes. NEVER CANCEL. Set timeout to 10+ minutes.
  - Automatically installs required tools: golangci-lint, staticcheck 
  - Runs tests with coverage and race detection
  - Performs security checks (go vet, staticcheck, golangci-lint)
  - Builds secure binary with hardening flags (-trimpath, -ldflags=-s -ldflags=-w, -buildmode=pie)
  - Generates coverage report (coverage.html) and build report (bin/build-report.txt)
- `go test -v ./...` -- runs test suite, takes ~1 minute 18 seconds. NEVER CANCEL. Set timeout to 5+ minutes.

### Run the AlxNet Platform
- ALWAYS run the build steps first before running the application.
- `./bin/alxnet start` -- starts complete platform with default ports
- `./bin/alxnet start -node-port 4001 -browser-port 8080 -wallet-port 8081 -node-ui-port 8082` -- custom ports
- **Platform provides 4 interfaces when running:**
  - Browser Interface: http://localhost:8080 (access decentralized websites)
  - Wallet Management: http://localhost:8081 (create and manage sites)
  - Node Management: http://localhost:8082 (network monitoring)
  - P2P Node: Auto-assigned port (peer discovery and content distribution)

### Security Audit and Validation
- `./security-audit.sh` -- comprehensive security analysis, takes ~8 seconds. NEVER CANCEL. Set timeout to 15+ minutes.
  - Runs go vet, staticcheck, golangci-lint, gosec security scanning
  - Analyzes dependencies for vulnerabilities with govulncheck
  - Generates detailed security reports in security-audit/ directory
- ALWAYS run `go vet ./...` and `./security-audit.sh` before committing changes.

## Validation

### Manual Testing Requirements
- ALWAYS manually validate any new code by running through complete user scenarios after making changes.
- **Essential validation workflow:**
  1. Build: `./build.sh` 
  2. Start platform: `./bin/alxnet start`
  3. Verify all 4 web interfaces are accessible (8080, 8081, 8082)
  4. Test P2P node connectivity and port assignment
  5. Stop gracefully with Ctrl+C
- **Complete end-to-end validation sequence:**
  ```bash
  # 1. Clean environment (optional, for thorough testing)
  rm -rf bin/ coverage.* data/ security-audit/
  
  # 2. Bootstrap dependencies
  go mod tidy
  
  # 3. Run test suite
  go test -v ./...
  
  # 4. Build with security scanning
  ./build.sh
  
  # 5. Start application and test interfaces
  ./bin/alxnet start
  # Navigate to http://localhost:8080, 8081, 8082 in browser
  # Press Ctrl+C to stop
  
  # 6. Security validation
  ./security-audit.sh
  ```
- **Additional integration testing:** Use multi-node setup for network testing:
  ```bash
  # Terminal 1 - Master node
  ./bin/alxnet start -node-port 4001 -data ./test-data-1
  
  # Terminal 2 - Test node (after master starts)
  ./bin/alxnet start -node-port 4002 -data ./test-data-2 -bootstrap /ip4/127.0.0.1/tcp/4001/p2p/...
  ```
- ALWAYS test the complete user workflow: node startup → web interface access → graceful shutdown.

### CI/CD Validation
- GitHub Actions automatically test network discovery and master node functionality
- CI builds and tests run on Go 1.21+ with comprehensive network testing
- ALWAYS run `./build.sh` locally before pushing to ensure CI will pass.

## Build Timing Expectations
- **NEVER CANCEL builds or long-running commands**
- **go mod tidy:** ~12 seconds (first run with downloads), ~1 second (subsequent runs)
- **go test -v ./...:** ~1 minute 10 seconds. NEVER CANCEL. Set timeout to 5+ minutes.
- **./build.sh:** ~2 minutes 9 seconds (clean), ~3+ minutes (with tool installation). NEVER CANCEL. Set timeout to 10+ minutes.
- **./security-audit.sh:** ~8 seconds (complete). NEVER CANCEL. Set timeout to 15+ minutes.
- **Application startup:** ~1 second to full operational state

## Common Tasks

### Project Structure
```
.
├── README.md                    # Comprehensive project documentation
├── build.sh                     # Enhanced build script with security
├── security-audit.sh            # Comprehensive security analysis
├── go.mod                       # Go 1.23+ dependencies
├── cmd/alxnet/main.go          # Single unified command entry point
├── internal/                    # Core implementation packages
│   ├── core/                   # Data structures and validation
│   ├── p2p/                    # Peer-to-peer networking
│   ├── store/                  # Data storage layer
│   ├── webserver/              # Web interface servers
│   ├── wallet/                 # Site creation and management
│   ├── config/                 # Configuration management
│   ├── crypto/                 # Cryptographic operations
│   └── network/                # Network discovery and consensus
├── network/                     # Master node configuration
│   ├── masterlist.json         # Global network bootstrap nodes
│   └── consensus-rules.json    # Network consensus configuration
└── .github/workflows/          # CI/CD automation
```

### Key Dependencies and Tools
- **Go 1.23+** (required, uses toolchain go1.24.6)
- **Build dependencies:** Automatically installed by build.sh
  - golangci-lint: Advanced linting and code analysis
  - staticcheck: Static analysis tool
  - gosec: Security vulnerability scanner (optional)
  - govulncheck: Dependency vulnerability checker (optional)
- **Core libraries:** libp2p (P2P networking), badger (storage), zap (logging)

### Application Capabilities
- **Single Binary Deployment:** All functionality in one executable (`./bin/alxnet`)
- **Decentralized Website Hosting:** Create, publish, and serve websites via P2P network
- **Multi-Interface Platform:** Browser, wallet management, and node monitoring UIs
- **Cryptographic Security:** Content signing, peer validation, secure key management
- **Network Discovery:** Automatic peer discovery via bootstrap nodes and DHT
- **Data Storage:** Persistent local storage with content deduplication

### Development Best Practices
- ALWAYS run `./build.sh` before committing - it includes comprehensive validation
- Use `go test -v ./internal/core` to test specific components during development  
- Run `./security-audit.sh` for security analysis before major releases
- Test multi-node scenarios for network-related changes
- Monitor coverage reports (coverage.html) to ensure adequate test coverage
- Validate all web interfaces are accessible after changes to webserver components

### Troubleshooting Common Issues
- **Build failures:** Ensure Go 1.23+ is installed and GOPATH is configured
- **Test failures:** Check that ports 8080-8082 and 4001+ are available
- **Network connectivity:** Verify firewall allows P2P traffic on configured ports
- **Permission errors:** Ensure binary has execute permissions (755)
- **Memory issues:** Platform uses BadgerDB - ensure sufficient disk space in data directory

### Performance and Security
- **Binary size:** ~38MB with security hardening (stripped, PIE, trimmed paths)
- **Memory usage:** Scales with content storage and peer connections
- **Security features:** Position-independent executables, input validation, rate limiting
- **Network limits:** Configurable via environment variables (ALXNET_MAX_PEERS, ALXNET_MAX_CONTENT_SIZE)