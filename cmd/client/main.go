package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"skald/internal/config"
)

var (
	version   string
	buildTime string
	gitCommit string
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

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	var verbose bool
	var continuous bool
	var showVersion bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&continuous, "continuous", false, "Enable continuous mode (for start command)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [start|stop|status|logs] [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		if version == "" {
			version = "development"
		}
		fmt.Printf("Skald-Go Client %s\n", version)
		if gitCommit != "" {
			fmt.Printf("Commit: %s\n", gitCommit)
		}
		if buildTime != "" {
			fmt.Printf("Built: %s\n", buildTime)
		}
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	action := flag.Arg(0)
	if !isValidAction(action) {
		fmt.Fprintf(os.Stderr, "Invalid action. Use 'start', 'stop', 'status', or 'logs'\n")
		os.Exit(1)
	}

	fmt.Printf("Connecting to server at %s\n", cfg.Server.SocketPath)
	conn, err := net.Dial("unix", cfg.Server.SocketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Sending command: %s\n", action)
	cmd := Command{
		Action:  action,
		Options: make(map[string]string),
	}
	if verbose {
		cmd.Options["verbose"] = "true"
	}
	if continuous && action == "start" {
		cmd.Options["continuous"] = "true"
	}
	if err := json.NewEncoder(conn).Encode(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send command: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Waiting for response...")
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read response: %v\n", err)
		os.Exit(1)
	}

	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Error)
		os.Exit(1)
	}
	
	if verbose && resp.Data != nil {
		fmt.Println("\nDetailed Status:")
		for key, value := range resp.Data {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
	
	fmt.Printf("Server response: %s\n", resp.Message)
}

func isValidAction(action string) bool {
	return action == "start" || action == "stop" || action == "status" || action == "logs"
}
