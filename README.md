# Skald - Simple Audio Transcription Tool

A lightweight command-line tool for real-time audio transcription using OpenAI's Whisper model.

## Philosophy

Following Unix philosophy: **Do one thing well**. Skald transcribes live audio into text from the command line.

## Features

- Real-time audio capture and transcription
- Automatic silence detection
- Clipboard integration
- Continuous transcription mode
- Multiple language support

## Installation

```bash
# Build from source
make build

# Install system-wide
sudo make install

# Or run directly
make run
```

## Usage

### Basic Usage

```bash
# Start transcribing (press Ctrl+C to stop)
./bin/skald

# Transcribe with a specific model
./bin/skald -model models/ggml-large-v3-turbo-q8_0.bin

# Specify language (auto-detect by default)
./bin/skald -language en

# Continuous mode (keeps transcribing after each pause)
./bin/skald -continuous

# Check version
./bin/skald --version
```

### Options

- `-model`: Path to Whisper model file (default: "models/ggml-large-v3-turbo-q8_0.bin")
- `-language`: Language code (e.g., en, es, fr) or "auto" for auto-detection
- `-continuous`: Enable continuous transcription mode
- `-sample-rate`: Audio sample rate (default: 16000)
- `-silence-threshold`: Silence detection threshold (default: 0.01)
- `-silence-duration`: Silence duration in seconds (default: 1.5)
- `-no-clipboard`: Disable clipboard output
- `-version`: Show version and exit

## How It Works

1. **Audio Capture**: Records audio from your microphone
2. **Silence Detection**: Detects when you stop speaking
3. **Transcription**: Converts speech to text using Whisper
4. **Output**: Prints text to stdout and copies to clipboard

## Architecture

```
Audio Input → Buffer → Silence Detection → Whisper → Text Output
```

Simple pipeline with minimal components:
- Single process, no client-server complexity
- Direct audio processing without intermediate files
- Interface-based design for easy testing and modification

## Requirements

- Go 1.21 or later
- Whisper model file
- Linux with ALSA support
- xclip (for clipboard support)

## Building

```bash
# Download dependencies
go mod download

# Build binary
make build

# Build release version
make release
```

### Models

The default model is `ggml-large-v3-turbo-q8_0.bin` (included). For different speed/accuracy trade-offs:

```bash
# Download smaller/faster models
make download-tiny-model     # ~39MB, fastest
make download-base-model     # ~142MB, good balance

# Use different models
./bin/skald -model models/ggml-tiny.bin      # Fastest
./bin/skald -model models/ggml-base.bin      # Good balance  
./bin/skald                                  # Default: large-turbo (best quality)
```

## Version Management

```bash
# Check current version
make version

# Show binary version
skald --version

# Create git tag for release
make tag
```

Version is managed through the `VERSION` file and automatically embedded into the binary at build time.

## License

MIT License - See LICENSE file for details