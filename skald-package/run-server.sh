#!/bin/bash

# Add error handling
set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Change to script directory (package root)
cd "$SCRIPT_DIR"

echo "Starting Skald-Go server..."

# Check if config says to be silent
if grep -q '"silent": true' config.json; then
    echo "Running in silent mode..."
    # Redirect stderr to suppress whisper.cpp verbose output
    LD_LIBRARY_PATH="$SCRIPT_DIR/lib" ./bin/skald-server 2>/dev/null
else
    echo "Running with verbose whisper output..."
    LD_LIBRARY_PATH="$SCRIPT_DIR/lib" ./bin/skald-server
fi
