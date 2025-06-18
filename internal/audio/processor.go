package audio

import (
	"context"
	"fmt"
	"log"
	"math"

	"skald/internal/config"
)

type Processor struct {
	cfg                      *config.Config
	logger                   *log.Logger
	silenceDuration          float32
	buffer                   *CircularBuffer
	consecutiveSilentSamples int
}

func NewProcessor(cfg *config.Config, logger *log.Logger) (*Processor, error) {
	bufferSize := cfg.Audio.SampleRate * cfg.Audio.FrameLength * cfg.Audio.BufferedFrames

	if cfg.Verbose {
		logger.Printf("Initializing audio processor with buffer size: %d samples (%.2f seconds)",
			bufferSize, float64(bufferSize)/float64(cfg.Audio.SampleRate))
	}

	return &Processor{
		cfg:                      cfg,
		logger:                   logger,
		silenceDuration:          0,
		buffer:                   NewCircularBuffer(bufferSize),
		consecutiveSilentSamples: 0,
	}, nil
}

func (p *Processor) Process(ctx context.Context, samples []float32, outChan chan<- []float32) error {
	if p.isSilent(samples) {
		p.silenceDuration += float32(len(samples)) / float32(p.cfg.Audio.SampleRate)
		if p.silenceDuration >= p.cfg.Audio.SilenceDuration {
			return ErrSilenceDetected
		}
	} else {
		p.silenceDuration = 0
	}

	_, err := p.buffer.Write(samples)
	if err != nil {
		return fmt.Errorf("buffer write error: %w", err)
	}

	frameSize := p.cfg.Audio.FrameLength * p.cfg.Audio.BufferedFrames
	if p.buffer.Available() >= frameSize {
		chunk := p.buffer.Read(frameSize)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case outChan <- chunk:
		}
	}

	return nil
}

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

	var isSilent bool
	if p.consecutiveSilentSamples > 0 {
		isSilent = rms < (threshold * 2.0)
	} else {
		isSilent = rms < threshold
	}

	return isSilent
}

func (p *Processor) Clear() {
	p.silenceDuration = 0
	p.consecutiveSilentSamples = 0
	p.buffer.Clear()
}

func (p *Processor) ProcessBuffer(buffer []float32) (int, error) {
	return p.buffer.Write(buffer)
}

func (p *Processor) ProcessSamples(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}

	isSilent := p.isSilent(samples)

	if isSilent {
		p.consecutiveSilentSamples++
		if p.consecutiveSilentSamples > 5 {
			p.silenceDuration += float32(len(samples)) / float32(p.cfg.Audio.SampleRate)
			if p.silenceDuration >= p.cfg.Audio.SilenceDuration {
				if p.cfg.Verbose {
					p.logger.Printf("Silence threshold reached: %.2f >= %.2f",
						p.silenceDuration, p.cfg.Audio.SilenceDuration)
				}
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

	_, err := p.buffer.Write(samples)
	if err != nil {
		return fmt.Errorf("buffer write error: %w", err)
	}

	return nil
}

func (p *Processor) GetBuffer() []float32 {
	available := p.buffer.Available()
	if available == 0 {
		return []float32{}
	}
	return p.buffer.Read(available)
}

func (p *Processor) ClearBuffer() {
	p.buffer.Clear()
	p.silenceDuration = 0
	p.consecutiveSilentSamples = 0
}
