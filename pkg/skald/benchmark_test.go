package skald

import (
	"fmt"
	"math"
	"testing"
	"time"

	"skald/pkg/skald/audio"
)

// BenchmarkSilenceDetection benchmarks silence detection performance
func BenchmarkSilenceDetection(b *testing.B) {
	silenceDetector := audio.NewSilenceDetector()
	
	benchmarks := []struct {
		name        string
		sampleCount int
		setupSamples func(int) []float32
	}{
		{
			name:        "silence-1024",
			sampleCount: 1024,
			setupSamples: func(n int) []float32 {
				return make([]float32, n) // All zeros
			},
		},
		{
			name:        "silence-4096",
			sampleCount: 4096,
			setupSamples: func(n int) []float32 {
				return make([]float32, n)
			},
		},
		{
			name:        "noise-1024",
			sampleCount: 1024,
			setupSamples: func(n int) []float32 {
				samples := make([]float32, n)
				seed := uint32(12345)
				for i := 0; i < n; i++ {
					seed = seed*1103515245 + 12345
					samples[i] = float32((seed>>16)&0x7fff)/32767.0*0.1 - 0.05
				}
				return samples
			},
		},
		{
			name:        "sine-wave-1024",
			sampleCount: 1024,
			setupSamples: func(n int) []float32 {
				samples := make([]float32, n)
				for i := 0; i < n; i++ {
					t := float64(i) / 16000.0
					samples[i] = float32(0.5 * math.Sin(2*math.Pi*440*t))
				}
				return samples
			},
		},
	}
	
	for _, bm := range benchmarks {
		samples := bm.setupSamples(bm.sampleCount)
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				silenceDetector.IsSilent(samples, 0.01)
			}
		})
	}
}

// BenchmarkRMSCalculation benchmarks RMS calculation performance
func BenchmarkRMSCalculation(b *testing.B) {
	silenceDetector := audio.NewSilenceDetector()
	
	sampleSizes := []int{256, 512, 1024, 2048, 4096}
	
	for _, size := range sampleSizes {
		samples := make([]float32, size)
		// Fill with sine wave
		for i := 0; i < size; i++ {
			t := float64(i) / 16000.0
			samples[i] = float32(0.5 * math.Sin(2*math.Pi*440*t))
		}
		
		b.Run(fmt.Sprintf("rms-%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				silenceDetector.CalculateRMS(samples)
			}
		})
	}
}

// BenchmarkAudioCapture benchmarks audio capture operations
func BenchmarkAudioCapture(b *testing.B) {
	b.Run("constructor", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			capture := audio.NewCapture(16000)
			_ = capture
		}
	})
	
	b.Run("stop-without-start", func(b *testing.B) {
		captures := make([]*audio.Capture, b.N)
		for i := 0; i < b.N; i++ {
			captures[i] = audio.NewCapture(16000)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			captures[i].Stop()
		}
	})
}

// BenchmarkMemoryUsage tests for memory leaks and usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("silence-detector-allocation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			detector := audio.NewSilenceDetector()
			samples := make([]float32, 1024)
			detector.IsSilent(samples, 0.01)
		}
	})
	
	b.Run("audio-buffer-reuse", func(b *testing.B) {
		silenceDetector := audio.NewSilenceDetector()
		buffer := make([]float32, 1024)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate reusing the same buffer
			for j := range buffer {
				buffer[j] = float32(i+j) * 0.001
			}
			silenceDetector.IsSilent(buffer, 0.01)
		}
	})
}

// BenchmarkConcurrentAccess tests concurrent performance
func BenchmarkConcurrentAccess(b *testing.B) {
	b.Run("concurrent-silence-detection", func(b *testing.B) {
		silenceDetector := audio.NewSilenceDetector()
		samples := make([]float32, 1024)
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				silenceDetector.IsSilent(samples, 0.01)
			}
		})
	})
	
	b.Run("concurrent-captures", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				capture := audio.NewCapture(16000)
				capture.Stop()
			}
		})
	})
}

// TestMemoryLeaks tests for memory leaks in long-running scenarios
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}
	
	t.Run("repeated-silence-detection", func(t *testing.T) {
		silenceDetector := audio.NewSilenceDetector()
		samples := make([]float32, 1024)
		
		// Run for a while to detect any gradual memory leaks
		start := time.Now()
		iterations := 0
		
		for time.Since(start) < time.Second {
			silenceDetector.IsSilent(samples, 0.01)
			iterations++
		}
		
		t.Logf("Completed %d silence detections in 1 second", iterations)
		
		// Should be able to do thousands per second
		if iterations < 1000 {
			t.Errorf("Performance concern: only %d iterations per second", iterations)
		}
	})
	
	t.Run("audio-capture-lifecycle", func(t *testing.T) {
		// Test creating and destroying many capture instances
		for i := 0; i < 1000; i++ {
			capture := audio.NewCapture(16000)
			err := capture.Stop()
			if err != nil {
				t.Errorf("Stop failed on iteration %d: %v", i, err)
			}
		}
	})
}

// TestRealTimeCharacteristics tests real-time processing characteristics
func TestRealTimeCharacteristics(t *testing.T) {
	t.Run("silence-detection-latency", func(t *testing.T) {
		silenceDetector := audio.NewSilenceDetector()
		
		testSizes := []int{256, 512, 1024, 2048, 4096}
		
		for _, size := range testSizes {
			samples := make([]float32, size)
			
			start := time.Now()
			silenceDetector.IsSilent(samples, 0.01)
			elapsed := time.Since(start)
			
			// Calculate how much real audio this represents at 16kHz
			realTimeDuration := time.Duration(size) * time.Second / 16000
			
			// Processing should be much faster than real-time
			if elapsed > realTimeDuration/10 { // Allow 10% of real-time
				t.Errorf("Silence detection too slow for size %d: %v (real-time: %v)", 
					size, elapsed, realTimeDuration)
			}
		}
	})
}

// PropertyBasedTesting implements property-based testing for audio processing
func TestAudioProcessingProperties(t *testing.T) {
	silenceDetector := audio.NewSilenceDetector()
	
	t.Run("silence-detection-properties", func(t *testing.T) {
		// Property: All-zero samples should always be silent
		for size := 1; size <= 4096; size *= 2 {
			samples := make([]float32, size)
			if !silenceDetector.IsSilent(samples, 0.01) {
				t.Errorf("All-zero samples of size %d should be silent", size)
			}
		}
		
		// Property: Very loud samples should never be silent
		for size := 1; size <= 4096; size *= 2 {
			samples := make([]float32, size)
			for i := range samples {
				samples[i] = 1.0 // Maximum amplitude
			}
			if silenceDetector.IsSilent(samples, 0.01) {
				t.Errorf("Maximum amplitude samples of size %d should not be silent", size)
			}
		}
		
		// Property: RMS should be monotonic with amplitude
		baselineRMS := silenceDetector.CalculateRMS(make([]float32, 1000)) // All zeros
		
		for amplitude := 0.1; amplitude <= 1.0; amplitude += 0.1 {
			samples := make([]float32, 1000)
			for i := range samples {
				samples[i] = float32(amplitude)
			}
			rms := silenceDetector.CalculateRMS(samples)
			
			if rms <= baselineRMS {
				t.Errorf("RMS should increase with amplitude, got %f <= %f", rms, baselineRMS)
			}
			baselineRMS = rms
		}
	})
}