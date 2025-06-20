package utils

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestClipboardManager_IsValidText(t *testing.T) {
	cm := NewClipboardManager(false)
	testCases := []struct {
		name             string
		text             string
		wantValid        bool
		allowPunctuation bool
	}{
		// Valid cases (should pass with default settings)
		{"Simple text", "hello world", true, true},
		{"Text with spaces", "  hello world  ", true, true},
		{"Text with punctuation", "Hello, how are you?", true, true},
		{"Numbers", "12345", true, true},
		{"Mixed case", "HeLlO WoRlD", true, true},
		{"Empty string", "", false, true}, // Empty string is invalid
		{"Tab character", "hello\tworld", true, true},
		{"Unicode text", "こんにちは世界", true, true},
		{"Emoji", "👋", true, true},
		{"Accented characters", "éàçü", true, true},

		// Invalid cases (punctuation and control characters) - should be invalid when punctuation is disallowed
		{"Text too long", strings.Repeat("a", 1000001), false, false},
		{"Semicolon", "hello; world", false, false},
		{"Ampersand", "hello & world", false, false},
		{"Pipe", "hello | world", false, false},
		{"Dollar sign", "hello $ world", false, false},
		{"Parentheses", "hello (world)", false, false},
		{"Curly braces", "hello {world}", false, false},
		{"Square brackets", "hello [world]", false, false},
		{"Redirection", "hello > file", false, false},
		{"History expansion", "hello !", false, false},
		{"Escape character", "hello \\ world", false, false},
		{"Single quotes", "hello 'world'", false, false},
		{"Double quotes", "hello \"world\"", false, false},
		{"Null byte", "hello\x00world", false, false},
		{"Newline", "hello\nworld", false, false},
		{"Carriage return", "hello\rworld", false, false},

		// Invalid cases (always blocked regardless of punctuation setting)
		{"Backtick", "`command`", false, true},
		{"Command substitution", "$(command)", false, true},
		{"Variable substitution", "${VAR}", false, true},
		{"Command chaining &&", "cmd1 && cmd2", false, true},
		{"Command chaining ||", "cmd1 || cmd2", false, true},
		{"rm command", "please rm -rf /", false, true},
		{"sudo command", "sudo reboot", false, true},
		{"chmod command", "chmod 777 file", false, true},
		{"curl command", "curl http://example.com", false, true},
		{"wget command", "wget http://example.com", false, true},
		{"bash command", "bash -c 'echo pwned'", false, true},
		{"python command", "python -c 'import os'", false, true},
		{"eval command", "eval 'rm -rf'", false, true},
		{"exec command", "exec 'reboot'", false, true},
		{"source command", "source ~/.bashrc", false, true},
		{". command", ". ~/.bashrc", false, true},
		{"export command", "export EVIL=pwned", false, true},
		{"Control character", "hello\x07world", false, true},
		{"Private use area", "text \uE000", false, true},
		{"Invalid Unicode", string(rune(0xFFFE)), false, true},
		{"Multiple dangerous", "eval `rm -rf` && sudo", false, true},
		{"Hidden command", "innocent text rm -rf hidden", false, true},
		{"Case variation", "SUDO Rm -Rf /", false, true},
		{"Tab instead of space", "rm\t-rf", false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := cm.IsValidTextWithMode(tc.text, "security_focused", tc.allowPunctuation, []string{})
			if got != tc.wantValid {
				t.Errorf("IsValidTextWithMode(%q, security_focused, %v) = %v, want %v", tc.text, tc.allowPunctuation, got, tc.wantValid)
			}
		})
	}
}

func TestClipboardManager_Copy(t *testing.T) {
	cm := NewClipboardManager(true)

	// Test Copy with valid text
	validText := "This is a valid text."
	if err := cm.Copy(validText); err != nil {
		t.Errorf("Expected no error when copying valid text, but got %v", err)
	}

	// Test Copy with invalid text that should be caught by the security-focused validator
	invalidText := "rm -rf /"
	// We must call the validator explicitly since IsValidText allows some punctuation by default
	if cm.IsValidTextWithMode(invalidText, "security_focused", false, []string{}) {
		t.Error("Invalid text was considered valid")
	}

	// Ensure that the Copy function itself returns an error for such text
	if err := cm.Copy(invalidText); err == nil {
		t.Error("Expected an error when copying invalid text, but got nil")
	}
}

func TestClipboardManager_RateLimiting(t *testing.T) {
	cm := NewClipboardManager(false)

	// Override minInterval for testing
	cm.minInterval = 50 * time.Millisecond

	// First copy should succeed immediately
	start := time.Now()
	err := cm.Copy("First text")
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("First copy failed: %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("First copy took too long: %v", elapsed)
	}

	// Second copy should be rate limited
	start = time.Now()
	err = cm.Copy("Second text")
	elapsed = time.Since(start)

	if err != nil {
		t.Errorf("Second copy failed: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Second copy was not rate limited, elapsed: %v", elapsed)
	}

	// Wait for rate limit to expire
	time.Sleep(60 * time.Millisecond)

	// Third copy should succeed immediately again
	start = time.Now()
	err = cm.Copy("Third text")
	elapsed = time.Since(start)

	if err != nil {
		t.Errorf("Third copy failed: %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Third copy took too long after rate limit expired: %v", elapsed)
	}
}

func TestClipboardManager_ConcurrentAccess(t *testing.T) {
	cm := NewClipboardManager(false)
	cm.minInterval = 10 * time.Millisecond

	// Test concurrent access doesn't cause race conditions
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(n int) {
			text := fmt.Sprintf("Concurrent text %d", n)
			err := cm.Copy(text)
			if err != nil && !strings.Contains(err.Error(), "invalid text") {
				t.Errorf("Goroutine %d: unexpected error: %v", n, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestClipboardManager_ValidationPerformance(t *testing.T) {
	cm := NewClipboardManager(false)

	// Test with a large but valid text
	largeText := strings.Repeat("This is a valid sentence. ", 10000)

	start := time.Now()
	// This should be invalid because punctuation is disallowed
	result := cm.IsValidTextWithMode(largeText, "security_focused", false, []string{})
	elapsed := time.Since(start)

	if result {
		t.Error("Large text with punctuation should be invalid when punctuation is disallowed")
	}
	if elapsed > 50*time.Millisecond { // Allow a bit more time for large text validation
		t.Errorf("Validation took too long: %v", elapsed)
	}
}

func TestClipboardManager_EdgeCases(t *testing.T) {
	cm := NewClipboardManager(false)

	testCases := []struct {
		name      string
		text      string
		wantValid bool
	}{
		{"Text at exactly max length", strings.Repeat("a", 1000000), true},
		{"Text with dangerous pattern at end", strings.Repeat("a", 999990) + " rm -rf /", false},
		{"Text with private use Unicode", "Valid text 世界 \uE000 more text", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if cm.IsValidText(tc.text) != tc.wantValid {
				t.Errorf("IsValidText(%q) = %v, want %v", tc.text, !tc.wantValid, tc.wantValid)
			}
		})
	}
}
