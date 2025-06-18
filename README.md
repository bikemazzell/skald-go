```
╔═╗╦╔═╔═╗╦  ╔╦╗   ╔═╗╔═╗
╚═╗╠╩╗╠═╣║   ║║   ║ ╦║ ║
╚═╝╩ ╩╩ ╩╩═╝═╩╝   ╚═╝╚═╝
```
# Skald-Go - Voice to Text Transcriber
> Created by [@shoewind1997](https://github.com/bikemazzell)

Skald-Go is a lightweight speech-to-text tool that converts your voice to text in real-time using whisper.cpp. It runs quietly in the background without any graphical interface, automatically copying transcriptions to your clipboard. The application consists of two parts: a background server that handles the transcription, and a client that can be bound to a hotkey for easy start/stop control. Named after the ancient Nordic poets and storytellers known as skalds, this tool makes it effortless to transform your spoken words into written text with a single keystroke.

## Features

- 🎤 Real-time microphone input capture using miniaudio
- 🤖 Advanced speech recognition using whisper.cpp
- 📋 Automatic clipboard copying of transcribed text
- ⌨️ Auto-paste support (configurable)
- 🔄 **Continuous recording mode** - Keep recording until manually stopped
- 🎵 **Audio feedback** - Customizable tones for start, completion, and error states
- 🛑 Silence detection for automatic stopping
- 🔒 **Security-focused text validation** - Allows natural punctuation while blocking command injection
- 💪 Multiple whisper models supported
- 🎯 OpenMP optimized processing

## Privacy & Offline Usage

Skald-Go is designed with privacy in mind:
- 🔒 Completely offline after initial model download
- 🚫 No data sent to external servers
- 💻 All processing happens locally on your machine
- 🗑️ Audio data is processed in memory and not saved to disk
- 🤖 Uses local AI models for transcription

## Quick Start
```bash
Clone the repository
git clone https://github.com/bikemazzell/skald-go.git
cd skald-go
```

Download a model
```bash
./scripts/download-model.sh
```

Build the project
```bash
make build
```

OR

Clean, build dependencies, and build the project
```bash
make clean && make deps && make build
```

Run the server
```bash
./scripts/run-server.sh
```
In another terminal, control recording
```bash
./scripts/run-client.sh start # Begin recording
./scripts/run-client.sh stop # Stop recording
./scripts/run-client.sh status # Check status
```
## System Requirements

### Dependencies
- Go 1.23 or higher
- GCC/Clang compiler
- OpenMP support
- whisper.cpp (included as dependency)

### Linux-specific Dependencies
Clipboard support (Required - install either xclip or xsel)

```bash
sudo apt install xclip
```
OR

```bash
sudo apt install xsel
```
Auto-paste support (Optional)

```bash
sudo apt install xdotool
```

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/bikemazzell/skald-go.git
cd skald-go
```

2. Download a model:
```bash
./scripts/download-model.sh
```
This script will:
- Read available models from your config.json
- Allow you to choose which model to download
- Download the selected model(s) to the models directory

3. Update dependencies:
```bash
./scripts/update_deps.sh
```
This script will:
- Clone or update whisper.cpp to the latest stable version
- Copy the Go bindings from whisper.cpp
- Copy necessary header files to the appropriate locations
- Update Go module dependencies
- Update the vendor directory

4. Build the project:
```bash
make build
```
This will compile both server and client binaries.

## Creating a Distributable Package

Skald-Go includes a packaging script that creates a self-contained directory with all necessary files for distribution:

```bash
make package
```

This script will:
- Build the application
- Create a package directory (`skald-package`)
- Copy binaries, libraries, and configuration files
- Create wrapper scripts for running the server and client
- Include a systemd service file for easy installation
- Generate a README with installation and usage instructions

The resulting package can be distributed to other users or systems without requiring them to build from source. Users can simply run the included wrapper scripts to start using Skald-Go.

To install the packaged application:

1. Copy the package directory to the desired location:
   ```bash
   cp -r skald-package ~/skald-go
   ```

2. Run the server:
   ```bash
   cd ~/skald-go
   ./run-server.sh
   ```

3. In another terminal, control the transcription:
   ```bash
   cd ~/skald-go
   ./run-client.sh start   # Start transcription
   ./run-client.sh stop    # Stop transcription
   ./run-client.sh status  # Check status
   ```

## Model Management

Skald-Go includes a script to easily download and manage whisper models:

```bash
./scripts/download-model.sh
```

This script will:
- Read available models from your config.json file
- Display a list of available models with their sizes
- Allow you to choose which model to download or download all models
- Check if models already exist and ask before redownloading
- Show download progress
- Store models in the `models` directory

The script supports multiple download methods:
- curl (preferred, with progress bar)
- wget (with progress display)
- Python's urllib (as a fallback)

When you run the script, you'll see a list of available models:

```
Available models:
----------------
1) tiny (14.6MB)
2) base (146MB)
3) small (466MB)
4) large-v3-turbo-q8_0 (874MB)
5) tiny.en (77.7MB)
----------------
```

You can then enter the number of the model you want to download, or 'a' to download all models.

### Adding New Models

To add new models to the available options, edit the `config.json` file and add entries to the `whisper.models` section:

```json
"models": {
    "tiny": {
        "url": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
        "size": "14.6MB"
    },
    "new-model": {
        "url": "https://example.com/path/to/model.bin",
        "size": "100MB"
    }
}
```

## Dependency Management

Skald-Go uses a custom dependency management approach for whisper.cpp:

- Dependencies are stored in the `deps/` directory (excluded from git)
- `scripts/update_deps.sh` script handles downloading and updating dependencies
- You can specify a specific whisper.cpp version: `./scripts/update_deps.sh v1.7.4`
- The script ensures all necessary header files are properly copied
- Go modules are configured to use the local dependencies

## Project Structure

Skald-Go follows standard Go project layout:

```
skald-go/
├── bin/                  # Compiled binaries
├── cmd/                  # Application entry points
│   ├── client/           # Client application
│   └── service/          # Server application
├── config.json           # Application configuration
├── deps/                 # External dependencies
├── internal/             # Private application code
│   ├── audio/            # Audio processing
│   ├── config/           # Configuration handling
│   ├── model/            # Model management
│   ├── server/           # Server implementation
│   ├── transcriber/      # Transcription logic
│   └── whisper/          # Whisper integration
├── lib/                  # Compiled libraries
├── models/               # Downloaded whisper models
├── pkg/                  # Public libraries
│   └── utils/            # Utility functions
├── scripts/              # Utility scripts
│   ├── build-static.sh   # Build static binaries
│   ├── download-model.sh # Download whisper models
│   ├── package.sh        # Create distributable package
│   ├── run-client.sh     # Run client wrapper
│   ├── run-server.sh     # Run server wrapper
│   ├── skald-server.service # Systemd service file
│   └── update_deps.sh    # Update dependencies
├── vendor/               # Vendored dependencies
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── LICENSE               # License file
├── Makefile              # Build automation
└── README.md             # This file
```

## Configuration

The application uses a JSON configuration file. Default location is `config.json` in the working directory.

Example configuration:
```json
{
"version": "0.1",
"audio": {
"sample_rate": 16000,
"channels": 1,
"silence_threshold": 0.008,
"silence_duration": 3.0,
"frame_length": 512,
"buffered_frames": 10,
"device_index": -1,
"start_tone": {
"enabled": true,
"frequency": 440,
"duration": 150,
"fade_ms": 5
},
"completion_tone": {
"enabled": true,
"frequency": 660,
"duration": 200,
"fade_ms": 10
},
"error_tone": {
"enabled": true,
"frequency": 220,
"duration": 300,
"fade_ms": 15
}
},
"processing": {
"shutdown_timeout": 30,
"event_wait_timeout": 0.1,
"auto_paste": true,
"channel_buffer_size": 10,
"continuous_mode": {
"enabled": true,
"max_session_duration": 300,
"inter_speech_timeout": 10,
"auto_stop_on_idle": true
},
"text_validation": {
"mode": "security_focused",
"allow_punctuation": true,
"custom_blocklist": []
}
},
"whisper": {
"model": "base",
"language": "en",
"beam_size": 5
},
"server": {
"socket_path": "/tmp/skald.sock",
"socket_timeout": 5.0,
"keyboard_enabled": true
},
"debug": {
"print_status": true,
"print_transcriptions": true
}
}
```

## Keyboard Interactions

When running the server with `keyboard_enabled: true` in the config, you can use the following keyboard shortcuts:

- `r` - Start transcription (same as running `./scripts/run-client.sh start`)
- `s` - Stop transcription (same as running `./scripts/run-client.sh stop`)
- `i` - Show transcriber status
- `q` - Quit the application
- `?` - Show available commands

This allows you to control the transcription directly from the terminal running the server without needing to use the client.

## Audio Configuration
- silence_threshold: Volume level below which audio is considered silence (0.0-1.0)
- silence_duration: Seconds of silence before recording stops
- start_tone: Audio feedback when recording starts
- completion_tone: Audio feedback when transcription completes successfully
- error_tone: Audio feedback when an error occurs

## Processing Configuration
- auto_paste: Automatically paste transcribed text (requires xdotool on Linux)
- channel_buffer_size: Buffer size for audio processing
- continuous_mode: Settings for continuous recording mode
  - enabled: Enable/disable continuous recording
  - max_session_duration: Maximum recording session time in seconds
  - inter_speech_timeout: Silence timeout between speech segments
  - auto_stop_on_idle: Automatically stop after extended idle time
- text_validation: Security settings for transcribed text
  - mode: Validation mode ("security_focused" or "strict")
  - allow_punctuation: Allow natural punctuation in transcriptions
  - custom_blocklist: Additional words/phrases to block

## Recording

### Single-Shot Mode (continuous_mode.enabled = false)
Recording will automatically stop when:
- Silence is detected for `silence_duration` seconds
- Manual stop command is sent

### Continuous Mode (continuous_mode.enabled = true)
Recording will continue through silence periods and only stop when:
- Maximum session duration is reached
- Manual stop command is sent
- Extended idle timeout (if auto_stop_on_idle is enabled)

**Benefits of Continuous Mode:**
- Natural speech patterns with pauses
- No need to repeatedly trigger recording
- Better for dictation and long-form content
- Automatic audio feedback when transcription segments complete

## Model Selection
| Model | Size | Use Case | Speed | Memory Usage |
|-------|------|----------|-------|--------------|
| tiny.en | 77.7MB | Quick tests, low resource environments | ⚡⚡⚡⚡⚡ | 🟢 Low |
| base | ~150MB | Basic transcription | ⚡⚡⚡⚡ | 🟢 Low |
| small | ~500MB | Balanced performance | ⚡⚡⚡ | 🟡 Medium |
| medium | ~1.5GB | Better accuracy | ⚡⚡ | 🟠 High |
| large-v3 | ~3GB | Best accuracy | ⚡ | 🔴 Very High |

## Troubleshooting

### Common Issues:

1. **No microphone input detected:**
   - Check your microphone settings in your OS
   - Ensure miniaudio/malgo dependencies are properly installed
   - Verify microphone permissions
   - Check device_index in config.json (-1 for default device)

2. **Compilation errors:**
   - Ensure OpenMP is installed
   - Check GCC/Clang installation
   - Run `./scripts/update_deps.sh` to ensure whisper.cpp and its headers are properly set up
   - Run `make clean && make deps && make build`

3. **Clipboard issues on Linux:**
   - Verify xclip or xsel is installed
   - Check xdotool installation for auto-paste feature
   - Verify X11 display is available (note: Wayland users may need to set `export DISPLAY=:0` in their shell)
   - Check clipboard permissions
4. **Audio feedback issues:**
   - Adjust start_tone settings in config.json
   - Verify audio output device is working
   - Check system volume levels

5. **Library not found errors:**
   - Use the provided wrapper scripts (`scripts/run-server.sh` and `scripts/run-client.sh`) which set the correct library paths
   - If running the binaries directly, set the LD_LIBRARY_PATH environment variable:
     ```bash
     LD_LIBRARY_PATH=/path/to/skald-go/deps/whisper.cpp/build/src:/path/to/skald-go/lib ./bin/skald-server
     ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

If you find this project helpful, please consider giving it a star ⭐️

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Running the Server

There are several ways to run the Skald-Go server:

### Using the wrapper script

The simplest way to run the server is to use the provided wrapper script:

```bash
./scripts/run-server.sh
```

This script automatically sets the correct `LD_LIBRARY_PATH` environment variable to find the required libraries.

### Using the client

To interact with the server, use the provided client wrapper script:

```bash
./scripts/run-client.sh start   # Start transcription
./scripts/run-client.sh stop    # Stop transcription
./scripts/run-client.sh status  # Check server status
```

### Using systemd (for Linux users)

For a more permanent solution, you can install the provided systemd service:

1. Copy the service file to your user's systemd directory:
   ```bash
   mkdir -p ~/.config/systemd/user/
   cp scripts/skald-server.service ~/.config/systemd/user/
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

## Keyboard Interactions

When running the server with `keyboard_enabled: true` in the config, you can use the following keyboard shortcuts:

- `r` - Start transcription (same as running `./scripts/run-client.sh start`)
- `s` - Stop transcription (same as running `./scripts/run-client.sh stop`)
- `i` - Show transcriber status
- `q` - Quit the application
- `?` - Show available commands
