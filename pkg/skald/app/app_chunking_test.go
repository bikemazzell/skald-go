package app

import (
	"context"
	"testing"
	"time"

	"skald/pkg/skald/mocks"
)

// TestProcessSession_ChunkingAt25Seconds tests that audio is chunked at 25 seconds
func TestProcessSession_ChunkingAt25Seconds(t *testing.T) {
	// Setup
	mockTranscriber := &mocks.MockTranscriber{}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false // Never silent
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	// Create a session with max 25 seconds
	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0), // 25 seconds
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create audio channel
	audioChan := make(chan []float32, 10)

	// Simulate exactly 25 seconds of continuous audio to trigger chunking
	// Send in smaller chunks to simulate real audio capture
	go func() {
		// Send 25 seconds worth of audio in 0.1 second chunks
		chunkSize := 1600 // 0.1 seconds at 16000 Hz
		totalChunks := 250 // 25 seconds / 0.1 seconds
		
		for i := 0; i < totalChunks; i++ {
			chunk := make([]float32, chunkSize)
			for j := range chunk {
				chunk[j] = 0.1 // Non-silent audio
			}
			select {
			case audioChan <- chunk:
			case <-ctx.Done():
				close(audioChan)
				return
			}
		}
		// Don't send more audio, just close to trigger processing
		close(audioChan)
	}()

	// Process session in a goroutine
	done := make(chan error)
	go func() {
		done <- app.processSession(ctx, audioChan, session)
	}()

	// Wait for processing or timeout
	select {
	case <-done:
		// Expected: should have transcribed at 25 seconds mark
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}

	// Should have exactly 1 call at 25s mark (buffer is reset after processing, nothing left)
	if mockTranscriber.TranscribeCalled != 1 {
		t.Errorf("Expected 1 transcription call at 25s mark, got %d calls", 
			mockTranscriber.TranscribeCalled)
	}
}

// TestProcessSession_MultipleChunks tests handling of very long audio
func TestProcessSession_MultipleChunks(t *testing.T) {
	transcriptionCount := 0
	mockTranscriber := &mocks.MockTranscriber{
		TranscribeFunc: func(audio []float32) (string, error) {
			transcriptionCount++
			return "transcribed", nil
		},
	}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false // Never silent
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       true, // Continuous mode to process multiple chunks
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create audio channel
	audioChan := make(chan []float32, 10)

	// Simulate 60 seconds of audio (should trigger 2 chunks)
	go func() {
		for i := 0; i < 60; i++ {
			chunk := make([]float32, 16000) // 1 second chunks
			for j := range chunk {
				chunk[j] = 0.1
			}
			select {
			case audioChan <- chunk:
			case <-ctx.Done():
				close(audioChan)
				return
			}
		}
		// After 60 seconds, send silence to end
		silentChunk := make([]float32, 16000*2) // 2 seconds of silence
		audioChan <- silentChunk
		close(audioChan)
	}()

	// Create mock audio capture
	mockAudio := &mocks.MockAudioCapture{
		StartFunc: func(ctx context.Context) (<-chan []float32, error) {
			return audioChan, nil
		},
		StopFunc: func() error {
			return nil
		},
	}
	app.audio = mockAudio

	// Run the app
	go func() {
		app.Run(ctx)
	}()

	// Wait for processing
	time.Sleep(1 * time.Second)
	cancel()

	// Should have at least 2 transcriptions (one at 25s, one at 50s)
	if transcriptionCount < 2 {
		t.Errorf("Expected at least 2 transcriptions for 60s audio, got %d", transcriptionCount)
	}
}

// TestProcessSession_ExactBoundary tests edge case at exactly 25 seconds
func TestProcessSession_ExactBoundary(t *testing.T) {
	mockTranscriber := &mocks.MockTranscriber{}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0),
	}

	ctx := context.Background()
	audioChan := make(chan []float32, 1)

	// Send exactly 25 seconds of audio
	exactSamples := 25 * 16000
	audioChan <- make([]float32, exactSamples)
	close(audioChan)

	app.processSession(ctx, audioChan, session)

	// Should have transcribed exactly once
	if mockTranscriber.TranscribeCalled != 1 {
		t.Errorf("Expected exactly 1 transcription at boundary, got %d", 
			mockTranscriber.TranscribeCalled)
	}
}

// TestProcessSession_RemainingAudioOnClose tests that remaining audio is processed
func TestProcessSession_RemainingAudioOnClose(t *testing.T) {
	mockTranscriber := &mocks.MockTranscriber{}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0),
	}

	ctx := context.Background()
	audioChan := make(chan []float32, 1)

	// Send 10 seconds of audio then close (less than max)
	audioChan <- make([]float32, 10*16000)
	close(audioChan)

	app.processSession(ctx, audioChan, session)

	// Should have transcribed the remaining audio
	if mockTranscriber.TranscribeCalled != 1 {
		t.Errorf("Expected remaining audio to be transcribed, got %d calls", 
			mockTranscriber.TranscribeCalled)
	}

	// Verify the correct amount was transcribed
	if len(mockTranscriber.LastAudio) != 10*16000 {
		t.Errorf("Expected 10 seconds of audio, got %.1f seconds", 
			float64(len(mockTranscriber.LastAudio))/16000.0)
	}
}

// TestProcessSession_ContextCancellation tests proper cleanup on context cancellation
func TestProcessSession_ContextCancellation(t *testing.T) {
	mockTranscriber := &mocks.MockTranscriber{}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0),
	}

	ctx, cancel := context.WithCancel(context.Background())
	audioChan := make(chan []float32, 1)

	// Send 5 seconds of audio
	audioChan <- make([]float32, 5*16000)

	// Start processing in goroutine
	done := make(chan error)
	go func() {
		done <- app.processSession(ctx, audioChan, session)
	}()

	// Cancel context after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for processing to complete
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Processing didn't complete after context cancellation")
	}

	// Should have transcribed the buffered audio before exiting
	if mockTranscriber.TranscribeCalled != 1 {
		t.Errorf("Expected buffered audio to be transcribed on cancel, got %d calls", 
			mockTranscriber.TranscribeCalled)
	}
}

// TestProcessSession_ChunkingWithSilence tests interaction between chunking and silence detection
func TestProcessSession_ChunkingWithSilence(t *testing.T) {
	callCount := 0
	mockTranscriber := &mocks.MockTranscriber{
		TranscribeFunc: func(audio []float32) (string, error) {
			callCount++
			return "text", nil
		},
	}
	mockOutput := &mocks.MockOutput{}
	
	// Silence detector that returns true after sample 20000
	sampleCount := 0
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			sampleCount += len(samples)
			// Silence after 20 seconds
			return sampleCount > 20*16000
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0),
	}

	ctx := context.Background()
	audioChan := make(chan []float32, 100)

	// Send 22 seconds of audio (20s speech + 2s silence)
	for i := 0; i < 22; i++ {
		audioChan <- make([]float32, 16000)
	}
	close(audioChan)

	app.processSession(ctx, audioChan, session)

	// Should transcribe at silence detection (before hitting 25s limit)
	if callCount != 1 {
		t.Errorf("Expected 1 transcription at silence, got %d", callCount)
	}

	// Audio should be approximately 21.5 seconds (20s speech + 1.5s silence threshold)
	expectedSamples := int(21.5 * 16000)
	actualSamples := len(mockTranscriber.LastAudio)
	tolerance := 16000 // 1 second tolerance
	
	if actualSamples < expectedSamples-tolerance || actualSamples > expectedSamples+tolerance {
		t.Errorf("Expected ~%d samples (21.5s), got %d samples (%.1fs)", 
			expectedSamples, actualSamples, float64(actualSamples)/16000.0)
	}
}

// TestProcessSession_EmptyBuffer tests that empty buffers are not transcribed
func TestProcessSession_EmptyBuffer(t *testing.T) {
	mockTranscriber := &mocks.MockTranscriber{}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return true // Always silent
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  0.1, // Short silence duration for quick test
		Continuous:       false,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	session := &TranscriptionSession{
		buffer:          make([]float32, 0),
		silentSamples:   0,
		silentThreshold: int(float32(config.SampleRate) * config.SilenceDuration),
		maxSamples:      int(float32(config.SampleRate) * 25.0),
	}

	ctx := context.Background()
	audioChan := make(chan []float32, 1)

	// Close immediately without sending audio
	close(audioChan)

	app.processSession(ctx, audioChan, session)

	// Should not have called transcribe for empty buffer
	if mockTranscriber.TranscribeCalled != 0 {
		t.Errorf("Expected no transcription for empty buffer, got %d calls", 
			mockTranscriber.TranscribeCalled)
	}
}

// TestProcessSession_VeryLongContinuousAudio tests handling of extremely long audio
func TestProcessSession_VeryLongContinuousAudio(t *testing.T) {
	transcriptionTimes := []time.Time{}
	mockTranscriber := &mocks.MockTranscriber{
		TranscribeFunc: func(audio []float32) (string, error) {
			transcriptionTimes = append(transcriptionTimes, time.Now())
			return "text", nil
		},
	}
	mockOutput := &mocks.MockOutput{}
	mockSilence := &mocks.MockSilenceDetector{
		IsSilentFunc: func(samples []float32, threshold float32) bool {
			return false // Never silent
		},
	}

	config := Config{
		SampleRate:       16000,
		SilenceThreshold: 0.01,
		SilenceDuration:  1.5,
		Continuous:       true,
	}

	app := &App{
		transcriber:     mockTranscriber,
		output:          mockOutput,
		silenceDetector: mockSilence,
		config:          config,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audioChan := make(chan []float32, 100)

	// Simulate 2 minutes of continuous audio
	go func() {
		for i := 0; i < 120; i++ { // 120 seconds
			select {
			case audioChan <- make([]float32, 16000):
				time.Sleep(5 * time.Millisecond) // Simulate real-time
			case <-ctx.Done():
				close(audioChan)
				return
			}
		}
		close(audioChan)
	}()

	// Create mock audio capture
	mockAudio := &mocks.MockAudioCapture{
		StartFunc: func(ctx context.Context) (<-chan []float32, error) {
			return audioChan, nil
		},
		StopFunc: func() error {
			return nil
		},
	}
	app.audio = mockAudio

	// Run in goroutine
	go app.Run(ctx)

	// Wait for some transcriptions
	time.Sleep(2 * time.Second)
	cancel()

	// Should have multiple transcriptions (at least 4 for 2 minutes: 25s, 50s, 75s, 100s)
	if len(transcriptionTimes) < 4 {
		t.Errorf("Expected at least 4 transcriptions for 2 minutes audio, got %d", 
			len(transcriptionTimes))
	}

	// Verify transcriptions happen at regular intervals (approximately every 25 seconds)
	for i := 1; i < len(transcriptionTimes); i++ {
		interval := transcriptionTimes[i].Sub(transcriptionTimes[i-1])
		// Allow some tolerance for processing time
		if interval > 30*time.Second {
			t.Errorf("Transcription interval too long: %v", interval)
		}
	}
}