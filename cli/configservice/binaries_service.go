package configservice

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
)

// DevrigBinariesService manages the devrig binaries configuration
type DevrigBinariesService interface {
	// ReadDevrigSection reads and parses the devrig section from devrig.yaml
	// Returns error if file doesn't exist, can't be parsed, or validation fails
	ReadDevrigSection() (*DevrigSection, error)

	// UpdateBinaries updates or creates devrig.yaml with the given binaries information
	// If the file doesn't exist, it creates it with proper headers
	// If the file exists, it updates only the devrig section while preserving comments and formatting
	UpdateBinaries(section *DevrigSection) error
}

// UpdateBinaries updates or creates devrig.yaml with the given binaries information
func (s *configServiceImpl) UpdateBinaries(section *DevrigSection) error {
	configPath := s.configPath
	// Validate the section first
	if err := validateDevrigSection(section); err != nil {
		return fmt.Errorf("invalid section: %w", err)
	}

	// Check if file exists
	_, err := os.Stat(configPath)
	fileExists := err == nil

	if !fileExists {
		// Create new file
		return s.createNewConfig(section)
	}

	// Update existing file
	return s.updateExistingConfig(section)
}

// createNewConfig creates a new devrig.yaml file
func (s *configServiceImpl) createNewConfig(section *DevrigSection) error {
	// Marshal the section
	yamlBytes, err := yaml.Marshal(map[string]interface{}{
		"devrig": section,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal section: %w", err)
	}

	// Add header comments
	header := "# devrig.yaml - Main configuration file for devrig tool\n"
	header += "# This file contains URLs and hash sums for devrig binaries across all supported platforms\n\n"
	yamlBytes = []byte(header + string(yamlBytes))

	devrigDir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(devrigDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devrig directory: %w", err)
	}
	log.Printf("Created .devrig directory at: %s\n", devrigDir)

	// Write to file
	if err := os.WriteFile(s.configPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}
	return nil
}

// updateExistingConfig updates an existing devrig.yaml file while preserving formatting
func (s *configServiceImpl) updateExistingConfig(section *DevrigSection) error {
	// Read the original file
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to read existing configuration: %w", err)
	}

	// Parse with comments to preserve formatting
	file, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse existing configuration: %w", err)
	}

	// Update the devrig section in the AST using path-based approach
	path, err := yaml.PathString("$.devrig")
	if err != nil {
		return fmt.Errorf("failed to create path: %w", err)
	}

	// Marshal the new section
	newYaml, err := yaml.Marshal(section)
	if err != nil {
		return fmt.Errorf("failed to marshal new section: %w", err)
	}

	// Parse the new section to get an AST node
	newFile, err := parser.ParseBytes(newYaml, 0)
	if err != nil {
		return fmt.Errorf("failed to parse new section: %w", err)
	}

	if len(newFile.Docs) == 0 || newFile.Docs[0].Body == nil {
		return fmt.Errorf("new section has no body")
	}

	newNode := newFile.Docs[0].Body

	// Replace the node at the path
	if err := path.ReplaceWithNode(file, newNode); err != nil {
		return fmt.Errorf("failed to replace node: %w", err)
	}

	// Write the updated AST back to file
	if err := os.WriteFile(s.configPath, []byte(file.String()), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}
