# Skald-Go - Voice to Text Transcriber
> A Go implementation of Skald using whisper.cpp
> Created by @shoewind1997

Skald-Go is a lightweight speech-to-text tool that converts your voice to text in real-time using whisper.cpp. It runs quietly in the background without any graphical interface, automatically copying transcriptions to your clipboard. The application consists of two parts: a background server that handles the transcription, and a client that can be bound to a hotkey for easy start/stop control. Named after the ancient Nordic poets and storytellers known as skalds, this tool makes it effortless to transform your spoken words into written text with a single keystroke.

## Features

- 🎤 Real-time microphone input capture using PvRecorder
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
- whisper.cpp (included as submodule)

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

1. Clone the repository with submodules:
```bash
git clone --recursive https://github.com/bikemazzell/skald-go.git
cd skald-go
```

2. Build the project:
```bash
make build
```
This will:
- Build whisper.cpp library
- Build Go bindings
- Compile both server and client binaries

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
"model": "large-v3-turbo-q8_0",
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

## Usage

1. Start the server:
```bash
./bin/skald-server
```
2. Use the client to control recording:
```bash
./bin/skald-client start # Begin recording
./bin/skald-client stop # Stop recording
./bin/skald-client status # Check status
```

Recording will automatically stop when:
- Silence is detected for `silence_duration` seconds
- Manual stop command is sent

## Model Selection
```
| Model | Size | Use Case | Speed | Memory Usage |
|-------|------|----------|-------|--------------|
| tiny.en | 77.7MB | Quick tests, low resource environments | ⚡⚡⚡⚡⚡ | 🟢 Low |
| base | ~150MB | Basic transcription | ⚡⚡⚡⚡ | 🟢 Low |
| small | ~500MB | Balanced performance | ⚡⚡⚡ | 🟡 Medium |
| medium | ~1.5GB | Better accuracy | ⚡⚡ | 🟠 High |
| large-v3 | ~3GB | Best accuracy | ⚡ | 🔴 Very High |
```
## Troubleshooting

### Common Issues:

1. **No microphone input detected:**
   - Check your microphone settings in your OS
   - Ensure PvRecorder is properly installed
   - Verify microphone permissions

2. **Compilation errors:**
   - Ensure OpenMP is installed
   - Check GCC/Clang installation
   - Verify whisper.cpp submodule is properly initialized

3. **Clipboard issues on Linux:**
   - Verify xclip or xsel is installed
   - Check xdotool installation for auto-paste feature
   - Verify X11 display is available

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

If you find this project helpful, please consider giving it a star ⭐️

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.