package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper function to create and parse test config
func parseTestConfig(t *testing.T, yaml string) (*ideConfigImpl, error) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".ides.yaml")
	err := os.WriteFile(configPath, []byte(yaml), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return parseConfigFile(configPath)
}

// helper function to verify IDE config
func assertIDEConfig(t *testing.T, got *ideConfigImpl, want *ideConfigImpl) {
	t.Helper()
	if got.Name() != want.Name() {
		t.Errorf("NameV() = %v, want %v", got.Name(), want.Name())
	}
	if got.Version() != want.Version() {
		t.Errorf("Version() = %v, want %v", got.Version(), want.Version())
	}
	if got.Build() != want.Build() {
		t.Errorf("BuildV() = %v, want %v", got.Build(), want.Build())
	}
}

func TestParseValidConfig(t *testing.T) {
	yaml := `
ide:
  name: GoLand
  version: 2024.3
  build: 243.123
`

	got, err := parseTestConfig(t, yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := &ideConfigImpl{
		NameV:    "GoLand",
		VersionV: "2024.3",
		BuildV:   "243.123",
	}
	assertIDEConfig(t, got, want)
}

func TestParseMissingName(t *testing.T) {
	yaml := `ide:
  version: 2024.3`

	_, err := parseTestConfig(t, yaml)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "IDE name is required") {
		t.Errorf("error message = %v, want containing 'IDE name is required'", err)
	}
}

func TestParseMissingVersion(t *testing.T) {
	yaml := `
ide:
  name: GoLand
`

	_, err := parseTestConfig(t, yaml)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "IDE version is required") {
		t.Errorf("error message = %v, want containing 'IDE version is required'", err)
	}
}

func TestParseEmptyConfig(t *testing.T) {
	_, err := parseTestConfig(t, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing ide configuration") {
		t.Errorf("error message = %v, want containing 'missing ide configuration'", err)
	}
}

func TestParseOptionalBuild(t *testing.T) {
	yaml := `
ide:
  name: GoLand
  version: 2024.3
`

	got, err := parseTestConfig(t, yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := &ideConfigImpl{
		NameV:    "GoLand",
		VersionV: "2024.3",
		BuildV:   "",
	}
	assertIDEConfig(t, got, want)
}
