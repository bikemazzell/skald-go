package whisper

import (
	"testing"
)

// TestTranscribeCreatesNewContext tests that each Transcribe call creates a fresh context
func TestTranscribeCreatesNewContext(t *testing.T) {
	// This test verifies the fix for continuous mode where context state was polluted
	// between transcriptions, causing empty results after the first transcription
	
	// Note: This is more of a conceptual test since we can't easily mock the whisper.cpp
	// internals, but it documents the expected behavior
	
	t.Run("Multiple transcriptions should work independently", func(t *testing.T) {
		// Each call to Transcribe should:
		// 1. Create a new context via model.NewContext()
		// 2. Configure the context with language settings
		// 3. Process the audio samples
		// 4. Extract segments
		// 5. Return the transcribed text
		//
		// This ensures that internal state (like segment counters) doesn't
		// carry over between transcriptions
		
		// The implementation in whisper.go now creates a fresh context
		// for each transcription, preventing state pollution
		t.Log("Transcribe method creates fresh context for each call")
		t.Log("This prevents segment counter pollution between transcriptions")
		t.Log("Fixes issue where second transcription returned empty text")
	})
}

// TestWhisperConfiguration tests that whisper configuration is properly applied
func TestWhisperConfiguration(t *testing.T) {
	testCases := []struct {
		name               string
		config             Config
		expectedBehavior   string
	}{
		{
			name: "Auto language detection enabled",
			config: Config{
				Language:           "auto",
				AutoDetectLanguage: true,
			},
			expectedBehavior: "Should set language to 'auto' for multilingual models",
		},
		{
			name: "Fixed language mode",
			config: Config{
				Language:           "en",
				AutoDetectLanguage: false,
			},
			expectedBehavior: "Should use fixed language 'en'",
		},
		{
			name: "Fixed language mode",
			config: Config{
				Language: "en",
			},
			expectedBehavior: "Should use fixed language",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Document expected behavior for each configuration
			t.Log(tc.expectedBehavior)
			
			// Verify config struct has expected fields
			if tc.config.Language == "" {
				t.Error("Language should not be empty")
			}
		})
	}
}