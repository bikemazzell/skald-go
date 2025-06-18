package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"skald/internal/config"
)

type Command struct {
	Action string `json:"action"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [start|stop|status]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	action := flag.Arg(0)
	if !isValidAction(action) {
		fmt.Fprintf(os.Stderr, "Invalid action. Use 'start', 'stop', or 'status'\n")
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
	cmd := Command{Action: action}
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
	fmt.Printf("Server response: %s\n", resp.Message)
}

func isValidAction(action string) bool {
	return action == "start" || action == "stop" || action == "status"
}
