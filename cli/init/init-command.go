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

var scriptsOnly bool
var initFromLocal bool

var Cmd *cobra.Command

func init() {
	Cmd = &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize the devrig.dev environment",
		Args:  cobra.MaximumNArgs(1),
		RunE:  doTheCommand,
	}
	Cmd.Flags().BoolVar(&scriptsOnly, "scripts-only", false, "Only generate bootstrap scripts")
	Cmd.Flags().BoolVar(&initFromLocal, "init-from-local", false, "Initialize with the current binary and generate devrig.yaml")
}

func doTheCommand(cmd *cobra.Command, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory path: %w", err)
	}
	log.Printf("Resolved target directory to: %s\n", absPath)

	// Ensure directory exists
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	log.Printf("Created directory: %s\n", absPath)

	cmd.Printf("Initializing devrig.dev environment in: %s\n", absPath)

	// Copy bootstrap scripts
	if err := bootstrap.CopyBootstrapScripts(absPath); err != nil {
		return fmt.Errorf("failed to copy bootstrap scripts: %w", err)
	}
	log.Println("Bootstrap scripts created successfully!")

	cmd.Println("Bootstrap scripts created successfully!")

	if scriptsOnly {
		cmd.Println("Scripts-only mode: Skipping additional initialization")
		return nil
	}

	if initFromLocal {
		cmd.Println("Initializing from local binary...")
		if err := initializeFromLocalBinary(absPath); err != nil {
			return fmt.Errorf("failed to initialize from local binary: %w", err)
		}
		log.Println("Local initialization completed successfully!")
		return nil
	}

	return nil
}

// initializeFromLocalBinary creates devrig.yaml and copies the current binary to .devrig folder
func initializeFromLocalBinary(targetDir string) error {
	log.Println("Initializing from local binary...")

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	log.Printf("Executable path: %s\n", execPath)

	// Resolve symlinks if any
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}
	log.Printf("Resolved executable path: %s\n", execPath)

	// Calculate hash of the current binary
	hash, err := calculateFileHash(execPath)
	if err != nil {
		return fmt.Errorf("failed to calculate binary hash: %w", err)
	}
	log.Printf("Calculated binary hash: %s\n", hash)

	// Determine OS and architecture
	osName := runtime.GOOS
	archName := runtime.GOARCH
	if archName == "amd64" {
		archName = "x86_64"
	}
	platform := fmt.Sprintf("%s-%s", osName, archName)
	log.Printf("Determined platform: %s\n", platform)

	// Generate devrig.yaml content
	yamlContent := generateDevrigYaml(platform, hash)

	// Write devrig.yaml
	yamlPath := filepath.Join(targetDir, "devrig.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write devrig.yaml: %w", err)
	}
	log.Printf("Created devrig.yaml at: %s\n", yamlPath)

	// Create .devrig directory
	devrigDir := filepath.Join(targetDir, ".devrig")
	if err := os.MkdirAll(devrigDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devrig directory: %w", err)
	}
	log.Printf("Created .devrig directory at: %s\n", devrigDir)

	// Determine binary name based on the layout: .devrig/<tool-name>-<os>-<cpu-type>-<hash>/binary
	binaryName := fmt.Sprintf("devrig-%s-%s-%s", osName, archName, hash)
	if osName == "windows" {
		binaryName += ".exe"
	}
	log.Printf("Determined binary name: %s\n", binaryName)

	// Copy binary to .devrig folder
	destPath := filepath.Join(devrigDir, binaryName)
	if err := copyFile(execPath, destPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	log.Printf("Copied binary to: %s\n", destPath)

	// Set executable permissions (Unix-like systems)
	if osName != "windows" {
		if err := os.Chmod(destPath, 0755); err != nil {
			return fmt.Errorf("failed to set executable permissions: %w", err)
		}
		log.Printf("Set executable permissions for: %s\n", destPath)
	}

	log.Println("Local initialization completed successfully!")
	return nil
}

// calculateFileHash calculates the SHA512 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()

	hash := sha512.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

type DevrigYamlConfig struct {
	Devrig DevrigSection `yaml:"devrig"`
}

type DevrigSection struct {
	Binaries map[string]BinaryInfo `yaml:"binaries"`
}

type BinaryInfo struct {
	URL    string `yaml:"url"`
	SHA512 string `yaml:"sha512"`
}

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

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}
