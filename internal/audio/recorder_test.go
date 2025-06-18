package audio

import (
	"skald/internal/config"
	"testing"
	"log"
	"os"
)

func TestRecorderToneMethods(t *testing.T) {
	// Create a test config
	cfg := config.DefaultConfig()
	cfg.Audio.StartTone.Enabled = false // Disable to avoid actual audio output during tests
	cfg.Audio.CompletionTone.Enabled = false
	cfg.Audio.ErrorTone.Enabled = false
	
	logger := log.New(os.Stderr, "test: ", log.LstdFlags)
	
	recorder, err := NewRecorder(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}
	defer recorder.Close()
	
	// Test PlayCompletionTone (should return nil when disabled)
	err = recorder.PlayCompletionTone()
	if err != nil {
		t.Errorf("PlayCompletionTone failed: %v", err)
	}
	
	// Test PlayErrorTone (should return nil when disabled)
	err = recorder.PlayErrorTone()
	if err != nil {
		t.Errorf("PlayErrorTone failed: %v", err)
	}
}

func TestToneConfigValidation(t *testing.T) {
	// Test that ToneConfig struct has expected fields
	toneConfig := ToneConfig{
		Enabled:   true,
		Frequency: 440,
		Duration:  200,
		FadeMs:    10,
	}
	
	if toneConfig.Enabled != true {
		t.Error("ToneConfig.Enabled field not working")
	}
	if toneConfig.Frequency != 440 {
		t.Error("ToneConfig.Frequency field not working")
	}
	if toneConfig.Duration != 200 {
		t.Error("ToneConfig.Duration field not working")
	}
	if toneConfig.FadeMs != 10 {
		t.Error("ToneConfig.FadeMs field not working")
	}
}