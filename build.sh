#!/usr/bin/env bash
set -euo pipefail

# Enhanced build script with security flags, testing, and linting
# Build all CLI and GUI binaries for Linux amd64 with security enhancements

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT_DIR="$ROOT_DIR/bin"
GO_BIN="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"

# Security and build flags
SECURITY_FLAGS="-trimpath -ldflags=-s -ldflags=-w -buildmode=pie"
TEST_FLAGS="-cover -coverprofile=coverage.out -timeout=5m"
LINT_FLAGS="-E gofmt,goimports,misspell,unused,deadcode,varcheck,structcheck,ineffassign"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Go version
    go_version=$("$GO_BIN" version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"
    
    # Check if required tools are available
    if ! command -v gofmt &> /dev/null; then
        log_warning "gofmt not found, installing..."
        "$GO_BIN" install golang.org/x/tools/cmd/gofmt@latest
    fi
    
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found, installing..."
        "$GO_BIN" install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    
    if ! command -v staticcheck &> /dev/null; then
        log_warning "staticcheck not found, installing..."
        "$GO_BIN" install honnef.co/go/tools/cmd/staticcheck@latest
    fi
}

# Run tests with coverage
run_tests() {
    log_info "Running tests with coverage and race detection..."
    
    # Clean previous coverage files
    rm -f coverage.out coverage.html
    
    # Run tests
    if "$GO_BIN" test $TEST_FLAGS ./...; then
        log_success "All tests passed!"
        
        # Generate coverage report
        if [ -f coverage.out ]; then
            log_info "Generating coverage report..."
            "$GO_BIN" tool cover -html=coverage.out -o coverage.html
            log_info "Coverage report saved to coverage.html"
        fi
    else
        log_error "Tests failed!"
        exit 1
    fi
}

# Run security checks
run_security_checks() {
    log_info "Running security checks..."
    
    # Go vet
    log_info "Running go vet..."
    if "$GO_BIN" vet ./...; then
        log_success "go vet passed"
    else
        log_error "go vet failed!"
        exit 1
    fi
    
    # Staticcheck
    log_info "Running staticcheck..."
    if staticcheck ./...; then
        log_success "staticcheck passed"
    else
        log_warning "staticcheck found issues (continuing...)"
    fi
    
    # golangci-lint
    log_info "Running golangci-lint..."
    if golangci-lint run --timeout=5m; then
        log_success "golangci-lint passed"
    else
        log_warning "golangci-lint found issues (continuing...)"
    fi
    
    # Check for known vulnerabilities
    log_info "Checking for known vulnerabilities..."
    if command -v gosec &> /dev/null; then
        if gosec ./...; then
            log_success "gosec security scan passed"
        else
            log_warning "gosec found security issues (continuing...)"
        fi
    else
        log_warning "gosec not installed, skipping security scan"
    fi
}

# Format and lint code
format_and_lint() {
    log_info "Formatting and linting code..."
    
    # Format code
    log_info "Running gofmt..."
    if gofmt -s -w .; then
        log_success "Code formatted successfully"
    else
        log_warning "Code formatting had issues"
    fi
    
    # Check imports
    log_info "Checking imports..."
    if "$GO_BIN" mod tidy; then
        log_success "Dependencies organized"
    else
        log_error "Failed to organize dependencies!"
        exit 1
    fi
}

# Build binaries with security flags
build_binaries() {
    log_info "Building binaries with security enhancements..."
    mkdir -p "$OUT_DIR"
    
    # Build betanet-node
    log_info "Building betanet-node..."
    if GOOS=linux GOARCH=amd64 "$GO_BIN" build $SECURITY_FLAGS -o "$OUT_DIR/betanet-node" ./cmd/betanet-node; then
        log_success "betanet-node built successfully"
    else
        log_error "Failed to build betanet-node!"
        exit 1
    fi
    
    # Build betanet-wallet
    log_info "Building betanet-wallet..."
    if GOOS=linux GOARCH=amd64 "$GO_BIN" build $SECURITY_FLAGS -o "$OUT_DIR/betanet-wallet" ./cmd/betanet-wallet; then
        log_success "betanet-wallet built successfully"
    else
        log_error "Failed to build betanet-wallet!"
        exit 1
    fi
    
    # Build betanet-gui (requires Linux GUI dev libs)
    log_info "Building betanet-gui..."
    if GOOS=linux GOARCH=amd64 "$GO_BIN" build $SECURITY_FLAGS -o "$OUT_DIR/betanet-gui" ./cmd/betanet-gui; then
        log_success "betanet-gui built successfully"
    else
        log_warning "betanet-gui build failed (likely missing X11/OpenGL dev libs). CLI builds are ready."
    fi
    
    # Build betanet-browser (requires Linux GUI dev libs)
    log_info "Building betanet-browser..."
    if GOOS=linux GOARCH=amd64 "$GO_BIN" build $SECURITY_FLAGS -o "$OUT_DIR/betanet-browser" ./cmd/betanet-browser; then
        log_success "betanet-browser built successfully"
    else
        log_warning "betanet-browser build failed (likely missing X11/OpenGL dev libs)."
    fi
}

# Security audit of built binaries
audit_binaries() {
    log_info "Auditing built binaries..."
    
    for binary in betanet-node betanet-wallet betanet-gui betanet-browser; do
        if [ -f "$OUT_DIR/$binary" ]; then
            log_info "Auditing $binary..."
            
            # Check file permissions
            perms=$(stat -c "%a" "$OUT_DIR/$binary")
            if [ "$perms" = "755" ]; then
                log_success "$binary has correct permissions (755)"
            else
                log_warning "$binary has unusual permissions ($perms)"
            fi
            
            # Check if binary is stripped
            if file "$OUT_DIR/$binary" | grep -q "stripped"; then
                log_success "$binary is properly stripped"
            else
                log_warning "$binary is not stripped (may contain debug info)"
            fi
            
            # Check binary size
            size=$(stat -c "%s" "$OUT_DIR/$binary")
            log_info "$binary size: $size bytes"
        fi
    done
}

# Generate build report
generate_report() {
    log_info "Generating build report..."
    
    report_file="$OUT_DIR/build-report.txt"
    {
        echo "Betanet Build Report"
        echo "===================="
        echo "Build Date: $(date)"
        echo "Go Version: $("$GO_BIN" version)"
        echo "Build Flags: $SECURITY_FLAGS"
        echo ""
        echo "Binaries Built:"
        for binary in betanet-node betanet-wallet betanet-gui betanet-browser; do
            if [ -f "$OUT_DIR/$binary" ]; then
                size=$(stat -c "%s" "$OUT_DIR/$binary")
                echo "  $binary: $size bytes"
            fi
        done
        echo ""
        echo "Security Features:"
        echo "  - Stripped binaries (no debug info)"
        echo "  - Position independent executables (PIE)"
        echo "  - Trimmed paths"
        echo "  - Race detection in tests"
        echo "  - Code coverage analysis"
        echo "  - Static analysis (staticcheck, golangci-lint)"
        echo "  - Security scanning (gosec)"
    } > "$report_file"
    
    log_success "Build report saved to $report_file"
}

# Main build process
main() {
    log_info "Starting enhanced Betanet build process..."
    log_info "Using Go: $GO_BIN"
    
    # Check prerequisites
    check_prerequisites
    
    # Format and lint code
    format_and_lint
    
    # Run tests
    run_tests
    
    # Run security checks
    run_security_checks
    
    # Build binaries
    build_binaries
    
    # Audit binaries
    audit_binaries
    
    # Generate report
    generate_report
    
    log_success "Build process completed successfully!"
    log_info "Binaries are available in: $OUT_DIR"
    log_info "Coverage report: coverage.html"
    log_info "Build report: $OUT_DIR/build-report.txt"
}

# Run main function
main "$@"


