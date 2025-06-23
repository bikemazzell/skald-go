#!/bin/bash

# Add error handling
set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Change to project root
cd "$PROJECT_ROOT"

# Check if built binary exists
if [ ! -f "bin/skald-server" ]; then
    echo "Binary not found. Building..."
    make build
fi

echo "Starting Skald-Go server..."

LD_LIBRARY_PATH="$PROJECT_ROOT/lib" ./bin/skald-server "$@" 