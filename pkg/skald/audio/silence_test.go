package audio

import (
	"math"
	"testing"
)

func TestSilenceDetector_IsSilent(t *testing.T) {
	detector := NewSilenceDetector()

	tests := []struct {
		name      string
		samples   []float32
		threshold float32
		want      bool
	}{
		{
			name:      "empty samples should be silent",
			samples:   []float32{},
			threshold: 0.01,
			want:      true,
		},
		{
			name:      "zero samples should be silent",
			samples:   []float32{0, 0, 0, 0, 0},
			threshold: 0.01,
			want:      true,
		},
		{
			name:      "very quiet samples should be silent",
			samples:   []float32{0.005, -0.005, 0.003, -0.003},
			threshold: 0.01,
			want:      true,
		},
		{
			name:      "loud samples should not be silent",
			samples:   []float32{0.5, -0.5, 0.3, -0.3},
			threshold: 0.01,
			want:      false,
		},
		{
			name:      "mixed samples with one loud should not be silent",
			samples:   []float32{0.001, 0.001, 0.5, 0.001},
			threshold: 0.01,
			want:      false,
		},
		{
			name:      "threshold edge case - just below",
			samples:   []float32{0.009, 0.009, 0.009, 0.009},
			threshold: 0.01,
			want:      true,
		},
		{
			name:      "threshold edge case - just above",
			samples:   []float32{0.011, 0.011, 0.011, 0.011},
			threshold: 0.01,
			want:      false,
		},
		{
			name:      "sine wave pattern",
			samples:   generateSineWave(100, 0.1),
			threshold: 0.05,
			want:      false,
		},
		{
			name:      "sine wave pattern - quiet",
			samples:   generateSineWave(100, 0.01),
			threshold: 0.05,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.IsSilent(tt.samples, tt.threshold)
			if got != tt.want {
				rms := detector.CalculateRMS(tt.samples)
				t.Errorf("IsSilent() = %v, want %v (RMS: %f, threshold: %f)", 
					got, tt.want, rms, tt.threshold)
			}
		})
	}
}

func TestSilenceDetector_CalculateRMS(t *testing.T) {
	detector := NewSilenceDetector()

	tests := []struct {
		name    string
		samples []float32
		want    float32
		epsilon float32
	}{
		{
			name:    "empty samples",
			samples: []float32{},
			want:    0,
			epsilon: 0.0001,
		},
		{
			name:    "zero samples",
			samples: []float32{0, 0, 0, 0},
			want:    0,
			epsilon: 0.0001,
		},
		{
			name:    "constant positive samples",
			samples: []float32{0.5, 0.5, 0.5, 0.5},
			want:    0.5,
			epsilon: 0.0001,
		},
		{
			name:    "constant negative samples",
			samples: []float32{-0.5, -0.5, -0.5, -0.5},
			want:    0.5,
			epsilon: 0.0001,
		},
		{
			name:    "alternating samples",
			samples: []float32{1, -1, 1, -1},
			want:    1,
			epsilon: 0.0001,
		},
		{
			name:    "sine wave RMS",
			samples: generateSineWave(1000, 1.0),
			want:    0.7071, // RMS of sine wave is amplitude / sqrt(2)
			epsilon: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.CalculateRMS(tt.samples)
			if math.Abs(float64(got-tt.want)) > float64(tt.epsilon) {
				t.Errorf("CalculateRMS() = %v, want %v ± %v", got, tt.want, tt.epsilon)
			}
		})
	}
}

// generateSineWave generates a sine wave with given number of samples and amplitude
func generateSineWave(samples int, amplitude float32) []float32 {
	result := make([]float32, samples)
	for i := 0; i < samples; i++ {
		result[i] = amplitude * float32(math.Sin(2*math.Pi*float64(i)/float64(samples)))
	}
	return result
}

func BenchmarkSilenceDetector_IsSilent(b *testing.B) {
	detector := NewSilenceDetector()
	samples := generateSineWave(1024, 0.1)
	threshold := float32(0.01)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.IsSilent(samples, threshold)
	}
}

func BenchmarkSilenceDetector_CalculateRMS(b *testing.B) {
	detector := NewSilenceDetector()
	samples := generateSineWave(1024, 0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.CalculateRMS(samples)
	}
}