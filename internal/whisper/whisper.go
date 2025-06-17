package whisper

import (
	"fmt"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

type Config struct {
	Language string
}

type Whisper struct {
	model whisper.Model
	ctx   whisper.Context
	cfg   Config
}

func New(modelPath string, cfg Config) (*Whisper, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	ctx, err := model.NewContext()
	if err != nil {
		model.Close()
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	// Set the language if specified
	if cfg.Language != "" {
		if err := ctx.SetLanguage(cfg.Language); err != nil {
			model.Close()
			return nil, fmt.Errorf("failed to set language: %w", err)
		}
	}

	// Ensure translation is disabled - we want transcription in the original language
	ctx.SetTranslate(false)

	return &Whisper{
		model: model,
		ctx:   ctx,
		cfg:   cfg,
	}, nil
}

func (w *Whisper) Transcribe(samples []float32) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("empty audio samples")
	}

	if err := w.ctx.Process(samples, nil, nil); err != nil {
		return "", fmt.Errorf("failed to process audio: %w", err)
	}

	var result string
	for {
		segment, err := w.ctx.NextSegment()
		if err != nil {
			break
		}
		result += segment.Text + " "
	}

	return result, nil
}

func (w *Whisper) Close() {
	if w.model != nil {
		w.model.Close()
	}
}
