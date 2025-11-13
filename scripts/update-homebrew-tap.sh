#!/usr/bin/env bash
set -euo pipefail

# Script to update the Homebrew tap with new checksums after a release
# Usage: ./scripts/update-homebrew-tap.sh [version]

VERSION="${1:-0.1.0}"
REPO="euforicio/wikimd"
TAP_REPO="euforicio/homebrew-taps"

echo "🍺 Updating Homebrew tap for wikimd v${VERSION}"

# Download checksums from GitHub release
echo "📥 Downloading checksums from GitHub release..."
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/v${VERSION}/checksums.txt"
CHECKSUMS=$(curl -fsSL "${CHECKSUMS_URL}")

if [ -z "$CHECKSUMS" ]; then
  echo "❌ Failed to download checksums from ${CHECKSUMS_URL}"
  exit 1
fi

# Extract checksums for each platform
echo "🔍 Extracting checksums..."
DARWIN_ARM64=$(echo "$CHECKSUMS" | grep "Darwin_arm64.tar.gz" | awk '{print $1}')
DARWIN_AMD64=$(echo "$CHECKSUMS" | grep "Darwin_x86_64.tar.gz" | awk '{print $1}')
LINUX_ARM64=$(echo "$CHECKSUMS" | grep "Linux_arm64.tar.gz" | awk '{print $1}')
LINUX_AMD64=$(echo "$CHECKSUMS" | grep "Linux_x86_64.tar.gz" | awk '{print $1}')

# Verify all checksums were found
if [ -z "$DARWIN_ARM64" ] || [ -z "$DARWIN_AMD64" ] || [ -z "$LINUX_ARM64" ] || [ -z "$LINUX_AMD64" ]; then
  echo "❌ Failed to extract all checksums"
  echo "Darwin arm64: ${DARWIN_ARM64:-missing}"
  echo "Darwin amd64: ${DARWIN_AMD64:-missing}"
  echo "Linux arm64: ${LINUX_ARM64:-missing}"
  echo "Linux amd64: ${LINUX_AMD64:-missing}"
  exit 1
fi

echo "✅ Checksums extracted:"
echo "  Darwin arm64:  $DARWIN_ARM64"
echo "  Darwin amd64:  $DARWIN_AMD64"
echo "  Linux arm64:   $LINUX_ARM64"
echo "  Linux amd64:   $LINUX_AMD64"

# Clone or update tap repository
TAP_DIR="${TMPDIR:-/tmp}/homebrew-taps"
if [ -d "$TAP_DIR" ]; then
  echo "🔄 Updating existing tap repository..."
  cd "$TAP_DIR"
  git fetch origin
  git reset --hard origin/main
else
  echo "📦 Cloning tap repository..."
  git clone "https://github.com/${TAP_REPO}.git" "$TAP_DIR"
  cd "$TAP_DIR"
fi

# Update the formula
FORMULA_FILE="Formula/wikimd.rb"
if [ ! -f "$FORMULA_FILE" ]; then
  echo "❌ Formula file not found: $FORMULA_FILE"
  exit 1
fi

echo "✏️  Updating formula..."

# Create a backup
cp "$FORMULA_FILE" "${FORMULA_FILE}.bak"

# Update version and checksums
sed -i.tmp "s/version \".*\"/version \"${VERSION}\"/" "$FORMULA_FILE"
sed -i.tmp "s/sha256 \"[a-f0-9]*\" # Darwin arm64/sha256 \"${DARWIN_ARM64}\"/" "$FORMULA_FILE" || \
  sed -i.tmp "/Darwin_arm64/,/sha256/ s/sha256 \"[a-f0-9]*\"/sha256 \"${DARWIN_ARM64}\"/" "$FORMULA_FILE"
sed -i.tmp "/Darwin_x86_64/,/sha256/ s/sha256 \"[a-f0-9]*\"/sha256 \"${DARWIN_AMD64}\"/" "$FORMULA_FILE"
sed -i.tmp "/Linux_arm64/,/sha256/ s/sha256 \"[a-f0-9]*\"/sha256 \"${LINUX_ARM64}\"/" "$FORMULA_FILE"
sed -i.tmp "/Linux_x86_64/,/sha256/ s/sha256 \"[a-f0-9]*\"/sha256 \"${LINUX_AMD64}\"/" "$FORMULA_FILE"

# Remove temp files
rm -f "${FORMULA_FILE}.tmp"

# Show the diff
echo ""
echo "📝 Changes to be committed:"
git diff "$FORMULA_FILE"

# Commit and push
echo ""
read -p "🚀 Commit and push changes? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  git add "$FORMULA_FILE"
  git commit -m "Update wikimd to v${VERSION}"
  git push origin main
  echo "✅ Homebrew tap updated successfully!"
  echo ""
  echo "Users can now install/upgrade with:"
  echo "  brew upgrade wikimd"
else
  echo "⏸️  Changes not committed. You can review them at:"
  echo "  $TAP_DIR"
  git restore "$FORMULA_FILE"
fi

# Cleanup
rm -f "${FORMULA_FILE}.bak"
