package utils

import (
	"testing"
)

func TestClipboardManager(t *testing.T) {
	// Create a clipboard manager with auto-paste disabled for testing
	cm := NewClipboardManager(false)
	
	// Test initialization
	if cm.autoPaste != false {
		t.Error("Expected autoPaste to be false")
	}
	
	// Test copying empty text (should be a no-op)
	err := cm.Copy("")
	if err != nil {
		t.Errorf("Expected no error copying empty text, got: %v", err)
	}
	
	// Test text validation
	err = cm.Copy("text with ; semicolon")
	if err == nil {
		t.Error("Expected error copying text with semicolon, got nil")
	}
	
	err = cm.Copy("text with & ampersand")
	if err == nil {
		t.Error("Expected error copying text with ampersand, got nil")
	}
	
	err = cm.Copy("text with | pipe")
	if err == nil {
		t.Error("Expected error copying text with pipe, got nil")
	}
	
	// Test valid text
	// Note: We can't actually test clipboard operations in a unit test
	// without affecting the system clipboard, so we'll just check that
	// the validation passes
	if !cm.IsValidText("This is valid text") {
		t.Error("Expected 'This is valid text' to be valid")
	}
	
	// Test paste with auto-paste disabled
	err = cm.Paste()
	if err != nil {
		t.Errorf("Expected no error with auto-paste disabled, got: %v", err)
	}
}

func TestTextValidation(t *testing.T) {
	cm := NewClipboardManager(false)
	
	testCases := []struct {
		text  string
		valid bool
	}{
		{"", false},
		{"Hello world", true},
		{"Text with semicolon;", false},
		{"Text with ampersand&", false},
		{"Text with pipe|", false},
		{"Text with both; characters|", false},
		{"Normal text 123", true},
		{"Text with newlines\nand spaces", false}, // newlines are dangerous
	}
	
	for _, tc := range testCases {
		result := cm.IsValidText(tc.text)
		if result != tc.valid {
			t.Errorf("For text '%s', expected valid=%v, got %v", tc.text, tc.valid, result)
		}
	}
}
