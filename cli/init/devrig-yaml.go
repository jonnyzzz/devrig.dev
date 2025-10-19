package init

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type DevrigYamlConfig struct {
	Devrig DevrigSection `yaml:"devrig"`
}

type DevrigSection struct {
	Version     string                `yaml:"version,omitempty"`
	ReleaseDate string                `yaml:"release_date,omitempty"`
	Binaries    map[string]BinaryInfo `yaml:"binaries"`
}

type BinaryInfo struct {
	URL    string `yaml:"url"`
	SHA512 string `yaml:"sha512"`
}

func writeNewDevrigYaml(targetDir string, config DevrigYamlConfig) error {
	yamlContent := generateDevrigYamlContent(config)

	// Write devrig.yaml
	yamlPath := filepath.Join(targetDir, "devrig.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write devrig.yaml: %w", err)
	}
	log.Printf("Created devrig.yaml at: %s\n", yamlPath)
	return nil
}

func generateDevrigYamlContent(config DevrigYamlConfig) string {
	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(&config)
	if err != nil {
		// Fallback to simple string if marshaling fails
		log.Fatalf("# Error generating YAML: %v\n", err)
		return ""
	}

	// Add header comments
	header := "# devrig.yaml - Main configuration file for devrig tool\n"
	header += "# This file contains URLs and hash sums for devrig binaries across all supported platforms\n\n"

	return header + string(yamlBytes)
}

func generateDevrigYamlModel(currentPlatform string, currentHash string) DevrigYamlConfig {
	url := fmt.Sprintf("https://devrig.dev/local-build-fake-url/%s", currentPlatform)
	if strings.Contains(currentPlatform, "windows") {
		url += ".exe"
	}

	config := DevrigYamlConfig{
		Devrig: DevrigSection{
			Binaries: map[string]BinaryInfo{
				currentPlatform: {
					URL:    url,
					SHA512: currentHash,
				},
			},
		},
	}
	return config
}
