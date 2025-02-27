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
- 🛑 Silence detection for automatic stopping
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
./bin/skald-server
```
In another terminal, control recording
```bash
./bin/skald-client start # Begin recording
./bin/skald-client stop # Stop recording
./bin/skald-client status # Check status
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

2. Update dependencies:
```bash
./update_deps.sh
```
This script will:
- Clone or update whisper.cpp to the latest stable version
- Copy the Go bindings from whisper.cpp
- Copy necessary header files to the appropriate locations
- Update Go module dependencies
- Update the vendor directory

3. Build the project:
```bash
make build
```
This will compile both server and client binaries.

## Dependency Management

Skald-Go uses a custom dependency management approach for whisper.cpp:

- Dependencies are stored in the `deps/` directory (excluded from git)
- `update_deps.sh` script handles downloading and updating dependencies
- You can specify a specific whisper.cpp version: `./update_deps.sh v1.7.4`
- The script ensures all necessary header files are properly copied
- Go modules are configured to use the local dependencies

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
}
},
"processing": {
"shutdown_timeout": 30,
"event_wait_timeout": 0.1,
"auto_paste": true,
"channel_buffer_size": 10
},
"whisper": {
"model": "base",
"language": "en",
"beam_size": 5
},
"server": {
"socket_path": "/tmp/skald.sock",
"socket_timeout": 5.0
},
"debug": {
"print_status": true,
"print_transcriptions": true
}
}
```

## Audio Configuration
- silence_threshold: Volume level below which audio is considered silence (0.0-1.0)
- silence_duration: Seconds of silence before recording stops
- start_tone: Configurable audio feedback when recording starts

## Processing Configuration
- auto_paste: Automatically paste transcribed text (requires xdotool on Linux)
- channel_buffer_size: Buffer size for audio processing

## Recording
Recording will automatically stop when:
- Silence is detected for `silence_duration` seconds
- Manual stop command is sent

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
   - Run `./update_deps.sh` to ensure whisper.cpp and its headers are properly set up
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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

If you find this project helpful, please consider giving it a star ⭐️

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.