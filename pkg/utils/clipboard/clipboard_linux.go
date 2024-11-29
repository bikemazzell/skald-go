package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

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
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (m *linuxManager) copyXsel(text string) error {
	cmd := exec.Command("xsel", "--input", "--clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (m *linuxManager) Paste() error {
	if !m.autoPaste {
		return nil
	}

	// Try xdotool first, then fallback to xte
	if err := exec.Command("xdotool", "key", "ctrl+v").Run(); err != nil {
		if err := exec.Command("xte", "key Control_L+v").Run(); err != nil {
			return fmt.Errorf("paste simulation failed: %w", err)
		}
	}
	return nil
}
