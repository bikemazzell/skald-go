package validation

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

func TestValidateModelPath(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (string, func())
		expectError   bool
		errorContains string
	}{
		{
			name: "valid GGML model file",
			setupFunc: func() (string, func()) {
				return createValidGGMLFile(t)
			},
			expectError: false,
		},
		{
			name: "non-existent file",
			setupFunc: func() (string, func()) {
				return "/non/existent/file.bin", func() {}
			},
			expectError:   true,
			errorContains: "model file not found",
		},
		{
			name: "path traversal attempt cleaned",
			setupFunc: func() (string, func()) {
				// Create a valid file and try path with ../ which should be cleaned
				path, cleanup := createValidGGMLFile(t)
				// Create a path like "/tmp/../tmp/file" which should resolve to "/tmp/file"
				return "/tmp/../" + path, cleanup
			},
			expectError: false, // Should work after cleaning the path
		},
		{
			name: "file with invalid magic bytes",
			setupFunc: func() (string, func()) {
				return createInvalidGGMLFile(t)
			},
			expectError:   true,
			errorContains: "invalid GGML magic number",
		},
		{
			name: "file too small",
			setupFunc: func() (string, func()) {
				return createTooSmallFile(t)
			},
			expectError:   true,
			errorContains: "model file too small",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupFunc()
			defer cleanup()

			result, err := ValidateModelPath(path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result path")
				}
			}
		})
	}
}

func TestValidateGGMLHeader(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (string, func())
		expectError   bool
		errorContains string
	}{
		{
			name: "valid GGML header",
			setupFunc: func() (string, func()) {
				return createValidGGMLFile(t)
			},
			expectError: false,
		},
		{
			name: "invalid magic bytes",
			setupFunc: func() (string, func()) {
				return createInvalidGGMLFile(t)
			},
			expectError:   true,
			errorContains: "invalid GGML magic number",
		},
		{
			name: "file too small",
			setupFunc: func() (string, func()) {
				return createTooSmallFile(t)
			},
			expectError:   true,
			errorContains: "model file too small",
		},
		{
			name: "empty file",
			setupFunc: func() (string, func()) {
				return createEmptyFile(t)
			},
			expectError:   true,
			errorContains: "failed to read magic bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setupFunc()
			defer cleanup()

			err := ValidateGGMLHeader(path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper functions for creating test files

func createValidGGMLFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "test_ggml_*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Write GGML magic number
	err = binary.Write(tmpFile, binary.LittleEndian, uint32(ggmlMagic))
	if err != nil {
		t.Fatalf("Failed to write magic: %v", err)
	}

	// Write dummy header parameters (11 int32 values)
	for i := 0; i < 11; i++ {
		err = binary.Write(tmpFile, binary.LittleEndian, int32(i+1))
		if err != nil {
			t.Fatalf("Failed to write header param %d: %v", i, err)
		}
	}

	tmpFile.Close()
	
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}

func createInvalidGGMLFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "test_invalid_ggml_*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Write invalid magic number
	err = binary.Write(tmpFile, binary.LittleEndian, uint32(0x12345678))
	if err != nil {
		t.Fatalf("Failed to write invalid magic: %v", err)
	}

	// Write dummy data to make it large enough
	for i := 0; i < 11; i++ {
		err = binary.Write(tmpFile, binary.LittleEndian, int32(i+1))
		if err != nil {
			t.Fatalf("Failed to write dummy data %d: %v", i, err)
		}
	}

	tmpFile.Close()
	
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}

func createTooSmallFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "test_small_*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Write GGML magic number but make file too small
	err = binary.Write(tmpFile, binary.LittleEndian, uint32(ggmlMagic))
	if err != nil {
		t.Fatalf("Failed to write magic: %v", err)
	}

	tmpFile.Close()
	
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}

func createEmptyFile(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "test_empty_*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tmpFile.Close()
	
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }
}