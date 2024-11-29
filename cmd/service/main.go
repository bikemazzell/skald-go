package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"skald/internal/config"
	"skald/internal/server"
)

var (
	configPath string
	version    string // Will be set by linker during build
)

func init() {
	// Parse command line flags
	flag.StringVar(&configPath, "config", "config.json", "path to configuration file")
	flag.Parse()
}

func displayBanner() {
	banner := `
╔═══════════════════════════════════╗
║ 👄   Skald (STT Transcriber)   🎙️  ║
║      Created by @shoewind1997     ║
║ 👂     Version %-6s          📝 ║
╚═══════════════════════════════════╝`
	fmt.Printf(banner+"\n", version)
}

func main() {
	displayBanner()

	// Create logger
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Load configuration
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.Fatalf("Failed to resolve config path: %v", err)
	}

	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	logger.Printf("Configuration loaded from: %s", absConfigPath)

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create server: %v", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Printf("Received signal: %v", sig)
		logger.Printf("Shutting down...")

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(cfg.Processing.ShutdownTimeout)*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		shutdownChan := make(chan struct{})
		go func() {
			if err := srv.Stop(); err != nil {
				logger.Printf("Error during shutdown: %v", err)
			}
			close(shutdownChan)
		}()

		// Wait for shutdown or timeout
		select {
		case <-ctx.Done():
			logger.Printf("Shutdown timeout exceeded, forcing exit")
		case <-shutdownChan:
			logger.Printf("Graceful shutdown completed")
		}

	case err := <-errChan:
		logger.Printf("Server error: %v", err)
	}

	logger.Printf("Cleaning up...")
	os.Exit(0)
}

// ensureDirectory ensures the directory for the socket file exists
func ensureDirectory(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
