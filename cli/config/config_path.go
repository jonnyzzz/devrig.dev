package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindConfigFile searches for .idew.yaml file starting from the given directory
// and moving up the directory tree until it finds the file or reaches the root.
func FindConfigFile(startDir string) (string, error) {
	configFileName := ".idew.yaml"
	dir := startDir

	visitedDirs := make(map[string]bool) // Track visited directories for symlink safety

	for {
		// Resolve the absolute path to prevent issues with symlinks and duplicates
		absDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlink in directory %s: %w", dir, err)
		}

		// Check for infinite loops due to symlinks
		if visitedDirs[absDir] {
			return "", fmt.Errorf("potential symlink loop detected in directory %s", absDir)
		}
		visitedDirs[absDir] = true

		// Construct the full path to the potential config file
		configPath := filepath.Join(absDir, configFileName)

		// Check if the file exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Found configuration file at: %s\n", configPath)
			return configPath, nil
		}

		// Get the parent directory
		parent := filepath.Dir(absDir)

		// If we've reached the root directory, stop searching
		if parent == absDir {
			return "", fmt.Errorf("configuration file %s not found in any parent directory", configFileName)
		}

		// Move up to the parent directory
		dir = parent
	}
}
