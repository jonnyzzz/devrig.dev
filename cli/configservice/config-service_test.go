package configservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigService_ReadDevrigSection_Basic(t *testing.T) {
	service := NewConfigService("testdata/basic.yaml")

	section, err := service.Binaries().ReadDevrigSection()
	if err != nil {
		t.Fatalf("Failed to read devrig section: %v", err)
	}

	if section.Version != "v0.79.6" {
		t.Errorf("Expected version v0.79.6, got: %s", section.Version)
	}

	if section.ReleaseDate != "2025-01-15" {
		t.Errorf("Expected release_date 2025-01-15, got: %s", section.ReleaseDate)
	}

	if len(section.Binaries) != 2 {
		t.Errorf("Expected 2 binaries, got: %d", len(section.Binaries))
	}

	// Check darwin-arm64
	darwin, ok := section.Binaries["darwin-arm64"]
	if !ok {
		t.Fatal("Expected darwin-arm64 binary")
	}
	if !strings.Contains(darwin.URL, "devrig-darwin-arm64") {
		t.Errorf("Unexpected URL: %s", darwin.URL)
	}
	if len(darwin.SHA512) != 128 {
		t.Errorf("Expected SHA512 length 128, got: %d", len(darwin.SHA512))
	}
}

func TestConfigService_ReadDevrigSection_AllTestFixtures(t *testing.T) {
	testCases := []string{
		"basic.yaml",
		"with-inline-comments.yaml",
		"with-multiline-comments.yaml",
		"mixed-indentation.yaml",
		"with-other-sections.yaml",
		"quoted-strings.yaml",
		"flow-style.yaml",
		"minimal-no-version.yaml",
		"extra-blank-lines.yaml",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			service := NewConfigService(filepath.Join("testdata", tc))
			section, err := service.Binaries().ReadDevrigSection()
			if err != nil {
				t.Fatalf("Failed to read devrig section from %s: %v", tc, err)
			}

			// All test files should have at least one binary
			if len(section.Binaries) == 0 {
				t.Errorf("Expected at least one binary in %s", tc)
			}
		})
	}
}

func TestConfigService_ReadDevrigSection_MissingFile(t *testing.T) {
	service := NewConfigService("/nonexistent/devrig.yaml")

	_, err := service.Binaries().ReadDevrigSection()
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestConfigService_ReadDevrigSection_MissingDevrigSection(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	// Create a YAML file without devrig section
	yamlContent := "other_section:\n  key: value\n"
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	service := NewConfigService(testFile)

	_, err := service.Binaries().ReadDevrigSection()
	if err == nil {
		t.Error("Expected error for missing devrig section, got nil")
	}
	if !strings.Contains(err.Error(), "devrig section not found") {
		t.Errorf("Expected 'devrig section not found' error, got: %v", err)
	}
}

func TestConfigService_ReadDevrigSection_InvalidHash(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	// Create a YAML with invalid hash (too short)
	yamlContent := `devrig:
  binaries:
    linux-x86_64:
      url: https://example.com/binary
      sha512: tooshort
`
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	service := NewConfigService(testFile)

	_, err := service.Binaries().ReadDevrigSection()
	if err == nil {
		t.Error("Expected validation error for invalid hash, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SHA512 hash length") {
		t.Errorf("Expected hash length error, got: %v", err)
	}
}

func TestConfigService_ReadDevrigSection_NonHexHash(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	// Create a YAML with non-hex characters in hash
	invalidHash := strings.Repeat("g", 128) // 'g' is not a hex character
	yamlContent := "devrig:\n  binaries:\n    linux-x86_64:\n      url: https://example.com/binary\n      sha512: " + invalidHash + "\n"
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	service := NewConfigService(testFile)

	_, err := service.Binaries().ReadDevrigSection()
	if err == nil {
		t.Error("Expected validation error for non-hex hash, got nil")
	}
	if !strings.Contains(err.Error(), "non-hexadecimal") {
		t.Errorf("Expected non-hexadecimal error, got: %v", err)
	}
}

func TestConfigService_EnsureValidConfig_FileExists(t *testing.T) {
	service := NewConfigService("testdata/basic.yaml")

	err := service.EnsureValidConfig()
	if err != nil {
		t.Errorf("EnsureValidConfig failed for valid file: %v", err)
	}
}

func TestConfigService_EnsureValidConfig_FileMissing(t *testing.T) {
	service := NewConfigService("/nonexistent/devrig.yaml")

	err := service.EnsureValidConfig()
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "devrig init") {
		t.Errorf("Expected helpful message about 'devrig init', got: %v", err)
	}
}

func TestConfigService_EnsureValidConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	// Create invalid YAML
	yamlContent := "devrig:\n  binaries: {invalid syntax here"
	if err := os.WriteFile(testFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	service := NewConfigService(testFile)

	err := service.EnsureValidConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected 'invalid' in error message, got: %v", err)
	}
}
