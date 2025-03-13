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
mkdir -p "$PACKAGE_DIR/models"

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

# Copy models if they exist
if [ -d "models" ] && [ "$(ls -A models)" ]; then
    echo "Copying models to package directory..."
    cp models/* "$PACKAGE_DIR/models/"
fi

# Copy download-model script
echo "Copying download-model script..."
cp download-model.sh "$PACKAGE_DIR/"
chmod +x "$PACKAGE_DIR/download-model.sh"

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

## Downloading Models

If you need to download additional models, use:

```bash
./download-model.sh
```

This script will allow you to choose which model to download from the available options in config.json.

## Keyboard Commands

When the server is running, you can use the following keyboard commands:

- `r` - Start transcription
- `s` - Stop transcription
- `i` - Show transcriber status
- `q` - Quit the application
- `?` - Show available commands

## Installation as a Systemd Service

To install as a systemd service:

EOF

# Add installation instructions to README
cat >> "$PACKAGE_DIR/README.md" << 'EOF'
1. Copy the service file to your user's systemd directory:
   ```bash
   mkdir -p ~/.config/systemd/user/
   cp skald-server.service ~/.config/systemd/user/
   ```

2. Reload systemd to recognize the new service:
   ```bash
   systemctl --user daemon-reload
   ```

3. Enable and start the service:
   ```bash
   systemctl --user enable skald-server.service
   systemctl --user start skald-server.service
   ```

4. Check the status of the service:
   ```bash
   systemctl --user status skald-server.service
   ```

5. View logs:
   ```bash
   journalctl --user -u skald-server.service -f
   ```
EOF

echo "Package created at $PACKAGE_DIR"
echo "To run the server from the package directory:"
echo "  cd $PACKAGE_DIR && ./run-server.sh" 