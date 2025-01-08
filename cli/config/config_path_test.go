package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindConfigFileInCurrentDir tests that the config file is found in the current directory
func TestFindConfigFileInCurrentDir(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create the config file in the current directory
	configFilePath := filepath.Join(tempDir, ".idew.yaml")
	err := os.WriteFile(configFilePath, []byte("config: test"), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Test: Should find the file in the current directory
	foundPath, err := FindConfigFile(tempDir)
	if err != nil {
		t.Errorf("expected to find config file, but got error: %v", err)
	}
	if foundPath != configFilePath {
		t.Errorf("expected config file path %s, but got %s", configFilePath, foundPath)
	}
}

// TestFindConfigFileInParentDir tests that the function finds the file in a parent directory
func TestFindConfigFileInParentDir(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Create the config file in the parent directory
	configFilePath := filepath.Join(tempDir, ".idew.yaml")
	err = os.WriteFile(configFilePath, []byte("config: test"), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Test: Should find the file in the parent directory
	foundPath, err := FindConfigFile(subDir)
	if err != nil {
		t.Errorf("expected to find config file, but got error: %v", err)
	}
	if foundPath != configFilePath {
		t.Errorf("expected config file path %s, but got %s", configFilePath, foundPath)
	}
}

// TestFindConfigFileNotFound tests that an error is returned when the config file is not found
func TestFindConfigFileNotFound(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Test: Should return an error since no config file exists
	_, err := FindConfigFile(tempDir)
	if err == nil {
		t.Errorf("expected an error when config file does not exist, but got none")
	}
}

// TestFindConfigFileSymlinkLoop tests that the function handles symlink loops gracefully
func TestFindConfigFileSymlinkLoop(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	symlinkDir := filepath.Join(tempDir, "symlinkDir")

	// Create a symlink pointing back to the same directory
	err := os.Symlink(tempDir, symlinkDir)
	if err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Test: Should return an error due to symlink loop detection
	_, err = FindConfigFile(symlinkDir)
	if err == nil {
		t.Errorf("expected an error due to symlink loop, but got none")
	}
}
