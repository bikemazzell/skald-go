package model

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"skald/internal/config"
)

type ModelManager struct {
	cfg       *config.Config
	logger    *log.Logger
	modelPath string
}

func New(cfg *config.Config, logger *log.Logger) *ModelManager {
	return &ModelManager{
		cfg:    cfg,
		logger: logger,
	}
}

func (m *ModelManager) Initialize(modelName string) error {
	// Get model info
	modelInfo, ok := m.cfg.Whisper.Models[modelName]
	if !ok {
		return fmt.Errorf("unknown model: %s", modelName)
	}

	// Create models directory if it doesn't exist
	modelsDir := "models"
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Construct model path
	m.modelPath = filepath.Join(modelsDir, fmt.Sprintf("ggml-%s.bin", modelName))

	// Check if model already exists
	if _, err := os.Stat(m.modelPath); err == nil {
		m.logger.Printf("Model already exists at: %s", m.modelPath)
		return nil
	}

	// Download model
	m.logger.Printf("Downloading model %s (%s)...", modelName, modelInfo.Size)
	if err := m.downloadModel(modelInfo.URL, m.modelPath); err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}

	m.logger.Printf("Model downloaded successfully to: %s", m.modelPath)
	return nil
}

func (m *ModelManager) GetModelPath() string {
	return m.modelPath
}

func (m *ModelManager) downloadModel(url, destPath string) error {
	// Create the destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create progress counter
	counter := &WriteCounter{
		Total:    resp.ContentLength,
		progress: new(int),
		logger:   m.logger,
	}

	// Copy data with progress updates
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// WriteCounter counts the number of bytes written to it
type WriteCounter struct {
	Total    int64
	progress *int
	logger   *log.Logger
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	current := int(*wc.progress+n) * 100 / int(wc.Total)
	if current != *wc.progress {
		*wc.progress = current
		wc.logger.Printf("Downloading... %d%%", current)
	}
	return n, nil
}
