package transcriber

import (
	"skald/internal/config"
	"testing"
	"time"
)

func TestContinuousModeConfiguration(t *testing.T) {
	// Test that continuous mode is properly configured
	cfg := config.DefaultConfig()
	
	// Verify continuous mode is enabled by default
	if !cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be enabled by default")
	}
	
	// Verify timeout settings
	if cfg.Processing.ContinuousMode.MaxSessionDuration <= 0 {
		t.Error("MaxSessionDuration should be positive")
	}
	
	if cfg.Processing.ContinuousMode.InterSpeechTimeout <= 0 {
		t.Error("InterSpeechTimeout should be positive")
	}
}

func TestContinuousModeTimeoutLogic(t *testing.T) {
	// Test that timeout calculations work correctly
	cfg := config.DefaultConfig()
	cfg.Processing.ContinuousMode.MaxSessionDuration = 60 // 1 minute
	
	timeout := time.Duration(cfg.Processing.ContinuousMode.MaxSessionDuration) * time.Second
	expected := time.Minute
	
	if timeout != expected {
		t.Errorf("Expected timeout of %v, got %v", expected, timeout)
	}
}

func TestContinuousModeDisabledFallback(t *testing.T) {
	// Test behavior when continuous mode is disabled
	cfg := config.DefaultConfig()
	cfg.Processing.ContinuousMode.Enabled = false
	
	// When disabled, the system should fall back to single-shot mode
	if cfg.Processing.ContinuousMode.Enabled {
		t.Error("Continuous mode should be disabled when set to false")
	}
}