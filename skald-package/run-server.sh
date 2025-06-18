#!/bin/bash

# Add error handling
set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Change to script directory (package root)
cd "$SCRIPT_DIR"

echo "Starting Skald-Go server..."

LD_LIBRARY_PATH="$SCRIPT_DIR/lib" ./bin/skald-server
