package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Manager defines the interface for clipboard operations
type Manager interface {
	Copy(text string) error
	Paste() error
}

type linuxManager struct {
	autoPaste bool
}

func newPlatformManager(autoPaste bool) (Manager, error) {
	return &linuxManager{autoPaste: autoPaste}, nil
}

func (m *linuxManager) Copy(text string) error {
	// Try xclip first, then fallback to xsel
	if err := m.copyXclip(text); err != nil {
		if err := m.copyXsel(text); err != nil {
			return fmt.Errorf("clipboard copy failed: %w", err)
		}
	}
	return nil
}

func (m *linuxManager) copyXclip(text string) error {
	// Validate text before copying
	if !isValidText(text) {
		return fmt.Errorf("invalid text contains potentially unsafe characters")
	}

	// Use full path to xclip for security
	xclipPath, err := exec.LookPath("xclip")
	if err != nil {
		return fmt.Errorf("xclip not found: %w", err)
	}

	cmd := exec.Command(xclipPath, "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (m *linuxManager) copyXsel(text string) error {
	// Validate text before copying
	if !isValidText(text) {
		return fmt.Errorf("invalid text contains potentially unsafe characters")
	}

	// Use full path to xsel for security
	xselPath, err := exec.LookPath("xsel")
	if err != nil {
		return fmt.Errorf("xsel not found: %w", err)
	}

	cmd := exec.Command(xselPath, "--input", "--clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// isValidText checks if the text is safe to copy to clipboard
func isValidText(text string) bool {
	// Check if text is empty
	if text == "" {
		return false
	}

	// Check for potentially dangerous characters that could be used for command injection
	dangerousChars := []string{";", "&", "|"}
	for _, char := range dangerousChars {
		if strings.Contains(text, char) {
			return false
		}
	}

	return true
}

func (m *linuxManager) Paste() error {
	if !m.autoPaste {
		return nil
	}

	// Try xdotool first with full path for security
	xdotoolPath, err := exec.LookPath("xdotool")
	if err == nil {
		// Use a timeout to prevent hanging
		cmd := exec.Command(xdotoolPath, "key", "ctrl+v")
		cmd.Start()

		// Create a channel to signal command completion
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		// Wait with timeout
		select {
		case err := <-done:
			if err == nil {
				return nil
			}
		case <-time.After(2 * time.Second):
			// Kill the process if it takes too long
			cmd.Process.Kill()
			return fmt.Errorf("xdotool paste timed out")
		}
	}

	// Fallback to xte with full path for security
	xtePath, err := exec.LookPath("xte")
	if err == nil {
		cmd := exec.Command(xtePath, "key Control_L+v")
		cmd.Start()

		// Create a channel to signal command completion
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		// Wait with timeout
		select {
		case err := <-done:
			if err == nil {
				return nil
			}
		case <-time.After(2 * time.Second):
			// Kill the process if it takes too long
			cmd.Process.Kill()
			return fmt.Errorf("xte paste timed out")
		}
	}

	return fmt.Errorf("paste simulation failed: no suitable tool found")
}
