package transcriber

import (
	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

// WhisperModelWrapper wraps the actual whisper model to implement our interface
type WhisperModelWrapper struct {
	model whisper.Model
}

func (w *WhisperModelWrapper) NewContext() (WhisperContext, error) {
	ctx, err := w.model.NewContext()
	if err != nil {
		return nil, err
	}
	return &WhisperContextWrapper{context: ctx}, nil
}

func (w *WhisperModelWrapper) Close() error {
	return w.model.Close()
}

// WhisperContextWrapper wraps the actual whisper context
type WhisperContextWrapper struct {
	context whisper.Context
}

func (w *WhisperContextWrapper) SetLanguage(lang string) error {
	return w.context.SetLanguage(lang)
}

func (w *WhisperContextWrapper) Process(audio []float32, cb1, cb2 interface{}) error {
	// Type assertions for whisper callback types
	var encoderBeginCallback whisper.EncoderBeginCallback
	var segmentCallback whisper.SegmentCallback
	var progressCallback whisper.ProgressCallback
	
	// Default encoder begin callback that allows processing
	encoderBeginCallback = func() bool { return true }
	
	if cb1 != nil {
		if sc, ok := cb1.(whisper.SegmentCallback); ok {
			segmentCallback = sc
		}
	}
	
	if cb2 != nil {
		if pc, ok := cb2.(whisper.ProgressCallback); ok {
			progressCallback = pc
		}
	}
	
	return w.context.Process(audio, encoderBeginCallback, segmentCallback, progressCallback)
}

func (w *WhisperContextWrapper) NextSegment() (WhisperSegment, error) {
	segment, err := w.context.NextSegment()
	if err != nil {
		return nil, err
	}
	return &WhisperSegmentWrapper{segment: segment}, nil
}

// WhisperSegmentWrapper wraps the actual whisper segment
type WhisperSegmentWrapper struct {
	segment whisper.Segment
}

func (w *WhisperSegmentWrapper) GetText() string {
	return w.segment.Text
}

// DefaultWhisperModelFactory creates real whisper models
type DefaultWhisperModelFactory struct{}

func (f *DefaultWhisperModelFactory) NewModel(modelPath string) (WhisperModel, error) {
	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, err
	}
	return &WhisperModelWrapper{model: model}, nil
}

// Global factory instance
var whisperFactory WhisperModelFactory = &DefaultWhisperModelFactory{}