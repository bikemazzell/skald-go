package audio

import "math"

// SilenceDetector implements silence detection
type SilenceDetector struct{}

// NewSilenceDetector creates a new silence detector
func NewSilenceDetector() *SilenceDetector {
	return &SilenceDetector{}
}

// IsSilent checks if audio samples are below the threshold
func (s *SilenceDetector) IsSilent(samples []float32, threshold float32) bool {
	if len(samples) == 0 {
		return true
	}

	// Calculate RMS (Root Mean Square) for better silence detection
	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	rms := math.Sqrt(sum / float64(len(samples)))
	
	return float32(rms) < threshold
}

// CalculateRMS calculates the root mean square of samples
func (s *SilenceDetector) CalculateRMS(samples []float32) float32 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample * sample)
	}
	return float32(math.Sqrt(sum / float64(len(samples))))
}