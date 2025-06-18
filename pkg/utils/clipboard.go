package utils

import (
    "fmt"
    "os/exec"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/atotto/clipboard"
)

// ClipboardManager handles clipboard operations and auto-paste functionality
type ClipboardManager struct {
    autoPaste    bool
    mu           sync.Mutex
    lastOp       time.Time
    minInterval  time.Duration
}

// NewClipboardManager creates a new clipboard manager
func NewClipboardManager(autoPaste bool) *ClipboardManager {
    return &ClipboardManager{
        autoPaste:   autoPaste,
        minInterval: 100 * time.Millisecond, // Minimum 100ms between operations
    }
}

// Copy copies text to system clipboard
func (cm *ClipboardManager) Copy(text string) error {
    if text == "" {
        return nil // Don't copy empty text
    }

    // Rate limiting
    cm.mu.Lock()
    timeSinceLastOp := time.Since(cm.lastOp)
    if timeSinceLastOp < cm.minInterval {
        timeToWait := cm.minInterval - timeSinceLastOp
        cm.mu.Unlock()
        time.Sleep(timeToWait)
        cm.mu.Lock()
    }
    cm.lastOp = time.Now()
    cm.mu.Unlock()

    // Validate text before copying to clipboard
    if !cm.IsValidText(text) {
        return fmt.Errorf("invalid text contains potentially unsafe characters")
    }

    return clipboard.WriteAll(text)
}

// IsValidText checks if the text is safe to copy to clipboard
func (cm *ClipboardManager) IsValidText(text string) bool {
    // Check if text is empty
    if text == "" {
        return false
    }

    // Check text length (prevent extremely large clipboard operations)
    const maxTextLength = 1000000 // 1MB of text
    if len(text) > maxTextLength {
        return false
    }

    // Check for potentially dangerous characters and patterns
    dangerousPatterns := []string{
        ";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", 
        "\n", "\r", "\x00", // Control characters
        "&&", "||", // Command chaining
        "$(", "${", // Command substitution
        "<", ">", // Redirection
        ">>", "<<", // Append/heredoc
        "!", // History expansion
        "\\", // Escape character
        "'", "\"", // Quotes that could break out of strings
    }
    
    for _, pattern := range dangerousPatterns {
        if strings.Contains(text, pattern) {
            return false
        }
    }

    // Check for common shell commands that shouldn't be in transcribed text
    dangerousCommands := []string{
        "rm ", "rm\t", "sudo", "chmod", "chown", "mkfs", "dd ",
        "curl ", "wget ", "sh ", "bash ", "zsh ", "fish ",
        "python ", "perl ", "ruby ", "node ", "php ",
        "eval ", "exec ", "source ", ". ", "export ",
    }
    
    lowerText := strings.ToLower(text)
    for _, cmd := range dangerousCommands {
        if strings.Contains(lowerText, cmd) {
            return false
        }
    }

    // Check for Unicode control characters
    for _, r := range text {
        if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, newline, carriage return
            return false
        }
        // Block private use and undefined Unicode ranges
        if (r >= 0xE000 && r <= 0xF8FF) || // Private use area
           (r >= 0xF0000 && r <= 0xFFFFF) || // Supplementary private use area A
           (r >= 0x100000 && r <= 0x10FFFF) { // Supplementary private use area B
            return false
        }
    }

    return true
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