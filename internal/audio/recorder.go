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

type ToneConfig struct {
	Enabled   bool
	Frequency int
	Duration  int
	FadeMs    int
}

func (r *Recorder) playStartTone() error {
	toneConfig := ToneConfig{
		Enabled:   r.cfg.Audio.StartTone.Enabled,
		Frequency: r.cfg.Audio.StartTone.Frequency,
		Duration:  r.cfg.Audio.StartTone.Duration,
		FadeMs:    r.cfg.Audio.StartTone.FadeMs,
	}
	return r.playTone(toneConfig)
}

func (r *Recorder) PlayCompletionTone() error {
	toneConfig := ToneConfig{
		Enabled:   r.cfg.Audio.CompletionTone.Enabled,
		Frequency: r.cfg.Audio.CompletionTone.Frequency,
		Duration:  r.cfg.Audio.CompletionTone.Duration,
		FadeMs:    r.cfg.Audio.CompletionTone.FadeMs,
	}
	return r.playTone(toneConfig)
}

func (r *Recorder) PlayErrorTone() error {
	toneConfig := ToneConfig{
		Enabled:   r.cfg.Audio.ErrorTone.Enabled,
		Frequency: r.cfg.Audio.ErrorTone.Frequency,
		Duration:  r.cfg.Audio.ErrorTone.Duration,
		FadeMs:    r.cfg.Audio.ErrorTone.FadeMs,
	}
	return r.playTone(toneConfig)
}

func (r *Recorder) playTone(toneConfig ToneConfig) error {
	if !toneConfig.Enabled {
		return nil
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = malgo.FormatF32
	deviceConfig.Playback.Channels = 1
	deviceConfig.SampleRate = uint32(r.cfg.Audio.SampleRate)
	deviceConfig.Alsa.NoMMap = 1

	totalSamples := (r.cfg.Audio.SampleRate * toneConfig.Duration) / 1000

	doneChan := make(chan struct{})
	var sampleCount uint32 = 0
	var err error
	r.playback, err = malgo.InitDevice(r.context.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, framecount uint32) {
			freq := float32(toneConfig.Frequency)
			duration := float32(toneConfig.Duration) / 1000.0
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
					fadeTime := float32(toneConfig.FadeMs) / 1000.0
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

	if err := r.playback.Stop(); err != nil {
		return fmt.Errorf("failed to stop playback: %w", err)
	}
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

	if r.cfg.Verbose {
		if r.cfg.Audio.DeviceIndex >= 0 {
			r.logger.Printf("Note: Specific device selection not yet implemented, using default device")
		} else {
			r.logger.Printf("Using default audio device")
		}
	}

	var err error
	r.device, err = malgo.InitDevice(r.context.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, framecount uint32) {
			floatSamples := make([]float32, framecount)
			for i := range floatSamples {
				bits := uint32(inputSamples[i*4]) |
					uint32(inputSamples[i*4+1])<<8 |
					uint32(inputSamples[i*4+2])<<16 |
					uint32(inputSamples[i*4+3])<<24
				floatSamples[i] = math.Float32frombits(bits)
			}

			if r.cfg.Verbose {
				var sum float32
				for _, sample := range floatSamples {
					sum += sample * sample
				}
				rms := math.Sqrt(float64(sum / float32(len(floatSamples))))
				if rms > 0.001 {
					r.logger.Printf("Audio input RMS: %.6f (samples: %d)", rms, len(floatSamples))
				}
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
		_ = r.context.Uninit() 
		r.context = nil
	}

	r.running = false
	return nil
}

func sinf(x float32) float32 {
	return float32(math.Sin(float64(x)))
}
