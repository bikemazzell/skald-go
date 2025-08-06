package transcriber

import (
	"errors"
	"strings"
	"testing"
)

func TestWhisper_NewWhisper_WithMocks(t *testing.T) {
	// Save original factory
	originalFactory := whisperFactory
	defer func() { whisperFactory = originalFactory }()
	
	tests := []struct {
		name           string
		modelPath      string
		language       string
		setupFactory   func() *MockWhisperModelFactory
		expectError    bool
		expectedError  string
		validateResult func(*testing.T, *Whisper)
	}{
		{
			name:      "successful model creation",
			modelPath: "test-model.bin",
			language:  "en",
			setupFactory: func() *MockWhisperModelFactory {
				return NewMockFactory()
			},
			expectError: false,
			validateResult: func(t *testing.T, w *Whisper) {
				if w == nil {
					t.Error("Expected whisper instance, got nil")
				}
				if w.language != "en" {
					t.Errorf("Expected language 'en', got %s", w.language)
				}
			},
		},
		{
			name:      "model creation failure",
			modelPath: "nonexistent-model.bin",
			language:  "en",
			setupFactory: func() *MockWhisperModelFactory {
				factory := NewMockFactory()
				factory.ShouldFailCreation = true
				factory.CreationError = errors.New("file not found")
				return factory
			},
			expectError:   true,
			expectedError: "failed to load model",
		},
		{
			name:      "different language",
			modelPath: "test-model.bin", 
			language:  "es",
			setupFactory: func() *MockWhisperModelFactory {
				return NewMockFactory()
			},
			expectError: false,
			validateResult: func(t *testing.T, w *Whisper) {
				if w.language != "es" {
					t.Errorf("Expected language 'es', got %s", w.language)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock factory
			mockFactory := tt.setupFactory()
			SetModelFactory(mockFactory)
			
			// Test NewWhisper
			whisper, err := NewWhisper(tt.modelPath, tt.language)
			
			// Verify error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if tt.validateResult != nil {
					tt.validateResult(t, whisper)
				}
			}
			
			// Verify factory usage
			if !tt.expectError {
				if len(mockFactory.CreatedModels) != 1 {
					t.Errorf("Expected 1 model created, got %d", len(mockFactory.CreatedModels))
				}
			}
		})
	}
}

func TestWhisper_Transcribe_WithMocks(t *testing.T) {
	originalFactory := whisperFactory
	defer func() { whisperFactory = originalFactory }()
	
	tests := []struct {
		name           string
		audio          []float32
		language       string
		setupMock      func(*MockWhisperModel)
		expectedResult string
		expectError    bool
		expectedError  string
		validateMock   func(*testing.T, *MockWhisperModel)
	}{
		{
			name:     "empty audio returns empty string",
			audio:    []float32{},
			language: "en",
			setupMock: func(model *MockWhisperModel) {
				// No setup needed for empty audio
			},
			expectedResult: "",
			expectError:    false,
		},
		{
			name:     "successful transcription single segment",
			audio:    []float32{0.1, 0.2, 0.3},
			language: "en",
			setupMock: func(model *MockWhisperModel) {
				// Context will be created during transcription
			},
			expectedResult: "Hello world",
			expectError:    false,
			validateMock: func(t *testing.T, model *MockWhisperModel) {
				if len(model.Contexts) != 1 {
					t.Errorf("Expected 1 context, got %d", len(model.Contexts))
				}
				ctx := model.Contexts[0]
				if ctx.Language != "en" {
					t.Errorf("Expected language 'en', got %s", ctx.Language)
				}
				if len(ctx.ProcessedAudio) != 1 {
					t.Errorf("Expected 1 audio processed, got %d", len(ctx.ProcessedAudio))
				}
			},
		},
		{
			name:     "successful transcription multiple segments",
			audio:    []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			language: "auto",
			setupMock: func(model *MockWhisperModel) {
				// Will be configured during test
			},
			expectedResult: "Hello world from multiple segments",
			expectError:    false,
		},
		{
			name:     "context creation failure",
			audio:    []float32{0.1, 0.2},
			language: "en",
			setupMock: func(model *MockWhisperModel) {
				model.ShouldFailContext = true
				model.ContextCreationError = errors.New("out of memory")
			},
			expectError:   true,
			expectedError: "failed to create context",
		},
		{
			name:     "language setting failure",
			audio:    []float32{0.1, 0.2},
			language: "invalid-lang",
			setupMock: func(model *MockWhisperModel) {
				// Will be configured during transcription
			},
			expectError:   true,
			expectedError: "failed to set language",
		},
		{
			name:     "audio processing failure",
			audio:    []float32{0.1, 0.2, 0.3},
			language: "en",
			setupMock: func(model *MockWhisperModel) {
				// Will be configured during transcription
			},
			expectError:   true,
			expectedError: "failed to process audio",
		},
		{
			name:     "auto language detection",
			audio:    []float32{0.1, 0.2},
			language: "auto",
			setupMock: func(model *MockWhisperModel) {
				// Language setting should be skipped
			},
			expectedResult: "Auto detected text",
			expectError:    false,
			validateMock: func(t *testing.T, model *MockWhisperModel) {
				if len(model.Contexts) != 1 {
					t.Errorf("Expected 1 context, got %d", len(model.Contexts))
				}
				ctx := model.Contexts[0]
				// Language should be empty for auto detection
				if ctx.Language != "" {
					t.Errorf("Expected empty language for auto, got %s", ctx.Language)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock factory and model
			mockFactory := NewMockFactory()
			SetModelFactory(mockFactory)
			
			// Create whisper instance
			whisper, err := NewWhisper("test-model.bin", tt.language)
			if err != nil {
				t.Fatalf("Failed to create whisper: %v", err)
			}
			
			// Get the created mock model
			if len(mockFactory.CreatedModels) != 1 {
				t.Fatalf("Expected 1 model, got %d", len(mockFactory.CreatedModels))
			}
			mockModel := mockFactory.CreatedModels[0]
			
			// Setup mock behavior
			tt.setupMock(mockModel)
			
			// Configure mock context behavior based on test case
			contextSetup := func(ctx *MockWhisperContext) {
				switch tt.name {
				case "successful transcription single segment":
					ctx.AddSegment("Hello world")
				case "successful transcription multiple segments":
					ctx.AddSegment("Hello ")
					ctx.AddSegment("world ")
					ctx.AddSegment("from ")
					ctx.AddSegment("multiple ")
					ctx.AddSegment("segments")
				case "language setting failure":
					ctx.ShouldFailSetLanguage = true
					ctx.SetLanguageError = errors.New("invalid language")
				case "audio processing failure":
					ctx.ShouldFailProcess = true
					ctx.ProcessError = errors.New("processing failed")
				case "auto language detection":
					ctx.AddSegment("Auto detected text")
				}
			}
			
			// Override NewContext to configure the mock context
			mockModel.NewContextFunc = func() (WhisperContext, error) {
				if mockModel.ShouldFailContext {
					return nil, mockModel.ContextCreationError
				}
				ctx := NewMockContext()
				ctx.Model = mockModel
				contextSetup(ctx)
				mockModel.Contexts = append(mockModel.Contexts, ctx)
				return ctx, nil
			}
			
			// Test transcription
			result, err := whisper.Transcribe(tt.audio)
			
			// Verify results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
				}
			}
			
			// Validate mock state
			if tt.validateMock != nil {
				tt.validateMock(t, mockModel)
			}
			
			// Clear override function
			mockModel.NewContextFunc = nil
		})
	}
}

func TestWhisper_Close_WithMocks(t *testing.T) {
	originalFactory := whisperFactory
	defer func() { whisperFactory = originalFactory }()
	
	tests := []struct {
		name          string
		setupMock     func(*MockWhisperModel)
		expectError   bool
		expectedError string
		validateMock  func(*testing.T, *MockWhisperModel)
	}{
		{
			name: "successful close",
			setupMock: func(model *MockWhisperModel) {
				// Default behavior
			},
			expectError: false,
			validateMock: func(t *testing.T, model *MockWhisperModel) {
				if !model.IsClosed {
					t.Error("Expected model to be closed")
				}
			},
		},
		{
			name: "close with error",
			setupMock: func(model *MockWhisperModel) {
				model.CloseError = errors.New("close failed")
			},
			expectError:   true,
			expectedError: "close failed",
		},
		{
			name: "close nil model", 
			setupMock: func(model *MockWhisperModel) {
				// Will be tested with nil model
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var whisper *Whisper
			var mockModel *MockWhisperModel
			
			if tt.name == "close nil model" {
				whisper = &Whisper{model: nil}
			} else {
				// Setup mock factory
				mockFactory := NewMockFactory()
				SetModelFactory(mockFactory)
				
				// Create whisper instance
				var err error
				whisper, err = NewWhisper("test-model.bin", "en")
				if err != nil {
					t.Fatalf("Failed to create whisper: %v", err)
				}
				
				mockModel = mockFactory.CreatedModels[0]
				tt.setupMock(mockModel)
			}
			
			// Test close
			err := whisper.Close()
			
			// Verify results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
			
			// Validate mock state
			if tt.validateMock != nil && mockModel != nil {
				tt.validateMock(t, mockModel)
			}
		})
	}
}

func TestWhisper_ConcurrentAccess(t *testing.T) {
	originalFactory := whisperFactory
	defer func() { whisperFactory = originalFactory }()
	
	// Setup mock factory
	mockFactory := NewMockFactory()
	SetModelFactory(mockFactory)
	
	// Create whisper instance
	whisper, err := NewWhisper("test-model.bin", "en")
	if err != nil {
		t.Fatalf("Failed to create whisper: %v", err)
	}
	
	// Test concurrent transcription calls
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			audio := []float32{float32(id) * 0.1}
			_, err := whisper.Transcribe(audio)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Good
		case err := <-errors:
			t.Errorf("Concurrent transcription error: %v", err)
		}
	}
	
	// Verify multiple contexts were created (one per transcription)
	mockModel := mockFactory.CreatedModels[0]
	if len(mockModel.Contexts) != numGoroutines {
		t.Errorf("Expected %d contexts for concurrent access, got %d", numGoroutines, len(mockModel.Contexts))
	}
}