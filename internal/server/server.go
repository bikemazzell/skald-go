package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"skald/internal/config"
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

// Server handles client connections and manages the transcriber
type Server struct {
	cfg         *config.Config
	transcriber *transcriber.Transcriber
	listener    net.Listener
	logger      *log.Logger

	mu        sync.Mutex
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new Server instance
func New(cfg *config.Config, logger *log.Logger) (*Server, error) {
	t, err := transcriber.New(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcriber: %w", err)
	}

	return &Server{
		cfg:         cfg,
		transcriber: t,
		logger:      logger,
	}, nil
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

	s.listener = listener
	s.isRunning = true
	s.mu.Unlock()

	s.logger.Printf("Server listening on %s", s.cfg.Server.SocketPath)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil // Normal shutdown
			default:
				return fmt.Errorf("accept error: %w", err)
			}
		}
		go s.handleConnection(conn)
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

func (s *Server) sendResponse(conn net.Conn, resp Response) {
	s.logger.Printf("Attempting to send response: %+v", resp)
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		s.logger.Printf("Failed to send response: %v", err)
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.logger.Printf("Stopping server...")

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
