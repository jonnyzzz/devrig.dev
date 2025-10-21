package configservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevrigBinariesService_UpdateBinaries_CreateNewFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	configService := NewConfigService(testFile)

	section := &DevrigSection{
		Version:     "v0.80.0",
		ReleaseDate: "2025-01-15",
		Binaries: map[string]BinaryInfo{
			"darwin-arm64": {
				URL:    "https://example.com/devrig-darwin-arm64",
				SHA512: strings.Repeat("a", 128),
			},
			"linux-x86_64": {
				URL:    "https://example.com/devrig-linux-x86_64",
				SHA512: strings.Repeat("b", 128),
			},
		},
	}

	err := configService.Binaries().UpdateBinaries(section)
	if err != nil {
		t.Fatalf("Failed to create new config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Read and verify content
	readSection, err := configService.Binaries().ReadDevrigSection()
	if err != nil {
		t.Fatalf("Failed to read created config: %v", err)
	}

	if readSection.Version != "v0.80.0" {
		t.Errorf("Expected version v0.80.0, got: %s", readSection.Version)
	}

	if readSection.ReleaseDate != "2025-01-15" {
		t.Errorf("Expected release_date 2025-01-15, got: %s", readSection.ReleaseDate)
	}

	if len(readSection.Binaries) != 2 {
		t.Errorf("Expected 2 binaries, got: %d", len(readSection.Binaries))
	}

	// Verify header comments were added
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# devrig.yaml") {
		t.Error("Expected header comment not found")
	}
}

func TestDevrigBinariesService_UpdateBinaries_UpdateExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	configService := NewConfigService(testFile)

	// Create initial config with comments
	initialContent := `# This is my custom header
# Don't overwrite this

devrig:
  version: v0.79.0
  release_date: "2025-01-10"
  binaries:
    darwin-arm64:
      url: https://example.com/old-binary
      sha512: ` + strings.Repeat("a", 128) + `

# Some other section
other:
  key: value
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Update with new binaries
	newSection := &DevrigSection{
		Version:     "v0.80.0",
		ReleaseDate: "2025-01-15",
		Binaries: map[string]BinaryInfo{
			"darwin-arm64": {
				URL:    "https://example.com/new-darwin-arm64",
				SHA512: strings.Repeat("c", 128),
			},
			"linux-x86_64": {
				URL:    "https://example.com/new-linux-x86_64",
				SHA512: strings.Repeat("d", 128),
			},
		},
	}

	err := configService.Binaries().UpdateBinaries(newSection)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Read and verify updated content
	readSection, err := configService.Binaries().ReadDevrigSection()
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	if readSection.Version != "v0.80.0" {
		t.Errorf("Expected version v0.80.0, got: %s", readSection.Version)
	}

	if readSection.ReleaseDate != "2025-01-15" {
		t.Errorf("Expected release_date 2025-01-15, got: %s", readSection.ReleaseDate)
	}

	if len(readSection.Binaries) != 2 {
		t.Errorf("Expected 2 binaries, got: %d", len(readSection.Binaries))
	}

	linux, ok := readSection.Binaries["linux-x86_64"]
	if !ok {
		t.Fatal("Expected linux-x86_64 binary")
	}
	if !strings.Contains(linux.URL, "new-linux-x86_64") {
		t.Errorf("Expected updated URL, got: %s", linux.URL)
	}

	// Verify comments and other sections were preserved
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "This is my custom header") {
		t.Error("Custom header comment was not preserved")
	}
	if !strings.Contains(content, "Some other section") {
		t.Error("Comment before other section was not preserved")
	}
	if !strings.Contains(content, "other:") {
		t.Error("Other YAML section was not preserved")
	}
}

func TestDevrigBinariesService_UpdateBinaries_InvalidSection(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	configService := NewConfigService(testFile)

	testCases := []struct {
		name    string
		section *DevrigSection
		errText string
	}{
		{
			name:    "nil section",
			section: nil,
			errText: "devrig section is empty",
		},
		{
			name: "no binaries",
			section: &DevrigSection{
				Version:  "v0.80.0",
				Binaries: map[string]BinaryInfo{},
			},
			errText: "no binaries configured",
		},
		{
			name: "missing URL",
			section: &DevrigSection{
				Version: "v0.80.0",
				Binaries: map[string]BinaryInfo{
					"darwin-arm64": {
						URL:    "",
						SHA512: strings.Repeat("a", 128),
					},
				},
			},
			errText: "missing URL",
		},
		{
			name: "missing SHA512",
			section: &DevrigSection{
				Version: "v0.80.0",
				Binaries: map[string]BinaryInfo{
					"darwin-arm64": {
						URL:    "https://example.com/binary",
						SHA512: "",
					},
				},
			},
			errText: "missing SHA512",
		},
		{
			name: "invalid SHA512 length",
			section: &DevrigSection{
				Version: "v0.80.0",
				Binaries: map[string]BinaryInfo{
					"darwin-arm64": {
						URL:    "https://example.com/binary",
						SHA512: "tooshort",
					},
				},
			},
			errText: "invalid SHA512 hash length",
		},
		{
			name: "non-hex SHA512",
			section: &DevrigSection{
				Version: "v0.80.0",
				Binaries: map[string]BinaryInfo{
					"darwin-arm64": {
						URL:    "https://example.com/binary",
						SHA512: strings.Repeat("g", 128),
					},
				},
			},
			errText: "non-hexadecimal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := configService.Binaries().UpdateBinaries(tc.section)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errText) {
				t.Errorf("Expected error containing '%s', got: %v", tc.errText, err)
			}
		})
	}
}

func TestDevrigBinariesService_UpdateBinaries_PreservesFormatting(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	configService := NewConfigService(testFile)

	// Create initial config with specific formatting
	initialContent := `# Header comment
devrig:
  version: v0.79.0  # inline comment
  binaries:
    darwin-arm64:
      url: https://example.com/old
      sha512: ` + strings.Repeat("a", 128) + `

other_section:
  key: value
  nested:
    deep: "data"
`
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Update binaries
	newSection := &DevrigSection{
		Version:     "v0.80.0",
		ReleaseDate: "2025-01-15",
		Binaries: map[string]BinaryInfo{
			"linux-x86_64": {
				URL:    "https://example.com/linux",
				SHA512: strings.Repeat("b", 128),
			},
		},
	}

	err := configService.Binaries().UpdateBinaries(newSection)
	if err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Verify other_section still exists and is valid YAML
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)

	// Check that other sections are preserved
	if !strings.Contains(content, "other_section:") {
		t.Error("other_section was removed")
	}
	if !strings.Contains(content, "nested:") {
		t.Error("Nested structure was removed")
	}

	// Verify the entire file is still valid YAML by attempting to read it with the config service
	_, err = configService.Binaries().ReadDevrigSection()
	if err != nil {
		t.Fatalf("Updated file is not valid YAML: %v", err)
	}
}

func TestDevrigBinariesService_UpdateBinaries_MinimalSection(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "devrig.yaml")

	configService := NewConfigService(testFile)

	// Create minimal section (no version or release_date)
	section := &DevrigSection{
		Binaries: map[string]BinaryInfo{
			"darwin-arm64": {
				URL:    "https://example.com/binary",
				SHA512: strings.Repeat("a", 128),
			},
		},
	}

	err := configService.Binaries().UpdateBinaries(section)
	if err != nil {
		t.Fatalf("Failed to create minimal config: %v", err)
	}

	// Read and verify
	readSection, err := configService.Binaries().ReadDevrigSection()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	if len(readSection.Binaries) != 1 {
		t.Errorf("Expected 1 binary, got: %d", len(readSection.Binaries))
	}

	// Version and ReleaseDate should be empty
	if readSection.Version != "" {
		t.Errorf("Expected empty version, got: %s", readSection.Version)
	}
	if readSection.ReleaseDate != "" {
		t.Errorf("Expected empty release_date, got: %s", readSection.ReleaseDate)
	}
}
