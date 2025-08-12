#!/usr/bin/env bash
set -euo pipefail

# Build all CLI and GUI binaries for Linux amd64

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT_DIR="$ROOT_DIR/bin"
GO_BIN="${GO_BIN:-$(command -v go || echo /usr/local/go/bin/go)}"
mkdir -p "$OUT_DIR"

echo "==> Using Go: $GO_BIN"

echo "==> Building betanet-node"
GOOS=linux GOARCH=amd64 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_DIR/betanet-node" ./cmd/betanet-node

echo "==> Building betanet-wallet"
GOOS=linux GOARCH=amd64 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_DIR/betanet-wallet" ./cmd/betanet-wallet

echo "==> Building betanet-gui (requires Linux GUI dev libs; see README)"
GOOS=linux GOARCH=amd64 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_DIR/betanet-gui" ./cmd/betanet-gui || {
  echo "[warn] betanet-gui build failed (likely missing X11/OpenGL dev libs). CLI builds are ready." >&2
}

echo "==> Building betanet-browser (requires Linux GUI dev libs; see README)"
GOOS=linux GOARCH=amd64 "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_DIR/betanet-browser" ./cmd/betanet-browser || {
  echo "[warn] betanet-browser build failed (likely missing X11/OpenGL dev libs)." >&2
}

echo "==> Done. Binaries in: $OUT_DIR"


