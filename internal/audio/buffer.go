package audio

import (
	"fmt"
	"sync"
)

type CircularBuffer struct {
	data  []float32
	size  int
	head  int
	tail  int
	count int
	mu    sync.Mutex
}

func NewCircularBuffer(size int) *CircularBuffer {
	return &CircularBuffer{
		data: make([]float32, size),
		size: size,
	}
}

func (b *CircularBuffer) Write(samples []float32) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(samples) == 0 {
		return 0, nil
	}

	space := b.size - b.count
	if space == 0 {
		return 0, fmt.Errorf("buffer full")
	}

	toWrite := min(len(samples), space)

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

func (b *CircularBuffer) Read(n int) []float32 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		return nil
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

func (b *CircularBuffer) Available() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

func (b *CircularBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.tail = 0
	b.count = 0
}

func (b *CircularBuffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count == b.size
}
