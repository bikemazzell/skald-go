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
	if err := m.EnsureModelExists(modelName); err != nil {
		return err
	}
	m.modelPath = filepath.Join("models", fmt.Sprintf("ggml-%s.bin", modelName))
	return nil
}

func (m *ModelManager) GetModelPath() string {
	if m.modelPath == "" {
		m.logger.Printf("Warning: modelPath is empty, model may not be initialized")
		return ""
	}

	absPath, err := filepath.Abs(m.modelPath)
	if err != nil {
		m.logger.Printf("Warning: failed to get absolute path for model: %v", err)
		return m.modelPath
	}

	if _, err := os.Stat(absPath); err != nil {
		m.logger.Printf("Warning: model file not accessible at %s: %v", absPath, err)
	}

	return absPath
}

func (m *ModelManager) downloadModel(url, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	counter := &WriteCounter{
		Total:    resp.ContentLength,
		progress: new(int),
		logger:   m.logger,
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

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

func (m *ModelManager) EnsureModelExists(modelName string) error {
	if modelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	modelInfo, exists := m.cfg.Whisper.Models[modelName]
	if !exists {
		return fmt.Errorf("model %s not found in configuration", modelName)
	}

	modelsDir := "models"
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	modelPath := filepath.Join(modelsDir, fmt.Sprintf("ggml-%s.bin", modelName))

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		m.logger.Printf("Model %s not found locally, downloading from %s...", modelName, modelInfo.URL)
		if err := m.downloadModel(modelInfo.URL, modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		m.logger.Printf("Model %s downloaded successfully", modelName)
	}

	return nil
}
