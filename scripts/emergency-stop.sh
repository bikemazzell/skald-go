#!/bin/bash

# Emergency stop - force stop all recording and kill server if needed

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_BIN="$SCRIPT_DIR/../bin/skald-client"
SOCKET_PATH="/tmp/skald.sock"

# Try normal stop first
if [ -f "$CLIENT_BIN" ]; then
    "$CLIENT_BIN" stop 2>/dev/null
fi

# Remove state files
rm -f /tmp/skald-continuous-state

# Check if server is still running
if pgrep -f "skald-server" > /dev/null; then
    echo "Server still running, sending SIGTERM..."
    pkill -TERM -f "skald-server"
    sleep 1
fi

# Force kill if still running
if pgrep -f "skald-server" > /dev/null; then
    echo "Force killing server..."
    pkill -KILL -f "skald-server"
fi

# Clean up socket
rm -f "$SOCKET_PATH"

echo "Emergency stop completed"