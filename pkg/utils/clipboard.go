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

type ClipboardManager struct {
    autoPaste    bool
    mu           sync.Mutex
    lastOp       time.Time
    minInterval  time.Duration
}

func NewClipboardManager(autoPaste bool) *ClipboardManager {
    return &ClipboardManager{
        autoPaste:   autoPaste,
        minInterval: 100 * time.Millisecond,
    }
}

func (cm *ClipboardManager) Copy(text string) error {
    if text == "" {
        return nil
    }

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

    if !cm.IsValidText(text) {
        return fmt.Errorf("invalid text contains potentially unsafe characters")
    }

    return clipboard.WriteAll(text)
}

func (cm *ClipboardManager) IsValidText(text string) bool {
    return cm.IsValidTextWithMode(text, "security_focused", true, []string{})
}

func (cm *ClipboardManager) IsValidTextWithMode(text, mode string, allowPunctuation bool, customBlocklist []string) bool {
    if text == "" {
        return false
    }

    const maxTextLength = 1000000
    if len(text) > maxTextLength {
        return false
    }

    switch mode {
    case "security_focused":
        return cm.validateSecurityFocused(text, allowPunctuation, customBlocklist)
    case "strict":
        return cm.validateStrict(text)
    default:
        return cm.validateSecurityFocused(text, allowPunctuation, customBlocklist)
    }
}

func (cm *ClipboardManager) validateSecurityFocused(text string, allowPunctuation bool, customBlocklist []string) bool {
    dangerousPatterns := []string{
        "$(", "${",
        "&&", "||",
        ">>", "<<",
        "`",
    }
    
    if !allowPunctuation {
        dangerousPatterns = append(dangerousPatterns, 
            ";", "&", "|", "<", ">", "\\", "'", "\"", "!", "(", ")", "{", "}", "[", "]", "$")
    }
    
    for _, pattern := range dangerousPatterns {
        if strings.Contains(text, pattern) {
            return false
        }
    }

    lowerText := strings.ToLower(text)
    for _, blocked := range customBlocklist {
        if strings.Contains(lowerText, strings.ToLower(blocked)) {
            return false
        }
    }

    dangerousCommands := []string{
        "rm -rf", "sudo ", "chmod ", "chown ", "mkfs", "dd if=",
        "curl -", "wget -", "bash -", "sh -", "eval ", "exec ",
    }
    
    for _, cmd := range dangerousCommands {
        if strings.Contains(lowerText, cmd) {
            return false
        }
    }

    for _, r := range text {
        if r < 32 && r != 9 && r != 10 && r != 13 {
            return false
        }
        if (r >= 0xE000 && r <= 0xF8FF) ||
           (r >= 0xF0000 && r <= 0xFFFFF) ||
           (r >= 0x100000 && r <= 0x10FFFF) {
            return false
        }
    }

    return true
}

func (cm *ClipboardManager) validateStrict(text string) bool {
    dangerousPatterns := []string{
        ";", "&", "|", "`", "$", "(", ")", "{", "}", "[", "]", 
        "\n", "\r", "\x00",
        "&&", "||",
        "$(", "${",
        "<", ">",
        ">>", "<<",
        "!",
        "\\",
        "'", "\"",
    }
    
    for _, pattern := range dangerousPatterns {
        if strings.Contains(text, pattern) {
            return false
        }
    }

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

    for _, r := range text {
        if r < 32 && r != 9 && r != 10 && r != 13 {
            return false
        }
        if (r >= 0xE000 && r <= 0xF8FF) ||
           (r >= 0xF0000 && r <= 0xFFFFF) ||
           (r >= 0x100000 && r <= 0x10FFFF) {
            return false
        }
    }

    return true
}

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

func CheckClipboardDependencies() error {
    if runtime.GOOS == "linux" {
        _, err := exec.LookPath("xdotool")
        if err != nil {
            return fmt.Errorf("xdotool not found. Auto-paste will be disabled. Install with: sudo apt-get install xdotool")
        }
    }
    return nil
}