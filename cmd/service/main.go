package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"skald/internal/config"
	"skald/internal/model"
	"skald/internal/server"
)

var (
	configPath  string
	verbose     bool
	version     string
	buildTime   string
	gitCommit   string
	showVersion bool
)

func init() {
	flag.StringVar(&configPath, "config", "config.json", "path to configuration file")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.Parse()
}

func displayBanner() {
	banner := `
══════════════════════════════════
👄      Skald-Go Transcriber    🎙️ 
👂     Version %-10.10s       📝 
══════════════════════════════════`
	fmt.Printf(banner+"\n", version)
}

func printVersion() {
	if version == "" {
		version = "development"
	}
	fmt.Printf("Skald-Go %s\n", version)
	if gitCommit != "" {
		fmt.Printf("Commit: %s\n", gitCommit)
	}
	if buildTime != "" {
		fmt.Printf("Built: %s\n", buildTime)
	}
}

func main() {
	if showVersion {
		printVersion()
		return
	}

	displayBanner()

	logger := setupLogger()
	
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.Fatalf("Failed to resolve config path: %v", err)
	}

	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	cfg.Verbose = verbose
	
	if verbose {
		logger.Printf("Configuration loaded from: %s", absConfigPath)
	}

	modelMgr := model.New(cfg, logger)
	if err := modelMgr.Initialize(cfg.Whisper.Model); err != nil {
		logger.Fatalf("Failed to ensure model exists: %v", err)
	}
	if verbose {
		logger.Printf("Model initialized successfully")
	}

	srv, err := server.New(cfg, logger, modelMgr)
	if err != nil {
		logger.Fatalf("Failed to create server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case sig := <-sigChan:
		logger.Printf("Received signal: %v", sig)
		logger.Printf("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(cfg.Processing.ShutdownTimeout)*time.Second)
		defer cancel()

		shutdownChan := make(chan struct{})
		go func() {
			if err := srv.Stop(); err != nil {
				logger.Printf("Error during shutdown: %v", err)
			}
			close(shutdownChan)
		}()

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

