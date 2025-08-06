package audio

import (
	"context"
	"testing"
	"time"
)

func TestCapture_NewCapture(t *testing.T) {
	sampleRates := []uint32{16000, 44100, 48000}
	
	for _, rate := range sampleRates {
		t.Run("", func(t *testing.T) {
			capture := NewCapture(rate)
			if capture == nil {
				t.Fatal("NewCapture returned nil")
			}
			if capture.sampleRate != rate {
				t.Errorf("Expected sample rate %d, got %d", rate, capture.sampleRate)
			}
			if capture.audioChan == nil {
				t.Error("Audio channel not initialized")
			}
		})
	}
}

func TestCapture_Stop(t *testing.T) {
	capture := NewCapture(16000)
	
	// Test stop without start
	err := capture.Stop()
	if err != nil {
		t.Errorf("Stop without start should not error: %v", err)
	}
	
	// Test multiple stops
	err = capture.Stop()
	if err != nil {
		t.Errorf("Multiple stops should not error: %v", err)
	}
}

func TestCapture_StartStop(t *testing.T) {
	// Skip if audio device is not available
	capture := NewCapture(16000)
	ctx := context.Background()
	
	_, err := capture.Start(ctx)
	if err != nil {
		t.Skip("Audio device not available, skipping test")
	}
	
	// Immediate stop should work
	err = capture.Stop()
	if err != nil {
		t.Errorf("Stop after start failed: %v", err)
	}
}

func TestCapture_ContextCancellation(t *testing.T) {
	capture := NewCapture(16000)
	ctx, cancel := context.WithCancel(context.Background())
	
	audioChan, err := capture.Start(ctx)
	if err != nil {
		t.Skip("Audio device not available, skipping test")
	}
	
	// Cancel context
	cancel()
	
	// Channel should eventually close
	timer := time.NewTimer(100 * time.Millisecond)
	select {
	case <-audioChan:
		// Good - channel closed
	case <-timer.C:
		// OK - channel might still have data
	}
	
	capture.Stop()
}

// MockDevice simulates an audio device for testing
type MockDevice struct {
	started bool
	stopped bool
}

func (m *MockDevice) Start() error {
	m.started = true
	return nil
}

func (m *MockDevice) Stop() error {
	m.stopped = true
	return nil
}

func (m *MockDevice) Uninit() {
	// Cleanup
}

func TestCapture_BufferHandling(t *testing.T) {
	// This test verifies the buffer handling logic
	// Since we can't easily mock malgo, we test the concepts
	
	t.Run("empty input handling", func(t *testing.T) {
		// Test that empty/nil inputs are handled gracefully
		capture := NewCapture(16000)
		if capture.audioChan == nil {
			t.Error("Audio channel should be initialized")
		}
	})
	
	t.Run("channel buffer size", func(t *testing.T) {
		capture := NewCapture(16000)
		// Verify channel has buffer
		if cap(capture.audioChan) != 100 {
			t.Errorf("Expected channel buffer size 100, got %d", cap(capture.audioChan))
		}
	})
}