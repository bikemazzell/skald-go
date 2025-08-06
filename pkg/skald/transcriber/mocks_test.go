package transcriber

import (
	"errors"
	"fmt"
)

// MockWhisperModelFactory creates mock whisper models for testing
type MockWhisperModelFactory struct {
	ShouldFailCreation bool
	CreationError      error
	CreatedModels      []*MockWhisperModel
}

func (f *MockWhisperModelFactory) NewModel(modelPath string) (WhisperModel, error) {
	if f.ShouldFailCreation {
		if f.CreationError != nil {
			return nil, f.CreationError
		}
		return nil, errors.New("mock model creation failed")
	}
	
	model := &MockWhisperModel{
		ModelPath:  modelPath,
		IsClosed:   false,
		Contexts:   make([]*MockWhisperContext, 0),
	}
	f.CreatedModels = append(f.CreatedModels, model)
	return model, nil
}

// MockWhisperModel simulates a whisper model
type MockWhisperModel struct {
	ModelPath           string
	IsClosed            bool
	ShouldFailContext   bool
	ContextCreationError error
	Contexts            []*MockWhisperContext
	CloseError          error
	NewContextFunc      func() (WhisperContext, error) // Allow override for tests
}

func (m *MockWhisperModel) NewContext() (WhisperContext, error) {
	// Use override function if provided
	if m.NewContextFunc != nil {
		return m.NewContextFunc()
	}
	
	if m.IsClosed {
		return nil, errors.New("model is closed")
	}
	
	if m.ShouldFailContext {
		if m.ContextCreationError != nil {
			return nil, m.ContextCreationError
		}
		return nil, errors.New("context creation failed")
	}
	
	context := &MockWhisperContext{
		Model:    m,
		Segments: make([]*MockWhisperSegment, 0),
	}
	m.Contexts = append(m.Contexts, context)
	return context, nil
}

func (m *MockWhisperModel) Close() error {
	if m.IsClosed {
		return nil // Already closed
	}
	
	m.IsClosed = true
	if m.CloseError != nil {
		return m.CloseError
	}
	return nil
}

// MockWhisperContext simulates a whisper context
type MockWhisperContext struct {
	Model                *MockWhisperModel
	Language             string
	Segments             []*MockWhisperSegment
	CurrentSegmentIndex  int
	ShouldFailSetLanguage bool
	SetLanguageError     error
	ShouldFailProcess    bool
	ProcessError         error
	ProcessedAudio       [][]float32
}

func (c *MockWhisperContext) SetLanguage(lang string) error {
	if c.ShouldFailSetLanguage {
		if c.SetLanguageError != nil {
			return c.SetLanguageError
		}
		return fmt.Errorf("failed to set language to %s", lang)
	}
	
	c.Language = lang
	return nil
}

func (c *MockWhisperContext) Process(audio []float32, cb1, cb2 interface{}) error {
	if c.ShouldFailProcess {
		if c.ProcessError != nil {
			return c.ProcessError
		}
		return errors.New("audio processing failed")
	}
	
	// Store processed audio for verification
	audioCopy := make([]float32, len(audio))
	copy(audioCopy, audio)
	c.ProcessedAudio = append(c.ProcessedAudio, audioCopy)
	
	return nil
}

func (c *MockWhisperContext) NextSegment() (WhisperSegment, error) {
	if c.CurrentSegmentIndex >= len(c.Segments) {
		return nil, errors.New("no more segments")
	}
	
	segment := c.Segments[c.CurrentSegmentIndex]
	c.CurrentSegmentIndex++
	return segment, nil
}

// AddSegment adds a mock segment to the context
func (c *MockWhisperContext) AddSegment(text string) {
	segment := &MockWhisperSegment{Text: text}
	c.Segments = append(c.Segments, segment)
}

// MockWhisperSegment simulates a whisper segment
type MockWhisperSegment struct {
	Text string
}

func (s *MockWhisperSegment) GetText() string {
	return s.Text
}

// TestHelper functions for setting up mocks

// NewMockFactory creates a new mock factory with default settings
func NewMockFactory() *MockWhisperModelFactory {
	return &MockWhisperModelFactory{
		ShouldFailCreation: false,
		CreatedModels:      make([]*MockWhisperModel, 0),
	}
}

// NewMockModel creates a mock model with default settings
func NewMockModel() *MockWhisperModel {
	return &MockWhisperModel{
		ModelPath:  "test-model.bin",
		IsClosed:   false,
		Contexts:   make([]*MockWhisperContext, 0),
	}
}

// NewMockContext creates a mock context with default settings
func NewMockContext() *MockWhisperContext {
	return &MockWhisperContext{
		Segments:            make([]*MockWhisperSegment, 0),
		CurrentSegmentIndex: 0,
		ProcessedAudio:      make([][]float32, 0),
	}
}

// NewMockSegment creates a mock segment
func NewMockSegment(text string) *MockWhisperSegment {
	return &MockWhisperSegment{Text: text}
}