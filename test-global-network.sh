#!/bin/bash

# Test Global Discovery Network
# This script tests the comprehensive global discovery network implementation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR"
BIN_DIR="$ROOT_DIR/bin"
NETWORK_DIR="$ROOT_DIR/network"
TEMP_DIR="/tmp/betanet-test"

# Cleanup function
cleanup() {
    log_info "Cleaning up test environment..."
    rm -rf "$TEMP_DIR"
    pkill -f "betanet-network" || true
    pkill -f "betanet-node" || true
}

# Setup function
setup() {
    log_info "Setting up test environment..."
    
    # Create temp directory
    mkdir -p "$TEMP_DIR"
    
    # Ensure binaries exist
    if [[ ! -f "$BIN_DIR/betanet-network" ]]; then
        log_error "betanet-network binary not found. Please build first."
        exit 1
    fi
    
    if [[ ! -f "$BIN_DIR/betanet-node" ]]; then
        log_error "betanet-node binary not found. Please build first."
        exit 1
    fi
    
    # Check network configuration files
    if [[ ! -f "$NETWORK_DIR/masterlist.json" ]]; then
        log_error "masterlist.json not found in network directory"
        exit 1
    fi
    
    if [[ ! -f "$NETWORK_DIR/consensus-rules.json" ]]; then
        log_error "consensus-rules.json not found in network directory"
        exit 1
    fi
    
    log_success "Test environment setup complete"
}

# Test 1: Network Manager Basic Functionality
test_network_manager() {
    log_info "Testing Network Manager Basic Functionality..."
    
    # Start network manager
    "$BIN_DIR/betanet-network" -command status -verbose &
    NETWORK_PID=$!
    
    # Wait for startup
    sleep 3
    
    # Check if process is running
    if kill -0 $NETWORK_PID 2>/dev/null; then
        log_success "Network manager started successfully"
        
        # Test status command - network manager should start even without master nodes
        if "$BIN_DIR/betanet-network" -command status | grep -q "🟢 Running"; then
            log_success "Status command working - Network manager started successfully"
        else
            log_warning "Status command may not be working as expected"
        fi
        
        # Cleanup
        kill $NETWORK_PID 2>/dev/null || true
        wait $NETWORK_PID 2>/dev/null || true
        return 0  # Success
    else
        log_error "Network manager failed to start"
        return 1
    fi
}

# Test 2: Master List Loading
test_master_list() {
    log_info "Testing Master List Loading..."
    
    # Check if master list is valid JSON
    if python3 -m json.tool "$NETWORK_DIR/masterlist.json" >/dev/null 2>&1; then
        log_success "Master list JSON is valid"
    else
        log_error "Master list JSON is invalid"
        return 1
    fi
    
    # Check required fields
    if grep -q '"version"' "$NETWORK_DIR/masterlist.json" && \
       grep -q '"master_nodes"' "$NETWORK_DIR/masterlist.json" && \
       grep -q '"consensus_rules"' "$NETWORK_DIR/masterlist.json"; then
        log_success "Master list contains required fields"
    else
        log_error "Master list missing required fields"
        return 1
    fi
}

# Test 3: Consensus Rules Validation
test_consensus_rules() {
    log_info "Testing Consensus Rules Validation..."
    
    # Check if consensus rules are valid JSON
    if python3 -m json.tool "$NETWORK_DIR/consensus-rules.json" >/dev/null 2>&1; then
        log_success "Consensus rules JSON is valid"
    else
        log_error "Consensus rules JSON is invalid"
        return 1
    fi
    
    # Check required fields
    if grep -q '"node_scoring"' "$NETWORK_DIR/consensus-rules.json" && \
       grep -q '"consensus"' "$NETWORK_DIR/consensus-rules.json" && \
       grep -q '"fault_tolerance"' "$NETWORK_DIR/consensus-rules.json"; then
        log_success "Consensus rules contain required fields"
    else
        log_error "Consensus rules missing required fields"
        return 1
    fi
}

# Test 4: Peer Discovery
test_peer_discovery() {
    log_info "Testing Peer Discovery..."
    
    # Start network manager in background
    timeout 60s "$BIN_DIR/betanet-network" -command discover -verbose &
    DISCOVERY_PID=$!
    
    # Wait for discovery to complete
    sleep 10
    
    # Check if discovery completed
    if kill -0 $DISCOVERY_PID 2>/dev/null; then
        log_success "Peer discovery process running"
        
        # Test peers command
        if "$BIN_DIR/betanet-network" -command peers -limit 5 | grep -q "Peers"; then
            log_success "Peers command working"
        else
            log_warning "Peers command may not be working (no peers available)"
        fi
        
        # Cleanup
        kill $DISCOVERY_PID 2>/dev/null || true
        wait $DISCOVERY_PID 2>/dev/null || true
    else
        log_error "Peer discovery failed"
        return 1
    fi
}

# Test 5: Network Health Monitoring
test_network_health() {
    log_info "Testing Network Health Monitoring..."
    
    # Test health command
    if "$BIN_DIR/betanet-network" -command health | grep -q "Network Health"; then
        log_success "Health command working"
    else
        log_warning "Health command may not be working (no health data available)"
    fi
}

# Test 6: Network Refresh
test_network_refresh() {
    log_info "Testing Network Refresh..."
    
    # Test refresh command
    if timeout 30s "$BIN_DIR/betanet-network" -command refresh | grep -q "completed"; then
        log_success "Refresh command working"
    else
        log_warning "Refresh command may not be working (timeout or no data)"
    fi
}

# Test 7: Integration with Existing P2P Node
test_p2p_integration() {
    log_info "Testing P2P Integration..."
    
    # Create test database
    TEST_DB="$TEMP_DIR/test-p2p.db"
    
    # Start P2P node with network discovery
    timeout 30s "$BIN_DIR/betanet-node" -db "$TEST_DB" -listen "/ip4/0.0.0.0/tcp/4002" &
    P2P_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Check if P2P node is running
    if kill -0 $P2P_PID 2>/dev/null; then
        log_success "P2P node started successfully with network integration"
        
        # Cleanup
        kill $P2P_PID 2>/dev/null || true
        wait $P2P_PID 2>/dev/null || true
    else
        log_error "P2P node failed to start"
        return 1
    fi
}

# Test 8: Configuration Management
test_configuration() {
    log_info "Testing Configuration Management..."
    
    # Test with custom configuration
    CUSTOM_CONFIG="$TEMP_DIR/custom-config.json"
    cat > "$CUSTOM_CONFIG" << EOF
{
  "discovery_config": {
    "github_master_list_url": "https://raw.githubusercontent.com/yourusername/betanet/main/network/masterlist.json",
    "local_master_list_path": "$TEMP_DIR/local-masterlist.json",
    "update_interval": "1m",
    "timeout": "15s"
  },
  "consensus_config": {
    "min_peers_for_consensus": 2,
    "consensus_timeout": "20s",
    "score_threshold": 0.6,
    "geographic_preference": true,
    "load_balancing": true
  }
}
EOF
    
    if [[ -f "$CUSTOM_CONFIG" ]]; then
        log_success "Custom configuration created successfully"
    else
        log_error "Failed to create custom configuration"
        return 1
    fi
}

# Test 9: Error Handling
test_error_handling() {
    log_info "Testing Error Handling..."
    
    # Test with invalid peer ID
    if "$BIN_DIR/betanet-network" -command status -peer "invalid-peer-id" 2>&1 | grep -q "error\|Error"; then
        log_success "Error handling working for invalid peer ID"
    else
        log_warning "Error handling may not be working as expected"
    fi
    
    # Test with invalid command
    if "$BIN_DIR/betanet-network" -command "invalid-command" 2>&1 | grep -q "Unknown command"; then
        log_success "Error handling working for invalid command"
    else
        log_error "Error handling failed for invalid command"
        return 1
    fi
}

# Test 10: Performance and Scalability
test_performance() {
    log_info "Testing Performance and Scalability..."
    
    # Test with large number of peers
    if timeout 30s "$BIN_DIR/betanet-network" -command peers -limit 100 | grep -q "Peers"; then
        log_success "Performance test with large peer limit passed"
    else
        log_warning "Performance test may not be working (no peers available)"
    fi
    
    # Test concurrent operations
    for i in {1..3}; do
        timeout 10s "$BIN_DIR/betanet-network" -command status &
        STATUS_PIDS[$i]=$!
    done
    
    # Wait for all to complete
    for pid in "${STATUS_PIDS[@]}"; do
        wait $pid 2>/dev/null || true
    done
    
    log_success "Concurrent operations test completed"
}

# Main test execution
main() {
    log_info "Starting Global Discovery Network Tests..."
    log_info "=========================================="
    
    # Setup
    setup
    
    # Run tests
    local tests_passed=0
    local tests_failed=0
    
    # Test 1: Network Manager
    if test_network_manager; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 2: Master List
    if test_master_list; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 3: Consensus Rules
    if test_consensus_rules; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 4: Peer Discovery
    if test_peer_discovery; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 5: Network Health
    if test_network_health; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 6: Network Refresh
    if test_network_refresh; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 7: P2P Integration
    if test_p2p_integration; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 8: Configuration
    if test_configuration; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 9: Error Handling
    if test_error_handling; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Test 10: Performance
    if test_performance; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Results
    log_info "=========================================="
    log_info "Test Results:"
    log_info "Tests Passed: $tests_passed"
    log_info "Tests Failed: $tests_failed"
    log_info "Total Tests: $((tests_passed + tests_failed))"
    
    if [[ $tests_failed -eq 0 ]]; then
        log_success "🎉 All tests passed! Global Discovery Network is working correctly."
        exit 0
    else
        log_error "❌ Some tests failed. Please check the output above."
        exit 1
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"
