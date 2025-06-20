# Skald-Go Release Package

This is a self-contained build of Skald-Go with all dependencies included.

## Quick Start

1. **Start the server:**
   ```bash
   ./skald server
   ```

2. **In another terminal, start transcription:**
   ```bash
   ./skald start
   ```

3. **Stop transcription:**
   ```bash
   ./skald stop
   ```

## Installation

For system-wide installation:

```bash
# Copy to user binaries
mkdir -p ~/.local/bin/skald-go
cp -r * ~/.local/bin/skald-go/

# Add to PATH (add to ~/.bashrc for persistence)
export PATH="$HOME/.local/bin/skald-go:$PATH"

# Install as systemd service
systemctl --user daemon-reload
systemctl --user enable ~/.local/bin/skald-go/skald.service
systemctl --user start skald.service
```

## Model Download

Download whisper models from HuggingFace and place them in this directory:
- https://huggingface.co/ggerganov/whisper.cpp/tree/main

Recommended: `ggml-base.bin` (good balance of speed/quality)

## Configuration

Edit `config.json` to customize audio settings, hotkeys, and behavior.
