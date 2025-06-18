package model

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func (m *ModelManager) downloadModel(url, destPath, expectedSHA256 string) error {
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		out.Close()
		os.Remove(tmpPath)
	}()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	
	if strings.HasPrefix(url, "https://127.0.0.1") || strings.HasPrefix(url, "https://localhost") {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
	
	client := &http.Client{
		Timeout: 30 * time.Minute,
		Transport: transport,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	hasher := sha256.New()

	counter := &WriteCounter{
		Total:    resp.ContentLength,
		progress: new(int),
		logger:   m.logger,
	}

	multiWriter := io.MultiWriter(out, hasher, counter)
	_, err = io.Copy(multiWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	if expectedSHA256 != "" {
		actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
		if actualSHA256 != expectedSHA256 {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
		}
		m.logger.Printf("Checksum verified: %s", actualSHA256)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	if err := os.Chmod(destPath, 0644); err != nil {
		m.logger.Printf("Warning: failed to set permissions on model file: %v", err)
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
		if err := m.downloadModel(modelInfo.URL, modelPath, modelInfo.SHA256); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		m.logger.Printf("Model %s downloaded successfully", modelName)
	} else if modelInfo.SHA256 != "" {
		if err := m.verifyModelChecksum(modelPath, modelInfo.SHA256); err != nil {
			m.logger.Printf("Warning: %v. Re-downloading model...", err)
			if err := m.downloadModel(modelInfo.URL, modelPath, modelInfo.SHA256); err != nil {
				return fmt.Errorf("failed to re-download model: %w", err)
			}
		}
	}

	return nil
}

func (m *ModelManager) verifyModelChecksum(filePath, expectedSHA256 string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum verification: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}
