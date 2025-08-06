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
		session := &TranscriptionSession{
			buffer:          make([]float32, 0),
			silentSamples:   0,
			silentThreshold: int(float32(app.config.SampleRate) * app.config.SilenceDuration),
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
}

// processSession processes a single transcription session
func (app *App) processSession(ctx context.Context, audioChan <-chan []float32, session *TranscriptionSession) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case samples, ok := <-audioChan:
			if !ok {
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

			// Process buffer when silence detected
			if session.silentSamples >= session.silentThreshold && len(session.buffer) > 0 {
				if err := app.transcribeAndOutput(session.buffer); err != nil {
					log.Printf("Transcription error: %v", err)
				}
				return nil
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