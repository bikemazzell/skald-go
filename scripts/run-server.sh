#!/bin/bash

# Add error handling
set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to project root
cd "$PROJECT_ROOT"

echo "Starting Skald-Go server..."

# Check if built binary exists
if [ ! -f "bin/skald-server" ]; then
    echo "Binary not found. Building..."
    make build
fi

# Check if config says to be silent
if grep -q '"silent": true' config.json; then
    echo "Running in silent mode..."
    # Redirect stderr to suppress whisper.cpp verbose output
    LD_LIBRARY_PATH="$PROJECT_ROOT/lib" ./bin/skald-server 2>/dev/null
else
    echo "Running with verbose whisper output..."
    LD_LIBRARY_PATH="$PROJECT_ROOT/lib" ./bin/skald-server
fi 