package validation

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

const ggmlMagic = 0x67676d6c // "ggml" in hex

// ValidateModelPath validates and secures the model file path
func ValidateModelPath(path string) (string, error) {
	// Clean the path to prevent path traversal
	cleanPath := filepath.Clean(path)
	
	// Check if file exists
	if _, err := os.Stat(cleanPath); err != nil {
		return "", fmt.Errorf("model file not found: %s", cleanPath)
	}
	
	// Validate GGML model header
	if err := ValidateGGMLHeader(cleanPath); err != nil {
		return "", fmt.Errorf("invalid model file: %w", err)
	}
	
	// Get absolute path for consistent validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve model path: %w", err)
	}
	
	return absPath, nil
}

// ValidateGGMLHeader validates that the file has a proper GGML header
func ValidateGGMLHeader(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()
	
	// Read the magic number (first 4 bytes)
	var magic uint32
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return fmt.Errorf("failed to read magic bytes: %w", err)
	}
	
	// Verify GGML magic number
	if magic != ggmlMagic {
		return fmt.Errorf("invalid GGML magic number: got 0x%x, expected 0x%x", magic, ggmlMagic)
	}
	
	// Basic sanity check on header size (should have at least basic parameters)
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	
	// GGML header should have magic + at least 11 int32 parameters (44 bytes minimum)
	if fileInfo.Size() < 48 {
		return fmt.Errorf("model file too small to be valid GGML format: %d bytes", fileInfo.Size())
	}
	
	return nil
}