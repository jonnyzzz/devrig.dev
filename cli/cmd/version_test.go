package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Test command output
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.Execute()

	expected := "Version: " + version + "\n"
	if got := buf.String(); got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}

	// Test command properties
	if got := versionCmd.Use; got != "version" {
		t.Errorf("Expected Use to be 'version', got %q", got)
	}
	if got := versionCmd.Short; got == "" {
		t.Error("Short description should not be empty")
	}
}
