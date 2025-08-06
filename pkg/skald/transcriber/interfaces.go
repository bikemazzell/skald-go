package transcriber

// WhisperModel defines the interface for whisper model operations
// This allows us to mock the whisper model for testing
type WhisperModel interface {
	NewContext() (WhisperContext, error)
	Close() error
}

// WhisperContext defines the interface for whisper context operations
type WhisperContext interface {
	SetLanguage(lang string) error
	Process(audio []float32, cb1, cb2 interface{}) error
	NextSegment() (WhisperSegment, error)
}

// WhisperSegment represents a transcribed text segment
type WhisperSegment interface {
	GetText() string
}

// WhisperModelFactory creates whisper models
type WhisperModelFactory interface {
	NewModel(modelPath string) (WhisperModel, error)
}