#!/bin/bash

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run the skald-client with the correct library path
LD_LIBRARY_PATH="$SCRIPT_DIR/deps/whisper.cpp/build/src:$SCRIPT_DIR/lib:$LD_LIBRARY_PATH" "$SCRIPT_DIR/bin/skald-client" "$@" 