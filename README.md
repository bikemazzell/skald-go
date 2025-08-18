# Skald - Simple Audio Transcription Tool

A lightweight command-line tool for real-time audio transcription using OpenAI's Whisper model.

## Philosophy

Following Unix philosophy: **Do one thing well**. Skald transcribes live audio into text from the command line.
Initially, this program had more configurations, parameters, and complexity. But all it really needs to do is work when invoked from CLI. And that is what it does now: run it and transcribe audio.

## Features

- Real-time audio capture and transcription
- Automatic silence detection
- Clipboard integration
- Continuous transcription mode (processed at 30s chunks - optimal for Whisper.cpp)
- Multiple language support

## Installation

### Quick Start

```bash
# Clone the repository
git clone https://github.com/bikemazzell/skald-go.git
cd skald-go

# Build from source (automatically downloads and builds dependencies)
make build

# Or build optimized release version
make release

# Install system-wide
sudo make install

# Or run directly
make run
```

The `make build` command will automatically:
1. Clone whisper.cpp if not present
2. Set up Go bindings
3. Build static libraries
4. Compile the skald binary

### System dependencies (Debian/Ubuntu)

Install required runtime libraries once on the target machine:

```bash
sudo apt-get update
sudo apt-get install -y libstdc++6 libgomp1 libasound2 xclip
```

- libstdc++6 and libgomp1 are needed for C++ and OpenMP runtime
- libasound2 is needed for ALSA audio input
- xclip is optional (clipboard output); the app still runs without it
- On very minimal systems you may also need: `sudo apt-get install -y libgcc-s1`

## Usage

### Basic Usage

```bash
# Start transcribing (press Ctrl+C to stop)
skald

# Transcribe with a specific model
skald -model models/ggml-large-v3-turbo.bin

# Specify language (auto-detect by default)
skald -language en

# Continuous mode (keeps transcribing after each pause)
skald -continuous
```

### Options

- `-model`: Path to Whisper model file (default: "models/ggml-large-v3-turbo.bin")
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

## Building from Source

### Prerequisites

- Go 1.21 or later
- CMake (for building whisper.cpp)
- C++ compiler (g++ or clang++)
- Git

### Build Steps

```bash
# Initialize dependencies and build
make deps    # Downloads whisper.cpp and builds static libraries
make build   # Builds the skald binary

# Or simply run (deps is automatic)
make build

# Build optimized release version
make release

# Download a Whisper model (required for transcription)
make download-model  # Downloads large-v3-turbo model
# Or for a smaller, faster model:
make download-tiny-model
```

### Troubleshooting

If you encounter build errors:

```bash
# Clean everything and rebuild
make clean
make deps
make build

## Portable builds (glibc 2.17 via Zig)

To produce a binary that runs on older Linux distributions (e.g., glibc 2.17+), build with Zig targeting an older glibc:

```bash
# Install zig (see https://ziglang.org/)
# Then build a portable release
make release-glibc217

# Inspect required GLIBC versions
make abi-check
```

This uses zig cc targeting x86_64-linux-gnu.2.17 for both whisper.cpp and the cgo link, and statically links libstdc++/libgcc to reduce runtime dependencies.

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