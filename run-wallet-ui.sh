#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_BIN="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"

# Build and run the GUI focused on the Wallet tab
if [[ ! -f "$SCRIPT_DIR/bin/betanet-gui" ]]; then
  echo "Building betanet-gui..."
  GOOS=linux GOARCH=amd64 "$GO_BIN" build -o "$SCRIPT_DIR/bin/betanet-gui" ./cmd/betanet-gui
fi
"$SCRIPT_DIR/bin/betanet-gui" -tab wallet


