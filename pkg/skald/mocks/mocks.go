package mocks

import (
	"context"
	"sync"
)

// MockAudioCapture is a mock implementation of AudioCapture
type MockAudioCapture struct {
	mu          sync.Mutex
	StartFunc   func(ctx context.Context) (<-chan []float32, error)
	StopFunc    func() error
	StartCalled int
	StopCalled  int
}

func (m *MockAudioCapture) Start(ctx context.Context) (<-chan []float32, error) {
	m.mu.Lock()
	m.StartCalled++
	m.mu.Unlock()
	
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	
	// Default implementation
	ch := make(chan []float32, 1)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func (m *MockAudioCapture) Stop() error {
	m.mu.Lock()
	m.StopCalled++
	m.mu.Unlock()
	
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

// MockTranscriber is a mock implementation of Transcriber
type MockTranscriber struct {
	mu               sync.Mutex
	TranscribeFunc   func(audio []float32) (string, error)
	CloseFunc        func() error
	TranscribeCalled int
	CloseCalled      int
	LastAudio        []float32
}

func (m *MockTranscriber) Transcribe(audio []float32) (string, error) {
	m.mu.Lock()
	m.TranscribeCalled++
	m.LastAudio = make([]float32, len(audio))
	copy(m.LastAudio, audio)
	m.mu.Unlock()
	
	if m.TranscribeFunc != nil {
		return m.TranscribeFunc(audio)
	}
	return "mock transcription", nil
}

func (m *MockTranscriber) Close() error {
	m.mu.Lock()
	m.CloseCalled++
	m.mu.Unlock()
	
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockOutput is a mock implementation of Output
type MockOutput struct {
	mu          sync.Mutex
	WriteFunc   func(text string) error
	WriteCalled int
	LastText    string
	AllTexts    []string
}

func (m *MockOutput) Write(text string) error {
	m.mu.Lock()
	m.WriteCalled++
	m.LastText = text
	m.AllTexts = append(m.AllTexts, text)
	m.mu.Unlock()
	
	if m.WriteFunc != nil {
		return m.WriteFunc(text)
	}
	return nil
}

// MockSilenceDetector is a mock implementation of SilenceDetector
type MockSilenceDetector struct {
	mu             sync.Mutex
	IsSilentFunc   func(samples []float32, threshold float32) bool
	IsSilentCalled int
}

func (m *MockSilenceDetector) IsSilent(samples []float32, threshold float32) bool {
	m.mu.Lock()
	m.IsSilentCalled++
	m.mu.Unlock()
	
	if m.IsSilentFunc != nil {
		return m.IsSilentFunc(samples, threshold)
	}
	return false
}