#!/bin/bash

# Script to package the skald-server with its dependencies
# This creates a self-contained directory that can be distributed

set -e

echo "Packaging skald-server..."

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Create package directory
PACKAGE_DIR="$SCRIPT_DIR/skald-package"
mkdir -p "$PACKAGE_DIR/lib"
mkdir -p "$PACKAGE_DIR/bin"

# Build the application
echo "Building application..."
make build

# Copy binaries and libraries
echo "Copying files to package directory..."
cp bin/skald-server "$PACKAGE_DIR/bin/"
cp bin/skald-client "$PACKAGE_DIR/bin/"

# Copy libraries
if [ -f "deps/whisper.cpp/build/src/libwhisper.so" ]; then
    cp deps/whisper.cpp/build/src/libwhisper.so* "$PACKAGE_DIR/lib/"
fi

# Create run scripts
echo "Creating run scripts..."
cat > "$PACKAGE_DIR/run-server.sh" << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export LD_LIBRARY_PATH="$SCRIPT_DIR/lib:$LD_LIBRARY_PATH"
"$SCRIPT_DIR/bin/skald-server" "$@"
EOF

cat > "$PACKAGE_DIR/run-client.sh" << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export LD_LIBRARY_PATH="$SCRIPT_DIR/lib:$LD_LIBRARY_PATH"
"$SCRIPT_DIR/bin/skald-client" "$@"
EOF

chmod +x "$PACKAGE_DIR/run-server.sh"
chmod +x "$PACKAGE_DIR/run-client.sh"

# Copy config file
cp config.json "$PACKAGE_DIR/"

# Create systemd service file
cat > "$PACKAGE_DIR/skald-server.service" << EOF
[Unit]
Description=Skald-Go Transcriber Service
After=network.target

[Service]
Type=simple
User=%u
WorkingDirectory=%h/skald-go
ExecStart=%h/skald-go/run-server.sh
Restart=on-failure
RestartSec=5
Environment=DISPLAY=:0

[Install]
WantedBy=default.target
EOF

# Create README
cat > "$PACKAGE_DIR/README.md" << 'EOF'
# Skald-Go Package

This is a self-contained package of the Skald-Go transcription service.

## Running the Server

To run the server, simply execute:

```bash
./run-server.sh
```

## Running the Client

To run the client, use:

```bash
./run-client.sh start   # Start transcription
./run-client.sh stop    # Stop transcription
./run-client.sh status  # Check server status
```

## Keyboard Commands

When the server is running, you can use the following keyboard commands:

- `r` - Start transcription
- `s` - Stop transcription
- `i` - Show transcriber status
- `q` - Quit the application
- `?` - Show available commands

## Installation as a Systemd Service

To install as a systemd service:

1. Copy the files to your home directory:
   ```bash
   mkdir -p ~/skald-go
   cp -r * ~/skald-go/
   ```

2. Copy the service file to your systemd user directory:
   ```bash
   mkdir -p ~/.config/systemd/user/
   cp skald-server.service ~/.config/systemd/user/
   ```

3. Reload systemd:
   ```bash
   systemctl --user daemon-reload
   ```

4. Enable and start the service:
   ```bash
   systemctl --user enable skald-server.service
   systemctl --user start skald-server.service
   ```

5. Check the status:
   ```bash
   systemctl --user status skald-server.service
   ```
EOF

echo "Package created at $PACKAGE_DIR"
echo "To run the server: cd $PACKAGE_DIR && ./run-server.sh" 