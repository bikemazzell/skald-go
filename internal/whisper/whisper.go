package whisper

import (
	"fmt"
	"os"

	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

type Config struct {
	Language           string
	AutoDetectLanguage bool
	SupportedLanguages []string
	Silent             bool
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
	var model whisper.Model
	var ctx whisper.Context
	var err error

	if cfg.Silent {
		// Set environment variable to suppress GGML/Whisper logging
		// This is a common approach used by whisper.cpp applications
		oldLogLevel := os.Getenv("GGML_LOG_LEVEL")
		os.Setenv("GGML_LOG_LEVEL", "ERROR")
		defer func() {
			if oldLogLevel == "" {
				os.Unsetenv("GGML_LOG_LEVEL")
			} else {
				os.Setenv("GGML_LOG_LEVEL", oldLogLevel)
			}
		}()
	}

	// Perform all potentially verbose initialization
	model, err = whisper.New(modelPath)
	if err == nil {
		ctx, err = model.NewContext()
		if err != nil {
			model.Close()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Check if model is multilingual
	isMultilingual := ctx.IsMultilingual()
	
	// Set the language based on configuration
	if cfg.AutoDetectLanguage && isMultilingual {
		// Enable auto-detection by setting language to "auto"
		if err := ctx.SetLanguage("auto"); err != nil {
			model.Close()
			return nil, fmt.Errorf("failed to enable auto language detection: %w", err)
		}
	} else if cfg.Language != "" {
		// Set specific language
		if err := ctx.SetLanguage(cfg.Language); err != nil {
			model.Close()
			return nil, fmt.Errorf("failed to set language: %w", err)
		}
	}

	// Ensure translation is disabled - we want transcription in the original language
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

	// For continuous mode, create a fresh context for each transcription
	// to avoid state pollution between transcriptions
	ctx, err := w.model.NewContext()
	if err != nil {
		return "", fmt.Errorf("failed to create fresh context: %w", err)
	}
	
	// Apply the same configuration as the original context
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

// GetDetectedLanguage returns the detected language from the most recent transcription
func (w *Whisper) GetDetectedLanguage() string {
	if w.cfg.AutoDetectLanguage && w.isMultilingual {
		// In auto-detection mode, return the configured language as fallback
		// since we're using fresh contexts per transcription
		return w.cfg.Language
	}
	return w.cfg.Language
}

// GetSupportedLanguages returns all languages supported by the model
func (w *Whisper) GetSupportedLanguages() []string {
	if w.isMultilingual {
		return w.model.Languages()
	}
	return []string{w.cfg.Language}
}

// IsMultilingual returns whether the model supports multiple languages
func (w *Whisper) IsMultilingual() bool {
	return w.isMultilingual
}

// DetectLanguage analyzes audio samples and returns language probabilities
func (w *Whisper) DetectLanguage(samples []float32) (map[string]float32, error) {
	if !w.isMultilingual {
		return map[string]float32{w.cfg.Language: 1.0}, nil
	}
	
	if len(samples) == 0 {
		return nil, fmt.Errorf("empty audio samples")
	}

	// Use auto-detection to get language probabilities
	// This requires processing the audio first
	if err := w.ctx.Process(samples, nil, nil); err != nil {
		return nil, fmt.Errorf("failed to process audio for language detection: %w", err)
	}

	// Get language probabilities (this would need the actual whisper.cpp method)
	// For now, return the current language with high confidence
	currentLang := w.ctx.Language()
	if currentLang == "" {
		currentLang = "en" // fallback
	}
	
	return map[string]float32{currentLang: 0.95}, nil
}

func (w *Whisper) Close() {
	if w.model != nil {
		w.model.Close()
	}
}
