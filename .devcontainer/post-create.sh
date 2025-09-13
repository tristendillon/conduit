#!/usr/bin/env bash
set -euo pipefail

echo "🚀 Setting up Conduit development environment..."

# Ensure proper PATH
export PATH="$HOME/.bun/bin:$HOME/go/bin:/usr/local/go/bin:$PATH"

# Display versions
echo "--- Environment Versions ---"
echo "Go version: $(go version)"
echo "Node version: $(node --version)"
echo "Bun version: $(bun --version)"

# Install essential Go tools
echo "--- Installing Go development tools ---"
go install github.com/spf13/cobra-cli@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/air-verse/air@latest

# Install essential frontend tools
echo "--- Installing frontend tools ---"
bun install -g typescript prettier eslint
bun install -g opencode-ai
# Initialize Go module if it doesn't exist
if [ ! -f "go.mod" ]; then
  echo "Initializing go.mod..."
  go mod init github.com/conduit-framework/conduit
fi

# GitHub CLI setup if authenticated
if command -v gh &> /dev/null && gh auth status &> /dev/null; then
  echo "✅ GitHub CLI authenticated"
  gh auth setup-git || true
fi

echo ""
echo "✅ Conduit development environment ready!"
echo ""
echo "🎯 Quick commands:"
echo "  • go run ./cmd/conduit --help"
echo "  • air                    (hot reload)"
echo "  • bun run dev           (if package.json exists)"
echo ""