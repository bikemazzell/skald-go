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

type Command struct {
	Action string `json:"action"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type KeyAction struct {
	Key     rune
	Action  string
	Desc    string
	Handler func() error
}

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

	keyActions     []KeyAction
	keyboardActive bool
	keyboardCtx    context.Context
	keyboardCancel context.CancelFunc
}

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

	s.setupKeyActions()

	return s, nil
}

func (s *Server) setupKeyActions() {
	s.keyActions = []KeyAction{}
	
	actionHandlers := map[string]func() error{
		"start": func() error {
			err := s.transcriber.Start()
			if err == nil {
				fmt.Println("\nTranscription started - listening for speech...")
			}
			return err
		},
		"stop": func() error {
			err := s.transcriber.Stop()
			if err == nil {
				fmt.Println("\nTranscription stopped")
			}
			return err
		},
		"status": func() error {
			isRunning := s.transcriber.IsRunning()
			status := "stopped"
			if isRunning {
				status = "running"
			}
			fmt.Printf("\nTranscriber status: %s\n\n", status)
			return nil
		},
		"quit": func() error {
			s.logger.Printf("Quit requested via keyboard")

			if s.cancel != nil {
				s.cancel()
			}

			if err := s.Stop(); err != nil {
				s.logger.Printf("Error stopping server: %v", err)
			}

			s.logger.Printf("Exiting application...")
			go func() {
				time.Sleep(100 * time.Millisecond)
				os.Exit(0)
			}()
			return nil
		},
		"help": func() error {
			return s.printKeyboardHelp()
		},
		"resume": func() error {
			s.logger.Printf("Manual resume requested")
			fmt.Println("\nManual resume triggered (not yet implemented)")
			return nil
		},
	}
	
	actionDescs := map[string]string{
		"start":  "Start transcription",
		"stop":   "Stop transcription", 
		"status": "Show transcriber status",
		"quit":   "Quit the application",
		"help":   "Show available commands",
		"resume": "Resume continuous recording",
	}
	
	for keyStr, action := range s.cfg.Server.Hotkeys {
		if len(keyStr) != 1 {
			s.logger.Printf("Warning: Invalid hotkey '%s' - must be single character", keyStr)
			continue
		}
		
		key := rune(keyStr[0])
		handler, exists := actionHandlers[action]
		if !exists {
			s.logger.Printf("Warning: Unknown action '%s' for hotkey '%s'", action, keyStr)
			continue
		}
		
		desc, hasDesc := actionDescs[action]
		if !hasDesc {
			desc = action
		}
		
		s.keyActions = append(s.keyActions, KeyAction{
			Key:     key,
			Action:  action,
			Desc:    desc,
			Handler: handler,
		})
	}
	
	if s.cfg.Verbose {
		s.logger.Printf("Configured %d hotkeys from settings", len(s.keyActions))
	}
}

func (s *Server) printKeyboardHelp() error {
	fmt.Println("\nAvailable commands:")
	for _, action := range s.keyActions {
		fmt.Printf("  %c - %s\n", action.Key, action.Desc)
	}
	fmt.Println()
	return nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	if err := s.ensureSocketPathIsSafe(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("socket path is unsafe: %w", err)
	}

	if err := os.Remove(s.cfg.Server.SocketPath); err != nil && !os.IsNotExist(err) {
		s.mu.Unlock()
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", s.cfg.Server.SocketPath)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to create socket: %w", err)
	}

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

	if s.cfg.Server.KeyboardEnabled {
		s.startKeyboardListener()
		s.printKeyboardHelp()
	}

	for {
		if err := s.listener.(*net.UnixListener).SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
			s.logger.Printf("Failed to set deadline: %v", err)
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if strings.Contains(err.Error(), "use of closed network connection") {
					return nil
				}
				return fmt.Errorf("accept error: %w", err)
			}
		}
		go s.handleConnection(conn)
	}
}

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

		var b = make([]byte, 1)
		for {
			select {
			case <-s.keyboardCtx.Done():
				return
			default:
				_ = os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				_, err := os.Stdin.Read(b)
				if err != nil {
					continue
				}

				s.handleKeyPress(rune(b[0]))
			}
		}
	}()
}

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

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("New connection received")

	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		s.logger.Printf("Failed to decode command: %v", err)
		return
	}
	s.logger.Printf("Received command: %s", cmd.Action)

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

	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		s.logger.Printf("Failed to encode response: %v", err)
		return
	}
	s.logger.Printf("Response sent")
}


func (s *Server) Stop() error {
	s.logger.Printf("Stopping server...")

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

func (s *Server) ensureSocketPathIsSafe() error {
	if s.cfg.Server.SocketPath == "" {
		return fmt.Errorf("socket path cannot be empty")
	}

	if !filepath.IsAbs(s.cfg.Server.SocketPath) {
		return fmt.Errorf("socket path must be absolute")
	}

	dir := filepath.Dir(s.cfg.Server.SocketPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("socket directory does not exist: %w", err)
		}
		return fmt.Errorf("failed to stat socket directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("socket path parent is not a directory")
	}

	if info, err := os.Stat(s.cfg.Server.SocketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("path exists but is not a socket")
		}

		conn, err := net.Dial("unix", s.cfg.Server.SocketPath)
		if err == nil {
			conn.Close()
			return fmt.Errorf("socket is already in use by another process")
		}
	}

	return nil
}
