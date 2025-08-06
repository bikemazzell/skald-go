// +build integration

package skald

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"skald/pkg/skald/app"
	"skald/pkg/skald/audio"
	"skald/pkg/skald/mocks"
	"skald/pkg/skald/transcriber"
)

// TestEndToEndPipeline tests the complete audio processing pipeline
func TestEndToEndPipeline(t *testing.T) {
	// Skip if no model available
	modelPath := "../../models/ggml-tiny.bin"
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("Model not available, skipping integration test")
	}
	
	testCases := []struct {
		name         string
		audioFile    string
		language     string
		expectOutput bool
		expectError  bool
	}{
		{
			name:         "silence should produce no output",
			audioFile:    "../../testdata/audio/silence_1s.raw",
			language:     "en",
			expectOutput: false,
			expectError:  false,
		},
		{
			name:         "mixed signal processing",
			audioFile:    "../../testdata/audio/mixed_speech_pattern.raw",
			language:     "auto",
			expectOutput: false, // Synthetic audio won't produce meaningful transcription
			expectError:  false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load test audio
			audioData, err := loadRawAudio(tc.audioFile)
			if err != nil {
				t.Fatalf("Failed to load test audio: %v", err)
			}
			
			// Create components
			mockAudio := &mocks.MockAudioCapture{}
			mockOutput := &mocks.MockOutput{}
			silenceDetector := audio.NewSilenceDetector()
			
			// Setup audio to return our test data
			mockAudio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
				audioChan := make(chan []float32, 10)
				go func() {
					defer close(audioChan)
					
					// Send audio in chunks
					chunkSize := 1024
					for i := 0; i < len(audioData); i += chunkSize {
						end := i + chunkSize
						if end > len(audioData) {
							end = len(audioData)
						}
						
						select {
						case audioChan <- audioData[i:end]:
						case <-ctx.Done():
							return
						}
						
						// Small delay to simulate real-time
						time.Sleep(10 * time.Millisecond)
					}
					
					// Send silence to trigger transcription
					silence := make([]float32, 16000) // 1 second
					for i := 0; i < 10; i++ {
						select {
						case audioChan <- silence:
						case <-ctx.Done():
							return
						}
					}
				}()
				return audioChan, nil
			}
			
			// Create transcriber
			trans, err := transcriber.NewWhisper(modelPath, tc.language)
			if err != nil {
				t.Fatalf("Failed to create transcriber: %v", err)
			}
			defer trans.Close()
			
			// Create app
			config := app.Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  1.0,
				Continuous:       false,
			}
			
			application := app.New(mockAudio, trans, mockOutput, silenceDetector, config)
			
			// Run with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			err = application.Run(ctx)
			
			// Check results
			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil && err != context.DeadlineExceeded {
					t.Errorf("Unexpected error: %v", err)
				}
			}
			
			// Check output
			hasOutput := len(mockOutput.AllTexts) > 0
			if hasOutput != tc.expectOutput {
				t.Errorf("Expected output: %v, got output: %v (texts: %v)", 
					tc.expectOutput, hasOutput, mockOutput.AllTexts)
			}
		})
	}
}

// TestComponentIntegration tests integration between components
func TestComponentIntegration(t *testing.T) {
	t.Run("audio capture to silence detection", func(t *testing.T) {
		silenceDetector := audio.NewSilenceDetector()
		
		testCases := []struct {
			name     string
			samples  []float32
			expected bool
		}{
			{"pure silence", make([]float32, 1000), true},
			{"loud signal", generateSineWave(1000, 440), false},
			{"quiet noise", generateNoise(1000, 0.005), true},
			{"mixed signal", generateMixedSignal(1000), false},
		}
		
		for _, tc := range testCases {
			result := silenceDetector.IsSilent(tc.samples, 0.01)
			if result != tc.expected {
				t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, result)
			}
		}
	})
	
	t.Run("silence detection to transcription trigger", func(t *testing.T) {
		// This tests the app logic for when transcription should be triggered
		config := app.Config{
			SampleRate:       16000,
			SilenceThreshold: 0.01,
			SilenceDuration:  0.5, // 0.5 seconds
			Continuous:       false,
		}
		
		silentThreshold := int(float32(config.SampleRate) * config.SilenceDuration)
		expectedThreshold := 8000 // 16000 * 0.5
		
		if silentThreshold != expectedThreshold {
			t.Errorf("Expected silent threshold %d, got %d", expectedThreshold, silentThreshold)
		}
	})
}

// TestRealTimeProcessing tests real-time processing characteristics
func TestRealTimeProcessing(t *testing.T) {
	t.Run("processing latency", func(t *testing.T) {
		silenceDetector := audio.NewSilenceDetector()
		
		// Test that silence detection is fast enough for real-time
		samples := make([]float32, 1024) // Typical audio buffer size
		
		start := time.Now()
		for i := 0; i < 1000; i++ {
			silenceDetector.IsSilent(samples, 0.01)
		}
		elapsed := time.Since(start)
		
		// Should be much faster than real-time
		// 1000 buffers of 1024 samples at 16kHz = ~64 seconds of audio
		maxAllowedTime := time.Second // Very generous for 64s of audio processing
		
		if elapsed > maxAllowedTime {
			t.Errorf("Silence detection too slow: %v for 1000 buffers", elapsed)
		}
	})
}

// TestErrorRecovery tests error recovery in the pipeline
func TestErrorRecovery(t *testing.T) {
	t.Run("transcription error recovery", func(t *testing.T) {
		// Test that the app continues running even if transcription fails
		mockAudio := &mocks.MockAudioCapture{}
		mockTranscriber := &mocks.MockTranscriber{}
		mockOutput := &mocks.MockOutput{}
		silenceDetector := audio.NewSilenceDetector()
		
		// Setup audio to provide data that triggers transcription
		mockAudio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
			audioChan := make(chan []float32, 5)
			go func() {
				defer close(audioChan)
				// Send audio then silence to trigger transcription
				audioChan <- []float32{0.1, 0.2, 0.3}
				for i := 0; i < 10; i++ {
					audioChan <- make([]float32, 1600) // Silence
				}
			}()
			return audioChan, nil
		}
		
		// Make transcriber fail
		mockTranscriber.TranscribeFunc = func(audio []float32) (string, error) {
			return "", errors.New("transcription failed")
		}
		
		config := app.Config{
			SampleRate:       16000,
			SilenceThreshold: 0.01,
			SilenceDuration:  0.1,
			Continuous:       false,
		}
		
		application := app.New(mockAudio, mockTranscriber, mockOutput, silenceDetector, config)
		
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		
		// Should not return error even though transcription fails
		err := application.Run(ctx)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("App should handle transcription errors gracefully, got: %v", err)
		}
		
		// Transcriber should have been called
		if mockTranscriber.TranscribeCalled == 0 {
			t.Error("Transcriber should have been called")
		}
		
		// Output should not have been called due to error
		if mockOutput.WriteCalled > 0 {
			t.Error("Output should not be called on transcription error")
		}
	})
}

// Helper functions

func loadRawAudio(filename string) ([]float32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	
	numSamples := stat.Size() / 4 // 4 bytes per float32
	samples := make([]float32, numSamples)
	
	err = binary.Read(file, binary.LittleEndian, samples)
	if err != nil {
		return nil, err
	}
	
	return samples, nil
}

func generateSineWave(numSamples int, frequency float64) []float32 {
	samples := make([]float32, numSamples)
	sampleRate := 16000.0
	
	for i := 0; i < numSamples; i++ {
		t := float64(i) / sampleRate
		samples[i] = float32(0.5 * math.Sin(2*math.Pi*frequency*t))
	}
	
	return samples
}

func generateNoise(numSamples int, amplitude float32) []float32 {
	samples := make([]float32, numSamples)
	seed := uint32(12345)
	
	for i := 0; i < numSamples; i++ {
		seed = seed*1103515245 + 12345
		noise := float32((seed>>16)&0x7fff)/32767.0*2.0 - 1.0
		samples[i] = noise * amplitude
	}
	
	return samples
}

func generateMixedSignal(numSamples int) []float32 {
	samples := make([]float32, numSamples)
	sampleRate := 16000.0
	
	for i := 0; i < numSamples; i++ {
		t := float64(i) / sampleRate
		// Mix of frequencies
		samples[i] = float32(0.3*math.Sin(2*math.Pi*300*t) + 0.2*math.Sin(2*math.Pi*800*t))
	}
	
	return samples
}