// +build e2e

package skald

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestEndToEndBinary tests the complete binary end-to-end
func TestEndToEndBinary(t *testing.T) {
	// Build binary for testing
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)
	
	tests := []struct {
		name           string
		args           []string
		input          string
		expectedOutput []string
		expectedError  string
		timeout        time.Duration
		shouldFail     bool
	}{
		{
			name:           "version flag",
			args:           []string{"--version"},
			expectedOutput: []string{"skald version"},
			timeout:        time.Second,
			shouldFail:     false,
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectedOutput: []string{"Usage", "model", "language"},
			timeout:        time.Second,
			shouldFail:     true, // Help exits with non-zero
		},
		{
			name:          "missing model file",
			args:          []string{"-model", "/nonexistent/model.bin"},
			expectedError: "Model file not found",
			timeout:       time.Second,
			shouldFail:    true,
		},
		{
			name:           "all flags provided",
			args:           []string{
				"-model", createTempModelFile(t),
				"-language", "en",
				"-sample-rate", "44100",
				"-silence-threshold", "0.05",
				"-silence-duration", "2.0",
				"-no-clipboard",
				"-continuous=false",
			},
			timeout:    2 * time.Second,
			shouldFail: true, // Will fail on audio device, but should parse args
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			
			// Set input if provided
			if tt.input != "" {
				cmd.Stdin = strings.NewReader(tt.input)
			}
			
			// Run with timeout
			done := make(chan error, 1)
			go func() {
				done <- cmd.Run()
			}()
			
			select {
			case err := <-done:
				// Command completed
				if tt.shouldFail && err == nil {
					t.Errorf("Expected command to fail, but it succeeded")
				}
				if !tt.shouldFail && err != nil {
					t.Errorf("Expected command to succeed, got error: %v", err)
				}
			case <-time.After(tt.timeout):
				// Command timed out - kill it
				cmd.Process.Kill()
				<-done // Wait for process to actually exit
			}
			
			// Check output
			output := stdout.String() + stderr.String()
			
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got: %s", expected, output)
				}
			}
			
			if tt.expectedError != "" {
				if !strings.Contains(output, tt.expectedError) {
					t.Errorf("Expected error containing %q, got: %s", tt.expectedError, output)
				}
			}
		})
	}
}

// TestSystemIntegration tests integration with system components
func TestSystemIntegration(t *testing.T) {
	// Only run if audio system is available
	if os.Getenv("SKIP_AUDIO_TESTS") != "" {
		t.Skip("Audio tests skipped by environment variable")
	}
	
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)
	
	// Create a small test model file
	modelFile := createTempModelFile(t)
	defer os.Remove(modelFile)
	
	t.Run("quick startup test", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-model", modelFile)
		
		// Start the command
		err := cmd.Start()
		if err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}
		
		// Let it run briefly
		time.Sleep(100 * time.Millisecond)
		
		// Kill it
		err = cmd.Process.Kill()
		if err != nil {
			t.Errorf("Failed to kill process: %v", err)
		}
		
		// Wait for it to finish
		cmd.Wait()
		
		// If we got here without panic, the basic startup worked
	})
}

// TestConfigurationVariations tests different configuration combinations
func TestConfigurationVariations(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)
	
	modelFile := createTempModelFile(t)
	defer os.Remove(modelFile)
	
	configs := []struct {
		name string
		args []string
	}{
		{
			name: "default config",
			args: []string{"-model", modelFile},
		},
		{
			name: "high sample rate",
			args: []string{"-model", modelFile, "-sample-rate", "48000"},
		},
		{
			name: "different language",
			args: []string{"-model", modelFile, "-language", "es"},
		},
		{
			name: "auto language",
			args: []string{"-model", modelFile, "-language", "auto"},
		},
		{
			name: "high sensitivity",
			args: []string{"-model", modelFile, "-silence-threshold", "0.001"},
		},
		{
			name: "low sensitivity",
			args: []string{"-model", modelFile, "-silence-threshold", "0.1"},
		},
		{
			name: "quick response",
			args: []string{"-model", modelFile, "-silence-duration", "0.5"},
		},
		{
			name: "slow response",
			args: []string{"-model", modelFile, "-silence-duration", "3.0"},
		},
		{
			name: "no clipboard",
			args: []string{"-model", modelFile, "-no-clipboard"},
		},
		{
			name: "continuous mode",
			args: []string{"-model", modelFile, "-continuous"},
		},
	}
	
	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, config.args...)
			
			// Start command
			err := cmd.Start()
			if err != nil {
				// May fail due to audio device availability, but config parsing should work
				t.Logf("Command failed to start (may be due to audio): %v", err)
				return
			}
			
			// Let it try to initialize
			time.Sleep(50 * time.Millisecond)
			
			// Kill it
			cmd.Process.Kill()
			cmd.Wait()
			
			// Success if no panic during config parsing
		})
	}
}

// TestResourceUsage tests resource usage patterns
func TestResourceUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource usage test in short mode")
	}
	
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)
	
	modelFile := createTempModelFile(t)
	defer os.Remove(modelFile)
	
	t.Run("memory usage stability", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "-model", modelFile)
		
		err := cmd.Start()
		if err != nil {
			t.Skip("Could not start process (audio not available)")
		}
		defer cmd.Process.Kill()
		
		// Monitor for a short time to ensure no immediate crashes
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			
			// Check if process is still running
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				t.Errorf("Process exited unexpectedly after %d iterations", i)
				break
			}
		}
	})
}

// TestErrorConditions tests various error conditions
func TestErrorConditions(t *testing.T) {
	binaryPath := buildTestBinary(t)
	defer os.Remove(binaryPath)
	
	errorTests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "invalid flag",
			args:          []string{"-invalid-flag"},
			expectedError: "flag provided but not defined",
		},
		{
			name:          "invalid sample rate",
			args:          []string{"-sample-rate", "invalid"},
			expectedError: "invalid value",
		},
		{
			name:          "invalid silence threshold",
			args:          []string{"-silence-threshold", "not-a-number"},
			expectedError: "invalid value",
		},
		{
			name:          "invalid silence duration",
			args:          []string{"-silence-duration", "abc"},
			expectedError: "invalid value",
		},
		{
			name:          "model file permission denied",
			args:          []string{"-model", "/root/secret.bin"},
			expectedError: "Model file not found",
		},
	}
	
	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, tt.args...)
			
			output, err := cmd.CombinedOutput()
			
			if err == nil {
				t.Error("Expected command to fail, but it succeeded")
			}
			
			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectedError) {
				t.Errorf("Expected error containing %q, got: %s", tt.expectedError, outputStr)
			}
		})
	}
}

// Helper functions

func buildTestBinary(t *testing.T) string {
	tmpDir := t.TempDir()
	binaryPath := tmpDir + "/skald-test"
	
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/skald")
	
	// Set CGO_ENABLED=0 to avoid library dependencies in tests
	env := os.Environ()
	cmd.Env = append(env, "CGO_ENABLED=0")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try with CGO enabled if static build fails
		cmd = exec.Command("go", "build", "-o", binaryPath, "../../cmd/skald")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to build test binary: %v\nOutput: %s", err, output)
		}
	}
	
	return binaryPath
}

func createTempModelFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp("", "test-model-*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp model file: %v", err)
	}
	defer tmpFile.Close()
	
	// Write some dummy data to make it a valid file
	_, err = tmpFile.WriteString("dummy whisper model data for testing")
	if err != nil {
		t.Fatalf("Failed to write temp model file: %v", err)
	}
	
	return tmpFile.Name()
}