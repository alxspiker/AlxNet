#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_BIN="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"

# Build and run the Browser UI
if [[ ! -f "$SCRIPT_DIR/bin/betanet-browser" ]]; then
  echo "Building betanet-browser..."
  GOOS=linux GOARCH=amd64 "$GO_BIN" build -o "$SCRIPT_DIR/bin/betanet-browser" ./cmd/betanet-browser
fi
"$SCRIPT_DIR/bin/betanet-browser"


