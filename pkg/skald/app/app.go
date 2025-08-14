package app

import (
	"context"
	"fmt"
	"log"

	"skald/pkg/skald"
)

// Config holds application configuration
type Config struct {
	SampleRate       uint32
	SilenceThreshold float32
	SilenceDuration  float32
	Continuous       bool
}

// App represents the main application
type App struct {
	audio           skald.AudioCapture
	transcriber     skald.Transcriber
	output          skald.Output
	silenceDetector skald.SilenceDetector
	config          Config
}

// New creates a new application instance
func New(audio skald.AudioCapture, transcriber skald.Transcriber, output skald.Output, silenceDetector skald.SilenceDetector, config Config) *App {
	return &App{
		audio:           audio,
		transcriber:     transcriber,
		output:          output,
		silenceDetector: silenceDetector,
		config:          config,
	}
}

// Run starts the transcription process
func (app *App) Run(ctx context.Context) error {
	audioChan, err := app.audio.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start audio capture: %w", err)
	}
	defer app.audio.Stop()

	log.Println("Listening... Press Ctrl+C to stop")

	for {
		// Create session with 25-second max to stay safely under Whisper's 30s limit
		maxDurationSeconds := float32(25.0)
		session := &TranscriptionSession{
			buffer:          make([]float32, 0),
			silentSamples:   0,
			silentThreshold: int(float32(app.config.SampleRate) * app.config.SilenceDuration),
			maxSamples:      int(float32(app.config.SampleRate) * maxDurationSeconds),
		}

		if err := app.processSession(ctx, audioChan, session); err != nil {
			return err
		}

		if !app.config.Continuous {
			return nil
		}
	}
}

// TranscriptionSession holds state for a single transcription session
type TranscriptionSession struct {
	buffer          []float32
	silentSamples   int
	silentThreshold int
	maxSamples      int // Maximum samples before forced transcription (30s limit)
}

// processSession processes a single transcription session with automatic chunking
func (app *App) processSession(ctx context.Context, audioChan <-chan []float32, session *TranscriptionSession) error {
	for {
		select {
		case <-ctx.Done():
			// Process any remaining audio before exiting
			if len(session.buffer) > 0 {
				if err := app.transcribeAndOutput(session.buffer); err != nil {
					log.Printf("Final transcription error: %v", err)
				}
			}
			return ctx.Err()
		case samples, ok := <-audioChan:
			if !ok {
				// Channel closed, process any remaining audio
				if len(session.buffer) > 0 {
					if err := app.transcribeAndOutput(session.buffer); err != nil {
						log.Printf("Final transcription error: %v", err)
					}
				}
				return nil
			}

			// Append to buffer
			session.buffer = append(session.buffer, samples...)

			// Check for silence
			isSilent := app.silenceDetector.IsSilent(samples, app.config.SilenceThreshold)

			if isSilent {
				session.silentSamples += len(samples)
			} else {
				session.silentSamples = 0
			}

			// Determine if we should process the buffer
			shouldProcess := false
			resetBuffer := false

			// Condition 1: Silence detected (original behavior)
			if session.silentSamples >= session.silentThreshold && len(session.buffer) > 0 {
				shouldProcess = true
				resetBuffer = true
			}

			// Condition 2: Buffer reached max duration (25 seconds)
			// This prevents sending >30s to Whisper which degrades quality
			if len(session.buffer) >= session.maxSamples {
				shouldProcess = true
				resetBuffer = true
			}

			if shouldProcess {
				if err := app.transcribeAndOutput(session.buffer); err != nil {
					log.Printf("Transcription error: %v", err)
				}
				
				if resetBuffer {
					// Reset buffer and silence counter
					session.buffer = make([]float32, 0)
					session.silentSamples = 0
				}

				// Exit if not in continuous mode and silence was detected
				if !app.config.Continuous && session.silentSamples >= session.silentThreshold {
					return nil
				}
			}
		}
	}
}

// transcribeAndOutput transcribes audio and outputs the result
func (app *App) transcribeAndOutput(buffer []float32) error {
	text, err := app.transcriber.Transcribe(buffer)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	if text != "" {
		if err := app.output.Write(text); err != nil {
			return fmt.Errorf("output failed: %w", err)
		}
	}

	return nil
}