#!/bin/bash

# Build release binaries and packages for distribution
# This script creates both static binaries and packaged versions

set -e

echo "🚀 Building Skald-Go Release..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')

echo "📦 Version: $VERSION"
echo "🔧 Commit: $GIT_COMMIT"
echo "⏰ Build Time: $BUILD_TIME"

# Create release directory
RELEASE_DIR="$PROJECT_ROOT/release"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

# 1. Build optimized release binaries
echo ""
echo "🔨 Building release binaries..."

# Ensure dependencies are built first
if [ ! -d "deps/whisper.cpp" ]; then
    echo "📦 Installing dependencies first..."
    ./scripts/update_deps.sh
fi

# Build with optimizations for release
# Don't clean dependencies, just the binaries
rm -rf bin/
make debug  # Use the reliable dynamic build

# Test release binaries
echo "✅ Testing release binaries..."
if LD_LIBRARY_PATH=./lib ./bin/skald-server --version >/dev/null 2>&1; then
    echo "   ✓ Server binary works"
else
    echo "   ❌ Server binary test failed"
fi

if LD_LIBRARY_PATH=./lib ./bin/skald-client --version >/dev/null 2>&1; then
    echo "   ✓ Client binary works"
else
    echo "   ❌ Client binary test failed"
fi

# 2. Create release package
echo ""
echo "📦 Creating release package..."
RELEASE_PKG="$RELEASE_DIR/skald-go-$VERSION"
mkdir -p "$RELEASE_PKG/lib"

# Copy binaries and libraries
cp bin/skald-server "$RELEASE_PKG/"
cp bin/skald-client "$RELEASE_PKG/"
cp lib/* "$RELEASE_PKG/lib/" 2>/dev/null || true
cp config.json "$RELEASE_PKG/"

# Create simple run scripts
cat > "$RELEASE_PKG/skald" << 'EOF'
#!/bin/bash
# Skald-Go launcher script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export LD_LIBRARY_PATH="$SCRIPT_DIR/lib:$LD_LIBRARY_PATH"

case "${1:-help}" in
    server)
        shift
        exec "$SCRIPT_DIR/skald-server" "$@"
        ;;
    start|stop|status|logs)
        exec "$SCRIPT_DIR/skald-client" "$@"
        ;;
    download-model)
        echo "To download models, visit: https://huggingface.co/ggerganov/whisper.cpp"
        echo "Place .bin files in the same directory as this script"
        ;;
    help|*)
        echo "Skald-Go - Voice to Text Transcriber"
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  server              - Start transcription server"
        echo "  start              - Start transcription"
        echo "  stop               - Stop transcription" 
        echo "  status             - Show status"
        echo "  logs               - Show recent logs"
        echo "  download-model     - Model download instructions"
        echo "  help               - Show this help"
        ;;
esac
EOF

chmod +x "$RELEASE_PKG/skald"

# Create systemd service
cat > "$RELEASE_PKG/skald.service" << EOF
[Unit]
Description=Skald-Go Voice Transcription Service
After=network.target

[Service]
Type=simple
User=%i
WorkingDirectory=%h/.local/bin/skald-go
ExecStart=%h/.local/bin/skald-go/skald server
Restart=on-failure
RestartSec=5
Environment=DISPLAY=:0

[Install]
WantedBy=default.target
EOF

# Create README
cat > "$RELEASE_PKG/README.md" << 'EOF'
# Skald-Go Release Package

This is a self-contained build of Skald-Go with all dependencies included.

## Quick Start

1. **Start the server:**
   ```bash
   ./skald server
   ```

2. **In another terminal, start transcription:**
   ```bash
   ./skald start
   ```

3. **Stop transcription:**
   ```bash
   ./skald stop
   ```

## Installation

For system-wide installation:

```bash
# Copy to user binaries
mkdir -p ~/.local/bin/skald-go
cp -r * ~/.local/bin/skald-go/

# Add to PATH (add to ~/.bashrc for persistence)
export PATH="$HOME/.local/bin/skald-go:$PATH"

# Install as systemd service
systemctl --user daemon-reload
systemctl --user enable ~/.local/bin/skald-go/skald.service
systemctl --user start skald.service
```

## Model Download

Download whisper models from HuggingFace and place them in this directory:
- https://huggingface.co/ggerganov/whisper.cpp/tree/main

Recommended: `ggml-base.bin` (good balance of speed/quality)

## Configuration

Edit `config.json` to customize audio settings, hotkeys, and behavior.
EOF

# Create tarball
echo "📦 Creating release tarball..."
cd "$RELEASE_DIR"
tar -czf "skald-go-$VERSION-linux-x64.tar.gz" "skald-go-$VERSION"

# 4. Create checksums
echo ""
echo "🔐 Creating checksums..."
cd "$RELEASE_DIR"
sha256sum *.tar.gz > checksums.txt

echo ""
echo "✅ Release build complete!"
echo ""
echo "📂 Release files:"
ls -la "$RELEASE_DIR"
echo ""
echo "🚀 Release package: skald-go-$VERSION-linux-x64.tar.gz"
echo ""
echo "📖 Usage:"
echo "  1. Extract: tar -xzf skald-go-$VERSION-linux-x64.tar.gz"
echo "  2. Run: cd skald-go-$VERSION && ./skald server"