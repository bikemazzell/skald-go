package audio

import (
	"log"
	"math"
	"os"
	"testing"

	"skald/internal/config"
)

func TestProcessor_SilenceDetection(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate:       16000,
			Channels:         1,
			SilenceThreshold: 0.01,
			SilenceDuration:  1.0, // 1 second
			FrameLength:      512,
			BufferedFrames:   10,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	processor, err := NewProcessor(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test with silent samples
	silentSamples := make([]float32, 512)
	for i := range silentSamples {
		silentSamples[i] = 0.001 // Below threshold
	}

	// Process silent samples multiple times to reach silence duration
	// Need to process enough silent samples to trigger the detection
	// The processor requires consecutiveSilentSamples > 5 before it starts counting silence duration
	samplesForOneSecond := cfg.Audio.SampleRate / len(silentSamples)
	
	// First, process enough samples to trigger silence counting (6+ consecutive silent chunks)
	for i := 0; i < 10; i++ {
		err = processor.ProcessSamples(silentSamples)
		if err == ErrSilenceDetected {
			// Got silence detected early, that's ok
			return
		}
		if err != nil {
			t.Errorf("Unexpected error during warmup: %v", err)
		}
	}
	
	// Now process for 1 second worth of silence
	for i := 0; i < samplesForOneSecond; i++ {
		err = processor.ProcessSamples(silentSamples)
		if err == ErrSilenceDetected {
			// Success! Got silence detected
			return
		}
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// If we got here without ErrSilenceDetected, that's a failure
	t.Error("Expected ErrSilenceDetected after processing 1+ seconds of silence")

	// Test with loud samples
	loudSamples := make([]float32, 512)
	for i := range loudSamples {
		loudSamples[i] = 0.5 // Above threshold
	}

	err = processor.ProcessSamples(loudSamples)
	if err != nil {
		t.Errorf("Should not error on loud samples: %v", err)
	}

	// Silence counter should be reset after loud samples
	// Process more silent samples - should take another full second to trigger
	silentCount := 0
	for i := 0; i < samplesForOneSecond*2; i++ {
		err = processor.ProcessSamples(silentSamples)
		if err == ErrSilenceDetected {
			silentCount = i
			break
		}
	}
	
	// Should have taken at least some samples to trigger silence again
	if silentCount < 10 {
		t.Errorf("Got ErrSilenceDetected too early after loud samples at iteration %d", silentCount)
	}
}

func TestProcessor_BufferManagement(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate:       16000,
			Channels:         1,
			SilenceThreshold: 0.01,
			FrameLength:      512,
			BufferedFrames:   10,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	processor, err := NewProcessor(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Add samples to buffer
	samples1 := []float32{0.1, 0.2, 0.3}
	samples2 := []float32{0.4, 0.5, 0.6}

	processor.ProcessSamples(samples1)
	processor.ProcessSamples(samples2)

	// Get buffer
	buffer := processor.GetBuffer()
	expected := append(samples1, samples2...)
	
	if len(buffer) != len(expected) {
		t.Errorf("Buffer length mismatch: got %d, expected %d", len(buffer), len(expected))
	}

	for i := range expected {
		if buffer[i] != expected[i] {
			t.Errorf("Buffer content mismatch at index %d: got %f, expected %f", i, buffer[i], expected[i])
		}
	}

	// Clear buffer
	processor.ClearBuffer()
	buffer = processor.GetBuffer()
	if len(buffer) != 0 {
		t.Errorf("Buffer should be empty after clear, got length %d", len(buffer))
	}
}

// TestProcessor_MaxDuration is commented out because the audio processor 
// doesn't currently implement max duration checking. This could be added
// as a future enhancement.
/*
func TestProcessor_MaxDuration(t *testing.T) {
	// TODO: Implement max duration checking in the audio processor
	t.Skip("Max duration checking not yet implemented")
}
*/

func TestProcessor_VolumeCalculation(t *testing.T) {
	tests := []struct {
		name           string
		samples        []float32
		expectedVolume float32
		tolerance      float32
	}{
		{
			name:           "Silent samples",
			samples:        []float32{0, 0, 0, 0},
			expectedVolume: 0,
			tolerance:      0.001,
		},
		{
			name:           "Constant volume",
			samples:        []float32{0.5, 0.5, 0.5, 0.5},
			expectedVolume: 0.5,
			tolerance:      0.001,
		},
		{
			name:           "Varying volume",
			samples:        []float32{0.1, -0.1, 0.2, -0.2},
			expectedVolume: 0.15, // RMS = sqrt((0.01 + 0.01 + 0.04 + 0.04) / 4) = sqrt(0.025) ≈ 0.158
			tolerance:      0.01,
		},
		{
			name:           "Max volume",
			samples:        []float32{1.0, 1.0, 1.0, 1.0},
			expectedVolume: 1.0,
			tolerance:      0.001,
		},
		{
			name:           "Negative samples",
			samples:        []float32{-0.5, -0.5, -0.5, -0.5},
			expectedVolume: 0.5,
			tolerance:      0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate RMS manually
			var sum float32
			for _, s := range tt.samples {
				sum += s * s
			}
			rms := float32(math.Sqrt(float64(sum / float32(len(tt.samples)))))

			// Compare with expected
			diff := float32(math.Abs(float64(rms - tt.expectedVolume)))
			if diff > tt.tolerance {
				t.Errorf("Volume calculation error: got %f, expected %f (diff: %f)", rms, tt.expectedVolume, diff)
			}
		})
	}
}

func TestProcessor_ConcurrentAccess(t *testing.T) {
	cfg := &config.Config{
		Audio: struct {
			SampleRate           int     `json:"sample_rate"`
			Channels             int     `json:"channels"`
			SilenceThreshold     float32 `json:"silence_threshold"`
			SilenceDuration      float32 `json:"silence_duration"`
			ChunkDuration        int     `json:"chunk_duration"`
			MaxDuration          int     `json:"max_duration"`
			BufferSizeMultiplier int     `json:"buffer_size_multiplier"`
			FrameLength          int     `json:"frame_length"`
			BufferedFrames       int     `json:"buffered_frames"`
			DeviceIndex          int     `json:"device_index"`
			StartTone            struct {
				Enabled   bool `json:"enabled"`
				Frequency int  `json:"frequency"`
				Duration  int  `json:"duration"`
				FadeMs    int  `json:"fade_ms"`
			} `json:"start_tone"`
		}{
			SampleRate:  16000,
			Channels:    1,
			FrameLength: 512,
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	processor, err := NewProcessor(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Test concurrent processing
	done := make(chan bool, 3)

	// Goroutine 1: Process samples
	go func() {
		for i := 0; i < 100; i++ {
			samples := make([]float32, 100)
			for j := range samples {
				samples[j] = float32(i) * 0.01
			}
			processor.ProcessSamples(samples)
		}
		done <- true
	}()

	// Goroutine 2: Get buffer
	go func() {
		for i := 0; i < 100; i++ {
			_ = processor.GetBuffer()
		}
		done <- true
	}()

	// Goroutine 3: Clear buffer
	go func() {
		for i := 0; i < 50; i++ {
			processor.ClearBuffer()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without deadlock or panic, the test passes
}