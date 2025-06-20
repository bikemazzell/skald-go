package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check some default values
	if cfg.Audio.SampleRate != 16000 {
		t.Errorf("Expected default sample rate 16000, got %d", cfg.Audio.SampleRate)
	}

	if cfg.Audio.Channels != 1 {
		t.Errorf("Expected default channels 1, got %d", cfg.Audio.Channels)
	}

	if cfg.Whisper.Language != "en" {
		t.Errorf("Expected default language 'en', got %s", cfg.Whisper.Language)
	}

	// Check that models map is initialized
	if len(cfg.Whisper.Models) == 0 {
		t.Error("Expected models map to be initialized with default values")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	cfg := DefaultConfig()
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}

	// Test invalid sample rate
	invalidCfg := DefaultConfig()
	invalidCfg.Audio.SampleRate = 0
	err = invalidCfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid sample rate, got nil")
	}

	// Test invalid model
	invalidCfg = DefaultConfig()
	invalidCfg.Whisper.Model = "non-existent-model"
	err = invalidCfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid model, got nil")
	}
}

func TestSubStructValidation(t *testing.T) {
	// --- AudioConfig Validation ---
	ac := DefaultAudioConfig()
	if err := ac.Validate(); err != nil {
		t.Errorf("Expected valid AudioConfig, got error: %v", err)
	}

	ac.SampleRate = 0
	if err := ac.Validate(); err == nil {
		t.Error("Expected error for zero sample rate in AudioConfig")
	}

	// --- WhisperConfig Validation ---
	wc := DefaultWhisperConfig()
	if err := wc.Validate(16000); err != nil {
		t.Errorf("Expected valid WhisperConfig, got error: %v", err)
	}

	wc.Model = "foo"
	if err := wc.Validate(16000); err == nil {
		t.Error("Expected error for non-existent model in WhisperConfig")
	}

	wc = DefaultWhisperConfig()
	wc.Language = ""
	wc.AutoDetectLanguage = false
	if err := wc.Validate(16000); err == nil {
		t.Error("Expected error for empty language with auto-detect off")
	}

	// --- ProcessingConfig Validation ---
	pc := DefaultProcessingConfig()
	if err := pc.Validate(); err != nil {
		t.Errorf("Expected valid ProcessingConfig, got error: %v", err)
	}

	pc.ShutdownTimeout = 0
	if err := pc.Validate(); err == nil {
		t.Error("Expected error for zero shutdown timeout in ProcessingConfig")
	}
}

func TestLoadAndSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file path
	configPath := filepath.Join(tempDir, "test-config.json")

	// Get default config
	defaultCfg := DefaultConfig()

	// Save the config
	err = Save(configPath, defaultCfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the config
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check if loaded config matches default
	if loadedCfg.Version != defaultCfg.Version {
		t.Errorf("Expected version %s, got %s", defaultCfg.Version, loadedCfg.Version)
	}

	if loadedCfg.Audio.SampleRate != defaultCfg.Audio.SampleRate {
		t.Errorf("Expected sample rate %d, got %d",
			defaultCfg.Audio.SampleRate, loadedCfg.Audio.SampleRate)
	}

	// Test loading non-existent file (should create default)
	nonExistentPath := filepath.Join(tempDir, "non-existent.json")
	_, err = Load(nonExistentPath)
	if err != nil {
		t.Errorf("Expected Load to create default config, got error: %v", err)
	}

	// Check if file was created
	if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
		t.Error("Default config file was not created")
	}
}

func TestInvalidJSON(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an invalid JSON file
	invalidPath := filepath.Join(tempDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	// Try to load invalid JSON
	_, err = Load(invalidPath)
	if err == nil {
		t.Error("Expected error loading invalid JSON, got nil")
	}
}
