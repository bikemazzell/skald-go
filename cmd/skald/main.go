package main

import (
	"context"
	"flag"
	"fmt"
	"log"
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

	// Create components
	audioCapture := audio.NewCapture(uint32(*sampleRate))
	
	whisperTranscriber, err := transcriber.NewWhisper(validatedModelPath, *language)
	if err != nil {
		log.Fatalf("Failed to create transcriber: %v", err)
	}
	defer whisperTranscriber.Close()

	clipboardOutput := output.NewClipboardOutput(os.Stdout, !*noClipboard)
	silenceDetector := audio.NewSilenceDetector()

	// Create app configuration
	config := app.Config{
		SampleRate:       uint32(*sampleRate),
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