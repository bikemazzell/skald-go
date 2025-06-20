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
	logger := log.New(io.Discard, "", 0)
	mm := New(logger)

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
	logger := log.New(io.Discard, "", 0)
	mm := New(logger, WithHttpClient(server.Client()))

	// Test successful download with checksum
	destPath := filepath.Join(tempDir, "downloaded-model.bin")
	modelInfo := ModelInfo{
		Name:     "test-model",
		URL:      server.URL,
		SHA256:   expectedSHA256,
		DestPath: destPath,
	}
	err = mm.downloadModel(modelInfo)
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
	modelInfoWrongChecksum := ModelInfo{
		Name:     "test-model-wrong-checksum",
		URL:      server.URL,
		SHA256:   "wrong-checksum",
		DestPath: destPath2,
	}
	err = mm.downloadModel(modelInfoWrongChecksum)
	if err == nil {
		t.Error("Expected error for wrong checksum, got nil")
	}

	// Test download from invalid URL
	destPath3 := filepath.Join(tempDir, "downloaded-model3.bin")
	modelInfoInvalidURL := ModelInfo{
		Name:     "test-model-invalid-url",
		URL:      "http://invalid-url-that-does-not-exist.com/model.bin",
		DestPath: destPath3,
	}
	err = mm.downloadModel(modelInfoInvalidURL)
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

	logger := log.New(io.Discard, "", 0)
	mm := New(logger, WithHttpClient(server.Client()))

	modelPath := filepath.Join("models", "ggml-test-model.bin")
	modelInfo := ModelInfo{
		Name:     "test-model",
		URL:      server.URL,
		SHA256:   expectedSHA256,
		DestPath: modelPath,
	}

	// Test model download
	err = mm.EnsureModelExists(modelInfo)
	if err != nil {
		t.Errorf("Expected no error for model download, got: %v", err)
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); err != nil {
		t.Errorf("Model file should exist after download: %v", err)
	}

	// Test with existing model (should not re-download)
	err = mm.EnsureModelExists(modelInfo)
	if err != nil {
		t.Errorf("Expected no error for existing model, got: %v", err)
	}

	// Test with non-existent model URL but with a valid structure
	nonExistentModelInfo := ModelInfo{
		Name:     "non-existent-model",
		URL:      "http://invalid-url-that-does-not-exist.com/model.bin",
		DestPath: filepath.Join("models", "ggml-non-existent.bin"),
	}
	err = mm.EnsureModelExists(nonExistentModelInfo)
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
	err = mm.EnsureModelExists(modelInfo)
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
	progress := 0
	wc := &WriteCounter{
		Total:    100,
		progress: &progress,
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
	if *wc.progress != 50 {
		t.Errorf("Expected progress to be 50, got %d", *wc.progress)
	}

	// Write more data
	n, err = wc.Write(data)
	if err != nil {
		t.Errorf("Expected no error from Write, got: %v", err)
	}
	if n != 50 {
		t.Errorf("Expected 50 bytes written, got %d", n)
	}
	if *wc.progress != 100 {
		t.Errorf("Expected progress to be 100, got %d", *wc.progress)
	}
}
