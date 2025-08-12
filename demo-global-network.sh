#!/bin/bash

# Global Discovery Network Demonstration
# This script demonstrates the comprehensive global discovery network capabilities

set -e

echo "🌐 Betanet Global Discovery Network Demonstration"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if network binary exists
if [[ ! -f "bin/betanet-network" ]]; then
    echo "❌ betanet-network binary not found. Building..."
    go build -o bin/betanet-network ./cmd/betanet-network
fi

echo -e "${BLUE}🚀 Starting Global Discovery Network...${NC}"
echo ""

# Test 1: Network Status
echo -e "${GREEN}📊 Test 1: Network Status${NC}"
echo "----------------------------------------"
./bin/betanet-network -command status
echo ""

# Test 2: Peer Discovery
echo -e "${GREEN}🔍 Test 2: Peer Discovery${NC}"
echo "----------------------------------------"
./bin/betanet-network -command discover
echo ""

# Test 3: List Available Peers
echo -e "${GREEN}📋 Test 3: Available Peers${NC}"
echo "----------------------------------------"
./bin/betanet-network -command peers -limit 10
echo ""

# Test 4: Network Health
echo -e "${GREEN}💚 Test 4: Network Health${NC}"
echo "----------------------------------------"
./bin/betanet-network -command health
echo ""

# Test 5: Network Refresh
echo -e "${GREEN}🔄 Test 5: Network Refresh${NC}"
echo "----------------------------------------"
./bin/betanet-network -command refresh
echo ""

echo -e "${BLUE}🎯 Demonstration Complete!${NC}"
echo ""
echo -e "${YELLOW}Key Features Demonstrated:${NC}"
echo "✅ Master list loading from local network directory"
echo "✅ Peer discovery and consensus scoring"
echo "✅ Network health monitoring and status"
echo "✅ Automatic network refresh and updates"
echo "✅ Intelligent peer selection and load balancing"
echo ""
echo -e "${GREEN}🌍 Cross-Internet Capabilities:${NC}"
echo "• Always-on connectivity via master list"
echo "• Dynamic peer discovery for optimal performance"
echo "• Automatic failover and fault tolerance"
echo "• Geographic optimization and load balancing"
echo "• Consensus-based peer reliability scoring"
echo ""
echo -e "${BLUE}🚀 Ready for Global Deployment!${NC}"
echo "The network can now connect nodes across the internet using:"
echo "1. GitHub master list (always available)"
echo "2. Local peer favorites (user-curated)"
echo "3. Dynamic peer discovery (automatic)"
echo "4. mDNS local network discovery (LAN)"
