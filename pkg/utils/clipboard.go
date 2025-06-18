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
    return cm.IsValidTextWithMode(text, "security_focused", true, []string{})
}

// IsValidTextWithMode checks text validity with configurable validation mode
func (cm *ClipboardManager) IsValidTextWithMode(text, mode string, allowPunctuation bool, customBlocklist []string) bool {
    // Check if text is empty
    if text == "" {
        return false
    }

    // Check text length (prevent extremely large clipboard operations)
    const maxTextLength = 1000000 // 1MB of text
    if len(text) > maxTextLength {
        return false
    }

    // Apply mode-specific validation
    switch mode {
    case "security_focused":
        return cm.validateSecurityFocused(text, allowPunctuation, customBlocklist)
    case "strict":
        return cm.validateStrict(text)
    default:
        return cm.validateSecurityFocused(text, allowPunctuation, customBlocklist)
    }
}

// validateSecurityFocused only blocks actual security threats, allows normal punctuation
func (cm *ClipboardManager) validateSecurityFocused(text string, allowPunctuation bool, customBlocklist []string) bool {
    // Check for actual command injection patterns (not simple punctuation)
    dangerousPatterns := []string{
        "$(", "${",     // Command substitution
        "&&", "||",     // Command chaining
        ">>", "<<",     // Redirection/heredoc
        "`",            // Backtick command substitution
    }
    
    // If punctuation is not allowed, add basic dangerous chars
    if !allowPunctuation {
        dangerousPatterns = append(dangerousPatterns, 
            ";", "&", "|", "<", ">", "\\", "'", "\"", "!", "(", ")", "{", "}", "[", "]", "$")
    }
    
    for _, pattern := range dangerousPatterns {
        if strings.Contains(text, pattern) {
            return false
        }
    }

    // Check custom blocklist
    lowerText := strings.ToLower(text)
    for _, blocked := range customBlocklist {
        if strings.Contains(lowerText, strings.ToLower(blocked)) {
            return false
        }
    }

    // Check for dangerous shell commands at word boundaries
    dangerousCommands := []string{
        "rm -rf", "sudo ", "chmod ", "chown ", "mkfs", "dd if=",
        "curl -", "wget -", "bash -", "sh -", "eval ", "exec ",
    }
    
    for _, cmd := range dangerousCommands {
        if strings.Contains(lowerText, cmd) {
            return false
        }
    }

    // Check for control characters (but allow common whitespace)
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

// validateStrict uses the original strict validation
func (cm *ClipboardManager) validateStrict(text string) bool {
    // Original strict validation for backward compatibility
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