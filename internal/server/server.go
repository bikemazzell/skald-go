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
	"syscall"
	"time"

	"skald/internal/config"
	"skald/internal/model"
	"skald/internal/transcriber"
)

type Command struct {
	Action  string            `json:"action"`
	Options map[string]string `json:"options,omitempty"`
}

type Response struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
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

	keyActions     map[rune]KeyAction
	keyboardActive bool

	statsMu        sync.RWMutex
	stats          ServerStats
	keyboardCtx    context.Context
	keyboardCancel context.CancelFunc
}

type ServerStats struct {
	StartTime             time.Time
	LastTranscriptionTime time.Time
	TranscriptionCount    int
	ErrorCount            int
	LastError             string
	LastErrorTime         time.Time
	CurrentState          string
	SessionDuration       time.Duration
	RecentLogs            []LogEntry
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

func (s *Server) logActivity(level, message string) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	s.stats.RecentLogs = append(s.stats.RecentLogs, entry)
	if len(s.stats.RecentLogs) > 100 {
		s.stats.RecentLogs = s.stats.RecentLogs[1:]
	}
}

func (s *Server) updateErrorStats(err error) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.ErrorCount++
	s.stats.LastError = err.Error()
	s.stats.LastErrorTime = time.Now()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "ERROR",
		Message:   err.Error(),
	}

	s.stats.RecentLogs = append(s.stats.RecentLogs, entry)
	if len(s.stats.RecentLogs) > 100 {
		s.stats.RecentLogs = s.stats.RecentLogs[1:]
	}
}

func (s *Server) updateStateStats(state string) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()

	s.stats.CurrentState = state
	if state == "transcribing" {
		s.stats.TranscriptionCount++
		s.stats.LastTranscriptionTime = time.Now()
	}
}

func (s *Server) buildStatusResponse(isRunning bool, verbose bool) Response {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	status := "stopped"
	if isRunning {
		status = "running"
	}

	resp := Response{
		Status:  "success",
		Message: fmt.Sprintf("Transcriber is %s", status),
	}

	if verbose {
		resp.Data = map[string]interface{}{
			"state":               s.stats.CurrentState,
			"uptime":              time.Since(s.stats.StartTime).String(),
			"transcription_count": s.stats.TranscriptionCount,
			"error_count":         s.stats.ErrorCount,
			"last_transcription":  s.formatTime(s.stats.LastTranscriptionTime),
			"last_error":          s.stats.LastError,
			"last_error_time":     s.formatTime(s.stats.LastErrorTime),
			"continuous_mode":     s.cfg.Processing.ContinuousMode.Enabled,
			"model":               s.cfg.Whisper.Model,
			"language":            s.cfg.Whisper.Language,
		}
	}

	return resp
}

func (s *Server) buildLogsResponse() Response {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()

	logs := make([]map[string]string, len(s.stats.RecentLogs))
	for i, log := range s.stats.RecentLogs {
		logs[i] = map[string]string{
			"timestamp": log.Timestamp.Format("2006-01-02 15:04:05"),
			"level":     log.Level,
			"message":   log.Message,
		}
	}

	return Response{
		Status:  "success",
		Message: fmt.Sprintf("Retrieved %d log entries", len(logs)),
		Data: map[string]interface{}{
			"logs": logs,
		},
	}
}

func (s *Server) formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format("2006-01-02 15:04:05")
}

func New(cfg *config.Config, logger *log.Logger, modelMgr *model.ModelManager, t *transcriber.Transcriber) *Server {
	s := &Server{
		cfg:         cfg,
		transcriber: t,
		logger:      logger,
		modelMgr:    modelMgr,
		stats: ServerStats{
			StartTime:    time.Now(),
			CurrentState: "initialized",
			RecentLogs:   make([]LogEntry, 0, 100),
		},
	}
	s.setupKeyActions()
	return s
}

func NewDefaultServer(cfg *config.Config, logger *log.Logger, modelMgr *model.ModelManager) (*Server, error) {
	t, err := transcriber.New(cfg, logger, modelMgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcriber: %w", err)
	}
	return New(cfg, logger, modelMgr, t), nil
}

func (s *Server) setupKeyActions() {
	s.keyActions = map[rune]KeyAction{
		'r': {Key: 'r', Action: "start", Desc: "Start transcription", Handler: s.handleStart},
		's': {Key: 's', Action: "stop", Desc: "Stop transcription", Handler: s.handleStop},
		'i': {Key: 'i', Action: "status", Desc: "Show status", Handler: s.handleStatus},
		'q': {Key: 'q', Action: "quit", Desc: "Quit", Handler: s.handleQuit},
		'?': {Key: '?', Action: "help", Desc: "Show help", Handler: s.handleHelp},
		'c': {Key: 'c', Action: "resume", Desc: "Resume continuous mode", Handler: s.handleResume},
	}
}

func (s *Server) handleStart() error {
	err := s.transcriber.Start()
	if err == nil {
		fmt.Println("\nTranscription started - listening for speech...")
	}
	return err
}

func (s *Server) handleStop() error {
	err := s.transcriber.Stop()
	if err == nil {
		fmt.Println("\nTranscription stopped")
	}
	return err
}

func (s *Server) handleStatus() error {
	isRunning := s.transcriber.IsRunning()
	status := "stopped"
	if isRunning {
		status = "running"
	}
	fmt.Printf("\nTranscriber status: %s\n\n", status)
	return nil
}

func (s *Server) handleQuit() error {
	s.logger.Printf("Quit requested via keyboard")
	go func() {
		time.Sleep(10 * time.Millisecond)
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			s.logger.Printf("Error sending interrupt signal: %v", err)
		}
	}()
	return nil
}

func (s *Server) handleHelp() error {
	return s.printKeyboardHelp()
}

func (s *Server) handleResume() error {
	// Implementation for resuming continuous mode would go here.
	// This is a placeholder.
	fmt.Println("\nResuming continuous mode...")
	return nil
}

func (s *Server) printKeyboardHelp() error {
	fmt.Println("\nAvailable keyboard commands:")
	// The keyActions map is not ordered, so we need to define order for printing
	printOrder := []rune{'r', 's', 'i', 'c', '?', 'q'}
	for _, key := range printOrder {
		if action, ok := s.keyActions[key]; ok {
			fmt.Printf("  %c: %s\n", action.Key, action.Desc)
		}
	}
	fmt.Println()
	return nil
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.isRunning = true
	s.mu.Unlock()

	// Start keyboard listener if enabled
	if s.cfg.Server.KeyboardEnabled {
		s.keyboardCtx, s.keyboardCancel = context.WithCancel(context.Background())
		go s.startKeyboardListener()
	}

	// Start the main server loop in a goroutine
	go s.run()

	s.logger.Println("Server started successfully")
	return nil
}

func (s *Server) run() {
	if err := s.ensureSocketPathIsSafe(); err != nil {
		s.logger.Fatalf("Socket path validation failed: %v", err)
	}

	var lc net.ListenConfig
	listener, err := lc.Listen(s.ctx, "unix", s.cfg.Server.SocketPath)
	if err != nil {
		s.logger.Fatalf("Failed to listen on socket: %v", err)
	}
	s.listener = listener

	s.logger.Printf("Server listening on %s", s.cfg.Server.SocketPath)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Printf("Failed to accept connection: %v", err)
			}
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("Client connected")

	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		s.logger.Printf("Failed to decode command: %v", err)
		return
	}
	s.logger.Printf("Received command: %s", cmd.Action)
	s.logActivity("INFO", fmt.Sprintf("Command received: %s", cmd.Action))

	var resp Response
	switch cmd.Action {
	case "start":
		continuous := cmd.Options["continuous"] == "true"
		if continuous {
			s.cfg.Processing.ContinuousMode.Enabled = true
		}
		if err := s.transcriber.Start(); err != nil {
			s.updateErrorStats(err)
			resp = Response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to start transcriber: %v", err),
			}
		} else {
			s.updateStateStats("running")
			s.logActivity("INFO", "Transcriber started")
			resp = Response{
				Status:  "success",
				Message: "Transcriber started",
			}
		}
	case "stop":
		if err := s.transcriber.Stop(); err != nil {
			s.updateErrorStats(err)
			resp = Response{
				Status: "error",
				Error:  fmt.Sprintf("Failed to stop transcriber: %v", err),
			}
		} else {
			s.updateStateStats("stopped")
			s.logActivity("INFO", "Transcriber stopped")
			resp = Response{
				Status:  "success",
				Message: "Transcriber stopped",
			}
		}
	case "status":
		isRunning := s.transcriber.IsRunning()
		verbose := cmd.Options["verbose"] == "true"
		resp = s.buildStatusResponse(isRunning, verbose)
	case "logs":
		resp = s.buildLogsResponse()
	default:
		s.updateErrorStats(fmt.Errorf("invalid command: %s", cmd.Action))
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

	// Cancel main context to stop all goroutines
	if s.cancel != nil {
		s.cancel()
	}

	// Stop keyboard listener
	if s.keyboardActive && s.keyboardCancel != nil {
		s.keyboardCancel()
	}

	// Close transcriber
	if s.transcriber != nil {
		if err := s.transcriber.Close(); err != nil {
			s.logger.Printf("Error closing transcriber: %v", err)
		}
	}

	// Close listener (ignore "use of closed network connection" errors)
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				return fmt.Errorf("failed to close listener: %w", err)
			}
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
		
		// Socket exists but is not in use (stale), remove it
		s.logger.Printf("Removing stale socket file: %s", s.cfg.Server.SocketPath)
		if err := os.Remove(s.cfg.Server.SocketPath); err != nil {
			return fmt.Errorf("failed to remove stale socket: %w", err)
		}
	}

	return nil
}
