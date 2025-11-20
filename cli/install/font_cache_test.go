package install

import (
	"runtime"
	"testing"
)

// TestRefreshFontCacheLinux tests the Linux font cache refresh functionality
func TestRefreshFontCacheLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	// This should not return an error even if fc-cache is not installed
	err := refreshFontCacheLinux()
	if err != nil {
		t.Errorf("refreshFontCacheLinux should not return an error, got: %v", err)
	}

	// Test passes if we get here, regardless of whether fc-cache succeeded
	t.Log("Font cache refresh attempted (fc-cache may or may not be installed)")
}

// TestRefreshFontCacheLinuxDoesNotPanic tests that the function doesn't panic
func TestRefreshFontCacheLinuxDoesNotPanic(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("refreshFontCacheLinux panicked: %v", r)
		}
	}()

	// Should not panic even if fc-cache is not available
	_ = refreshFontCacheLinux()
}
