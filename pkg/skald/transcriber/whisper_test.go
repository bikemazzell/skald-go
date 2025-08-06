package transcriber

import (
	"testing"
)

func TestWhisper_Transcribe_EmptyAudio(t *testing.T) {
	// This test doesn't require a real model
	w := &Whisper{
		model:    nil,
		language: "en",
	}

	result, err := w.Transcribe([]float32{})
	if err != nil {
		t.Errorf("Expected no error for empty audio, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty result for empty audio, got: %q", result)
	}
}

func TestWhisper_NewWhisper_InvalidModel(t *testing.T) {
	// Test loading non-existent model
	_, err := NewWhisper("/non/existent/model.bin", "en")
	if err == nil {
		t.Error("Expected error for non-existent model, got nil")
	}
}

// Integration test - only run if model is available
func TestWhisper_Integration(t *testing.T) {
	modelPath := "../../../../models/ggml-base.bin"
	
	// Skip if model doesn't exist
	if _, err := NewWhisper(modelPath, "en"); err != nil {
		t.Skip("Model not available, skipping integration test")
	}

	tests := []struct {
		name     string
		language string
		audio    []float32
	}{
		{
			name:     "english language",
			language: "en",
			audio:    generateTestAudio(16000), // 1 second of test audio
		},
		{
			name:     "auto language detection",
			language: "auto",
			audio:    generateTestAudio(16000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewWhisper(modelPath, tt.language)
			if err != nil {
				t.Fatalf("Failed to create whisper: %v", err)
			}
			defer w.Close()

			// Just test that transcription doesn't panic/error
			_, err = w.Transcribe(tt.audio)
			if err != nil {
				t.Errorf("Transcribe() error = %v", err)
			}
		})
	}
}

func generateTestAudio(samples int) []float32 {
	// Generate silence for testing
	return make([]float32, samples)
}

// Benchmark transcription with different audio lengths
func BenchmarkWhisper_Transcribe(b *testing.B) {
	modelPath := "../../../../models/ggml-base.bin"
	
	w, err := NewWhisper(modelPath, "en")
	if err != nil {
		b.Skip("Model not available, skipping benchmark")
	}
	defer w.Close()

	audioLengths := []int{
		16000,     // 1 second
		16000 * 5, // 5 seconds
		16000 * 10, // 10 seconds
	}

	for _, length := range audioLengths {
		audio := generateTestAudio(length)
		b.Run(b.Name(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w.Transcribe(audio)
			}
		})
	}
}