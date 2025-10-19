package init

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"jonnyzzz.com/devrig.dev/bootstrap"
	"jonnyzzz.com/devrig.dev/updates"

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
	cmd.Printf("Initializing devrig.dev environment in: %s\n", absPath)

	// Copy bootstrap scripts
	if err := bootstrap.CopyBootstrapScripts(absPath); err != nil {
		return fmt.Errorf("failed to copy bootstrap scripts: %w", err)
	}
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
		cmd.Println("Local initialization completed successfully!")
		return nil
	}

	return initializeFromUpdates(cmd, targetDir)
}

func initializeFromUpdates(cmd *cobra.Command, targetDir string) error {
	client := updates.NewClient()
	updateInfo, err := client.FetchLatestUpdateInfo()
	if err != nil {
		cmd.PrintErr("Failed to fetch latest update information, ", err)
		return err
	}

	config := DevrigYamlConfig{
		Devrig: DevrigSection{
			Version:     updateInfo.Version,
			ReleaseDate: updateInfo.ReleaseDate,
			Binaries:    map[string]BinaryInfo{},
		},
	}

	for _, b := range updateInfo.Binaries {
		config.Devrig.Binaries[fmt.Sprintf("%s-%s", b.OS, b.Arch)] = BinaryInfo{
			URL:    b.URL,
			SHA512: b.SHA512,
		}
	}

	// Generate devrig.yaml content
	return writeNewDevrigYaml(targetDir, config)
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
	config := generateDevrigYamlModel(platform, hash)
	err = writeNewDevrigYaml(targetDir, config)
	if err != nil {
		return err
	}

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
