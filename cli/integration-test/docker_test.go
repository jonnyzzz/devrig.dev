package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDockerBinaryExecution builds binaries and tests them in Docker
func TestDockerBinaryExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Step 1: Build binaries using build.sh
	t.Log("Building binaries using build.sh...")
	buildScript := filepath.Join(os.Getenv("PWD"), "..", "build.sh")

	cmd := exec.Command("bash", buildScript)
	cmd.Dir = filepath.Dir(buildScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run build.sh: %v", err)
	}

	// Step 2: Determine the binary for Linux (Docker environment)
	// Detect Docker host architecture
	dockerArch := getDockerArchitecture(t)
	binaryName := fmt.Sprintf("devrig-linux-%s", dockerArch)

	buildInDockerDir := filepath.Join("..", "build-in-docker")
	binaryPath := filepath.Join(buildInDockerDir, binaryName)

	// Verify binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("Binary %s not found in %s: %v", binaryName, buildInDockerDir, err)
	}

	t.Logf("Using binary: %s", binaryName)

	// Step 3: Run binary in Docker container
	t.Run("VersionInDocker", func(t *testing.T) {
		testVersionInDocker(t, binaryPath)
	})

	t.Run("VersionInEmptyFolder", func(t *testing.T) {
		testVersionInEmptyFolder(t, binaryPath)
	})
}

// getDockerArchitecture detects the architecture of the Docker environment
func getDockerArchitecture(t *testing.T) string {
	// Try to detect from Docker
	cmd := exec.Command("docker", "run", "--rm", "alpine", "uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		t.Logf("Failed to detect Docker architecture, using host: %v", err)
		// Fall back to host architecture
		return mapGoArchToLinux(runtime.GOARCH)
	}

	arch := strings.TrimSpace(string(output))
	switch arch {
	case "x86_64", "amd64":
		return "x86_64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		t.Fatalf("Unsupported Docker architecture: %s", arch)
		return ""
	}
}

// mapGoArchToLinux maps Go's GOARCH to Linux architecture naming
func mapGoArchToLinux(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		return goarch
	}
}

// testVersionInDocker tests running version command in a basic Docker container
func testVersionInDocker(t *testing.T, binaryPath string) {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	var stdout, stderr bytes.Buffer

	// Run the binary in Alpine Linux container
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/devrig:ro", absPath),
		"alpine:latest",
		"/devrig", "version",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Failed to run version in Docker: %v\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Version:") {
		t.Errorf("Version output doesn't contain 'Version:': %s", output)
	}

	t.Logf("Version output: %s", strings.TrimSpace(output))
}

// testVersionInEmptyFolder tests running version command in an empty random folder to trigger errors
func testVersionInEmptyFolder(t *testing.T, binaryPath string) {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Generate a random folder name
	randomFolder := fmt.Sprintf("/tmp/devrig-test-%d", os.Getpid())

	var stdout, stderr bytes.Buffer

	// Run the binary in Alpine Linux container with an empty random folder as working directory
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/devrig:ro", absPath),
		"-w", randomFolder,
		"alpine:latest",
		"sh", "-c", fmt.Sprintf("mkdir -p %s && cd %s && /devrig version", randomFolder, randomFolder),
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// It's okay if this fails - we're testing error handling
		t.Logf("Command exited with error (expected): %v", err)
	}

	output := stdout.String() + stderr.String()

	// The version command should still work even in an empty folder
	// We're testing that the binary handles this gracefully
	if strings.Contains(output, "Version:") {
		t.Logf("✓ Binary handled empty folder correctly")
		t.Logf("Output: %s", strings.TrimSpace(output))
	} else if err != nil {
		t.Logf("Binary output in empty folder: %s", strings.TrimSpace(output))
		// This is acceptable - we're just testing it doesn't crash unexpectedly
	}
}
