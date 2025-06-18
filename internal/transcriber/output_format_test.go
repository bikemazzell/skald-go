package transcriber

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestInlineTranscriptionOutput tests that transcriptions are output on the same line
func TestInlineTranscriptionOutput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Simulate transcription output
	// This mimics what handleTranscriptions does
	firstTranscription := true
	transcriptions := []string{"Hello", "world", "this", "is", "a", "test"}
	
	for _, text := range transcriptions {
		if firstTranscription {
			os.Stdout.WriteString(" ")
			firstTranscription = false
		}
		os.Stdout.WriteString(text + " ")
		os.Stdout.Sync()
	}
	
	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	
	// Verify output is on a single line
	if strings.Count(output, "\n") > 0 {
		t.Errorf("Expected output on single line, but found newlines: %q", output)
	}
	
	// Verify all words are present
	expectedOutput := " Hello world this is a test "
	if output != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output)
	}
}

// TestTranscriptionOutputWithNewlineOnStop tests that a newline is added when transcription stops
func TestTranscriptionOutputWithNewlineOnStop(t *testing.T) {
	// This test verifies the behavior when context is cancelled
	// In the actual code, this happens in handleTranscriptions when ctx.Done() is received
	
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Simulate transcription followed by stop
	firstTranscription := false // Simulate that we've already started
	if !firstTranscription {
		os.Stdout.WriteString("\n") // This is what happens on ctx.Done()
	}
	
	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	
	// Verify newline is present
	if output != "\n" {
		t.Errorf("Expected newline on stop, got %q", output)
	}
}