package transcriber

import (
	"context"
	"fmt"
	"log"
	"math"
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

	// Initialize whisper model once at startup for persistence
	modelPath := modelMgr.GetModelPath()
	if cfg.Verbose {
		logger.Printf("Loading Whisper model at startup: %s", modelPath)
	}

	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("model file not accessible: %w", err)
	}

	whisperInstance, err := whisper.New(modelPath, whisper.Config{
		Language:           cfg.Whisper.Language,
		AutoDetectLanguage: cfg.Whisper.AutoDetectLanguage,
		SupportedLanguages: cfg.Whisper.SupportedLanguages,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize whisper model: %w", err)
	}

	// Log language configuration
	if cfg.Whisper.AutoDetectLanguage && whisperInstance.IsMultilingual() {
		if cfg.Verbose {
			logger.Printf("Language auto-detection enabled for multilingual model")
			supportedLangs := whisperInstance.GetSupportedLanguages()
			logger.Printf("Supported languages: %v", supportedLangs[:10])
		}
	} else {
		if cfg.Verbose {
			logger.Printf("Using fixed language: %s", cfg.Whisper.Language)
		}
	}

	if cfg.Verbose {
		logger.Printf("Whisper model loaded and ready for persistent use")
	}

	return &Transcriber{
		cfg:       cfg,
		logger:    logger,
		processor: processor,
		modelMgr:  modelMgr,
		recorder:  recorder,
		clipboard: clipboard,
		whisper:   whisperInstance,
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

	// Whisper model is already loaded and persistent - no need to reinitialize
	if t.cfg.Verbose {
		t.logger.Printf("Using persistent Whisper model for session")
	}

	// Create context with timeout if continuous mode has max session duration
	if t.cfg.Processing.ContinuousMode.Enabled && t.cfg.Processing.ContinuousMode.MaxSessionDuration > 0 {
		timeout := time.Duration(t.cfg.Processing.ContinuousMode.MaxSessionDuration) * time.Second
		t.ctx, t.cancel = context.WithTimeout(context.Background(), timeout)
		if t.cfg.Verbose {
			t.logger.Printf("Continuous mode enabled with %d second session limit", t.cfg.Processing.ContinuousMode.MaxSessionDuration)
		}
	} else {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}
	t.isRunning = true

	// Create channels for audio data
	audioChan := make(chan []float32, t.cfg.Processing.ChannelBufferSize)
	transcriptionChan := make(chan string, t.cfg.Processing.ChannelBufferSize)

	// Start recording goroutine
	go func() {
		if err := t.recorder.Start(t.ctx, audioChan); err != nil {
			if err != context.Canceled {
				t.logger.Printf("Recording error: %v", err)
			}
		}
	}()

	// Start processing goroutine
	go func() {
		if err := t.processAudio(t.ctx, audioChan, transcriptionChan); err != nil {
			if err != context.Canceled {
				t.logger.Printf("Processing error: %v", err)
			}
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

	// Don't close whisper - keep it persistent for reuse

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
			if t.cfg.Verbose {
				t.logger.Printf("Context cancelled, stopping audio processing")
			}
			return nil
		case samples := <-audioChan:
			if t.cfg.Verbose {
				// Calculate RMS for main processing loop debugging
				var sum float32
				for _, sample := range samples {
					sum += sample * sample
				}
				rms := math.Sqrt(float64(sum / float32(len(samples))))
				if rms > 0.001 { // Only log when there's significant audio
					t.logger.Printf("Main loop - Processing audio RMS: %.6f", rms)
				}
			}
			
			if err := t.processor.ProcessSamples(samples); err != nil {
				if err == audio.ErrSilenceDetected {
					if t.cfg.Verbose {
						t.logger.Printf("Silence detected in main processing loop")
					}
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
					t.mu.Unlock()

					if t.cfg.Processing.ContinuousMode.Enabled {
						if t.cfg.Verbose {
							t.logger.Printf("Silence detected, processed buffer, waiting for new speech in continuous mode")
						}
						
						// Enter waiting-for-speech state in continuous mode
						err := t.waitForNewSpeechInContinuousMode(ctx, audioChan, transcriptionChan)
						if err != nil {
							if t.cfg.Verbose {
								t.logger.Printf("Continuous mode wait ended: %v", err)
							}
							return err
						}
						// If we get here, new speech was detected and processed, continue the main loop
						continue
					} else {
						// Single-shot mode: stop after silence
						if err := t.Stop(); err != nil {
							t.logger.Printf("Error stopping transcriber: %v", err)
						}
						return nil
					}
				}
				return fmt.Errorf("processing error: %w", err)
			}
		}
	}
}

func (t *Transcriber) processBuffer(buffer []float32, transcriptionChan chan<- string) error {
	if len(buffer) == 0 {
		if t.cfg.Verbose {
			t.logger.Printf("processBuffer called with empty buffer")
		}
		return nil
	}

	if t.cfg.Verbose {
		t.logger.Printf("processBuffer called with %d samples", len(buffer))
	}

	// Create a copy of the buffer to avoid race conditions
	bufferCopy := make([]float32, len(buffer))
	copy(bufferCopy, buffer)

	if t.cfg.Verbose {
		t.logger.Printf("Starting whisper transcription...")
	}

	// Use whisper to transcribe the audio
	text, err := t.whisper.Transcribe(bufferCopy)
	if err != nil {
		if t.cfg.Verbose {
			t.logger.Printf("Whisper transcription failed: %v", err)
		}
		return fmt.Errorf("transcription failed: %w", err)
	}

	if t.cfg.Verbose {
		t.logger.Printf("Whisper transcription completed, raw text: '%s'", text)
	}

	// Log detected language if auto-detection is enabled
	if t.cfg.Whisper.AutoDetectLanguage && t.whisper.IsMultilingual() {
		detectedLang := t.whisper.GetDetectedLanguage()
		if t.cfg.Verbose && detectedLang != "" {
			t.logger.Printf("Detected language: %s", detectedLang)
		}
	}

	// Filter out special tokens
	filteredText := text
	if t.cfg.Verbose {
		t.logger.Printf("Before filtering: '%s'", text)
	}
	
	for _, token := range tokensToFilter {
		filteredText = strings.ReplaceAll(filteredText, token, "")
	}

	filteredText = strings.TrimSpace(filteredText)

	if t.cfg.Verbose {
		t.logger.Printf("After filtering and trimming: '%s'", filteredText)
	}

	if filteredText != "" {
		if t.cfg.Verbose {
			t.logger.Printf("Sending transcription to channel: '%s'", filteredText)
		}
		// Use a timeout to prevent blocking indefinitely
		select {
		case transcriptionChan <- filteredText:
			if t.cfg.Verbose {
				t.logger.Printf("Successfully sent transcription to channel")
			}
		case <-time.After(time.Duration(t.cfg.Audio.SilenceDuration) * time.Second):
			t.logger.Printf("Warning: transcription channel full, dropping text")
		}
	} else {
		if t.cfg.Verbose {
			t.logger.Printf("No text to transcribe after filtering (empty or filtered out)")
		}
	}

	return nil
}

// handleTranscriptions manages transcription output and clipboard operations
func (t *Transcriber) handleTranscriptions(ctx context.Context, transcriptionChan <-chan string) {
	// Create a ticker to limit clipboard operations
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Track if we need to start a new line
	firstTranscription := true

	for {
		select {
		case <-ctx.Done():
			// Print newline when done to move to next line
			if !firstTranscription && t.cfg.Debug.PrintTranscriptions {
				fmt.Println()
			}
			return
		case text := <-transcriptionChan:
			if text != "" {
				if t.cfg.Debug.PrintTranscriptions {
					if firstTranscription {
						// Start the transcription line without timestamp
						fmt.Print(" ")
						firstTranscription = false
					}
					// Append text to the same line with a space separator
					fmt.Printf("%s ", text)
					// Flush output to ensure it appears immediately
					os.Stdout.Sync()
				}

				// Wait for the next tick to avoid overwhelming the clipboard
				<-ticker.C

				if t.cfg.Processing.AutoPaste {
					// Validate text before copying to clipboard
					if !t.isValidText(text) {
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
					} else {
						// Play completion tone after successful clipboard operation
						// Check if recorder is still available (may be nil if Stop() was called)
						t.mu.Lock()
						recorder := t.recorder
						t.mu.Unlock()
						
						if recorder != nil {
							if err := recorder.PlayCompletionTone(); err != nil {
								t.logger.Printf("Warning: failed to play completion tone: %v", err)
							}
						}
					}
				}
			}
		}
	}
}

// waitForNewSpeechInContinuousMode waits for new speech after silence in continuous mode
func (t *Transcriber) waitForNewSpeechInContinuousMode(ctx context.Context, audioChan <-chan []float32, transcriptionChan chan<- string) error {
	// Reset processor state to start fresh for new speech
	t.mu.Lock()
	t.processor.ClearBuffer()
	t.mu.Unlock()
	
	if t.cfg.Verbose {
		t.logger.Printf("Waiting for new speech in continuous mode...")
	}
	
	// Wait for non-silent audio that indicates new speech
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case samples := <-audioChan:
			// Calculate audio level for debugging
			var sum float32
			for _, sample := range samples {
				sum += sample * sample
			}
			rms := math.Sqrt(float64(sum / float32(len(samples))))
			
			// Debug: Always log audio levels in verbose mode when waiting
			if t.cfg.Verbose {
				t.logger.Printf("Waiting for speech - Audio RMS: %.6f (threshold: %.6f)", 
					rms, t.cfg.Audio.SilenceThreshold)
			}
			
			// Check if this audio contains speech (non-silent)
			if t.isNonSilentAudio(samples) {
				if t.cfg.Verbose {
					t.logger.Printf("New speech detected (RMS: %.6f > %.6f), resuming processing", 
						rms, t.cfg.Audio.SilenceThreshold)
				}
				
				// Process this first non-silent sample
				t.mu.Lock()
				if err := t.processor.ProcessSamples(samples); err != nil {
					t.mu.Unlock()
					if err == audio.ErrSilenceDetected {
						// This shouldn't happen since we just detected non-silent audio,
						// but if it does, continue waiting
						if t.cfg.Verbose {
							t.logger.Printf("Unexpected silence detection on non-silent audio, continuing to wait")
						}
						continue
					}
					return fmt.Errorf("error processing new speech: %w", err)
				}
				t.mu.Unlock()
				
				// Successfully started processing new speech, return to main loop
				return nil
			}
			// Sample is still silent, continue waiting
		}
	}
}

// isNonSilentAudio checks if audio samples contain non-silent content
func (t *Transcriber) isNonSilentAudio(samples []float32) bool {
	if len(samples) == 0 {
		return false
	}
	
	// Calculate RMS (same logic as processor uses)
	var sum float32
	for _, sample := range samples {
		sum += sample * sample
	}
	rms := math.Sqrt(float64(sum / float32(len(samples))))
	
	// Use same threshold as configured for silence detection
	return rms > float64(t.cfg.Audio.SilenceThreshold)
}

// isValidText checks if the text is safe to copy to clipboard using config settings
func (t *Transcriber) isValidText(text string) bool {
	// Use configuration-aware validation
	cm := utils.NewClipboardManager(false)
	return cm.IsValidTextWithMode(
		text, 
		t.cfg.Processing.TextValidation.Mode,
		t.cfg.Processing.TextValidation.AllowPunctuation,
		t.cfg.Processing.TextValidation.CustomBlocklist,
	)
}

// Close cleans up resources
func (t *Transcriber) Close() error {
	var errs []error

	// Stop if running (Stop handles its own locking)
	// Use a timeout to prevent hanging
	done := make(chan error, 1)
	go func() {
		done <- t.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			errs = append(errs, fmt.Errorf("stop error: %w", err))
		}
	case <-time.After(5 * time.Second):
		t.logger.Printf("Warning: Stop() timed out, continuing with cleanup...")
		errs = append(errs, fmt.Errorf("stop timeout"))
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Recorder cleanup is already done in Stop

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
