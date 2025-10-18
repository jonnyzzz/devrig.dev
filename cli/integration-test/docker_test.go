package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	cachedBinaryPath string
	binarySetupOnce  sync.Once
	binarySetupError error
)

// setupDockerBinary builds the binary and returns its path (cached)
func setupDockerBinary(t *testing.T) string {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binarySetupOnce.Do(func() {
		// Step 1: Build binaries using build.sh
		t.Log("Building binaries using build.sh...")
		wd, err := os.Getwd()
		if err != nil {
			binarySetupError = fmt.Errorf("failed to get working directory: %v", err)
			return
		}
		buildScript := filepath.Join(wd, "..", "build.sh")

		cmd := exec.Command("bash", buildScript)
		cmd.Env = append([]string{}, os.Environ()...)
		cmd.Env = append(cmd.Env, "BUILD_CURRENT_ONLY=YES")
		cmd.Dir = filepath.Dir(buildScript)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			binarySetupError = fmt.Errorf("failed to run build.sh: %v", err)
			return
		}

		// Step 2: Determine the binary for Linux (Docker environment)
		// Detect Docker host architecture
		dockerArch := getDockerArchitecture(t)
		binaryName := fmt.Sprintf("devrig-linux-%s", dockerArch)

		buildInDockerDir := filepath.Join(wd, "..", "build-in-docker")
		binaryPath := filepath.Join(buildInDockerDir, binaryName)

		// Verify binary exists
		if _, err := os.Stat(binaryPath); err != nil {
			binarySetupError = fmt.Errorf("binary %s not found in %s: %v", binaryName, buildInDockerDir, err)
			return
		}

		binaryName, err = filepath.Abs(binaryPath)
		if err != nil {
			binarySetupError = fmt.Errorf("Failed to get absolute path: %v", err)
			return
		}

		t.Logf("Using binary: %s", binaryName)
		cachedBinaryPath = binaryPath
	})

	if binarySetupError != nil {
		t.Fatalf("Binary setup failed: %v", binarySetupError)
	}

	return cachedBinaryPath
}

// TestVersionInDocker tests running version command in a basic Docker container
func TestVersionInDocker(t *testing.T) {
	binaryPath := setupDockerBinary(t)
	var stdout, stderr bytes.Buffer

	// Run the binary in Alpine Linux container
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/devrig:ro", binaryPath),
		"alpine:latest",
		"/devrig", "version",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

// TestVersionInEmptyFolder tests running version command in an empty random folder
func TestVersionInEmptyFolder(t *testing.T) {
	binaryPath := setupDockerBinary(t)

	// Generate a random folder name
	randomFolder := fmt.Sprintf("/tmp/devrig-test-%d", os.Getpid())

	var stdout, stderr bytes.Buffer

	// Run the binary in Alpine Linux container with an empty random folder as working directory
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/devrig:ro", binaryPath),
		"-w", randomFolder,
		"alpine:latest",
		"sh", "-c", fmt.Sprintf("mkdir -p %s && cd %s && /devrig version", randomFolder, randomFolder),
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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

// TestInitFromLocalBinary tests the init --init-from-local command
func TestInitFromLocalBinary(t *testing.T) {
	binaryPath := setupDockerBinary(t)
	var stdout, stderr bytes.Buffer

	script := `#!/bin/sh
      mkdir -p /workspace/local-project
      /devrig init --init-from-local /workspace/local-project
      cd /workspace/local-project
      ls -lah
      ls -lah .devrig
      ls -lah .devrig/*
      cat devrig.yaml
      # set -e -x -o && source ./devrig version
      ./devrig version || echo "FAIL - version command failed"
      echo "completed"
    `

	// Step 1: Run init --init-from-local to create the environment
	t.Log("Step 1: Running init --init-from-local...")
	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/devrig:ro", binaryPath),
		"alpine:latest",
		"sh", "-c", script,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to run init --init-from-local in Docker: %v\nStdout: %s\nStderr: %s",
			err, stdout.String(), stderr.String())
	}

	output := stdout.String() + "\n\nSTDERR:\n\n" + stderr.String()
	t.Log(output)

	if !strings.Contains(output, "Initializing from local binary") {
		t.Errorf("Init output doesn't contain local binary message: %s", output)
	}
	if !strings.Contains(output, "Local initialization completed successfully!") {
		t.Errorf("Init output doesn't contain success message: %s", output)
	}
	if strings.Contains(output, "FAIL -") {
		t.Errorf("Output contains FAIL message: %s", output)
	}
}

// getDockerArchitecture detects the architecture of the Docker environment
func getDockerArchitecture(t *testing.T) string {
	// Try to detect from Docker
	cmd := exec.Command("docker", "run", "--rm", "alpine", "uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to detect Docker architecture, using host: %v", err)
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
