#!/usr/bin/env bash
set -euo pipefail

echo "🌐 Testing Complete Domain Workflow"
echo "=================================="
echo ""

# Clean up any existing test data
rm -rf /tmp/test-node /tmp/test-wallet.json

echo "1️⃣ Starting test node..."
./bin/betanet-node run -data /tmp/test-node -listen /ip4/0.0.0.0/tcp/4002 &
NODE_PID=$!
sleep 3

echo "2️⃣ Creating test wallet..."
./bin/betanet-wallet new -out /tmp/test-wallet.json > /tmp/wallet-output.txt
MNEMONIC=$(tail -n +3 /tmp/wallet-output.txt)
echo "   Wallet created with mnemonic (first 20 chars): ${MNEMONIC:0:20}..."

echo "3️⃣ Adding test site..."
./bin/betanet-wallet add-site -wallet /tmp/test-wallet.json -mnemonic "$MNEMONIC" -label testsite

echo "4️⃣ Registering domain 'test.bn'..."
./bin/betanet-wallet register-domain \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -domain test.bn \
  -data /tmp/test-node

echo "5️⃣ Creating and publishing content..."
echo "# Test Decentralized Site" > /tmp/test-content.md
echo "" >> /tmp/test-content.md
echo "This is a test site published to Betanet!" >> /tmp/test-content.md
echo "" >> /tmp/test-content.md
echo "Domain: test.bn" >> /tmp/test-content.md
echo "Site ID: $(./bin/betanet-wallet list -wallet /tmp/test-wallet.json -mnemonic "$MNEMONIC" | grep testsite | cut -d'=' -f3 | cut -d' ' -f1)" >> /tmp/test-content.md

./bin/betanet-wallet publish \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -content /tmp/test-content.md \
  -data /tmp/test-node

echo "6️⃣ Verifying domain resolution..."
./bin/betanet-wallet resolve-domain -data /tmp/test-node -domain test.bn

echo ""
echo "✅ Domain workflow complete!"
echo ""
echo "🌐 Now you can browse to 'test.bn' using:"
echo "   ./bin/betanet-browser -data /tmp/test-node"
echo ""
echo "   The browser will resolve 'test.bn' to your site!"
echo ""
echo "🧹 Cleanup: kill $NODE_PID && rm -rf /tmp/test-node /tmp/test-wallet.json"
