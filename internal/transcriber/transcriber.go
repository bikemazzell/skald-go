package transcriber

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"skald/internal/audio"
	"skald/internal/config"
	"skald/internal/model"
	"skald/internal/whisper"
	"skald/pkg/utils"
)

// Add these constants at the top of the file
var tokensToFilter = []string{
	"[BLANK_AUDIO]",
	"[SILENCE]",
	"[NOISE]",
	"[SPEECH]",
	"[MUSIC]",
}

// Transcriber handles the audio recording and transcription process
type Transcriber struct {
	cfg       *config.Config
	modelMgr  *model.ModelManager
	recorder  *audio.Recorder
	processor *audio.Processor
	whisper   *whisper.Whisper
	clipboard *utils.ClipboardManager
	logger    *log.Logger

	mu         sync.Mutex
	isRunning  bool
	ctx        context.Context
	cancel     context.CancelFunc
	processing atomic.Bool
}

// New creates a new Transcriber instance
func New(cfg *config.Config, logger *log.Logger, modelMgr *model.ModelManager) (*Transcriber, error) {
	// Create audio processor
	processor, err := audio.NewProcessor(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio processor: %w", err)
	}

	// Create recorder
	recorder, err := audio.NewRecorder(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create recorder: %w", err)
	}

	// Create clipboard manager with auto-paste setting from config
	clipboard := utils.NewClipboardManager(cfg.Processing.AutoPaste)

	return &Transcriber{
		cfg:       cfg,
		logger:    logger,
		processor: processor,
		modelMgr:  modelMgr,
		recorder:  recorder,
		clipboard: clipboard,
	}, nil
}

// Start begins the recording and transcription process
func (t *Transcriber) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.isRunning {
		return fmt.Errorf("transcriber is already running")
	}

	// Create new recorder instance
	var err error
	t.recorder, err = audio.NewRecorder(t.cfg, t.logger)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}

	// Get model path and verify it exists
	modelPath := t.modelMgr.GetModelPath()
	if t.cfg.Verbose {
		t.logger.Printf("Using model at path: %s", modelPath)
	}

	if _, err := os.Stat(modelPath); err != nil {
		return fmt.Errorf("model file not accessible: %w", err)
	}

	// Initialize whisper
	whisperInstance, err := whisper.New(modelPath, whisper.Config{
		Language: t.cfg.Whisper.Language,
		Silent:   t.cfg.Whisper.Silent,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize whisper: %w", err)
	}
	t.whisper = whisperInstance

	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.isRunning = true

	// Create channels for audio data
	audioChan := make(chan []float32, t.cfg.Processing.ChannelBufferSize)
	transcriptionChan := make(chan string, t.cfg.Processing.ChannelBufferSize)

	// Start recording goroutine
	go func() {
		if err := t.recorder.Start(t.ctx, audioChan); err != nil {
			t.logger.Printf("Recording error: %v", err)
		}
	}()

	// Start processing goroutine
	go func() {
		if err := t.processAudio(t.ctx, audioChan, transcriptionChan); err != nil {
			t.logger.Printf("Processing error: %v", err)
		}
	}()

	// Start transcription handling goroutine
	go func() {
		t.handleTranscriptions(t.ctx, transcriptionChan)
	}()

	return nil
}

// Stop ends the recording and transcription process
func (t *Transcriber) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isRunning {
		return nil
	}

	// Set isRunning to false first to prevent concurrent stops
	t.isRunning = false

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}

	if t.whisper != nil {
		t.whisper.Close()
		t.whisper = nil
	}

	if t.recorder != nil {
		if err := t.recorder.Close(); err != nil {
			return fmt.Errorf("failed to close recorder: %w", err)
		}
		t.recorder = nil
	}

	return nil
}

// processAudio handles the audio processing and transcription
func (t *Transcriber) processAudio(ctx context.Context, audioChan <-chan []float32, transcriptionChan chan<- string) error {
	// Set processing flag to true to prevent concurrent processing
	if !t.processing.CompareAndSwap(false, true) {
		return fmt.Errorf("audio processing already in progress")
	}

	// Ensure processing flag is reset when done
	defer t.processing.Store(false)

	for {
		select {
		case <-ctx.Done():
			t.logger.Printf("Context cancelled, stopping audio processing")
			return nil
		case samples := <-audioChan:
			if err := t.processor.ProcessSamples(samples); err != nil {
				if err == audio.ErrSilenceDetected {
					// Use a mutex to safely access the processor
					t.mu.Lock()
					audioData := t.processor.GetBuffer()
					if len(audioData) > 0 {
						if err := t.processBuffer(audioData, transcriptionChan); err != nil {
							t.mu.Unlock()
							t.logger.Printf("Failed to process buffer: %v", err)
							return fmt.Errorf("failed to process buffer: %w", err)
						}
					}
					t.processor.ClearBuffer()
					t.mu.Unlock()

					// Call Stop() outside the lock to avoid deadlock
					if err := t.Stop(); err != nil {
						t.logger.Printf("Error stopping transcriber: %v", err)
					}
					return nil
				}
				return fmt.Errorf("processing error: %w", err)
			}
		}
	}
}

func (t *Transcriber) processBuffer(buffer []float32, transcriptionChan chan<- string) error {
	if len(buffer) == 0 {
		return nil
	}

	// Create a copy of the buffer to avoid race conditions
	bufferCopy := make([]float32, len(buffer))
	copy(bufferCopy, buffer)

	// Use whisper to transcribe the audio
	text, err := t.whisper.Transcribe(bufferCopy)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	// Filter out special tokens
	filteredText := text
	for _, token := range tokensToFilter {
		filteredText = strings.ReplaceAll(filteredText, token, "")
	}

	filteredText = strings.TrimSpace(filteredText)

	if filteredText != "" {
		// Use a timeout to prevent blocking indefinitely
		select {
		case transcriptionChan <- filteredText:
		case <-time.After(time.Duration(t.cfg.Audio.SilenceDuration) * time.Second):
			t.logger.Printf("Warning: transcription channel full, dropping text")
		}
	}

	return nil
}

// handleTranscriptions manages transcription output and clipboard operations
func (t *Transcriber) handleTranscriptions(ctx context.Context, transcriptionChan <-chan string) {
	// Create a ticker to limit clipboard operations
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case text := <-transcriptionChan:
			if text != "" {
				if t.cfg.Debug.PrintTranscriptions {
					t.logger.Printf("Transcription: %s", text)
				}

				// Wait for the next tick to avoid overwhelming the clipboard
				<-ticker.C

				if t.cfg.Processing.AutoPaste {
					// Validate text before copying to clipboard
					if !isValidText(text) {
						t.logger.Printf("Invalid text detected, skipping clipboard operation")
						continue
					}

					// First copy to clipboard
					if err := t.clipboard.Copy(text); err != nil {
						t.logger.Printf("Failed to copy to clipboard: %v", err)
						continue
					}

					// Then simulate paste with a small delay to ensure clipboard is ready
					time.Sleep(50 * time.Millisecond)
					if err := t.clipboard.Paste(); err != nil {
						t.logger.Printf("Failed to paste from clipboard: %v", err)
					}
				}
			}
		}
	}
}

// isValidText checks if the text is safe to copy to clipboard
func isValidText(text string) bool {
	// Use the clipboard manager's validation logic
	cm := utils.NewClipboardManager(false)
	return cm.IsValidText(text)
}

// Close cleans up resources
func (t *Transcriber) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var errs []error

	// Cancel context if running
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}

	// Stop if running
	if t.isRunning {
		if err := t.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop error: %w", err))
		}
	}

	// Clean up recorder
	if t.recorder != nil {
		if err := t.recorder.Close(); err != nil {
			errs = append(errs, fmt.Errorf("recorder close error: %w", err))
		}
		t.recorder = nil
	}

	// Clean up whisper
	if t.whisper != nil {
		t.whisper.Close()
		t.whisper = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// IsRunning returns whether the transcriber is currently running
func (t *Transcriber) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isRunning
}
