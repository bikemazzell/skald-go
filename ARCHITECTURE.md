# Skald Architecture

## Overview

Skald is a lightweight command-line audio transcription tool built with Go that implements real-time speech-to-text using OpenAI's Whisper model. The architecture follows Unix philosophy principles: do one thing well, with a simple pipeline design and minimal dependencies.

## System Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────────┐     ┌──────────────┐
│  Microphone │────▶│ Audio Capture│────▶│ Audio Buffer   │────▶│   Silence    │
│   (ALSA)    │     │   (malgo)    │     │  ([]float32)   │     │  Detection   │
└─────────────┘     └──────────────┘     └────────────────┘     └──────────────┘
                                                   │                      │
                                                   │                      │
                                                   ▼                      ▼
┌─────────────┐     ┌──────────────┐     ┌────────────────┐     ┌──────────────┐
│  Clipboard  │◀────│    Output    │◀────│   Whisper      │◀────│   Trigger    │
│   (xclip)   │     │   Handler    │     │  Transcriber   │     │  Processing  │
└─────────────┘     └──────────────┘     └────────────────┘     └──────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │    stdout     │
                    └──────────────┘
```

## Core Components

### 1. Application Layer (`cmd/skald/`)

**Purpose**: Entry point and CLI interface

- **main.go**: Command-line argument parsing, component initialization, and signal handling
- Manages application lifecycle and graceful shutdown
- Version management through build-time injection

**Key Responsibilities**:
- Parse command-line flags
- Validate model file existence
- Initialize all components with proper configuration
- Handle OS signals (SIGINT, SIGTERM)
- Coordinate graceful shutdown

### 2. Core Package (`pkg/skald/`)

#### 2.1 Interfaces (`interfaces.go`)

Defines core abstractions for dependency injection and testability:

- **AudioCapture**: Audio input abstraction
  ```go
  Start(ctx context.Context) (<-chan []float32, error)
  Stop() error
  ```

- **Transcriber**: Speech-to-text abstraction
  ```go
  Transcribe(audio []float32) (string, error)
  Close() error
  ```

- **Output**: Text output abstraction
  ```go
  Write(text string) error
  ```

- **SilenceDetector**: Audio silence detection abstraction
  ```go
  IsSilent(samples []float32, threshold float32) bool
  ```

#### 2.2 Application Logic (`app/`)

**app.go**: Main application orchestration

- Manages transcription sessions
- Coordinates audio capture, silence detection, and transcription
- Implements continuous mode for ongoing transcription
- Buffer management for audio samples

**TranscriptionSession**: Session state management
- Audio buffer accumulation
- Silence tracking
- Threshold-based processing triggers

#### 2.3 Audio Module (`audio/`)

**capture.go**: Audio input implementation using malgo
- Real-time audio capture from system microphone
- ALSA backend on Linux
- Configurable sample rate (default: 16kHz)
- Non-blocking channel-based audio streaming
- Frame dropping when buffer is full

**silence.go**: Silence detection implementation
- RMS (Root Mean Square) based silence detection
- Configurable threshold (default: 0.01)
- Efficient sample processing

#### 2.4 Transcriber Module (`transcriber/`)

**whisper.go**: Whisper model integration
- Uses whisper.cpp Go bindings
- Language auto-detection or manual specification
- Context-based processing for efficient memory usage
- Segment-based text extraction

#### 2.5 Output Module (`output/`)

**clipboard.go**: Output handling
- Dual output to stdout and clipboard
- Linux clipboard integration via xclip
- Graceful fallback if clipboard unavailable
- Non-fatal clipboard errors

## Data Flow

1. **Audio Capture**: 
   - Microphone → malgo → float32 samples
   - Sample rate: 16kHz mono
   - Channel buffer: 100 frames

2. **Buffering & Detection**:
   - Accumulate audio samples in session buffer
   - Calculate RMS for silence detection
   - Track consecutive silent samples

3. **Transcription Trigger**:
   - Triggered when silence duration exceeds threshold (default: 1.5s)
   - Full buffer sent to Whisper model
   - Session reset for next utterance

4. **Text Output**:
   - Transcribed text to stdout
   - Optional clipboard copy via xclip
   - Empty transcriptions filtered out

## Build System

### Dependencies

**External Libraries**:
- **libwhisper.so**: Core Whisper C++ library
- **libggml*.so**: GGML tensor library (Whisper backend)
- Located in `lib/` directory with RPATH embedding

**Go Dependencies**:
- **malgo**: Cross-platform audio capture
- **whisper.cpp/bindings/go**: Go bindings for Whisper
- Vendored dependencies in `vendor/`

### Build Process

1. **CGO Configuration**:
   - CGO_ENABLED=1 for C++ library linking
   - RPATH set to `$ORIGIN/../lib` for portable binaries
   - Static linking of Go code, dynamic linking of Whisper

2. **Version Management**:
   - Version stored in `VERSION` file
   - Injected at build time via `-ldflags`
   - Accessible via `--version` flag

3. **Makefile Targets**:
   - `build`: Standard build with embedded RPATH
   - `install`: System-wide installation with ldconfig update
   - `release`: Optimized build with version tagging
   - `test`: Unit test execution
   - `test-coverage`: Coverage report generation

## Configuration

### Runtime Configuration

Command-line flags configure behavior:
- Model selection (`-model`)
- Language specification (`-language`)
- Continuous mode (`-continuous`)
- Audio parameters (`-sample-rate`, `-silence-threshold`, `-silence-duration`)
- Output control (`-no-clipboard`)

### Default Values

```go
defaultSampleRate       = 16000
defaultSilenceThreshold = 0.01
defaultSilenceDuration  = 1.5
defaultModelPath        = "models/ggml-large-v3-turbo.bin"
```

## Testing Strategy

### Unit Tests

Each module has corresponding test files:
- `app/app_test.go`: Application logic testing
- `audio/capture_test.go`: Audio capture mocking
- `audio/silence_test.go`: Silence detection validation
- `output/clipboard_test.go`: Output handling tests
- `transcriber/whisper_test.go`: Transcriber interface testing

### Mock Support

`mocks/mocks.go`: Test doubles for all interfaces
- Enables isolated unit testing
- Supports behavior verification
- Facilitates error scenario testing

## Performance Considerations

1. **Audio Processing**:
   - Real-time processing with minimal latency
   - Non-blocking audio capture
   - Frame dropping prevents memory buildup

2. **Memory Management**:
   - Bounded channel buffers
   - Session-scoped audio buffers
   - Whisper context reuse per transcription

3. **CPU Usage**:
   - Whisper model runs on CPU
   - RMS calculation optimized for efficiency
   - Silence detection prevents unnecessary transcription

## Security & Permissions

- **Audio Access**: Requires microphone permissions
- **Clipboard**: Optional xclip dependency
- **File System**: Read-only model file access
- **No Network**: Fully offline operation

## Platform Support

### Linux (Primary)
- ALSA audio backend
- xclip for clipboard
- Native library loading via LD_LIBRARY_PATH

### Potential Portability
- malgo supports Windows/macOS
- Clipboard abstraction ready for cross-platform
- Whisper.cpp is cross-platform

## Future Architecture Considerations

### Scalability
- Current design is single-process, single-user
- Could add queue-based processing for batch operations
- Server mode possible with minimal changes

### Extensibility
- Interface-based design allows easy component swapping
- New transcribers can implement Transcriber interface
- Output plugins possible via Output interface

### Optimization Opportunities
- GPU acceleration for Whisper (requires CUDA/Metal builds)
- VAD (Voice Activity Detection) preprocessing
- Streaming transcription for lower latency