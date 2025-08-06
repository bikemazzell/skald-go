package audio

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Capture implements audio capture using malgo
type Capture struct {
	device     *malgo.Device
	malgoCtx   *malgo.AllocatedContext
	sampleRate uint32
	audioChan  chan []float32
	closed     bool
}

// NewCapture creates a new audio capture instance
func NewCapture(sampleRate uint32) *Capture {
	return &Capture{
		sampleRate: sampleRate,
		audioChan:  make(chan []float32, 100),
	}
}

// Start begins audio capture
func (a *Capture) Start(ctx context.Context) (<-chan []float32, error) {
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = a.sampleRate
	deviceConfig.Alsa.NoMMap = 1

	onRecvFrames := func(pOutput, pInput []byte, framecount uint32) {
		if framecount == 0 || len(pInput) == 0 {
			return
		}
		
		samples := make([]float32, framecount)
		copy(samples, (*[1 << 30]float32)(unsafe.Pointer(&pInput[0]))[:framecount])
		
		select {
		case a.audioChan <- samples:
		case <-ctx.Done():
			return
		default:
			// Drop frames if channel is full
		}
	}

	malgoCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init malgo context: %w", err)
	}
	a.malgoCtx = malgoCtx

	device, err := malgo.InitDevice(malgoCtx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		malgoCtx.Uninit()
		return nil, fmt.Errorf("failed to init capture device: %w", err)
	}

	a.device = device

	if err := device.Start(); err != nil {
		device.Uninit()
		malgoCtx.Uninit()
		return nil, fmt.Errorf("failed to start device: %w", err)
	}

	return a.audioChan, nil
}

// Stop stops audio capture
func (a *Capture) Stop() error {
	if a.device != nil {
		a.device.Uninit()
		a.device = nil
	}
	if a.malgoCtx != nil {
		a.malgoCtx.Uninit()
		a.malgoCtx = nil
	}
	// Only close channel once
	if !a.closed {
		close(a.audioChan)
		a.closed = true
	}
	return nil
}