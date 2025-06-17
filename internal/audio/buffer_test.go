package audio

import (
	"testing"
)

func TestCircularBuffer(t *testing.T) {
	// Test buffer creation
	bufferSize := 10
	buffer := NewCircularBuffer(bufferSize)
	
	if buffer.size != bufferSize {
		t.Errorf("Expected buffer size %d, got %d", bufferSize, buffer.size)
	}
	
	if buffer.Available() != 0 {
		t.Errorf("Expected empty buffer, got %d elements", buffer.Available())
	}
	
	// Test writing to buffer
	samples := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	n, err := buffer.Write(samples)
	
	if err != nil {
		t.Errorf("Unexpected error writing to buffer: %v", err)
	}
	
	if n != len(samples) {
		t.Errorf("Expected to write %d samples, wrote %d", len(samples), n)
	}
	
	if buffer.Available() != len(samples) {
		t.Errorf("Expected buffer to have %d elements, got %d", len(samples), buffer.Available())
	}
	
	// Test reading from buffer
	readSamples := buffer.Read(3)
	
	if len(readSamples) != 3 {
		t.Errorf("Expected to read 3 samples, got %d", len(readSamples))
	}
	
	if buffer.Available() != 2 {
		t.Errorf("Expected buffer to have 2 elements after reading, got %d", buffer.Available())
	}
	
	// Test buffer full condition
	largeSamples := make([]float32, bufferSize)
	for i := range largeSamples {
		largeSamples[i] = float32(i)
	}
	
	buffer.Clear()
	n, err = buffer.Write(largeSamples)
	
	if err != nil {
		t.Errorf("Unexpected error writing to buffer: %v", err)
	}
	
	if n != bufferSize {
		t.Errorf("Expected to write %d samples, wrote %d", bufferSize, n)
	}
	
	if !buffer.IsFull() {
		t.Error("Expected buffer to be full")
	}
	
	// Test writing to full buffer
	_, err = buffer.Write([]float32{1.0})
	if err == nil {
		t.Error("Expected error when writing to full buffer")
	}
	
	// Test reading more than available
	readSamples = buffer.Read(bufferSize + 5)
	if len(readSamples) != bufferSize {
		t.Errorf("Expected to read %d samples, got %d", bufferSize, len(readSamples))
	}
	
	// Test reading from empty buffer
	readSamples = buffer.Read(1)
	if readSamples != nil {
		t.Errorf("Expected nil when reading from empty buffer, got %v", readSamples)
	}
	
	// Test clear
	if _, err := buffer.Write(samples); err != nil {
		t.Fatalf("Failed to write samples: %v", err)
	}
	buffer.Clear()
	if buffer.Available() != 0 {
		t.Errorf("Expected buffer to be empty after clear, got %d elements", buffer.Available())
	}
}

func TestCircularBufferWraparound(t *testing.T) {
	// Test buffer wraparound behavior
	bufferSize := 5
	buffer := NewCircularBuffer(bufferSize)
	
	// Fill buffer
	samples1 := []float32{1.0, 2.0, 3.0}
	if _, err := buffer.Write(samples1); err != nil {
		t.Fatalf("Failed to write samples1: %v", err)
	}
	
	// Read some data
	buffer.Read(2)
	
	// Write more data to trigger wraparound
	samples2 := []float32{4.0, 5.0, 6.0, 7.0}
	n, err := buffer.Write(samples2)
	
	if err != nil {
		t.Errorf("Unexpected error writing to buffer: %v", err)
	}
	
	if n != 4 {
		t.Errorf("Expected to write 4 samples, wrote %d", n)
	}
	
	// Buffer should now contain [3.0, 4.0, 5.0, 6.0, 7.0]
	if buffer.Available() != 5 {
		t.Errorf("Expected buffer to have 5 elements, got %d", buffer.Available())
	}
	
	// Read all data
	readSamples := buffer.Read(5)
	
	if len(readSamples) != 5 {
		t.Errorf("Expected to read 5 samples, got %d", len(readSamples))
	}
	
	// Verify the values
	expected := []float32{3.0, 4.0, 5.0, 6.0, 7.0}
	for i, val := range expected {
		if readSamples[i] != val {
			t.Errorf("Expected sample %d to be %f, got %f", i, val, readSamples[i])
		}
	}
}
