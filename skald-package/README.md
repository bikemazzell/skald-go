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
