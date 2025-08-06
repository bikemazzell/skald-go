package output

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ClipboardOutput implements clipboard and stdout output
type ClipboardOutput struct {
	writer io.Writer
	useClipboard bool
}

// NewClipboardOutput creates a new clipboard output
func NewClipboardOutput(writer io.Writer, useClipboard bool) *ClipboardOutput {
	return &ClipboardOutput{
		writer: writer,
		useClipboard: useClipboard,
	}
}

// Write writes text to output and optionally clipboard
func (c *ClipboardOutput) Write(text string) error {
	if text == "" {
		return nil
	}

	// Write to writer (usually stdout)
	if _, err := fmt.Fprintln(c.writer, text); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}
	
	// Copy to clipboard if enabled
	if c.useClipboard {
		if err := c.copyToClipboard(text); err != nil {
			// Non-fatal error - we already printed to stdout
			fmt.Fprintf(c.writer, "Warning: Failed to copy to clipboard: %v\n", err)
		}
	}
	
	return nil
}

// copyToClipboard copies text to system clipboard using xclip
func (c *ClipboardOutput) copyToClipboard(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}