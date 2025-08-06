package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"skald/pkg/skald/mocks"
)

func TestApp_Run(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		setupMocks     func(*mocks.MockAudioCapture, *mocks.MockTranscriber, *mocks.MockOutput, *mocks.MockSilenceDetector)
		expectedError  bool
		expectedOutput []string
	}{
		{
			name: "successful single transcription",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  0.001, // 1ms for faster tests, requires only 16 samples of silence
				Continuous:       false,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				// Setup audio capture to return samples then silence
				audioChan := make(chan []float32, 10)
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					// Send some audio samples
					go func() {
						defer close(audioChan)
						// First, send non-silent audio
						audioChan <- []float32{0.1, 0.2, 0.3}
						audioChan <- []float32{0.1, 0.2, 0.3}
						// Then send enough silence to trigger transcription
						// silence duration is 0.1s, sample rate is 16000, so need 1600 samples of silence
						for i := 0; i < 20; i++ {
							select {
							case audioChan <- []float32{0.001, 0.001}:
							case <-ctx.Done():
								return
							}
						}
					}()
					return audioChan, nil
				}

				// Setup silence detector - more predictable pattern
				callCount := 0
				silence.IsSilentFunc = func(samples []float32, threshold float32) bool {
					callCount++
					// First 2 calls are speech, rest are silence
					return callCount > 2
				}

				// Setup transcriber
				trans.TranscribeFunc = func(audio []float32) (string, error) {
					return "Hello, World!", nil
				}
			},
			expectedError:  false,
			expectedOutput: []string{"Hello, World!"},
		},
		{
			name: "audio start error",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  1.0,
				Continuous:       false,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					return nil, errors.New("audio device not found")
				}
				// Stop should still be called even on start error
			},
			expectedError:  true,
			expectedOutput: nil,
		},
		{
			name: "transcription error - continues operation",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  0.05,
				Continuous:       false,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				audioChan := make(chan []float32, 5)
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					go func() {
						audioChan <- []float32{0.1, 0.2}
						for i := 0; i < 5; i++ {
							audioChan <- []float32{0.001}
						}
						close(audioChan)
					}()
					return audioChan, nil
				}

				silence.IsSilentFunc = func(samples []float32, threshold float32) bool {
					return samples[0] < 0.01
				}

				trans.TranscribeFunc = func(audio []float32) (string, error) {
					return "", errors.New("transcription failed")
				}
			},
			expectedError:  false, // Error is logged but not returned
			expectedOutput: nil,
		},
		{
			name: "empty transcription result",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  0.05,
				Continuous:       false,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				audioChan := make(chan []float32, 5)
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					go func() {
						audioChan <- []float32{0.1}
						for i := 0; i < 5; i++ {
							audioChan <- []float32{0.001}
						}
						close(audioChan)
					}()
					return audioChan, nil
				}

				silence.IsSilentFunc = func(samples []float32, threshold float32) bool {
					return samples[0] < 0.01
				}

				trans.TranscribeFunc = func(audio []float32) (string, error) {
					return "", nil // Empty result
				}
			},
			expectedError:  false,
			expectedOutput: nil, // No output for empty transcription
		},
		{
			name: "context cancellation",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  1.0,
				Continuous:       true,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				audioChan := make(chan []float32)
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					go func() {
						<-ctx.Done()
						close(audioChan)
					}()
					return audioChan, nil
				}
			},
			expectedError:  false, // Context cancellation returns ctx.Err() which might be nil or context.Canceled
			expectedOutput: nil,
		},
		{
			name: "channel closed unexpectedly",
			config: Config{
				SampleRate:       16000,
				SilenceThreshold: 0.01,
				SilenceDuration:  0.05,
				Continuous:       false,
			},
			setupMocks: func(audio *mocks.MockAudioCapture, trans *mocks.MockTranscriber, out *mocks.MockOutput, silence *mocks.MockSilenceDetector) {
				audioChan := make(chan []float32, 2)
				audio.StartFunc = func(ctx context.Context) (<-chan []float32, error) {
					go func() {
						// Send one sample then close channel
						audioChan <- []float32{0.1, 0.2}
						close(audioChan)
					}()
					return audioChan, nil
				}
				
				silence.IsSilentFunc = func(samples []float32, threshold float32) bool {
					return false // Never silent
				}
			},
			expectedError:  false, // Channel close is handled gracefully
			expectedOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockAudio := &mocks.MockAudioCapture{}
			mockTrans := &mocks.MockTranscriber{}
			mockOutput := &mocks.MockOutput{}
			mockSilence := &mocks.MockSilenceDetector{}

			// Setup mocks
			tt.setupMocks(mockAudio, mockTrans, mockOutput, mockSilence)

			// Create app
			app := New(mockAudio, mockTrans, mockOutput, mockSilence, tt.config)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			// Run app
			err := app.Run(ctx)

			// Check error - be lenient about context errors
			hasError := err != nil && err != context.DeadlineExceeded && err != context.Canceled
			if hasError != tt.expectedError {
				t.Errorf("Run() error = %v, expectedError %v", err, tt.expectedError)
			}

			// Check output
			if len(mockOutput.AllTexts) != len(tt.expectedOutput) {
				t.Errorf("Expected %d outputs, got %d", len(tt.expectedOutput), len(mockOutput.AllTexts))
			} else {
				for i, expected := range tt.expectedOutput {
					if i < len(mockOutput.AllTexts) && mockOutput.AllTexts[i] != expected {
						t.Errorf("Output[%d] = %q, want %q", i, mockOutput.AllTexts[i], expected)
					}
				}
			}

			// Verify cleanup - Stop should only be called if Start succeeded
			if !tt.expectedError && mockAudio.StartCalled > 0 && mockAudio.StopCalled == 0 {
				t.Error("Audio Stop() was not called after successful start")
			}
		})
	}
}

func TestApp_transcribeAndOutput(t *testing.T) {
	tests := []struct {
		name          string
		buffer        []float32
		transcription string
		transcribeErr error
		outputErr     error
		expectError   bool
	}{
		{
			name:          "successful transcription and output",
			buffer:        []float32{0.1, 0.2, 0.3},
			transcription: "Test transcription",
			transcribeErr: nil,
			outputErr:     nil,
			expectError:   false,
		},
		{
			name:          "transcription error",
			buffer:        []float32{0.1, 0.2, 0.3},
			transcription: "",
			transcribeErr: errors.New("transcription failed"),
			outputErr:     nil,
			expectError:   true,
		},
		{
			name:          "output error",
			buffer:        []float32{0.1, 0.2, 0.3},
			transcription: "Test transcription",
			transcribeErr: nil,
			outputErr:     errors.New("output failed"),
			expectError:   true,
		},
		{
			name:          "empty transcription - no output",
			buffer:        []float32{0.1, 0.2, 0.3},
			transcription: "",
			transcribeErr: nil,
			outputErr:     nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTrans := &mocks.MockTranscriber{
				TranscribeFunc: func(audio []float32) (string, error) {
					return tt.transcription, tt.transcribeErr
				},
			}

			mockOutput := &mocks.MockOutput{
				WriteFunc: func(text string) error {
					return tt.outputErr
				},
			}

			app := &App{
				transcriber: mockTrans,
				output:      mockOutput,
			}

			err := app.transcribeAndOutput(tt.buffer)

			if (err != nil) != tt.expectError {
				t.Errorf("transcribeAndOutput() error = %v, expectError %v", err, tt.expectError)
			}

			// Verify transcriber was called
			if mockTrans.TranscribeCalled != 1 {
				t.Errorf("Expected Transcribe to be called once, got %d", mockTrans.TranscribeCalled)
			}

			// Verify output was called only for non-empty transcriptions
			expectedOutputCalls := 0
			if tt.transcription != "" && tt.transcribeErr == nil {
				expectedOutputCalls = 1
			}
			if mockOutput.WriteCalled != expectedOutputCalls {
				t.Errorf("Expected Write to be called %d times, got %d", expectedOutputCalls, mockOutput.WriteCalled)
			}
		})
	}
}

func TestApp_processSession_TranscriptionErrorLogging(t *testing.T) {
	// Test specifically targets app.go:98-100 error logging path
	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  0.001, // 1ms for fast test  
		Continuous:       false,
	}

	// Create mocks
	mockTrans := &mocks.MockTranscriber{
		TranscribeFunc: func(audio []float32) (string, error) {
			return "", errors.New("transcription model error")
		},
	}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			// Always return true to trigger transcription immediately
			return true
		},
	}

	app := &App{
		transcriber:     mockTrans,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	// Create audio channel with sample data
	audioChan := make(chan []float32, 10)
	// First add some samples to build up the buffer
	audioChan <- []float32{0.1, 0.2, 0.3}
	// Add more silence samples to reach the threshold 
	// silenceThreshold = 16000 * 0.001 = 16 samples
	audioChan <- []float32{0.001, 0.001, 0.001, 0.001, 0.001, 0.001, 0.001, 0.001}
	audioChan <- []float32{0.001, 0.001, 0.001, 0.001, 0.001, 0.001, 0.001, 0.001}
	close(audioChan)

	// Create session
	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration), // 16 samples
	}

	ctx := context.Background()

	// This should trigger the error logging path at app.go:98-100
	err := app.processSession(ctx, audioChan, session)

	// processSession should return nil (not propagate transcription error)
	if err != nil {
		t.Errorf("processSession() should not return transcription errors, got: %v", err)
	}

	// Verify transcriber was called
	if mockTrans.TranscribeCalled != 1 {
		t.Errorf("Expected Transcribe to be called once, got %d", mockTrans.TranscribeCalled)
	}

	// Output should not be called due to transcription error
	if mockOutput.WriteCalled != 0 {
		t.Errorf("Expected Write to not be called due to transcription error, got %d calls", mockOutput.WriteCalled)
	}
}