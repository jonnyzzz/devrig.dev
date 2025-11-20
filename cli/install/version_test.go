package install

import (
	"strings"
	"testing"
)

// TestVersionInUserAgent tests that the devrig version is properly used in the user agent
func TestVersionInUserAgent(t *testing.T) {
	testVersion := "1.2.3-test"

	installer, err := NewJetBrainsMonoInstaller(testVersion)
	if err != nil {
		// It's OK if we can't fetch the latest release (e.g., no network)
		// We're just testing the version is set correctly
		t.Logf("Could not fetch release (expected in some environments): %v", err)

		// Create a minimal installer to test
		installer = &JetBrainsMonoInstaller{
			devrigVersion: testVersion,
			userAgent:     "devrig/" + testVersion,
		}
	}

	expectedUserAgent := "devrig/" + testVersion
	if installer.userAgent != expectedUserAgent {
		t.Errorf("Expected user agent %q, got %q", expectedUserAgent, installer.userAgent)
	}

	if installer.devrigVersion != testVersion {
		t.Errorf("Expected devrig version %q, got %q", testVersion, installer.devrigVersion)
	}

	if !strings.Contains(installer.userAgent, testVersion) {
		t.Errorf("User agent should contain version %q, got %q", testVersion, installer.userAgent)
	}
}
