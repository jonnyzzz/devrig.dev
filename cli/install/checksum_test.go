package install

import (
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	// Create a temporary file with known content
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.zip")

	testContent := []byte("test content for checksum verification")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate the expected checksum
	hash := sha512.New()
	hash.Write(testContent)
	expectedChecksum := hex.EncodeToString(hash.Sum(nil))

	// Test case 1: Valid checksum
	t.Run("ValidChecksum", func(t *testing.T) {
		installer := &JetBrainsMonoInstaller{
			fontVersion: "v9.9.9",
		}

		// Add the checksum to known checksums temporarily
		originalChecksums := KnownChecksums
		KnownChecksums = map[string]string{
			"v9.9.9": expectedChecksum,
		}
		defer func() { KnownChecksums = originalChecksums }()

		if err := installer.verifyChecksum(testFile); err != nil {
			t.Errorf("Expected valid checksum to pass, got error: %v", err)
		}
	})

	// Test case 2: Invalid checksum
	t.Run("InvalidChecksum", func(t *testing.T) {
		installer := &JetBrainsMonoInstaller{
			fontVersion: "v9.9.9",
		}

		// Add a wrong checksum to known checksums temporarily
		originalChecksums := KnownChecksums
		KnownChecksums = map[string]string{
			"v9.9.9": "wrongchecksumwrongchecksumwrongchecksum",
		}
		defer func() { KnownChecksums = originalChecksums }()

		if err := installer.verifyChecksum(testFile); err == nil {
			t.Error("Expected invalid checksum to fail, but it passed")
		}
	})

	// Test case 3: Unknown version (should warn but not fail)
	t.Run("UnknownVersion", func(t *testing.T) {
		installer := &JetBrainsMonoInstaller{
			fontVersion: "v99.99.99",
		}

		// Ensure this version is not in known checksums
		originalChecksums := KnownChecksums
		KnownChecksums = map[string]string{}
		defer func() { KnownChecksums = originalChecksums }()

		if err := installer.verifyChecksum(testFile); err != nil {
			t.Errorf("Expected unknown version to warn but not fail, got error: %v", err)
		}
	})

	// Test case 4: File doesn't exist (with known checksum)
	t.Run("FileNotFound", func(t *testing.T) {
		installer := &JetBrainsMonoInstaller{
			fontVersion: "v9.9.9",
		}

		// Add a checksum so it actually tries to verify the file
		originalChecksums := KnownChecksums
		KnownChecksums = map[string]string{
			"v9.9.9": expectedChecksum,
		}
		defer func() { KnownChecksums = originalChecksums }()

		nonExistentFile := filepath.Join(tempDir, "nonexistent.zip")
		if err := installer.verifyChecksum(nonExistentFile); err == nil {
			t.Error("Expected error for non-existent file, but got none")
		}
	})
}

func TestGetKnownChecksum(t *testing.T) {
	// Test getting known checksum for v2.304
	checksum := GetKnownChecksum("v2.304")
	if checksum == "" {
		t.Error("Expected v2.304 to have a known checksum")
	}

	// Verify the checksum is a valid SHA-512 hex string (128 characters)
	if len(checksum) != 128 {
		t.Errorf("Expected SHA-512 checksum to be 128 characters, got %d", len(checksum))
	}

	// Test getting checksum for unknown version
	unknownChecksum := GetKnownChecksum("v99.99.99")
	if unknownChecksum != "" {
		t.Error("Expected empty string for unknown version")
	}
}
