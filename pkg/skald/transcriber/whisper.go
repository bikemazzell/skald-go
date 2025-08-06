package transcriber

import (
	"fmt"
	"strings"
)

// Whisper implements transcription using whisper.cpp
type Whisper struct {
	model    WhisperModel
	language string
}

// NewWhisper creates a new whisper transcriber
func NewWhisper(modelPath, language string) (*Whisper, error) {
	model, err := whisperFactory.NewModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	return &Whisper{
		model:    model,
		language: language,
	}, nil
}

// SetModelFactory allows injection of a different model factory for testing
func SetModelFactory(factory WhisperModelFactory) {
	whisperFactory = factory
}

// Transcribe converts audio to text
func (w *Whisper) Transcribe(audio []float32) (string, error) {
	if len(audio) == 0 {
		return "", nil
	}

	context, err := w.model.NewContext()
	if err != nil {
		return "", fmt.Errorf("failed to create context: %w", err)
	}

	// Set language if specified
	if w.language != "" && w.language != "auto" {
		if err := context.SetLanguage(w.language); err != nil {
			return "", fmt.Errorf("failed to set language: %w", err)
		}
	}

	// Process audio
	if err := context.Process(audio, nil, nil); err != nil {
		return "", fmt.Errorf("failed to process audio: %w", err)
	}

	// Get text from all segments
	var text strings.Builder
	for {
		segment, err := context.NextSegment()
		if err != nil {
			break
		}
		text.WriteString(segment.GetText())
	}

	return strings.TrimSpace(text.String()), nil
}

// Close releases resources
func (w *Whisper) Close() error {
	if w.model != nil {
		return w.model.Close()
	}
	return nil
}