package server

import (
	"skald/internal/config"
	"testing"
)

func TestHotkeyConfiguration(t *testing.T) {
	// Test default hotkey configuration
	cfg := config.DefaultConfig()
	
	expectedHotkeys := map[string]string{
		"r": "start",
		"s": "stop",
		"i": "status", 
		"q": "quit",
		"?": "help",
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
	cfg := config.DefaultConfig()
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
	// Test that configuration handles various hotkey scenarios
	testCases := []struct {
		name     string
		hotkeys  map[string]string
		expected int // expected number of valid hotkeys
	}{
		{
			name: "All valid single character keys",
			hotkeys: map[string]string{
				"a": "start",
				"b": "stop",
				"c": "status",
			},
			expected: 3,
		},
		{
			name: "Mixed valid and invalid keys",
			hotkeys: map[string]string{
				"x": "start",        // valid
				"yz": "stop",        // invalid - multiple chars
				"": "status",        // invalid - empty
				"q": "quit",         // valid
			},
			expected: 2, // only 'x' and 'q' should be valid
		},
		{
			name: "Valid keys with invalid actions",
			hotkeys: map[string]string{
				"a": "start",        // valid
				"b": "invalid_action", // invalid action
				"c": "stop",         // valid
			},
			expected: 2, // only 'a' and 'c' should have valid actions
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Server.Hotkeys = tc.hotkeys
			
			// Count valid hotkeys by checking the format
			validCount := 0
			for key, action := range cfg.Server.Hotkeys {
				if len(key) == 1 && action != "" {
					// Check if action is in valid actions list
					validActions := []string{"start", "stop", "status", "quit", "help", "resume"}
					isValidAction := false
					for _, validAction := range validActions {
						if action == validAction {
							isValidAction = true
							break
						}
					}
					if isValidAction {
						validCount++
					}
				}
			}
			
			if validCount != tc.expected {
				t.Errorf("Test '%s': expected %d valid hotkeys, got %d", tc.name, tc.expected, validCount)
			}
		})
	}
}

func TestHotkeyActions(t *testing.T) {
	// Test that all required actions are supported
	requiredActions := []string{"start", "stop", "status", "quit", "help", "resume"}
	
	cfg := config.DefaultConfig()
	
	// Check that default config includes all required actions
	foundActions := make(map[string]bool)
	for _, action := range cfg.Server.Hotkeys {
		foundActions[action] = true
	}
	
	for _, requiredAction := range requiredActions {
		if !foundActions[requiredAction] {
			t.Errorf("Required action '%s' not found in default hotkey configuration", requiredAction)
		}
	}
}