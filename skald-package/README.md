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
