package transcriber

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"skald/internal/audio"
	"skald/internal/config"
	"skald/internal/model"
	"skald/internal/whisper"
	"skald/pkg/utils"
)

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
func New(cfg *config.Config, logger *log.Logger) (*Transcriber, error) {
	// Create audio processor
	processor, err := audio.NewProcessor(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio processor: %w", err)
	}

	// Create model manager
	modelMgr := model.New(cfg, logger)

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

	// Initialize model
	if err := t.modelMgr.Initialize(t.cfg.Whisper.Model); err != nil {
		return fmt.Errorf("failed to initialize model: %w", err)
	}

	// Create new recorder instance
	var err error
	t.recorder, err = audio.NewRecorder(t.cfg, t.logger)
	if err != nil {
		return fmt.Errorf("failed to create recorder: %w", err)
	}

	// Initialize whisper
	whisperInstance, err := whisper.New(t.modelMgr.GetModelPath(), whisper.Config{
		Language: t.cfg.Whisper.Language,
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
	t.logger.Printf("Starting audio processing...")
	for {
		select {
		case <-ctx.Done():
			t.logger.Printf("Context cancelled, stopping audio processing")
			return nil
		case samples := <-audioChan:
			t.logger.Printf("Received audio samples: %d", len(samples))
			if err := t.processor.ProcessSamples(samples); err != nil {
				if err == audio.ErrSilenceDetected {
					t.logger.Printf("Silence detected, getting buffer for transcription")
					audioData := t.processor.GetBuffer()
					t.logger.Printf("Got buffer of size: %d samples", len(audioData))

					if len(audioData) > 0 {
						t.logger.Printf("Processing buffer for transcription")
						if err := t.processBuffer(audioData, transcriptionChan); err != nil {
							t.logger.Printf("Failed to process buffer: %v", err)
							return fmt.Errorf("failed to process buffer: %w", err)
						}
					}

					t.logger.Printf("Clearing buffer and stopping recording")
					t.processor.ClearBuffer()

					if err := t.Stop(); err != nil {
						t.logger.Printf("Error stopping transcriber: %v", err)
					}
					return nil
				}
				t.logger.Printf("Processing error: %v", err)
				return fmt.Errorf("processing error: %w", err)
			}
		}
	}
}

func (t *Transcriber) processBuffer(buffer []float32, transcriptionChan chan<- string) error {
	if len(buffer) == 0 {
		return nil
	}

	// Transcribe the audio buffer
	text, err := t.whisper.Transcribe(buffer)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	if text != "" {
		select {
		case transcriptionChan <- text:
			t.logger.Printf("Transcribed: %s", text)
		default:
			t.logger.Printf("Warning: transcription channel full, dropping text")
		}
	}

	return nil
}

// handleTranscriptions manages transcription output and clipboard operations
func (t *Transcriber) handleTranscriptions(ctx context.Context, transcriptionChan <-chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case text := <-transcriptionChan:
			if text != "" {
				if t.cfg.Debug.PrintTranscriptions {
					t.logger.Printf("Transcription: %s", text)
				}
				if t.cfg.Processing.AutoPaste {
					// First copy to clipboard
					if err := t.clipboard.Copy(text); err != nil {
						t.logger.Printf("Failed to copy to clipboard: %v", err)
						continue
					}

					// Then simulate paste
					if err := t.clipboard.Paste(); err != nil {
						t.logger.Printf("Failed to paste from clipboard: %v", err)
					}
				}
			}
		}
	}
}

// Close cleans up resources
func (t *Transcriber) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var errs []error

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
