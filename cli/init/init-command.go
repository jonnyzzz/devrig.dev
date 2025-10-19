package init

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"jonnyzzz.com/devrig.dev/bootstrap"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)


var Cmd *cobra.Command






func generateDevrigYaml(currentPlatform, currentHash string) string {
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

