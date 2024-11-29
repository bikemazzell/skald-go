package audio

import (
	"context"
	"fmt"
	"log"
	"skald/internal/config"

	pvrecorder "github.com/Picovoice/pvrecorder/binding/go"
)

type Recorder struct {
	cfg      *config.Config
	recorder *pvrecorder.PvRecorder
	running  bool
	logger   *log.Logger
}

func NewRecorder(cfg *config.Config, logger *log.Logger) (*Recorder, error) {
	recorder := &pvrecorder.PvRecorder{
		DeviceIndex:         cfg.Audio.DeviceIndex,
		FrameLength:         cfg.Audio.FrameLength,
		BufferedFramesCount: cfg.Audio.BufferedFrames,
	}

	if err := recorder.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize recorder: %w", err)
	}

	return &Recorder{
		cfg:      cfg,
		recorder: recorder,
		logger:   logger,
	}, nil
}

func (r *Recorder) playStartTone() error {
	if !r.cfg.Audio.StartTone.Enabled {
		return nil
	}

	r.logger.Printf("Start tone disabled: audio output not implemented yet")
	// TODO: Implement audio output using PortAudio or similar
	// For now, we'll just log that we would have played a tone
	return nil
}

func (r *Recorder) Start(ctx context.Context, samples chan<- []float32) error {
	if r.running {
		return fmt.Errorf("recorder already running")
	}

	if err := r.playStartTone(); err != nil {
		r.logger.Printf("Warning: failed to play start tone: %v", err)
	}

	if err := r.recorder.Start(); err != nil {
		return fmt.Errorf("failed to start recorder: %w", err)
	}

	r.running = true

	go func() {
		defer close(samples)
		defer r.recorder.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				frame, err := r.recorder.Read()
				if err != nil {
					fmt.Printf("Error reading audio: %v\n", err)
					return
				}

				// Convert int16 samples to float32
				floatSamples := make([]float32, len(frame))
				for i, sample := range frame {
					floatSamples[i] = float32(sample) / 32768.0 // Normalize to [-1.0, 1.0]
				}

				samples <- floatSamples
			}
		}
	}()

	return nil
}

func (r *Recorder) Close() error {
	if r == nil || r.recorder == nil {
		return nil
	}

	r.recorder.Stop()
	r.recorder.Delete()
	r.recorder = nil
	return nil
}
