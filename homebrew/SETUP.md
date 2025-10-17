# Homebrew Setup Guide

This guide walks you through setting up the Homebrew tap for wikimd.

## What's Been Done

✅ **GoReleaser Configuration** (`.goreleaser.yaml`)
  - Configured to build wikimd binary for macOS, Linux, and Windows
  - Supports both amd64 and arm64 architectures
  - Embeds version, commit, and build date metadata

✅ **GitHub Actions Workflow** (`.github/workflows/release.yml`)
  - Automatically triggers on version tags (e.g., `v0.1.0`)
  - Builds frontend assets with Bun
  - Runs tests before release
  - Creates GitHub releases with binaries and checksums

✅ **Homebrew Formula** (`homebrew/wikimd.rb`)
  - Binary-only installation (no build dependencies needed)
  - Supports macOS (arm64/amd64) and Linux (arm64/amd64)
  - Includes version tests and helpful caveats

✅ **Documentation**
  - README updated with Homebrew installation instructions
  - RELEASE.md with step-by-step release process
  - homebrew/README.md with tap setup instructions

## Next Steps

### 1. Create Your First Release

```bash
# Tag and push to create a release
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

This will trigger the GitHub Actions workflow to:
- Build binaries for all platforms
- Create a GitHub release
- Generate checksums.txt

### 2. Tap Repository Already Created! ✅

The Homebrew tap repository has been created and set up at:
**https://github.com/euforicio/homebrew-taps**

The wikimd formula is already published. Users can install with:

```bash
brew tap euforicio/taps
brew install wikimd
```

### 3. Update the Formula with Real Checksums

After the release is created:

```bash
# Download checksums from the release
curl -L "https://github.com/euforicio/wikimd/releases/download/v0.1.0/checksums.txt" -o checksums.txt

# Extract checksums
DARWIN_ARM64=$(grep "Darwin_arm64" checksums.txt | awk '{print $1}')
DARWIN_AMD64=$(grep "Darwin_x86_64" checksums.txt | awk '{print $1}')
LINUX_ARM64=$(grep "Linux_arm64" checksums.txt | awk '{print $1}')
LINUX_AMD64=$(grep "Linux_x86_64" checksums.txt | awk '{print $1}')

# Clone the tap repository and update
git clone https://github.com/euforicio/homebrew-taps.git
cd homebrew-taps
VERSION="0.1.0"  # Without 'v' prefix

# Update version and checksums
sed -i '' "s/version \".*\"/version \"$VERSION\"/" Formula/wikimd.rb
sed -i '' "s/PLACEHOLDER_ARM64_MACOS_CHECKSUM/$DARWIN_ARM64/" Formula/wikimd.rb
sed -i '' "s/PLACEHOLDER_AMD64_MACOS_CHECKSUM/$DARWIN_AMD64/" Formula/wikimd.rb
sed -i '' "s/PLACEHOLDER_ARM64_LINUX_CHECKSUM/$LINUX_ARM64/" Formula/wikimd.rb
sed -i '' "s/PLACEHOLDER_AMD64_LINUX_CHECKSUM/$LINUX_AMD64/" Formula/wikimd.rb

# Commit and push
git add Formula/wikimd.rb
git commit -m "Update wikimd to v$VERSION"
git push origin main
```

### 4. Test the Installation

```bash
# Audit the formula
brew audit --strict --online Formula/wikimd.rb

# Test installation
brew install euforicio/taps/wikimd

# Verify it works
wikimd --version

# Test the formula tests
brew test wikimd
```

### 5. Users Can Now Install ✅

Users can install with:

```bash
brew tap euforicio/taps
brew install wikimd
```

## Updating for Future Releases

For each new release:

1. Tag and push: `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
2. Wait for GitHub Actions to complete
3. Download new checksums from the release
4. Update Formula/wikimd.rb in the tap repository
5. Test locally: `brew upgrade wikimd`

See `RELEASE.md` for detailed instructions.

## File Structure

```
wikimd/
├── .goreleaser.yaml          # GoReleaser config (already exists)
├── .github/workflows/
│   └── release.yml           # Release automation (already exists)
├── homebrew/
│   ├── wikimd.rb             # Formula template
│   ├── README.md             # Tap setup instructions
│   └── SETUP.md              # This file
├── RELEASE.md                # Release process documentation
└── README.md                 # Updated with Homebrew install instructions
```

## Resources

- [GoReleaser Documentation](https://goreleaser.com/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Creating Taps](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
