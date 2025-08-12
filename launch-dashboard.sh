#!/usr/bin/env bash
set -euo pipefail

# Betanet Dashboard Launcher
# Launches the unified dashboard for complete system management

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
DASHBOARD_BIN="$ROOT_DIR/bin/betanet-dashboard"
DEFAULT_DATA_DIR="$HOME/.betanet/dashboard"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Show usage
show_usage() {
    echo "🌐 Betanet Dashboard Launcher"
    echo "=============================="
    echo ""
    echo "Usage:"
    echo "  $0                    # Launch with default data directory"
    echo "  $0 -data /path/db     # Launch with specified data directory"
    echo "  $0 -h, --help         # Show this help message"
    echo ""
    echo "Features:"
    echo "  💼 Complete Wallet Management - All wallet operations"
    echo "  🖥️  Full Node Control - All node operations"
    echo "  🌐 Network Management - All network operations"
    echo "  🌍 Web Browser - Browse decentralized sites"
    echo "  🔧 Advanced Tools - All console functionality"
    echo ""
    echo "Data Directory:"
    echo "  Default: $DEFAULT_DATA_DIR"
    echo "  Custom:  Specify with -data flag"
    echo ""
    echo "Examples:"
    echo "  $0                           # Use default data directory"
    echo "  $0 -data /opt/betanet/data   # Use custom data directory"
    echo ""
}

# Check if dashboard binary exists
check_dashboard() {
    if [ ! -f "$DASHBOARD_BIN" ]; then
        log_error "Dashboard binary not found: $DASHBOARD_BIN"
        echo ""
        echo "Please build the dashboard first:"
        echo "  ./build.sh"
        echo ""
        echo "Or build just the dashboard:"
        echo "  go build -o bin/betanet-dashboard ./cmd/betanet-dashboard"
        echo ""
        exit 1
    fi
    
    if [ ! -x "$DASHBOARD_BIN" ]; then
        log_error "Dashboard binary is not executable: $DASHBOARD_BIN"
        echo "Fixing permissions..."
        chmod +x "$DASHBOARD_BIN"
        log_success "Permissions fixed"
    fi
}

# Create data directory if it doesn't exist
setup_data_dir() {
    local data_dir="$1"
    
    if [ ! -d "$data_dir" ]; then
        log_info "Creating data directory: $data_dir"
        mkdir -p "$data_dir"
        log_success "Data directory created"
    fi
    
    # Set proper permissions
    chmod 700 "$data_dir"
    log_info "Data directory permissions set to 700"
}

# Launch dashboard
launch_dashboard() {
    local data_dir="$1"
    
    log_info "Launching Betanet Dashboard..."
    log_info "Data directory: $data_dir"
    echo ""
    
    # Setup data directory
    setup_data_dir "$data_dir"
    
    # Launch dashboard
    log_success "Starting dashboard..."
    echo ""
    echo "🌐 Betanet Dashboard is starting..."
    echo "   This may take a few seconds on first launch."
    echo ""
    
    if "$DASHBOARD_BIN" -data "$data_dir"; then
        log_success "Dashboard closed successfully"
    else
        log_error "Dashboard exited with error code $?"
        exit 1
    fi
}

# Main function
main() {
    local data_dir="$DEFAULT_DATA_DIR"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -data)
                if [[ -z "${2:-}" ]]; then
                    log_error "-data flag requires a directory path"
                    exit 1
                fi
                data_dir="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Check if dashboard binary exists
    check_dashboard
    
    # Launch dashboard
    launch_dashboard "$data_dir"
}

# Run main function
main "$@"
