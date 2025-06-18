package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skald/internal/config"
	"skald/internal/model"
	"skald/internal/transcriber"
)

// Command represents a client command
type Command struct {
	Action string `json:"action"` // "start", "stop", or "status"
}

// Response represents a server response
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// KeyAction represents a keyboard action
type KeyAction struct {
	Key     rune
	Action  string
	Desc    string
	Handler func() error
}

// Server handles client connections and manages the transcriber
type Server struct {
	cfg         *config.Config
	transcriber *transcriber.Transcriber
	listener    net.Listener
	logger      *log.Logger
	modelMgr    *model.ModelManager

	mu        sync.Mutex
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc

	// Keyboard interaction
	keyActions     []KeyAction
	keyboardActive bool
	keyboardCtx    context.Context
	keyboardCancel context.CancelFunc
}

// New creates a new Server instance
func New(cfg *config.Config, logger *log.Logger, modelMgr *model.ModelManager) (*Server, error) {
	t, err := transcriber.New(cfg, logger, modelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcriber: %w", err)
	}

	s := &Server{
		cfg:         cfg,
		transcriber: t,
		logger:      logger,
		modelMgr:    modelMgr,
	}

	// Setup keyboard actions
	s.setupKeyActions()

	return s, nil
}

// setupKeyActions configures the available keyboard actions
func (s *Server) setupKeyActions() {
	s.keyActions = []KeyAction{
		{
			Key:    'r',
			Action: "start",
			Desc:   "Start transcription",
			Handler: func() error {
				return s.transcriber.Start()
			},
		},
		{
			Key:    's',
			Action: "stop",
			Desc:   "Stop transcription",
			Handler: func() error {
				return s.transcriber.Stop()
			},
		},
		{
			Key:    'i',
			Action: "status",
			Desc:   "Show transcriber status",
			Handler: func() error {
				isRunning := s.transcriber.IsRunning()
				status := "stopped"
				if isRunning {
					status = "running"
				}
				fmt.Printf("\nTranscriber status: %s\n\n", status)
				return nil
			},
		},
		{
			Key:    'q',
			Action: "quit",
			Desc:   "Quit the application",
			Handler: func() error {
				s.logger.Printf("Quit requested via keyboard")

				// Cancel the context to signal shutdown first
				if s.cancel != nil {
					s.cancel()
				}

				// Stop the server
				if err := s.Stop(); err != nil {
					s.logger.Printf("Error stopping server: %v", err)
				}

				// Exit the application
				s.logger.Printf("Exiting application...")
				go func() {
					// Give a short delay to allow logs to be written
					time.Sleep(100 * time.Millisecond)
					os.Exit(0)
				}()
				return nil
			},
		},
		{
			Key:    '?',
			Action: "help",
			Desc:   "Show available commands",
			Handler: func() error {
				fmt.Println("\nAvailable commands:")
				fmt.Println("  r - Start transcription")
				fmt.Println("  s - Stop transcription")
				fmt.Println("  i - Show transcriber status")
				fmt.Println("  q - Quit the application")
				fmt.Println("  ? - Show this help")
				return nil
			},
		},
	}
}

// printKeyboardHelp displays available keyboard commands
func (s *Server) printKeyboardHelp() {
	fmt.Println("\nCommands: r=record, s=stop, i=status, q=quit, ?=help")
}

// Start begins listening for connections
func (s *Server) Start() error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	// Create context
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Check if socket path is safe to use
	if err := s.ensureSocketPathIsSafe(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("socket path is unsafe: %w", err)
	}

	// Remove existing socket file if it exists
	if err := os.Remove(s.cfg.Server.SocketPath); err != nil && !os.IsNotExist(err) {
		s.mu.Unlock()
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", s.cfg.Server.SocketPath)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to create socket: %w", err)
	}

	// Set restrictive permissions on the socket (owner read/write only)
	if err := os.Chmod(s.cfg.Server.SocketPath, 0600); err != nil {
		listener.Close()
		s.mu.Unlock()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener
	s.isRunning = true
	s.mu.Unlock()

	if s.cfg.Verbose {
		s.logger.Printf("Server listening on %s", s.cfg.Server.SocketPath)
	}

	// Start keyboard listener in a separate goroutine if enabled
	if s.cfg.Server.KeyboardEnabled {
		s.startKeyboardListener()
		// Print available keyboard commands
		s.printKeyboardHelp()
	}

	// Accept connections
	for {
		// Use a deadline to allow for graceful shutdown
		if err := s.listener.(*net.UnixListener).SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
			s.logger.Printf("Failed to set deadline: %v", err)
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				// Normal shutdown, return without error
				return nil
			default:
				// Check if it's a timeout error, which we can ignore
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// This is just a timeout, continue the loop
					continue
				}
				// If the listener is closed, exit gracefully
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				// Otherwise, it's an unexpected error
				return fmt.Errorf("accept error: %w", err)
			}
		}
		go s.handleConnection(conn)
	}
}

// startKeyboardListener starts a goroutine to listen for keyboard input
func (s *Server) startKeyboardListener() {
	s.keyboardCtx, s.keyboardCancel = context.WithCancel(s.ctx)
	s.keyboardActive = true

	go func() {
		defer func() {
			s.keyboardActive = false
		}()

		if s.cfg.Verbose {
			s.logger.Printf("Keyboard listener started. Press '?' for help.")
		}

		// Buffer for reading a single character
		var b = make([]byte, 1)
		for {
			select {
			case <-s.keyboardCtx.Done():
				return
			default:
				// Non-blocking read from stdin
				_ = os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				_, err := os.Stdin.Read(b)
				if err != nil {
					// Timeout or other error, just continue
					continue
				}

				// Process the key
				s.handleKeyPress(rune(b[0]))
			}
		}
	}()
}

// handleKeyPress processes keyboard input
func (s *Server) handleKeyPress(key rune) {
	for _, action := range s.keyActions {
		if action.Key == key {
			if s.cfg.Verbose {
				s.logger.Printf("Executing keyboard action: %s", action.Desc)
			}
			if err := action.Handler(); err != nil {
				s.logger.Printf("Error executing action: %v", err)
			}
			return
		}
	}
}

// handleConnection processes a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("New connection received")

	// Read command
	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		s.logger.Printf("Failed to decode command: %v", err)
		return
	}
	s.logger.Printf("Received command: %s", cmd.Action)

	// Process command and prepare response
	var resp Response
	switch cmd.Action {
	case "start":
		if err := s.transcriber.Start(); err != nil {
			resp = Response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to start transcriber: %v", err),
			}
		} else {
			resp = Response{
				Status:  "success",
				Message: "Transcriber started",
			}
		}
	case "stop":
		if err := s.transcriber.Stop(); err != nil {
			resp = Response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to stop transcriber: %v", err),
			}
		} else {
			resp = Response{
				Status:  "success",
				Message: "Transcriber stopped",
			}
		}
	case "status":
		// Get transcriber status
		isRunning := s.transcriber.IsRunning()
		resp = Response{
			Status:  "success",
			Message: fmt.Sprintf("Transcriber is %s", map[bool]string{true: "running", false: "stopped"}[isRunning]),
		}
	default:
		resp = Response{
			Status: "error",
			Error:  "Invalid command",
		}
	}

	// Send response
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		s.logger.Printf("Failed to encode response: %v", err)
		return
	}
	s.logger.Printf("Response sent")
}


// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.logger.Printf("Stopping server...")

	// Stop keyboard listener
	if s.keyboardActive && s.keyboardCancel != nil {
		s.keyboardCancel()
	}

	if s.transcriber != nil {
		if err := s.transcriber.Close(); err != nil {
			s.logger.Printf("Error closing transcriber: %v", err)
		}
	}

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	return nil
}

// Add error types for better error handling
type ServerError struct {
	Code    string
	Message string
	Err     error
}

func (e *ServerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

const (
	ErrServerAlreadyRunning = "SERVER_RUNNING"
	ErrInvalidCommand       = "INVALID_COMMAND"
	ErrTranscriberFailed    = "TRANSCRIBER_FAILED"
)

func NewServerError(code string, message string, err error) *ServerError {
	return &ServerError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// ensureSocketPathIsSafe checks if the socket path is safe to use
func (s *Server) ensureSocketPathIsSafe() error {
	// Validate socket path
	if s.cfg.Server.SocketPath == "" {
		return fmt.Errorf("socket path cannot be empty")
	}

	// Check if the socket path is absolute
	if !filepath.IsAbs(s.cfg.Server.SocketPath) {
		return fmt.Errorf("socket path must be absolute")
	}

	// Check if the directory exists and is writable
	dir := filepath.Dir(s.cfg.Server.SocketPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("socket directory does not exist: %w", err)
		}
		return fmt.Errorf("failed to stat socket directory: %w", err)
	}

	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("socket path parent is not a directory")
	}

	// Check if the socket file exists and is a socket
	if info, err := os.Stat(s.cfg.Server.SocketPath); err == nil {
		// File exists, check if it's a socket
		if info.Mode()&os.ModeSocket == 0 {
			// Not a socket, could be a regular file or something else
			return fmt.Errorf("path exists but is not a socket")
		}

		// It's a socket, check if it's stale
		conn, err := net.Dial("unix", s.cfg.Server.SocketPath)
		if err == nil {
			// Socket is active
			conn.Close()
			return fmt.Errorf("socket is already in use by another process")
		}
		// Socket is stale, we can remove it
	}

	return nil
}
