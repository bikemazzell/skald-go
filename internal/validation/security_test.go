package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateModelPathStrictSecurity(t *testing.T) {
	// Create temporary test directories
	tempDir := t.TempDir()
	allowedDir := filepath.Join(tempDir, "allowed")
	forbiddenDir := filepath.Join(tempDir, "forbidden")
	
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(forbiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create a dummy GGML file in allowed directory
	allowedFile := filepath.Join(allowedDir, "test.bin")
	if err := createDummyGGMLFile(allowedFile); err != nil {
		t.Fatal(err)
	}
	
	// Create a dummy GGML file in forbidden directory
	forbiddenFile := filepath.Join(forbiddenDir, "test.bin")
	if err := createDummyGGMLFile(forbiddenFile); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		wantErr     bool
	}{
		{
			name:        "File in allowed directory",
			path:        allowedFile,
			allowedDirs: []string{allowedDir},
			wantErr:     false,
		},
		{
			name:        "File in forbidden directory",
			path:        forbiddenFile,
			allowedDirs: []string{allowedDir},
			wantErr:     true,
		},
		{
			name:        "No directory restrictions",
			path:        forbiddenFile,
			allowedDirs: []string{},
			wantErr:     false,
		},
		{
			name:        "Multiple allowed directories",
			path:        forbiddenFile,
			allowedDirs: []string{allowedDir, forbiddenDir},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateModelPathStrict(tt.path, tt.allowedDirs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModelPathStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathTraversalProtection(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	allowedDir := filepath.Join(tempDir, "allowed")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	// Create a dummy GGML file
	testFile := filepath.Join(allowedDir, "test.bin")
	if err := createDummyGGMLFile(testFile); err != nil {
		t.Fatal(err)
	}

	traversalPaths := []string{
		filepath.Join(allowedDir, "../../../test.bin"),
		filepath.Join(allowedDir, "..\\..\\..\\test.bin"),
		allowedDir + "/../test.bin",
		allowedDir + "/./test.bin",
		allowedDir + "//test.bin",
	}

	for _, maliciousPath := range traversalPaths {
		t.Run("PathTraversal", func(t *testing.T) {
			// Even if the file doesn't exist, the path cleaning should prevent traversal
			_, err := ValidateModelPathStrict(maliciousPath, []string{allowedDir})
			// Most should fail due to file not existing or path restrictions
			if err == nil {
				// If it passes, make sure the resolved path is still within allowed directory
				cleanPath := filepath.Clean(maliciousPath)
				absPath, _ := filepath.Abs(cleanPath)
				absAllowed, _ := filepath.Abs(allowedDir)
				if !filepath.HasPrefix(absPath, absAllowed) {
					t.Errorf("Path traversal not properly prevented for: %s", maliciousPath)
				}
			}
		})
	}
}

// createDummyGGMLFile creates a minimal GGML file for testing
func createDummyGGMLFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write GGML magic number (0x67676d6c = "ggml")
	magicBytes := []byte{0x6c, 0x6d, 0x67, 0x67} // little endian
	if _, err := file.Write(magicBytes); err != nil {
		return err
	}
	
	// Write dummy header data (44 more bytes to meet minimum size requirement)
	dummyHeader := make([]byte, 44)
	if _, err := file.Write(dummyHeader); err != nil {
		return err
	}
	
	return nil
}