#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Run the skald-server with the correct library path
LD_LIBRARY_PATH="$PROJECT_ROOT/deps/whisper.cpp/build/src:$PROJECT_ROOT/lib:$LD_LIBRARY_PATH" "$PROJECT_ROOT/bin/skald-server" "$@" 