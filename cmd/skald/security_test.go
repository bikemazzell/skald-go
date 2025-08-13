package main

import (
	"math"
	"testing"
)

func TestSampleRateValidation(t *testing.T) {
	tests := []struct {
		name    string
		rate    int
		wantErr bool
	}{
		{"Rate too low", 7999, true},
		{"Minimum valid rate", 8000, false},
		{"Common rate 16kHz", 16000, false},
		{"Common rate 44.1kHz", 44100, false},
		{"Common rate 48kHz", 48000, false},
		{"Maximum valid rate", 192000, false},
		{"Rate too high", 192001, true},
		{"Negative rate", -1, true},
		{"Zero rate", 0, true},
		{"MaxInt32 boundary", math.MaxInt32, true},
		{"MaxUint32 boundary", int(math.MaxUint32), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSampleRate(tt.rate)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSampleRate(%d) error = %v, wantErr %v", 
					tt.rate, err, tt.wantErr)
			}
		})
	}
}

func TestSampleRateEdgeCases(t *testing.T) {
	// Test specific edge cases for integer overflow
	edgeCases := []int{
		-2147483648, // MinInt32
		2147483647,  // MaxInt32
		-1,
		0,
		1,
		7999,   // Just below minimum
		8000,   // Minimum valid
		192000, // Maximum valid
		192001, // Just above maximum
	}

	for _, rate := range edgeCases {
		t.Run("EdgeCase", func(t *testing.T) {
			err := validateSampleRate(rate)
			
			// Valid rates: 8000 <= rate <= 192000
			shouldBeValid := rate >= 8000 && rate <= 192000
			
			if shouldBeValid && err != nil {
				t.Errorf("validateSampleRate(%d) should be valid but got error: %v", rate, err)
			}
			if !shouldBeValid && err == nil {
				t.Errorf("validateSampleRate(%d) should be invalid but got no error", rate)
			}
		})
	}
}

// BenchmarkSampleRateValidation benchmarks the validation function
func BenchmarkSampleRateValidation(b *testing.B) {
	testRates := []int{16000, 48000, 44100, 8000, 192000}
	
	for i := 0; i < b.N; i++ {
		for _, rate := range testRates {
			validateSampleRate(rate)
		}
	}
}