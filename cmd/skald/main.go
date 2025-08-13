package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"

	"skald/internal/validation"
	"skald/pkg/skald/app"
	"skald/pkg/skald/audio"
	"skald/pkg/skald/output"
	"skald/pkg/skald/transcriber"
)

const (
	defaultSampleRate       = 16000
	defaultSilenceThreshold = 0.01
	defaultSilenceDuration  = 1.5
	defaultModelPath        = "models/ggml-large-v3-turbo.bin"
)

// Version will be set at build time
var version = "dev"

// validateSampleRate ensures the sample rate is within safe bounds
func validateSampleRate(rate int) error {
	if rate < 8000 {
		return fmt.Errorf("sample rate too low: %d (minimum: 8000)", rate)
	}
	if rate > 192000 {
		return fmt.Errorf("sample rate too high: %d (maximum: 192000)", rate)
	}
	if rate < 0 || rate > math.MaxUint32 {
		return fmt.Errorf("sample rate out of uint32 range: %d", rate)
	}
	return nil
}


func main() {
	var (
		modelPath  = flag.String("model", defaultModelPath, "Path to whisper model")
		language   = flag.String("language", "auto", "Language code (e.g., en, es, auto)")
		continuous = flag.Bool("continuous", false, "Continuous transcription mode")
		sampleRate = flag.Int("sample-rate", defaultSampleRate, "Audio sample rate")
		silenceThreshold = flag.Float64("silence-threshold", defaultSilenceThreshold, "Silence threshold (0-1)")
		silenceDuration = flag.Float64("silence-duration", defaultSilenceDuration, "Silence duration in seconds")
		noClipboard = flag.Bool("no-clipboard", false, "Disable clipboard output")
		showVersion = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("skald version %s\n", version)
		return
	}

	// Validate and secure model path
	validatedModelPath, err := validation.ValidateModelPath(*modelPath)
	if err != nil {
		log.Fatalf("Invalid model path: %v", err)
	}

	// Validate sample rate before use
	if err := validateSampleRate(*sampleRate); err != nil {
		log.Fatalf("Invalid sample rate: %v", err)
	}

	// Create components with validated sample rate
	// Note: Safe conversion after validation - sampleRate already checked to be within uint32 range
	safeRate := uint32(*sampleRate) //nolint:gosec
	audioCapture := audio.NewCapture(safeRate)
	
	whisperTranscriber, err := transcriber.NewWhisper(validatedModelPath, *language)
	if err != nil {
		log.Fatalf("Failed to create transcriber: %v", err)
	}
	defer whisperTranscriber.Close()

	clipboardOutput := output.NewClipboardOutput(os.Stdout, !*noClipboard)
	silenceDetector := audio.NewSilenceDetector()

	// Create app configuration
	config := app.Config{
		SampleRate:       safeRate,
		SilenceThreshold: float32(*silenceThreshold),
		SilenceDuration:  float32(*silenceDuration),
		Continuous:       *continuous,
	}

	// Create and run app
	application := app.New(audioCapture, whisperTranscriber, clipboardOutput, silenceDetector, config)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nStopping...")
		cancel()
	}()

	// Run the app
	if err := application.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Error: %v", err)
	}
}