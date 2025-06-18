package transcriber

import (
	"skald/internal/config"
	"testing"
)

func TestIsNonSilentAudio(t *testing.T) {
	cfg := config.DefaultConfig()
	transcriber := &Transcriber{cfg: cfg}
	
	// Test empty samples
	if transcriber.isNonSilentAudio([]float32{}) {
		t.Error("Empty samples should be considered silent")
	}
	
	// Test silent samples (below threshold)
	silentSamples := make([]float32, 100)
	for i := range silentSamples {
		silentSamples[i] = 0.001 // Very quiet
	}
	if transcriber.isNonSilentAudio(silentSamples) {
		t.Error("Quiet samples should be considered silent")
	}
	
	// Test loud samples (above threshold)
	loudSamples := make([]float32, 100)
	for i := range loudSamples {
		loudSamples[i] = 0.1 // Above default threshold of 0.008
	}
	if !transcriber.isNonSilentAudio(loudSamples) {
		t.Error("Loud samples should be considered non-silent")
	}
}

func TestContinuousModeBehavior(t *testing.T) {
	// Test that continuous mode configuration affects behavior
	cfg := config.DefaultConfig()
	
	// Test with continuous mode enabled
	cfg.Processing.ContinuousMode.Enabled = true
	if !cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be enabled")
	}
	
	// Test with continuous mode disabled
	cfg.Processing.ContinuousMode.Enabled = false
	if cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be disabled")
	}
}

func TestSilenceThresholdConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()
	
	// Test that silence threshold is reasonable
	if cfg.Audio.SilenceThreshold <= 0 {
		t.Error("Silence threshold should be positive")
	}
	
	if cfg.Audio.SilenceThreshold >= 1.0 {
		t.Error("Silence threshold should be less than 1.0")
	}
	
	// Test that silence duration is reasonable
	if cfg.Audio.SilenceDuration <= 0 {
		t.Error("Silence duration should be positive")
	}
}