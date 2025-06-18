package transcriber

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"skald/internal/config"
	"skald/internal/model"
)

// Mock components for testing
type mockRecorder struct {
	mu        sync.Mutex
	started   bool
	stopped   bool
	audioChan chan<- []float32
	errToReturn error
}

func (m *mockRecorder) Start(ctx context.Context, samples chan<- []float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.errToReturn != nil {
		return m.errToReturn
	}
	
	m.started = true
	m.audioChan = samples
	
	// Simulate audio data
	go func() {
		// Send some test audio samples
		select {
		case samples <- []float32{0.1, 0.2, 0.3}:
		case <-ctx.Done():
			return
		}
	}()
	
	return nil
}

func (m *mockRecorder) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return nil
}

type mockProcessor struct {
	mu           sync.Mutex
	samples      []float32
	shouldError  bool
	errorToReturn error
}

func (m *mockProcessor) ProcessSamples(samples []float32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.shouldError {
		return m.errorToReturn
	}
	
	m.samples = append(m.samples, samples...)
	return nil
}

func (m *mockProcessor) GetBuffer() []float32 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.samples
}

func (m *mockProcessor) ClearBuffer() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.samples = nil
}

type mockWhisper struct {
	mu              sync.Mutex
	transcriptions  []string
	nextTranscript  string
	shouldError     bool
	closed          bool
}

func (m *mockWhisper) Transcribe(audio []float32) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.shouldError {
		return "", errors.New("transcription error")
	}
	
	if m.nextTranscript != "" {
		m.transcriptions = append(m.transcriptions, m.nextTranscript)
		return m.nextTranscript, nil
	}
	
	return "test transcription", nil
}

func (m *mockWhisper) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
}

type mockClipboard struct {
	mu          sync.Mutex
	copiedTexts []string
	copyError   error
	pasteError  error
	pasteCalled bool
}

func (m *mockClipboard) Copy(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.copyError != nil {
		return m.copyError
	}
	
	m.copiedTexts = append(m.copiedTexts, text)
	return nil
}

func (m *mockClipboard) Paste() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.pasteCalled = true
	return m.pasteError
}

func (m *mockClipboard) IsValidText(text string) bool {
	// Simple validation for testing
	return text != "" && len(text) < 1000000
}

func TestTranscriber_StartStop(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate: 16000,
			Channels:   1,
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
		}{
			ChannelBufferSize: 10,
			AutoPaste:         false,
		},
		Debug: struct {
			PrintStatus         bool `json:"print_status"`
			PrintTranscriptions bool `json:"print_transcriptions"`
		}{
			PrintTranscriptions: false,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	// Create transcriber
	transcriber, err := New(cfg, logger, modelMgr)
	if err != nil {
		t.Fatalf("Failed to create transcriber: %v", err)
	}

	// Test initial state
	if transcriber.IsRunning() {
		t.Error("Transcriber should not be running initially")
	}

	// Test stop when not running
	err = transcriber.Stop()
	if err != nil {
		t.Errorf("Stop on non-running transcriber should not error: %v", err)
	}

	// Test double stop
	err = transcriber.Stop()
	if err != nil {
		t.Errorf("Double stop should not error: %v", err)
	}

	// Clean up
	transcriber.Close()
}

func TestTranscriber_ConcurrentStartStop(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate: 16000,
			Channels:   1,
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
		}{
			ChannelBufferSize: 10,
			AutoPaste:         false,
		},
		Debug: struct {
			PrintStatus         bool `json:"print_status"`
			PrintTranscriptions bool `json:"print_transcriptions"`
		}{
			PrintTranscriptions: false,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	transcriber, err := New(cfg, logger, modelMgr)
	if err != nil {
		t.Fatalf("Failed to create transcriber: %v", err)
	}
	defer transcriber.Close()

	// Test concurrent stop calls
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			transcriber.Stop()
		}()
	}
	wg.Wait()

	// Verify transcriber is stopped
	if transcriber.IsRunning() {
		t.Error("Transcriber should be stopped after concurrent stops")
	}
}

func TestTranscriber_ProcessingAtomicBool(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate: 16000,
			Channels:   1,
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
		}{
			ChannelBufferSize: 10,
			AutoPaste:         false,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	transcriber, err := New(cfg, logger, modelMgr)
	if err != nil {
		t.Fatalf("Failed to create transcriber: %v", err)
	}
	defer transcriber.Close()

	// Set processing flag to true
	transcriber.processing.Store(true)

	// Try to process audio (should fail)
	ctx := context.Background()
	audioChan := make(chan []float32)
	transcriptionChan := make(chan string)

	err = transcriber.processAudio(ctx, audioChan, transcriptionChan)
	if err == nil {
		t.Error("Expected error when processing is already in progress")
	}
	if err.Error() != "audio processing already in progress" {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Verify flag is still true
	if !transcriber.processing.Load() {
		t.Error("Processing flag should still be true")
	}
}

func TestTranscriber_FilterSpecialTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No special tokens",
			input:    "This is a normal transcription",
			expected: "This is a normal transcription",
		},
		{
			name:     "Single special token",
			input:    "[BLANK_AUDIO] This is text",
			expected: "This is text",
		},
		{
			name:     "Multiple special tokens",
			input:    "[SILENCE] Some text [NOISE] more text [SPEECH]",
			expected: "Some text  more text",
		},
		{
			name:     "Only special tokens",
			input:    "[BLANK_AUDIO][SILENCE][NOISE]",
			expected: "",
		},
		{
			name:     "Special tokens with extra spaces",
			input:    "  [MUSIC]  Text with spaces  [SPEECH]  ",
			expected: "Text with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the filtering logic from processBuffer
			filteredText := tt.input
			for _, token := range tokensToFilter {
				filteredText = strings.ReplaceAll(filteredText, token, "")
			}
			filteredText = strings.TrimSpace(filteredText)

			if filteredText != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, filteredText)
			}
		})
	}
}

func TestTranscriber_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate:      16000,
			Channels:        1,
			SilenceDuration: 1.0,
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
		}{
			ChannelBufferSize: 10,
			AutoPaste:         false,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	transcriber, err := New(cfg, logger, modelMgr)
	if err != nil {
		t.Fatalf("Failed to create transcriber: %v", err)
	}
	defer transcriber.Close()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	audioChan := make(chan []float32, 1)
	transcriptionChan := make(chan string, 1)

	// Start processing in a goroutine
	processingDone := make(chan bool)
	go func() {
		err := transcriber.processAudio(ctx, audioChan, transcriptionChan)
		if err != nil {
			t.Logf("Processing ended with error: %v", err)
		}
		processingDone <- true
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for processing to end
	select {
	case <-processingDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Processing did not stop after context cancellation")
	}
}

func TestTranscriber_TranscriptionChannelTimeout(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate:      16000,
			Channels:        1,
			SilenceDuration: 0.1, // Short timeout for testing
		},
		Processing: struct {
			ShutdownTimeout   int     `json:"shutdown_timeout"`
			EventWaitTimeout  float64 `json:"event_wait_timeout"`
			AutoPaste         bool    `json:"auto_paste"`
			ChannelBufferSize int     `json:"channel_buffer_size"`
		}{
			ChannelBufferSize: 1, // Small buffer to test blocking
			AutoPaste:         false,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	transcriber, err := New(cfg, logger, modelMgr)
	if err != nil {
		t.Fatalf("Failed to create transcriber: %v", err)
	}
	defer transcriber.Close()

	// Create a full transcription channel (capacity 1, don't read from it)
	transcriptionChan := make(chan string, 1)
	transcriptionChan <- "blocking text"

	// Process buffer should handle the timeout gracefully
	buffer := []float32{0.1, 0.2, 0.3}
	err = transcriber.processBuffer(buffer, transcriptionChan)
	
	// Should not error - it should log a warning instead
	if err != nil {
		t.Errorf("processBuffer should not error on channel timeout: %v", err)
	}
}