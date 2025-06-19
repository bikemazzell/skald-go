#!/bin/bash

# Quick transcribe - one-shot transcription for hotkey integration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_BIN="$SCRIPT_DIR/../bin/skald-client"

if [ ! -f "$CLIENT_BIN" ]; then
    echo "Error: Client binary not found. Run 'make build' first."
    exit 1
fi

# Start transcription
"$CLIENT_BIN" start

# Wait for user to speak (they'll manually stop)
echo "Recording... Press hotkey to stop or wait for silence detection"

# Alternative: Auto-stop after a delay (uncomment if preferred)
# sleep 10
# "$CLIENT_BIN" stop