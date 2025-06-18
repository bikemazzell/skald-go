package server

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"skald/internal/config"
	"skald/internal/model"
)

func TestEnsureSocketPathIsSafe(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "server-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config
	cfg := &config.Config{
		Server: struct {
			SocketPath      string            `json:"socket_path"`
			SocketTimeout   float32           `json:"socket_timeout"`
			KeyboardEnabled bool              `json:"keyboard_enabled"`
			Hotkeys         map[string]string `json:"hotkeys"`
		}{
			SocketPath:      filepath.Join(tempDir, "test.sock"),
			SocketTimeout:   5.0,
			KeyboardEnabled: false,
			Hotkeys:         map[string]string{},
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)

	// Create a server instance
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	// Test with valid socket path
	err = s.ensureSocketPathIsSafe()
	if err != nil {
		t.Errorf("Expected no error for valid socket path, got: %v", err)
	}

	// Test with empty socket path
	s.cfg.Server.SocketPath = ""
	err = s.ensureSocketPathIsSafe()
	if err == nil {
		t.Error("Expected error for empty socket path, got nil")
	}

	// Test with non-absolute socket path
	s.cfg.Server.SocketPath = "relative/path.sock"
	err = s.ensureSocketPathIsSafe()
	if err == nil {
		t.Error("Expected error for non-absolute socket path, got nil")
	}

	// Test with non-existent directory
	s.cfg.Server.SocketPath = filepath.Join(tempDir, "nonexistent", "test.sock")
	err = s.ensureSocketPathIsSafe()
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}

	// Test with file instead of socket
	filePath := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(filePath, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	s.cfg.Server.SocketPath = filePath
	err = s.ensureSocketPathIsSafe()
	if err == nil {
		t.Error("Expected error for file instead of socket, got nil")
	}
}

func TestServerError(t *testing.T) {
	// Test ServerError creation and formatting
	err := NewServerError(ErrServerAlreadyRunning, "Server is already running", nil)

	if err.Code != ErrServerAlreadyRunning {
		t.Errorf("Expected error code %s, got %s", ErrServerAlreadyRunning, err.Code)
	}

	if err.Message != "Server is already running" {
		t.Errorf("Expected error message 'Server is already running', got '%s'", err.Message)
	}

	// Test error string formatting without underlying error
	errStr := err.Error()
	expected := "SERVER_RUNNING: Server is already running"
	if errStr != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, errStr)
	}

	// Test with underlying error
	underlyingErr := NewServerError(ErrInvalidCommand, "Invalid command", nil)
	err = NewServerError(ErrTranscriberFailed, "Transcriber failed", underlyingErr)

	_ = err.Error() // Just verifying it doesn't panic
	if err.Code != ErrTranscriberFailed {
		t.Errorf("Expected error code %s, got %s", ErrTranscriberFailed, err.Code)
	}
}

func TestKeyActions(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Server: struct {
			SocketPath      string            `json:"socket_path"`
			SocketTimeout   float32           `json:"socket_timeout"`
			KeyboardEnabled bool              `json:"keyboard_enabled"`
			Hotkeys         map[string]string `json:"hotkeys"`
		}{
			SocketPath:      "/tmp/test.sock",
			SocketTimeout:   5.0,
			KeyboardEnabled: true,
			Hotkeys: map[string]string{
				"r": "start",
				"s": "stop",
				"i": "status",
				"q": "quit",
				"?": "help",
			},
		},
	}

	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	modelMgr := &model.ModelManager{}

	// Create a server instance
	s := &Server{
		cfg:      cfg,
		logger:   logger,
		modelMgr: modelMgr,
	}

	// Setup key actions
	s.setupKeyActions()

	// Check that key actions were initialized
	if len(s.keyActions) == 0 {
		t.Fatal("Expected key actions to be initialized")
	}

	// Check specific key actions
	foundStart := false
	foundStop := false
	foundStatus := false
	foundQuit := false
	foundHelp := false

	for _, action := range s.keyActions {
		switch action.Key {
		case 'r':
			foundStart = true
			if action.Action != "start" {
				t.Errorf("Expected action 'start' for key 'r', got '%s'", action.Action)
			}
		case 's':
			foundStop = true
			if action.Action != "stop" {
				t.Errorf("Expected action 'stop' for key 's', got '%s'", action.Action)
			}
		case 'i':
			foundStatus = true
			if action.Action != "status" {
				t.Errorf("Expected action 'status' for key 'i', got '%s'", action.Action)
			}
		case 'q':
			foundQuit = true
			if action.Action != "quit" {
				t.Errorf("Expected action 'quit' for key 'q', got '%s'", action.Action)
			}
		case '?':
			foundHelp = true
			if action.Action != "help" {
				t.Errorf("Expected action 'help' for key '?', got '%s'", action.Action)
			}
		}
	}

	if !foundStart {
		t.Error("Missing key action for 'r' (start)")
	}
	if !foundStop {
		t.Error("Missing key action for 's' (stop)")
	}
	if !foundStatus {
		t.Error("Missing key action for 'i' (status)")
	}
	if !foundQuit {
		t.Error("Missing key action for 'q' (quit)")
	}
	if !foundHelp {
		t.Error("Missing key action for '?' (help)")
	}
}
