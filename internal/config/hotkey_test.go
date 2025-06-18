package config

import (
	"testing"
)

func TestHotkeyConfiguration(t *testing.T) {
	// Test default hotkey configuration
	cfg := DefaultConfig()
	
	expectedHotkeys := map[string]string{
		"r": "start",
		"s": "stop",
		"i": "status", 
		"q": "quit",
		"?": "help",
		"c": "resume",
	}
	
	for key, expectedAction := range expectedHotkeys {
		actualAction, exists := cfg.Server.Hotkeys[key]
		if !exists {
			t.Errorf("Expected hotkey '%s' not found in default configuration", key)
			continue
		}
		if actualAction != expectedAction {
			t.Errorf("Hotkey '%s': expected action '%s', got '%s'", key, expectedAction, actualAction)
		}
	}
	
	// Check that we have exactly 6 default hotkeys (including 'c' for resume)
	if len(cfg.Server.Hotkeys) != 6 {
		t.Errorf("Expected 6 default hotkeys, got %d", len(cfg.Server.Hotkeys))
	}
}

func TestCustomHotkeyConfiguration(t *testing.T) {
	// Test custom hotkey configuration
	cfg := DefaultConfig()
	cfg.Server.Hotkeys = map[string]string{
		"1": "start",
		"2": "stop",
		"3": "status",
		"0": "quit",
		"h": "help",
	}
	
	// Verify custom configuration is set correctly
	expectedCustom := map[string]string{
		"1": "start",
		"2": "stop", 
		"3": "status",
		"0": "quit",
		"h": "help",
	}
	
	for key, expectedAction := range expectedCustom {
		actualAction, exists := cfg.Server.Hotkeys[key]
		if !exists {
			t.Errorf("Expected custom hotkey '%s' not found", key)
			continue
		}
		if actualAction != expectedAction {
			t.Errorf("Custom hotkey '%s': expected action '%s', got '%s'", key, expectedAction, actualAction)
		}
	}
}

func TestHotkeyValidation(t *testing.T) {
	// Test valid actions
	validActions := []string{"start", "stop", "status", "quit", "help", "resume"}
	
	cfg := DefaultConfig()
	
	// Check that default config only uses valid actions
	for key, action := range cfg.Server.Hotkeys {
		isValid := false
		for _, validAction := range validActions {
			if action == validAction {
				isValid = true
				break
			}
		}
		if !isValid {
			t.Errorf("Invalid action '%s' found for hotkey '%s'", action, key)
		}
	}
}

func TestHotkeyConfigurationEditable(t *testing.T) {
	// Test that hotkey configuration can be modified
	cfg := DefaultConfig()
	
	// Modify a hotkey
	cfg.Server.Hotkeys["z"] = "start"
	
	// Verify it was set
	if cfg.Server.Hotkeys["z"] != "start" {
		t.Error("Failed to set custom hotkey")
	}
	
	// Remove a hotkey
	delete(cfg.Server.Hotkeys, "r")
	
	// Verify it was removed
	if _, exists := cfg.Server.Hotkeys["r"]; exists {
		t.Error("Failed to remove hotkey")
	}
	
	// Clear all hotkeys
	cfg.Server.Hotkeys = make(map[string]string)
	
	// Verify they were cleared
	if len(cfg.Server.Hotkeys) != 0 {
		t.Error("Failed to clear all hotkeys")
	}
}