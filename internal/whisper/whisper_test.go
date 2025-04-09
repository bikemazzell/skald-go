package whisper

import (
	"os"
	"testing"
)

func TestTranscribe(t *testing.T) {
	tests := []struct {
		name        string
		samples     []float32
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty samples",
			samples:     []float32{},
			wantErr:     true,
			errContains: "empty audio samples",
		},
		{
			name:    "valid samples",
			samples: make([]float32, 16000*2), // 2 seconds of audio at 16kHz to be safe
			wantErr: false,
		},
	}

	modelPath := "../../models/ggml-base.bin"

	// Check if model file exists, skip test if not available
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skip("Skipping test as model file is not available")
		return
	}

	w, err := New(modelPath, Config{})
	if err != nil {
		t.Fatalf("failed to create whisper instance: %v", err)
	}
	defer w.Close()

	// Fill the valid samples with some non-zero values to simulate audio
	for i := range tests[1].samples {
		tests[1].samples[i] = 0.1 // Add a simple constant tone
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := w.Transcribe(tt.samples)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr && result == "" {
				t.Error("expected non-empty result for valid samples")
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}
