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
)

type ModelInfo struct {
	Name               string
	URL                string
	SHA256             string
	DestPath           string
	InsecureSkipVerify bool
}

type ModelManager struct {
	client    *http.Client
	logger    *log.Logger
	modelPath string
}

type ModelManagerOption func(*ModelManager)

func WithHttpClient(client *http.Client) ModelManagerOption {
	return func(m *ModelManager) {
		m.client = client
	}
}

func New(logger *log.Logger, opts ...ModelManagerOption) *ModelManager {
	mm := &ModelManager{
		logger: logger,
	}

	for _, opt := range opts {
		opt(mm)
	}

	if mm.client == nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		mm.client = &http.Client{
			Timeout:   30 * time.Minute,
			Transport: transport,
		}
	}

	return mm
}

func (m *ModelManager) Initialize(info ModelInfo) error {
	if err := m.EnsureModelExists(info); err != nil {
		return err
	}
	m.modelPath = info.DestPath
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

func (m *ModelManager) downloadModel(info ModelInfo) error {
	tmpPath := info.DestPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		out.Close()
		os.Remove(tmpPath)
	}()

	transport, ok := m.client.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("http client transport is not a *http.Transport")
	}

	if info.InsecureSkipVerify {
		transport.TLSClientConfig.InsecureSkipVerify = true
	} else if strings.HasPrefix(info.URL, "https://127.0.0.1") || strings.HasPrefix(info.URL, "https://localhost") {
		// For local development, allow insecure connections to localhost
		transport.TLSClientConfig.InsecureSkipVerify = true
	} else {
		// Ensure it's disabled for other cases
		transport.TLSClientConfig.InsecureSkipVerify = false
	}

	resp, err := m.client.Get(info.URL)
	if err != nil {
		return fmt.Errorf("failed to download file from %s: %w", info.URL, err)
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

	if info.SHA256 != "" {
		actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
		if actualSHA256 != info.SHA256 {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", info.SHA256, actualSHA256)
		}
		m.logger.Printf("Checksum verified: %s", actualSHA256)
	}

	if err := os.Rename(tmpPath, info.DestPath); err != nil {
		return fmt.Errorf("failed to move file to final destination: %w", err)
	}

	if err := os.Chmod(info.DestPath, 0644); err != nil {
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
	*wc.progress += n
	if wc.Total > 0 {
		percentage := int(float64(*wc.progress) / float64(wc.Total) * 100)
		wc.logger.Printf("Downloading... %d%%", percentage)
	} else {
		wc.logger.Printf("Downloading... %d bytes", *wc.progress)
	}
	return n, nil
}

func (m *ModelManager) EnsureModelExists(info ModelInfo) error {
	if info.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(info.DestPath), 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	_, err := os.Stat(info.DestPath)
	if os.IsNotExist(err) {
		m.logger.Printf("Model %s not found locally, downloading from %s...", info.Name, info.URL)
		if err := m.downloadModel(info); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
		m.logger.Printf("Model %s downloaded successfully", info.Name)
	} else if err != nil {
		return fmt.Errorf("failed to stat model file: %w", err)
	} else if info.SHA256 != "" {
		if err := m.verifyModelChecksum(info.DestPath, info.SHA256); err != nil {
			m.logger.Printf("Warning: %v. Re-downloading model...", err)
			if err := m.downloadModel(info); err != nil {
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
