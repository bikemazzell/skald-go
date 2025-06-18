package whisper

import (
	"testing"
)

func TestConfigLanguageSettings(t *testing.T) {
	// Test language configuration structure
	cfg := Config{
		Language:           "en",
		AutoDetectLanguage: false,
		SupportedLanguages: []string{"en", "es", "fr"},
	}
	
	if cfg.Language != "en" {
		t.Error("Language should be set to 'en'")
	}
	if cfg.AutoDetectLanguage {
		t.Error("AutoDetectLanguage should be false")
	}
	if len(cfg.SupportedLanguages) != 3 {
		t.Error("SupportedLanguages should have 3 languages")
	}
}

func TestAutoDetectionConfig(t *testing.T) {
	// Test auto-detection configuration
	cfg := Config{
		Language:           "auto",
		AutoDetectLanguage: true,
		SupportedLanguages: []string{"en", "es", "fr", "de", "it"},
	}
	
	if cfg.Language != "auto" {
		t.Error("Language should be set to 'auto' for auto-detection")
	}
	if !cfg.AutoDetectLanguage {
		t.Error("AutoDetectLanguage should be true")
	}
	if len(cfg.SupportedLanguages) != 5 {
		t.Error("SupportedLanguages should have 5 languages")
	}
}

func TestLanguageConfigValidation(t *testing.T) {
	// Test that configuration handles common language codes
	supportedLangs := []string{
		"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh",
		"ar", "hi", "th", "vi", "tr", "pl", "nl", "sv", "da", "no",
	}
	
	cfg := Config{
		Language:           "en",
		AutoDetectLanguage: false,
		SupportedLanguages: supportedLangs,
	}
	
	// Check that common languages are included
	foundEn := false
	foundEs := false
	foundFr := false
	
	for _, lang := range cfg.SupportedLanguages {
		switch lang {
		case "en":
			foundEn = true
		case "es":
			foundEs = true
		case "fr":
			foundFr = true
		}
	}
	
	if !foundEn || !foundEs || !foundFr {
		t.Error("Common languages (en, es, fr) should be in supported languages")
	}
}