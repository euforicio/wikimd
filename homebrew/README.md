# Homebrew Formula for wikimd

This directory contains the Homebrew formula for installing wikimd via Homebrew.

## Installation (for users)

Users can install wikimd with:

```bash
brew tap euforicio/taps
brew install wikimd
```

## Setting up the Homebrew Tap (for maintainers)

### 1. Create the tap repository

The tap repository `euforicio/homebrew-taps` has been created at https://github.com/euforicio/homebrew-taps.

The tap is already set up and available at:
https://github.com/euforicio/homebrew-taps

To add or update formulas:
```bash
# Clone the tap repository
git clone https://github.com/euforicio/homebrew-taps.git
cd homebrew-taps

# Update the formula
cp /path/to/wikimd/homebrew/wikimd.rb Formula/

# Commit and push
git add Formula/wikimd.rb
git commit -m "Update wikimd formula"
git push origin main
```

### 2. Update the formula after each release

After creating a new release (e.g., v0.1.0):

1. **Get the checksums** from the GitHub release page:
   ```bash
   # Download checksums.txt from the release
   curl -L https://github.com/euforicio/wikimd/releases/download/v0.1.0/checksums.txt
   ```

2. **Update the formula**:
   - Update the `version` line
   - Replace the checksum placeholders with actual SHA256 values from `checksums.txt`

3. **Test locally**:
   ```bash
   brew install --build-from-source Formula/wikimd.rb
   brew test wikimd
   brew audit --strict --online Formula/wikimd.rb
   ```

4. **Commit and push**:
   ```bash
   git add Formula/wikimd.rb
   git commit -m "Update wikimd to v0.1.0"
   git push
   ```

### 3. Automate formula updates (optional)

You can automate this process with a GitHub Action in the tap repository. Create `.github/workflows/update-formula.yml`:

```yaml
name: Update Formula
on:
  repository_dispatch:
    types: [release]
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to update to (e.g., 0.1.0)'
        required: true

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Update formula
        env:
          VERSION: ${{ github.event.inputs.version || github.event.client_payload.version }}
        run: |
          # Download checksums
          curl -L "https://github.com/euforicio/wikimd/releases/download/v${VERSION}/checksums.txt" -o checksums.txt

          # Extract checksums for each platform
          DARWIN_ARM64=$(grep "Darwin_arm64" checksums.txt | awk '{print $1}')
          DARWIN_AMD64=$(grep "Darwin_x86_64" checksums.txt | awk '{print $1}')
          LINUX_ARM64=$(grep "Linux_arm64" checksums.txt | awk '{print $1}')
          LINUX_AMD64=$(grep "Linux_x86_64" checksums.txt | awk '{print $1}')

          # Update the formula
          sed -i "s/version \".*\"/version \"${VERSION}\"/" Formula/wikimd.rb
          sed -i "s/PLACEHOLDER_ARM64_MACOS_CHECKSUM/${DARWIN_ARM64}/" Formula/wikimd.rb
          sed -i "s/PLACEHOLDER_AMD64_MACOS_CHECKSUM/${DARWIN_AMD64}/" Formula/wikimd.rb
          sed -i "s/PLACEHOLDER_ARM64_LINUX_CHECKSUM/${LINUX_ARM64}/" Formula/wikimd.rb
          sed -i "s/PLACEHOLDER_AMD64_LINUX_CHECKSUM/${LINUX_AMD64}/" Formula/wikimd.rb

      - name: Commit changes
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add Formula/wikimd.rb
          git commit -m "Update wikimd to v${VERSION}"
          git push
```

Then trigger it from the main wikimd repository's release workflow by adding:

```yaml
- name: Trigger Homebrew formula update
  run: |
    curl -X POST \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer ${{ secrets.HOMEBREW_TAP_TOKEN }}" \
      https://api.github.com/repos/euforicio/homebrew-taps/dispatches \
      -d '{"event_type":"release","client_payload":{"version":"${{ github.ref_name }}"}}'
```

## Testing the Formula

```bash
# Test installation from source
brew install --build-from-source Formula/wikimd.rb

# Run formula tests
brew test wikimd

# Audit the formula
brew audit --strict --online Formula/wikimd.rb

# Uninstall
brew uninstall wikimd
```

## Manual Installation (without tap)

Users can also install directly from the formula file:

```bash
brew install https://raw.githubusercontent.com/euforicio/homebrew-taps/main/Formula/wikimd.rb
```
