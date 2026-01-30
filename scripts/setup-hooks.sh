#!/usr/bin/env bash
set -euo pipefail

# Setup git hooks for local development
# This script configures git to use .githooks directory for hooks

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "Setting up git hooks..."

# Configure git to use .githooks directory
git config core.hooksPath .githooks

# Ensure hooks are executable
chmod +x "$REPO_ROOT/.githooks/"*

echo "Git hooks installed successfully!"
echo ""
echo "The following hooks are now active:"
ls -la "$REPO_ROOT/.githooks/"
echo ""
echo "To disable hooks temporarily, run:"
echo "  git commit --no-verify"
