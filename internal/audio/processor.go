package audio

import (
	"context"
	"fmt"
	"log"
	"math"

	"skald/internal/config"
)

// Processor handles audio processing and silence detection
type Processor struct {
	cfg                      *config.Config
	logger                   *log.Logger
	silenceDuration          float32
	buffer                   *CircularBuffer
	consecutiveSilentSamples int
}

// NewProcessor creates a new audio processor
func NewProcessor(cfg *config.Config, logger *log.Logger) (*Processor, error) {
	// Calculate buffer size using frame settings from config
	bufferSize := cfg.Audio.SampleRate * cfg.Audio.FrameLength * cfg.Audio.BufferedFrames

	logger.Printf("Initializing audio processor with buffer size: %d samples (%.2f seconds)",
		bufferSize, float64(bufferSize)/float64(cfg.Audio.SampleRate))

	return &Processor{
		cfg:                      cfg,
		logger:                   logger,
		silenceDuration:          0,
		buffer:                   NewCircularBuffer(bufferSize),
		consecutiveSilentSamples: 0,
	}, nil
}

// Process handles incoming audio data
func (p *Processor) Process(ctx context.Context, samples []float32, outChan chan<- []float32) error {
	// Check for silence
	if p.isSilent(samples) {
		p.silenceDuration += float32(len(samples)) / float32(p.cfg.Audio.SampleRate)
		if p.silenceDuration >= p.cfg.Audio.SilenceDuration {
			return ErrSilenceDetected
		}
	} else {
		p.silenceDuration = 0
	}

	// Write to buffer
	_, err := p.buffer.Write(samples)
	if err != nil {
		return fmt.Errorf("buffer write error: %w", err)
	}

	// Check if we have enough data for a chunk
	frameSize := p.cfg.Audio.FrameLength * p.cfg.Audio.BufferedFrames
	if p.buffer.Available() >= frameSize {
		chunk := p.buffer.Read(frameSize)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case outChan <- chunk:
			// Chunk sent successfully
		}
	}

	return nil
}

// isSilent determines if an audio segment is silent
func (p *Processor) isSilent(samples []float32) bool {
	if len(samples) == 0 {
		return true
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}

	rms := math.Sqrt(sum / float64(len(samples)))
	threshold := float64(p.cfg.Audio.SilenceThreshold)

	// Add hysteresis: use a higher threshold to exit silence
	// and a lower threshold to enter silence
	var isSilent bool
	if p.consecutiveSilentSamples > 0 {
		// We're in silence, need more volume to break it
		isSilent = rms < (threshold * 2.0) // Need double the threshold to break silence
	} else {
		// We're not in silence, use normal threshold to enter it
		isSilent = rms < threshold
	}

	if p.cfg.Debug.PrintStatus {
		p.logger.Printf("RMS: %.6f, Threshold: %.6f, IsSilent: %v", rms, threshold, isSilent)
	}

	return isSilent
}

// Clear resets the processor state
func (p *Processor) Clear() {
	p.silenceDuration = 0
	p.consecutiveSilentSamples = 0
	p.buffer.Clear()
}

// ProcessBuffer writes a buffer of audio data to the processor's buffer
func (p *Processor) ProcessBuffer(buffer []float32) (int, error) {
	return p.buffer.Write(buffer)
}

// ProcessSamples processes a chunk of audio samples
func (p *Processor) ProcessSamples(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}

	// Log buffer status
	p.logger.Printf("Buffer status before write: %d/%d samples used",
		p.buffer.Available(), p.buffer.size)

	isSilent := p.isSilent(samples)
	p.logger.Printf("Processing samples: len=%d, isSilent=%v, silenceDuration=%.2f, consecutiveSilent=%d",
		len(samples), isSilent, p.silenceDuration, p.consecutiveSilentSamples)

	if isSilent {
		p.consecutiveSilentSamples++
		if p.consecutiveSilentSamples > 5 {
			p.silenceDuration += float32(len(samples)) / float32(p.cfg.Audio.SampleRate)
			if p.silenceDuration >= p.cfg.Audio.SilenceDuration {
				p.logger.Printf("Silence threshold reached: %.2f >= %.2f",
					p.silenceDuration, p.cfg.Audio.SilenceDuration)
				// Write the current samples before returning the error
				_, err := p.buffer.Write(samples)
				if err != nil {
					return fmt.Errorf("buffer write error: %w", err)
				}
				return ErrSilenceDetected
			}
		}
	} else {
		p.silenceDuration = 0
		p.consecutiveSilentSamples = 0
	}

	// Write to buffer
	_, err := p.buffer.Write(samples)
	if err != nil {
		return fmt.Errorf("buffer write error: %w", err)
	}

	// Log buffer status after write
	p.logger.Printf("Buffer status after write: %d/%d samples used",
		p.buffer.Available(), p.buffer.size)

	return nil
}

// GetBuffer returns the current audio buffer
func (p *Processor) GetBuffer() []float32 {
	available := p.buffer.Available()
	if available == 0 {
		return []float32{}
	}
	return p.buffer.Read(available)
}

// ClearBuffer clears the audio buffer
func (p *Processor) ClearBuffer() {
	p.buffer.Clear()
	p.silenceDuration = 0
	p.consecutiveSilentSamples = 0
}
