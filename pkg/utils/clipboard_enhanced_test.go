package utils

import (
	"strings"
	"fmt"
	"testing"
	"time"
)

func TestClipboardManager_IsValidText(t *testing.T) {
	cm := NewClipboardManager(false)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		// Valid cases
		{"Simple text", "Hello World", true},
		{"Text with spaces", "This is a valid transcription", true},
		{"Text with punctuation", "Hello, how are you?", true},
		{"Numbers", "The year is 2024", true},
		{"Mixed case", "Hello WORLD", true},

		// Invalid cases - empty
		{"Empty string", "", false},

		// Invalid cases - too long
		{"Text too long", strings.Repeat("a", 1000001), false},

		// Invalid cases - dangerous characters
		{"Semicolon", "hello; world", false},
		{"Ampersand", "hello & world", false},
		{"Pipe", "hello | world", false},
		{"Backtick", "hello ` world", false},
		{"Dollar sign", "hello $ world", false},
		{"Parentheses", "hello (world)", false},
		{"Curly braces", "hello {world}", false},
		{"Square brackets", "hello [world]", false},
		{"Command substitution", "hello $(command)", false},
		{"Variable substitution", "hello ${var}", false},
		{"Redirection", "hello > file", false},
		{"Append", "hello >> file", false},
		{"History expansion", "hello !", false},
		{"Escape character", "hello \\ world", false},
		{"Single quotes", "hello 'world'", false},
		{"Double quotes", "hello \"world\"", false},
		{"Command chaining &&", "cmd1 && cmd2", false},
		{"Command chaining ||", "cmd1 || cmd2", false},

		// Invalid cases - dangerous commands
		{"rm command", "please rm -rf /", false},
		{"sudo command", "sudo apt-get update", false},
		{"chmod command", "chmod 777 file", false},
		{"curl command", "curl http://example.com", false},
		{"wget command", "wget http://example.com", false},
		{"bash command", "bash script.sh", false},
		{"python command", "python script.py", false},
		{"eval command", "eval malicious", false},
		{"exec command", "exec malicious", false},
		{"source command", "source ~/.bashrc", false},
		{"dot command", ". ~/.bashrc", false},
		{"export command", "export PATH=/bad", false},

		// Invalid cases - control characters
		{"Null byte", "hello\x00world", false},
		{"Newline", "hello\nworld", false},
		{"Carriage return", "hello\rworld", false},
		{"Control character", "hello\x01world", false},

		// Invalid cases - Unicode issues
		{"Private use area", "hello\uE000world", false},
		{"Invalid Unicode", "hello\U000F0000world", false},

		// Edge cases that should be valid
		{"Tab character", "hello\tworld", true},
		{"Unicode text", "Hello 世界", true},
		{"Emoji", "Hello 👋", true},
		{"Accented characters", "Café résumé", true},

		// Mixed dangerous patterns
		{"Multiple dangerous", "rm -rf / && sudo chmod 777", false},
		{"Hidden command", "innocent text rm -rf hidden", false},
		{"Case variation", "RM -rf /", false},
		{"Tab instead of space", "rm\t-rf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.IsValidText(tt.text)
			if result != tt.expected {
				t.Errorf("IsValidText(%q) = %v, want %v", tt.text, result, tt.expected)
			}
		})
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
	result := cm.IsValidText(largeText)
	elapsed := time.Since(start)

	if result {
		t.Error("Large valid text should be invalid due to commas")
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("Validation took too long: %v", elapsed)
	}
}

func TestClipboardManager_EdgeCases(t *testing.T) {
	cm := NewClipboardManager(false)

	// Test with exactly max length
	maxLengthText := strings.Repeat("a", 1000000)
	if !cm.IsValidText(maxLengthText) {
		t.Error("Text at exactly max length should be valid")
	}

	// Test with dangerous pattern at the end
	textWithDangerAtEnd := strings.Repeat("a", 999990) + "rm -rf /"
	if cm.IsValidText(textWithDangerAtEnd) {
		t.Error("Text with dangerous pattern at end should be invalid")
	}

	// Test with mixed valid and invalid Unicode
	mixedUnicode := "Valid text 世界 \uE000 more text"
	if cm.IsValidText(mixedUnicode) {
		t.Error("Text with private use Unicode should be invalid")
	}
}