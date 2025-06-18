package whisper

import (
	"fmt"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

type Config struct {
	Language           string
	AutoDetectLanguage bool
	SupportedLanguages []string
}

type Whisper struct {
	model           whisper.Model
	ctx             whisper.Context
	cfg             Config
	firstCall       bool
	detectedLang    string
	isMultilingual  bool
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

	isMultilingual := ctx.IsMultilingual()
	
	if cfg.AutoDetectLanguage && isMultilingual {
		if err := ctx.SetLanguage("auto"); err != nil {
			model.Close()
			return nil, fmt.Errorf("failed to enable auto language detection: %w", err)
		}
	} else if cfg.Language != "" {
		if err := ctx.SetLanguage(cfg.Language); err != nil {
			model.Close()
			return nil, fmt.Errorf("failed to set language: %w", err)
		}
	}

	ctx.SetTranslate(false)

	return &Whisper{
		model:          model,
		ctx:            ctx,
		cfg:            cfg,
		firstCall:      true,
		detectedLang:   "",
		isMultilingual: isMultilingual,
	}, nil
}

func (w *Whisper) Transcribe(samples []float32) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("empty audio samples")
	}

	ctx, err := w.model.NewContext()
	if err != nil {
		return "", fmt.Errorf("failed to create fresh context: %w", err)
	}
	
	if w.cfg.AutoDetectLanguage && w.isMultilingual {
		if err := ctx.SetLanguage("auto"); err != nil {
			return "", fmt.Errorf("failed to set auto language detection: %w", err)
		}
	} else if w.cfg.Language != "" {
		if err := ctx.SetLanguage(w.cfg.Language); err != nil {
			return "", fmt.Errorf("failed to set language: %w", err)
		}
	}
	ctx.SetTranslate(false)

	if err := ctx.Process(samples, nil, nil); err != nil {
		return "", fmt.Errorf("failed to process audio: %w", err)
	}

	var result string
	for {
		segment, err := ctx.NextSegment()
		if err != nil {
			break
		}
		result += segment.Text + " "
	}

	return result, nil
}

func (w *Whisper) GetDetectedLanguage() string {
	if w.cfg.AutoDetectLanguage && w.isMultilingual {
		return w.cfg.Language
	}
	return w.cfg.Language
}

func (w *Whisper) GetSupportedLanguages() []string {
	if w.isMultilingual {
		return w.model.Languages()
	}
	return []string{w.cfg.Language}
}

func (w *Whisper) IsMultilingual() bool {
	return w.isMultilingual
}

func (w *Whisper) DetectLanguage(samples []float32) (map[string]float32, error) {
	if !w.isMultilingual {
		return map[string]float32{w.cfg.Language: 1.0}, nil
	}
	
	if len(samples) == 0 {
		return nil, fmt.Errorf("empty audio samples")
	}

	if err := w.ctx.Process(samples, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio for language detection: %w", err)
	}

	currentLang := w.ctx.Language()
	if currentLang == "" {
		currentLang = "en"
	}
	
	return map[string]float32{currentLang: 0.95}, nil
}

func (w *Whisper) Close() {
	if w.model != nil {
		w.model.Close()
	}
}
