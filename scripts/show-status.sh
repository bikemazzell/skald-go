#!/bin/bash

# Show detailed status of the transcription service

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_BIN="$SCRIPT_DIR/../bin/skald-client"

if [ ! -f "$CLIENT_BIN" ]; then
    echo "Error: Client binary not found. Run 'make build' first."
    exit 1
fi

echo "=== Skald Transcription Service Status ==="
echo

# Get verbose status
"$CLIENT_BIN" status --verbose

echo
echo "=== Recent Activity Logs ==="
echo

# Get logs
"$CLIENT_BIN" logs