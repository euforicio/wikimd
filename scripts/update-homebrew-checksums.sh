#!/usr/bin/env bash
set -euo pipefail

# Simple script to download and display new checksums for Homebrew formula
# Usage: ./scripts/update-homebrew-checksums.sh [version]

VERSION="${1:-0.1.0}"
REPO="euforicio/wikimd"

echo "🍺 Getting checksums for wikimd v${VERSION}"
echo ""

# Download checksums from GitHub release
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
echo "📥 Downloading from: ${CHECKSUMS_URL}"
echo ""

CHECKSUMS=$(curl -fsSL "${CHECKSUMS_URL}")

if [ -z "$CHECKSUMS" ]; then
  echo "❌ Failed to download checksums"
  exit 1
fi

# Extract checksums
DARWIN_ARM64=$(echo "$CHECKSUMS" | grep "Darwin_arm64.tar.gz" | awk '{print $1}')
DARWIN_AMD64=$(echo "$CHECKSUMS" | grep "Darwin_x86_64.tar.gz" | awk '{print $1}')
LINUX_ARM64=$(echo "$CHECKSUMS" | grep "Linux_arm64.tar.gz" | awk '{print $1}')
LINUX_AMD64=$(echo "$CHECKSUMS" | grep "Linux_x86_64.tar.gz" | awk '{print $1}')

echo "✅ New checksums for v${VERSION}:"
echo ""
echo "macOS arm64 (Apple Silicon):"
echo "  sha256 \"${DARWIN_ARM64}\""
echo ""
echo "macOS amd64 (Intel):"
echo "  sha256 \"${DARWIN_AMD64}\""
echo ""
echo "Linux arm64:"
echo "  sha256 \"${LINUX_ARM64}\""
echo ""
echo "Linux amd64:"
echo "  sha256 \"${LINUX_AMD64}\""
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 Next steps:"
echo ""
echo "1. Clone/update the tap repository:"
echo "   git clone https://github.com/${REPO%-*}/homebrew-taps.git"
echo "   cd homebrew-taps"
echo ""
echo "2. Edit Formula/wikimd.rb and update:"
echo "   - version \"${VERSION}\""
echo "   - Replace the 4 sha256 values with the ones above"
echo ""
echo "3. Commit and push:"
echo "   git add Formula/wikimd.rb"
echo "   git commit -m \"Update wikimd to v${VERSION}\""
echo "   git push origin main"
echo ""
