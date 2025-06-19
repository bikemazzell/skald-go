#!/bin/bash

# Toggle continuous mode - start/stop continuous transcription

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_BIN="$SCRIPT_DIR/../bin/skald-client"
STATE_FILE="/tmp/skald-continuous-state"

if [ ! -f "$CLIENT_BIN" ]; then
    echo "Error: Client binary not found. Run 'make build' first."
    exit 1
fi

# Check current state
STATUS=$("$CLIENT_BIN" status 2>&1)

if echo "$STATUS" | grep -q "running"; then
    # Stop if running
    "$CLIENT_BIN" stop
    rm -f "$STATE_FILE"
    echo "Continuous transcription stopped"
else
    # Start in continuous mode
    "$CLIENT_BIN" start --continuous
    echo "active" > "$STATE_FILE"
    echo "Continuous transcription started"
fi