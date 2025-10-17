# Release Process

This document describes the release process for wikimd, including creating releases, updating the Homebrew formula, and testing.

## Overview

wikimd uses [GoReleaser](https://goreleaser.com/) to automate binary releases for multiple platforms. When a version tag is pushed, GitHub Actions automatically:

1. Builds binaries for macOS, Linux, and Windows (amd64 & arm64)
2. Generates checksums
3. Creates a GitHub release with artifacts
4. Generates a changelog

## Creating a New Release

### 1. Prepare the Release

Before tagging, ensure:

- [ ] All tests pass: `go test ./...`
- [ ] Code is linted: `golangci-lint run ./...` (if available)
- [ ] Web assets build: `make web-build`
- [ ] Version information is correct in code
- [ ] CHANGELOG or release notes are ready (optional)

### 2. Create and Push the Tag

```bash
# Determine the new version (follow semantic versioning)
VERSION="v0.1.0"  # e.g., v0.1.0, v1.0.0, v1.2.3

# Create an annotated tag
git tag -a "$VERSION" -m "Release $VERSION"

# Push the tag to trigger the release workflow
git push origin "$VERSION"
```

### 3. Monitor the Release

1. Go to [GitHub Actions](https://github.com/euforicio/wikimd/actions)
2. Watch the "Release" workflow
3. Verify all builds complete successfully
4. Check the [Releases page](https://github.com/euforicio/wikimd/releases) for the new release

### 4. Update the Homebrew Formula

After the release is published:

1. **Download the checksums file:**
   ```bash
   curl -L "https://github.com/euforicio/wikimd/releases/download/$VERSION/checksums.txt" -o checksums.txt
   cat checksums.txt
   ```

2. **Extract platform-specific checksums:**
   ```bash
   # macOS ARM64 (Apple Silicon)
   DARWIN_ARM64=$(grep "Darwin_arm64" checksums.txt | awk '{print $1}')

   # macOS AMD64 (Intel)
   DARWIN_AMD64=$(grep "Darwin_x86_64" checksums.txt | awk '{print $1}')

   # Linux ARM64
   LINUX_ARM64=$(grep "Linux_arm64" checksums.txt | awk '{print $1}')

   # Linux AMD64
   LINUX_AMD64=$(grep "Linux_x86_64" checksums.txt | awk '{print $1}')

   echo "Darwin ARM64: $DARWIN_ARM64"
   echo "Darwin AMD64: $DARWIN_AMD64"
   echo "Linux ARM64: $LINUX_ARM64"
   echo "Linux AMD64: $LINUX_AMD64"
   ```

3. **Update the formula in the tap repository:**

   ```bash
   # Clone or navigate to the tap repository
   git clone https://github.com/euforicio/homebrew-taps.git
   cd homebrew-taps

   # Update Formula/wikimd.rb with the new version and checksums
   # Replace:
   #   - version "0.1.0" with the new version number (without 'v' prefix)
   #   - All PLACEHOLDER checksums with actual values from above

   # Example using sed (macOS/BSD sed):
   VERSION_NUMBER="${VERSION#v}"  # Remove 'v' prefix
   sed -i '' "s/version \".*\"/version \"$VERSION_NUMBER\"/" Formula/wikimd.rb
   sed -i '' "s/PLACEHOLDER_ARM64_MACOS_CHECKSUM/$DARWIN_ARM64/" Formula/wikimd.rb
   sed -i '' "s/PLACEHOLDER_AMD64_MACOS_CHECKSUM/$DARWIN_AMD64/" Formula/wikimd.rb
   sed -i '' "s/PLACEHOLDER_ARM64_LINUX_CHECKSUM/$LINUX_ARM64/" Formula/wikimd.rb
   sed -i '' "s/PLACEHOLDER_AMD64_LINUX_CHECKSUM/$LINUX_AMD64/" Formula/wikimd.rb
   ```

4. **Test the formula locally:**

   ```bash
   # Audit the formula
   brew audit --strict --online Formula/wikimd.rb

   # Test installation
   brew install --build-from-source Formula/wikimd.rb

   # Run tests
   brew test wikimd

   # Verify binaries work
   wikimd --version
   wikimd-export --version

   # Uninstall for clean state
   brew uninstall wikimd
   ```

5. **Commit and push the updated formula:**

   ```bash
   git add Formula/wikimd.rb
   git commit -m "Update wikimd to $VERSION"
   git push origin main
   ```

### 5. Announce the Release

- Update any relevant documentation
- Post on social media or community channels (if applicable)
- Close related issues or pull requests

## Semantic Versioning

wikimd follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v2.0.0): Incompatible API changes
- **MINOR** version (v1.1.0): New functionality in a backwards-compatible manner
- **PATCH** version (v1.0.1): Backwards-compatible bug fixes

## Release Checklist Template

Copy this checklist when preparing a release:

```markdown
## Release vX.Y.Z Checklist

### Pre-release
- [ ] All tests passing
- [ ] Code linted (if golangci-lint available)
- [ ] Web assets build successfully
- [ ] Version bumped in relevant files (if any)
- [ ] Release notes drafted

### Release
- [ ] Tag created and pushed: `git tag -a vX.Y.Z -m "Release vX.Y.Z" && git push origin vX.Y.Z`
- [ ] GitHub Actions workflow completed successfully
- [ ] Release published on GitHub with all artifacts

### Homebrew Formula Update
- [ ] Checksums downloaded and extracted
- [ ] Formula updated with new version and checksums
- [ ] Formula audited: `brew audit --strict --online`
- [ ] Formula tested locally: `brew install --build-from-source`
- [ ] Formula tests passed: `brew test wikimd`
- [ ] Binaries verified: `wikimd --version && wikimd-export --version`
- [ ] Formula committed and pushed to tap

### Post-release
- [ ] Documentation updated (if needed)
- [ ] Release announced (if applicable)
- [ ] Related issues/PRs closed
```

## Troubleshooting

### Release Workflow Fails

1. **Check the logs** in GitHub Actions for specific errors
2. **Common issues:**
   - Tests failing: Fix tests and push to main before tagging
   - Build errors: Ensure `go.mod` is up to date
   - Asset build fails: Verify Bun and frontend dependencies

### Formula Installation Fails

1. **Check the formula syntax:**
   ```bash
   brew audit --strict Formula/wikimd.rb
   ```

2. **Verify checksums:**
   - Ensure checksums match the release artifacts
   - Download and verify manually:
     ```bash
     curl -L URL | shasum -a 256
     ```

3. **Test with verbose output:**
   ```bash
   brew install --build-from-source --verbose Formula/wikimd.rb
   ```

### Version Mismatch

If `wikimd --version` doesn't show the expected version:

1. Check ldflags in `.goreleaser.yaml`
2. Verify `internal/buildinfo` package exists and is imported
3. Re-run the release process

## Automation (Future Enhancement)

Consider setting up automated formula updates using GitHub Actions. See `homebrew/README.md` for details on implementing automated formula updates via workflow dispatch or repository dispatch events.

## References

- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
