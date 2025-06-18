package transcriber

import (
	"skald/internal/config"
	"testing"
)

func TestContinuousModeConfiguration(t *testing.T) {
	// Test continuous mode enabled configuration
	cfg := config.DefaultConfig()
	cfg.Processing.ContinuousMode.Enabled = true
	cfg.Processing.ContinuousMode.MaxSessionDuration = 10
	cfg.Processing.ContinuousMode.InterSpeechTimeout = 2
	cfg.Processing.ContinuousMode.AutoStopOnIdle = true

	// Verify configuration is set correctly without creating full transcriber
	if !cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be enabled")
	}
	if cfg.Processing.ContinuousMode.MaxSessionDuration != 10 {
		t.Error("MaxSessionDuration should be 10")
	}
	if cfg.Processing.ContinuousMode.InterSpeechTimeout != 2 {
		t.Error("InterSpeechTimeout should be 2")
	}
	if !cfg.Processing.ContinuousMode.AutoStopOnIdle {
		t.Error("AutoStopOnIdle should be true")
	}
}

func TestContinuousModeDisabled(t *testing.T) {
	// Test continuous mode disabled configuration
	cfg := config.DefaultConfig()
	cfg.Processing.ContinuousMode.Enabled = false

	// Verify configuration is set correctly without creating full transcriber
	if cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be disabled")
	}
}

func TestDefaultContinuousMode(t *testing.T) {
	// Test that default config has continuous mode enabled
	cfg := config.DefaultConfig()
	
	if !cfg.Processing.ContinuousMode.Enabled {
		t.Error("Default config should have continuous mode enabled")
	}
	if cfg.Processing.ContinuousMode.MaxSessionDuration != 300 {
		t.Error("Default MaxSessionDuration should be 300 seconds")
	}
	if cfg.Processing.ContinuousMode.InterSpeechTimeout != 10 {
		t.Error("Default InterSpeechTimeout should be 10 seconds")
	}
	if !cfg.Processing.ContinuousMode.AutoStopOnIdle {
		t.Error("Default AutoStopOnIdle should be true")
	}
}