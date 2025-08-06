package skald

import "context"

// AudioCapture interface for audio input
type AudioCapture interface {
	Start(ctx context.Context) (<-chan []float32, error)
	Stop() error
}

// Transcriber interface for speech-to-text
type Transcriber interface {
	Transcribe(audio []float32) (string, error)
	Close() error
}

// Output interface for text output
type Output interface {
	Write(text string) error
}

// SilenceDetector interface for detecting silence in audio
type SilenceDetector interface {
	IsSilent(samples []float32, threshold float32) bool
}