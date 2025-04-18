package audio

import (
	"context"
	"fmt"
	"log"
	"math"
	"skald/internal/config"

	"github.com/gen2brain/malgo"
)

type Recorder struct {
	cfg      *config.Config
	context  *malgo.AllocatedContext
	device   *malgo.Device
	running  bool
	logger   *log.Logger
	playback *malgo.Device
}

func NewRecorder(cfg *config.Config, logger *log.Logger) (*Recorder, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize context: %w", err)
	}

	return &Recorder{
		cfg:     cfg,
		context: ctx,
		logger:  logger,
	}, nil
}

func (r *Recorder) playStartTone() error {
	if !r.cfg.Audio.StartTone.Enabled {
		return nil
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = uint32(r.cfg.Audio.SampleRate)
	deviceConfig.Alsa.NoMMap = 1

	// Calculate total samples needed for the duration
	totalSamples := (r.cfg.Audio.SampleRate * r.cfg.Audio.StartTone.Duration) / 1000

	doneChan := make(chan struct{})
	var sampleCount uint32 = 0
	var err error
	r.playback, err = malgo.InitDevice(r.context.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, framecount uint32) {
			freq := float32(r.cfg.Audio.StartTone.Frequency)
			duration := float32(r.cfg.Audio.StartTone.Duration) / 1000.0
			samples := make([]float32, framecount)

			for i := range samples {
				if sampleCount >= uint32(totalSamples) {
					samples[i] = 0
					if i == 0 {
						close(doneChan)
					}
					continue
				}

				t := float32(sampleCount) / float32(r.cfg.Audio.SampleRate)
				if t > duration {
					samples[i] = 0
				} else {
					fadeTime := float32(r.cfg.Audio.StartTone.FadeMs) / 1000.0
					var amp float32 = 1.0
					if t < fadeTime {
						amp = t / fadeTime
					} else if t > (duration - fadeTime) {
						amp = (duration - t) / fadeTime
					}
					samples[i] = amp * 0.5 * sinf(2*3.14159*freq*t)
				}
				sampleCount++
			}

			bytes := make([]byte, len(samples)*4)
			for i, sample := range samples {
				bits := math.Float32bits(sample)
				bytes[i*4] = byte(bits)
				bytes[i*4+1] = byte(bits >> 8)
				bytes[i*4+2] = byte(bits >> 16)
				bytes[i*4+3] = byte(bits >> 24)
			}

			copy(outputSamples, bytes)
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize playback device: %w", err)
	}

	if err := r.playback.Start(); err != nil {
		return fmt.Errorf("failed to start playback: %w", err)
	}

	<-doneChan

	r.playback.Stop()
	r.playback.Uninit()
	r.playback = nil

	return nil
}

func (r *Recorder) Start(ctx context.Context, samples chan<- []float32) error {
	if r.running {
		return fmt.Errorf("recorder already running")
	}

	if err := r.playStartTone(); err != nil {
		r.logger.Printf("Warning: failed to play start tone: %v", err)
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = uint32(r.cfg.Audio.SampleRate)
	deviceConfig.Alsa.NoMMap = 1

	var err error
	r.device, err = malgo.InitDevice(r.context.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, framecount uint32) {
			// Convert input bytes to float32 samples
			floatSamples := make([]float32, framecount)
			for i := range floatSamples {
				// Convert 4 bytes to float32 (little-endian)
				bits := uint32(inputSamples[i*4]) |
					uint32(inputSamples[i*4+1])<<8 |
					uint32(inputSamples[i*4+2])<<16 |
					uint32(inputSamples[i*4+3])<<24
				floatSamples[i] = math.Float32frombits(bits)
			}

			select {
			case <-ctx.Done():
				return
			case samples <- floatSamples:
			}
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initialize device: %w", err)
	}

	if err := r.device.Start(); err != nil {
		return fmt.Errorf("failed to start device: %w", err)
	}

	r.running = true
	return nil
}

func (r *Recorder) Close() error {
	if r.device != nil {
		r.device.Uninit()
		r.device = nil
	}

	if r.playback != nil {
		r.playback.Uninit()
		r.playback = nil
	}

	if r.context != nil {
		r.context.Uninit()
		r.context = nil
	}

	r.running = false
	return nil
}

func sinf(x float32) float32 {
	return float32(math.Sin(float64(x)))
}
