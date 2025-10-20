package init

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jonnyzzz.com/devrig.dev/updates"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// mockUpdateService is a mock implementation of UpdateService for testing
type mockUpdateService struct{}

func (t *mockUpdateService) LastUpdateInfo() (*updates.UpdateInfo, error) {
	return nil, fmt.Errorf("not implemented for tests")
}

func (t *mockUpdateService) IsUpdateAvailable() (bool, error) {
	return false, fmt.Errorf("not implemented for tests")
}

// newTestInitCommand creates a new init command with mock dependencies for testing
func newTestInitCommand() *cobra.Command {
	return NewInitCommand(&mockUpdateService{})
}

// DevrigConfig represents the structure of devrig.yaml
type DevrigConfig struct {
	Devrig struct {
		Binaries map[string]struct {
			URL    string `yaml:"url"`
			SHA512 string `yaml:"sha512"`
		} `yaml:"binaries"`
	} `yaml:"devrig"`
}

func TestInitCommand_DefaultDirectory(t *testing.T) {
	// Create a temporary directory and change to it
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute the command
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--scripts-only"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output
	output := stdout.String()
	if !strings.Contains(output, "Initializing devrig.dev environment") {
		t.Errorf("Expected initialization message in output: %s", output)
	}
	if !strings.Contains(output, "Bootstrap scripts created successfully!") {
		t.Errorf("Expected success message in output: %s", output)
	}

	// Verify files were created
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}
}

func TestInitCommand_SpecificDirectory(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "my-project")

	// Execute the command
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--scripts-only", targetDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify files were created in the target directory
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist in %s", file, targetDir)
		}
	}
}

func TestInitCommand_RelativePath(t *testing.T) {
	// Create a temporary directory and change to it
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Execute the command with relative path
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--scripts-only", "./subdir"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify files were created
	targetDir := filepath.Join(tempDir, "subdir")
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist in %s", file, targetDir)
		}
	}
}

func TestInitCommand_ScriptsOnly(t *testing.T) {
	tempDir := t.TempDir()

	// Execute the command with --scripts-only flag
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--scripts-only", tempDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output mentions scripts-only mode
	output := stdout.String()
	if !strings.Contains(output, "Scripts-only mode") {
		t.Errorf("Expected scripts-only message in output: %s", output)
	}

	// Verify scripts were still created
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}
}

func TestInitCommand_NestedDirectory(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "level1", "level2", "level3")

	// Execute the command
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--scripts-only", targetDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Errorf("Nested directory was not created")
	}

	// Verify files were created
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist in nested directory", file)
		}
	}
}

func TestInitCommand_TooManyArgs(t *testing.T) {
	tempDir := t.TempDir()

	// Execute the command with too many arguments
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{tempDir, "extra-arg"})

	err := cmd.Execute()
	if err == nil {
		t.Errorf("Expected error when providing too many arguments")
	}
}

func TestInitCommand_InitFromLocal(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "local-init")

	// Execute the command with --init-from-local flag
	cmd := newTestInitCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--init-from-local", targetDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify output messages
	output := stdout.String()
	if !strings.Contains(output, "Initializing from local binary") {
		t.Errorf("Expected local binary message in output: %s", output)
	}
	if !strings.Contains(output, "Local initialization completed successfully!") {
		t.Errorf("Expected success message in output: %s", output)
	}

	// Verify bootstrap scripts were created
	scripts := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, script := range scripts {
		path := filepath.Join(targetDir, script)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected script %s does not exist", script)
		}
	}

	// Verify devrig.yaml was created
	yamlPath := filepath.Join(targetDir, "devrig.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Fatalf("devrig.yaml was not created")
	}

	// Read and parse devrig.yaml content
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read devrig.yaml: %v", err)
	}

	var config DevrigConfig
	if err := yaml.Unmarshal(yamlContent, &config); err != nil {
		t.Fatalf("Failed to parse devrig.yaml: %v", err)
	}

	// Get current platform
	osName := runtime.GOOS
	archName := runtime.GOARCH
	if archName == "amd64" {
		archName = "x86_64"
	}
	currentPlatform := fmt.Sprintf("%s-%s", osName, archName)

	// Verify only current platform is present
	if len(config.Devrig.Binaries) != 1 {
		t.Errorf("Expected 1 platform (current), got %d", len(config.Devrig.Binaries))
	}

	if _, exists := config.Devrig.Binaries[currentPlatform]; !exists {
		t.Errorf("Current platform %s not found in devrig.yaml", currentPlatform)
	}

	// Verify platform has URL and SHA512
	for platform, binary := range config.Devrig.Binaries {
		if binary.URL == "" {
			t.Errorf("Platform %s has empty URL", platform)
		}
		if !strings.Contains(binary.URL, "https://devrig.dev/local-build-fake-url") {
			t.Errorf("Platform %s URL doesn't contain expected fake URL: %s", platform, binary.URL)
		}
		if binary.SHA512 == "" {
			t.Errorf("Platform %s has empty SHA512", platform)
		}
		if len(binary.SHA512) != 128 {
			t.Errorf("Platform %s has invalid SHA512 length: %d (expected 128)", platform, len(binary.SHA512))
		}

		// Verify Windows binaries have .exe extension in URL
		if strings.HasPrefix(platform, "windows-") {
			if !strings.HasSuffix(binary.URL, ".exe") {
				t.Errorf("Windows platform %s URL doesn't have .exe extension: %s", platform, binary.URL)
			}
		}
	}

	// Verify .devrig directory was created
	devrigDir := filepath.Join(targetDir, ".devrig")
	if _, err := os.Stat(devrigDir); os.IsNotExist(err) {
		t.Fatalf(".devrig directory was not created")
	}

	// Verify binary was copied to .devrig folder
	entries, err := os.ReadDir(devrigDir)
	if err != nil {
		t.Fatalf("Failed to read .devrig directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("No directories found in .devrig folder")
	}

	// Find directory matching pattern devrig-<os>-<arch>-<hash>
	var binaryFound bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "devrig-") {
			// Check if binary exists in this directory
			binaryPath := filepath.Join(devrigDir, entry.Name())
			if _, err := os.Stat(binaryPath); err == nil {
				binaryFound = true
				t.Logf("Found binary at: %s", binaryPath)

				// Verify binary is executable (Unix-like systems)
				if runtime.GOOS != "windows" {
					info, err := os.Stat(binaryPath)
					if err != nil {
						t.Fatalf("Failed to stat binary: %v", err)
					}
					if info.Mode().Perm()&0111 == 0 {
						t.Errorf("BinaryInfo is not executable, mode: %v", info.Mode())
					}
				}
				break
			}
		}
	}

	if !binaryFound {
		t.Errorf("BinaryInfo not found in .devrig directory")
	}
}

func TestCalculateFileHash(t *testing.T) {
	// Create a temporary file with known content
	tempFile := filepath.Join(t.TempDir(), "test-file")
	content := []byte("test content for hashing")
	if err := os.WriteFile(tempFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate hash
	hash, err := calculateFileHash(tempFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Verify hash is not empty and has correct length (SHA512 produces 128 hex chars)
	if len(hash) != 128 {
		t.Errorf("Expected hash length 128, got %d", len(hash))
	}

	// Verify hash is consistent
	hash2, err := calculateFileHash(tempFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash second time: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Hash calculation is not consistent: %s != %s", hash, hash2)
	}

	// Verify hash changes with different content
	if err := os.WriteFile(tempFile, []byte("different content"), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	hash3, err := calculateFileHash(tempFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash for different content: %v", err)
	}

	if hash == hash3 {
		t.Errorf("Hash should be different for different content")
	}
}

func TestGenerateDevrigYaml(t *testing.T) {
	testHash := "abc123def456abc123def456abc123def456abc123def456abc123def456abc123def456abc123def456abc123def456abc123def456abc123def456abc123de"
	testPlatform := "linux-x86_64"

	config2 := generateDevrigYamlModel(testPlatform, testHash)
	yamlStr := generateDevrigYamlContent(config2)

	// Parse the generated YAML
	var config DevrigConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}

	// Verify only test platform is present
	if len(config.Devrig.Binaries) != 1 {
		t.Errorf("Expected 1 platform, got %d", len(config.Devrig.Binaries))
	}

	if _, exists := config.Devrig.Binaries[testPlatform]; !exists {
		t.Errorf("Test platform %s not found in generated YAML", testPlatform)
	}

	// Verify platform has correct URL and SHA512
	if testBinary, exists := config.Devrig.Binaries[testPlatform]; exists {
		// Verify URL structure
		if testBinary.URL == "" {
			t.Errorf("Platform %s has empty URL", testPlatform)
		}
		if !strings.Contains(testBinary.URL, "https://devrig.dev/local-build-fake-url") {
			t.Errorf("Platform %s URL doesn't contain fake URL: %s", testPlatform, testBinary.URL)
		}
		if !strings.Contains(testBinary.URL, testPlatform) {
			t.Errorf("Platform %s URL doesn't contain platform name: %s", testPlatform, testBinary.URL)
		}

		// Verify SHA512
		if testBinary.SHA512 == "" {
			t.Errorf("Platform %s has empty SHA512", testPlatform)
		}
		if len(testBinary.SHA512) != 128 {
			t.Errorf("Platform %s has invalid SHA512 length: %d (expected 128)\nHash: %q", testPlatform, len(testBinary.SHA512), testBinary.SHA512)
		}
		if testBinary.SHA512 != testHash {
			t.Errorf("Test platform %s has incorrect hash.\nExpected: %s\nGot: %s", testPlatform, testHash, testBinary.SHA512)
		}
	}

	// Test with Windows platform to verify .exe extension
	t.Run("WindowsPlatform", func(t *testing.T) {
		config3 := generateDevrigYamlModel("windows-x86_64", testHash)
		winYaml := generateDevrigYamlContent(config3)
		var winConfig DevrigConfig
		if err := yaml.Unmarshal([]byte(winYaml), &winConfig); err != nil {
			t.Fatalf("Failed to parse Windows YAML: %v", err)
		}

		if winBinary, exists := winConfig.Devrig.Binaries["windows-x86_64"]; exists {
			if !strings.HasSuffix(winBinary.URL, ".exe") {
				t.Errorf("Windows platform URL doesn't have .exe extension: %s", winBinary.URL)
			}
		} else {
			t.Errorf("Windows platform not found in generated YAML")
		}
	})
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create source file
	srcPath := filepath.Join(tempDir, "source.txt")
	srcContent := []byte("test content for copying")
	if err := os.WriteFile(srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy to destination
	dstPath := filepath.Join(tempDir, "destination.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination exists
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Fatalf("Destination file was not created")
	}

	// Verify content matches
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if !bytes.Equal(srcContent, dstContent) {
		t.Errorf("Destination content doesn't match source.\nExpected: %s\nGot: %s", srcContent, dstContent)
	}

	// Test copying to nested directory
	nestedDst := filepath.Join(tempDir, "nested", "dir", "file.txt")
	if err := os.MkdirAll(filepath.Dir(nestedDst), 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	if err := copyFile(srcPath, nestedDst); err != nil {
		t.Fatalf("Failed to copy file to nested directory: %v", err)
	}

	if _, err := os.Stat(nestedDst); os.IsNotExist(err) {
		t.Fatalf("File was not copied to nested directory")
	}
}

func TestInitializeFromLocalBinary(t *testing.T) {
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "init-target")

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Run initializeFromLocalBinary
	err := initializeFromLocalBinary(targetDir)
	if err != nil {
		t.Fatalf("initializeFromLocalBinary failed: %v", err)
	}

	// Verify devrig.yaml was created
	yamlPath := filepath.Join(targetDir, "devrig.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		t.Errorf("devrig.yaml was not created")
	}

	// Verify .devrig directory was created
	devrigDir := filepath.Join(targetDir, ".devrig")
	if _, err := os.Stat(devrigDir); os.IsNotExist(err) {
		t.Errorf(".devrig directory was not created")
	}

	// Verify binary directory exists and contains binary
	entries, err := os.ReadDir(devrigDir)
	if err != nil {
		t.Fatalf("Failed to read .devrig directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatalf("No directories found in .devrig folder")
	}

	var binaryFound bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "devrig-") {
			binaryName := ""
			if runtime.GOOS == "windows" {
				binaryName = ".exe"
			}
			binaryPath := filepath.Join(devrigDir, entry.Name()+binaryName)
			if info, err := os.Stat(binaryPath); err == nil {
				binaryFound = true

				// Verify binary size is non-zero
				if info.Size() == 0 {
					t.Errorf("BinaryInfo has zero size")
				}

				// Verify executable permissions on Unix-like systems
				if runtime.GOOS != "windows" {
					if info.Mode().Perm()&0111 == 0 {
						t.Errorf("BinaryInfo is not executable")
					}
				}
				break
			}
		}
	}

	if !binaryFound {
		t.Errorf("BinaryInfo was not copied to .devrig directory")
	}

	// Read and parse devrig.yaml
	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to read devrig.yaml: %v", err)
	}

	var config DevrigConfig
	if err := yaml.Unmarshal(yamlContent, &config); err != nil {
		t.Fatalf("Failed to parse devrig.yaml: %v", err)
	}

	// Get current platform
	osName := runtime.GOOS
	archName := runtime.GOARCH
	if archName == "amd64" {
		archName = "x86_64"
	}
	platform := fmt.Sprintf("%s-%s", osName, archName)

	// Verify only current platform is present
	if len(config.Devrig.Binaries) != 1 {
		t.Errorf("Expected 1 platform (current), got %d", len(config.Devrig.Binaries))
	}

	// Verify current platform exists in config
	if _, exists := config.Devrig.Binaries[platform]; !exists {
		t.Errorf("Current platform %s not found in devrig.yaml", platform)
	}

	// Verify platform has URL and SHA512
	for plat, binary := range config.Devrig.Binaries {
		if binary.URL == "" {
			t.Errorf("Platform %s has empty URL", plat)
		}
		if !strings.Contains(binary.URL, "https://devrig.dev/local-build-fake-url") {
			t.Errorf("Platform %s URL doesn't contain fake URL: %s", plat, binary.URL)
		}
		if binary.SHA512 == "" {
			t.Errorf("Platform %s has empty SHA512", plat)
		}
		if len(binary.SHA512) != 128 {
			t.Errorf("Platform %s has invalid SHA512 length: %d", plat, len(binary.SHA512))
		}
	}
}
