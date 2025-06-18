package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"skald/internal/config"
)

func TestModelManager_verifyModelChecksum(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "model-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file with known content
	testContent := []byte("test model content")
	testFile := filepath.Join(tempDir, "test-model.bin")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected SHA256
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create model manager
	cfg := &config.Config{}
	logger := log.New(io.Discard, "", 0)
	mm := New(cfg, logger)

	// Test valid checksum
	err = mm.verifyModelChecksum(testFile, expectedSHA256)
	if err != nil {
		t.Errorf("Expected no error for valid checksum, got: %v", err)
	}

	// Test invalid checksum
	err = mm.verifyModelChecksum(testFile, "invalid-checksum")
	if err == nil {
		t.Error("Expected error for invalid checksum, got nil")
	}

	// Test non-existent file
	err = mm.verifyModelChecksum(filepath.Join(tempDir, "non-existent.bin"), expectedSHA256)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestModelManager_downloadModel(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "model-download-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test content
	testContent := []byte("test model content for download")
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	// Create model manager
	cfg := &config.Config{}
	logger := log.New(io.Discard, "", 0)
	mm := New(cfg, logger)

	// Test successful download with checksum
	destPath := filepath.Join(tempDir, "downloaded-model.bin")
	err = mm.downloadModel(server.URL, destPath, expectedSHA256)
	if err != nil {
		t.Errorf("Expected no error for successful download, got: %v", err)
	}

	// Verify file exists and has correct content
	downloadedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(downloadedContent) != string(testContent) {
		t.Error("Downloaded content doesn't match expected content")
	}

	// Test download with wrong checksum
	destPath2 := filepath.Join(tempDir, "downloaded-model2.bin")
	err = mm.downloadModel(server.URL, destPath2, "wrong-checksum")
	if err == nil {
		t.Error("Expected error for wrong checksum, got nil")
	}

	// Test download from invalid URL
	destPath3 := filepath.Join(tempDir, "downloaded-model3.bin")
	err = mm.downloadModel("http://invalid-url-that-does-not-exist.com/model.bin", destPath3, "")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestModelManager_EnsureModelExists(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "model-ensure-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory to create models folder there
	originalWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWd)

	// Test content
	testContent := []byte("test model content")
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedSHA256 := hex.EncodeToString(hasher.Sum(nil))

	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer server.Close()

	// Create config with test model
	cfg := config.DefaultConfig()
	cfg.Whisper.Models = map[string]config.WhisperModelInfo{
		"test-model": {
			URL:    server.URL,
			Size:   "1MB",
			SHA256: expectedSHA256,
		},
	}

	logger := log.New(io.Discard, "", 0)
	mm := New(cfg, logger)

	// Test model download
	err = mm.EnsureModelExists("test-model")
	if err != nil {
		t.Errorf("Expected no error for model download, got: %v", err)
	}

	// Verify model file exists
	modelPath := filepath.Join("models", "ggml-test-model.bin")
	if _, err := os.Stat(modelPath); err != nil {
		t.Errorf("Model file should exist after download: %v", err)
	}

	// Test with existing model (should not re-download)
	err = mm.EnsureModelExists("test-model")
	if err != nil {
		t.Errorf("Expected no error for existing model, got: %v", err)
	}

	// Test with non-existent model
	err = mm.EnsureModelExists("non-existent-model")
	if err == nil {
		t.Error("Expected error for non-existent model, got nil")
	}

	// Test with corrupted existing model
	// Write different content to simulate corruption
	corruptContent := []byte("corrupted content")
	if err := os.WriteFile(modelPath, corruptContent, 0644); err != nil {
		t.Fatalf("Failed to write corrupt model: %v", err)
	}

	// Should detect corruption and re-download
	err = mm.EnsureModelExists("test-model")
	if err != nil {
		t.Errorf("Expected no error for re-download after corruption, got: %v", err)
	}

	// Verify model was re-downloaded with correct content
	redownloadedContent, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("Failed to read re-downloaded model: %v", err)
	}
	if string(redownloadedContent) != string(testContent) {
		t.Error("Re-downloaded content doesn't match expected content")
	}
}

func TestWriteCounter(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	progress := new(int)
	wc := &WriteCounter{
		Total:    100,
		progress: progress,
		logger:   logger,
	}

	// Test writing data
	data := make([]byte, 50)
	n, err := wc.Write(data)
	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != 50 {
		t.Errorf("Expected 50 bytes written, got %d", n)
	}
	if *progress != 50 {
		t.Errorf("Expected progress to be 50, got %d", *progress)
	}

	// Write more data
	n, err = wc.Write(data)
	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != 50 {
		t.Errorf("Expected 50 bytes written, got %d", n)
	}
	if *progress != 100 {
		t.Errorf("Expected progress to be 100, got %d", *progress)
	}
}