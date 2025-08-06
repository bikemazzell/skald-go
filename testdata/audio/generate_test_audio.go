package main

import (
	"encoding/binary"
	"math"
	"os"
)

// generateTestAudio creates various test audio files for integration testing
func main() {
	// Generate silence
	generateSilence("silence_1s.raw", 16000, 1)
	generateSilence("silence_5s.raw", 16000, 5)
	
	// Generate sine waves
	generateSineWave("sine_440hz_1s.raw", 16000, 1, 440)
	generateSineWave("sine_1000hz_2s.raw", 16000, 2, 1000)
	
	// Generate noise patterns
	generateWhiteNoise("noise_1s.raw", 16000, 1)
	
	// Generate mixed content
	generateMixedSignal("mixed_speech_pattern.raw", 16000, 3)
}

func generateSilence(filename string, sampleRate, durationSecs int) {
	samples := make([]float32, sampleRate*durationSecs)
	// All zeros = silence
	writeRawAudio(filename, samples)
}

func generateSineWave(filename string, sampleRate, durationSecs int, frequency float64) {
	numSamples := sampleRate * durationSecs
	samples := make([]float32, numSamples)
	
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		samples[i] = float32(0.5 * math.Sin(2*math.Pi*frequency*t))
	}
	
	writeRawAudio(filename, samples)
}

func generateWhiteNoise(filename string, sampleRate, durationSecs int) {
	numSamples := sampleRate * durationSecs
	samples := make([]float32, numSamples)
	
	// Simple PRNG for consistent test results
	seed := uint32(12345)
	for i := 0; i < numSamples; i++ {
		// Linear congruential generator
		seed = seed*1103515245 + 12345
		// Convert to float in range [-0.1, 0.1] for quiet noise
		samples[i] = float32((seed>>16)&0x7fff)/32767.0*0.2 - 0.1
	}
	
	writeRawAudio(filename, samples)
}

func generateMixedSignal(filename string, sampleRate, durationSecs int) {
	numSamples := sampleRate * durationSecs
	samples := make([]float32, numSamples)
	
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		
		// Simulate speech-like patterns: bursts of activity with pauses
		segment := int(t * 2) % 3 // 0, 1, 2 pattern every 1.5 seconds
		
		switch segment {
		case 0: // Active speech simulation
			samples[i] = float32(0.3 * math.Sin(2*math.Pi*300*t) * math.Exp(-math.Mod(t*5, 1)))
		case 1: // Pause
			samples[i] = 0
		case 2: // Different frequency
			samples[i] = float32(0.2 * math.Sin(2*math.Pi*600*t))
		}
		
		// Add small amount of noise
		seed := uint32(i + 54321)
		seed = seed*1103515245 + 12345
		noise := float32((seed>>16)&0x7fff)/32767.0*0.02 - 0.01
		samples[i] += noise
	}
	
	writeRawAudio(filename, samples)
}

func writeRawAudio(filename string, samples []float32) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	
	for _, sample := range samples {
		err := binary.Write(file, binary.LittleEndian, sample)
		if err != nil {
			panic(err)
		}
	}
}