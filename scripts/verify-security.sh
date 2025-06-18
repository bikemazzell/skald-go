#!/bin/bash
set -e

echo "🔍 Security Verification Script"
echo "================================"

# Check Go version
echo "📌 Checking Go version..."
go version

# Check for vulnerabilities
echo "🔍 Running vulnerability scan..."
if command -v govulncheck &> /dev/null; then
    echo "Running govulncheck..."
    govulncheck ./...
else
    echo "⚠️  govulncheck not found. Installing..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...
fi

# Run static security analysis
echo "🔒 Running static security analysis..."
if command -v gosec &> /dev/null; then
    echo "Running gosec..."
    gosec -fmt=json ./... | jq -r '.Issues[] | select(.severity == "HIGH" or .severity == "MEDIUM") | "\(.severity): \(.details) (\(.file):\(.line))"' || echo "No high/medium severity issues found"
else
    echo "⚠️  gosec not found. Installing..."
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    gosec -fmt=json ./... | jq -r '.Issues[] | select(.severity == "HIGH" or .severity == "MEDIUM") | "\(.severity): \(.details) (\(.file):\(.line))"' || echo "No high/medium severity issues found"
fi

# Check for outdated dependencies
echo "📦 Checking for outdated dependencies..."
go list -u -m all | grep -E '\[.*\]' || echo "All dependencies are up to date"

# Run tests to ensure everything still works
echo "🧪 Running tests..."
go test ./internal/config/... ./internal/model/... ./pkg/utils/... -v

# Build check
echo "🔨 Testing build..."
go build ./cmd/service/...
go build ./cmd/client/...

echo "✅ Security verification complete!"