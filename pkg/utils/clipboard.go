package utils

import (
    "fmt"
    "os/exec"
    "runtime"
    
    "github.com/atotto/clipboard"
)

// ClipboardManager handles clipboard operations and auto-paste functionality
type ClipboardManager struct {
    autoPaste bool
}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager(autoPaste bool) *ClipboardManager {
    return &ClipboardManager{
        autoPaste: autoPaste,
    }
}

// Copy copies text to system clipboard
func (cm *ClipboardManager) Copy(text string) error {
    if text == "" {
        return nil // Don't copy empty text
    }

    return clipboard.WriteAll(text)
}

// Paste simulates Ctrl+V/Cmd+V if auto-paste is enabled
func (cm *ClipboardManager) Paste() error {
    if !cm.autoPaste {
        return nil
    }

    switch runtime.GOOS {
    case "linux":
        return cm.pasteLinux()
    case "darwin":
        return cm.pasteDarwin()
    case "windows":
        return cm.pasteWindows()
    default:
        return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
    }
}

// Platform-specific paste implementations
func (cm *ClipboardManager) pasteLinux() error {
    xdotool, err := exec.LookPath("xdotool")
    if err != nil {
        return fmt.Errorf("xdotool not found: %v", err)
    }
    return exec.Command(xdotool, "key", "ctrl+v").Run()
}

func (cm *ClipboardManager) pasteDarwin() error {
    script := `tell application "System Events" to keystroke "v" using command down`
    return exec.Command("osascript", "-e", script).Run()
}

func (cm *ClipboardManager) pasteWindows() error {
    script := `Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait("^v")`
    return exec.Command("powershell.exe", "-command", script).Run()
}

// CheckClipboardDependencies checks if required clipboard utilities are available
func CheckClipboardDependencies() error {
    // Only need to check for auto-paste dependencies
    if runtime.GOOS == "linux" {
        _, err := exec.LookPath("xdotool")
        if err != nil {
            return fmt.Errorf("xdotool not found. Auto-paste will be disabled. Install with: sudo apt-get install xdotool")
        }
    }
    return nil
}