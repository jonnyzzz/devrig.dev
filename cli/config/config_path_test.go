package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindConfigFileRelativePath tests that the function resolves the relative path "." correctly
func TestFindConfigFileRelativePath(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Switch to the temporary directory using os.Chdir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get the current working directory: %v", err)
	}
	defer os.Chdir(originalDir) // Ensure to restore the original directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("failed to change directory to temporary directory: %v", err)
	}

	// Create the config file in the current (temp) directory
	configFilePath := filepath.Join(tempDir, ".idew.yaml")
	err = os.WriteFile(configFilePath, []byte("config: test"), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	// Test: Should find the file using "."
	foundPath, err := FindConfigFile(".")
	if err != nil || foundPath == "" {
		t.Errorf("expected to find config file, but got error: %v", err)
	}
	foundPath, _ = filepath.EvalSymlinks(foundPath)
	configFilePath, _ = filepath.EvalSymlinks(configFilePath)
	if foundPath != configFilePath {
		t.Errorf("expected config file path %s, but got %s", configFilePath, foundPath)
	}
}

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
	if err != nil || foundPath == "" {
		t.Errorf("expected to find config file, but got error: %v", err)
	}
	foundPath, _ = filepath.EvalSymlinks(foundPath)
	configFilePath, _ = filepath.EvalSymlinks(configFilePath)
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
	if err != nil || foundPath == "" {
		t.Errorf("expected to find config file, but got error: %v", err)
	}
	foundPath, _ = filepath.EvalSymlinks(foundPath)
	configFilePath, _ = filepath.EvalSymlinks(configFilePath)
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
