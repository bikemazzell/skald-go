package transcriber

import (
	"strings"
	"testing"
)

// Helper function that mimics the exact filtering logic in processBuffer
func filterText(input string) string {
	filtered := input
	for _, token := range tokensToFilter {
		filtered = strings.ReplaceAll(filtered, token, "")
	}
	return strings.TrimSpace(filtered)
}

// Test for filtering tokens in processBuffer
func TestFilterTokens(t *testing.T) {
	// Test cases for token filtering
	testCases := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"Hello [SILENCE] world", "Hello  world"},
		{"[BLANK_AUDIO] Test", "Test"},
		{"[NOISE] [SPEECH] Test [MUSIC]", "Test"},
		{"[SILENCE][BLANK_AUDIO][NOISE][SPEECH][MUSIC]", ""},
	}

	for _, tc := range testCases {
		// Get the actual result using our helper function
		actual := filterText(tc.input)

		// Compare with the expected result
		if actual != tc.expected {
			// If they don't match, run the actual filtering logic step by step to debug
			filtered := tc.input
			for _, token := range tokensToFilter {
				before := filtered
				filtered = strings.ReplaceAll(filtered, token, "")
				if before != filtered {
					t.Logf("Replaced '%s' in '%s' -> '%s'", token, before, filtered)
				}
			}
			filtered = strings.TrimSpace(filtered)

			t.Errorf("Expected '%s' but got '%s' for input '%s'", tc.expected, actual, tc.input)
		}
	}
}

// Test for IsRunning function
func TestIsRunning(t *testing.T) {
	transcriber := &Transcriber{
		isRunning: true,
	}

	if !transcriber.IsRunning() {
		t.Error("Expected IsRunning to return true when isRunning is true")
	}

	transcriber.isRunning = false
	if transcriber.IsRunning() {
		t.Error("Expected IsRunning to return false when isRunning is false")
	}
}
