package audio

import (
	"fmt"
	"sync"
)

// CircularBuffer implements a thread-safe circular buffer for audio samples
type CircularBuffer struct {
	data  []float32
	size  int
	head  int // Write position
	tail  int // Read position
	count int // Current number of elements
	mu    sync.Mutex
}

// NewCircularBuffer creates a new circular buffer with given size
func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]float32, size),
		size: size,
	}
}

// Write adds samples to the buffer, returns number of samples written
func (b *CircularBuffer) Write(samples []float32) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(samples) == 0 {
		return 0, nil
	}

	// Calculate available space
	space := b.size - b.count
	if space == 0 {
		return 0, fmt.Errorf("buffer full")
	}

	// Calculate how many samples we can write
	toWrite := min(len(samples), space)

	// Write samples
	for i := 0; i < toWrite; i++ {
		b.data[b.head] = samples[i]
		b.head = (b.head + 1) % b.size
		b.count++
	}

	return toWrite, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Read retrieves samples from the buffer
func (b *CircularBuffer) Read(n int) []float32 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil // Return nil if buffer is empty
	}

	if n > b.count {
		n = b.count
	}

	result := make([]float32, n)
	for i := 0; i < n; i++ {
		result[i] = b.data[b.tail]
		b.tail = (b.tail + 1) % b.size
		b.count--
	}

	return result
}

// Available returns the number of samples available to read
func (b *CircularBuffer) Available() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// Clear empties the buffer
func (b *CircularBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.tail = 0
	b.count = 0
}

// IsFull returns true if the buffer is full
func (b *CircularBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count == b.size
}
