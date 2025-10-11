package bootstrap

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Env struct {
	scriptName string
	image      string
}

func downloadFile(url string) ([]byte, string, error) {
	// Create a client that follows redirects (default behavior)
	client := &http.Client{}

	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	hash := sha512.Sum512(data)
	hashStr := hex.EncodeToString(hash[:])

	return data, hashStr, nil
}

func setupTestConfig(t *testing.T, name, url, hash string) string {
	configPath := fmt.Sprintf("test-config-%s.yaml", name)
	content := fmt.Sprintf(`
devrig:
  binaries:
    linux-x86_64:
      url: "%s"
      sha512: "%s"
`, url, hash)

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	t.Cleanup(func() {
		//goland:noinspection GoUnhandledErrorResult
		os.Remove(configPath)
	})
	return configPath
}

type Run struct {
	env             Env
	environmentVars []string
	commandline     []string

	expectedExitCode int
	expectedOutput   []string
}

func runAndAssert(t *testing.T, run Run) {
	var stdout, stderr bytes.Buffer

	args := make([]string, 0)

	args = append(args,
		"run",
		"--rm",
		"-v"+os.Getenv("PWD")+":/image:ro",

		"--workdir", "/image",
		"-e", "BOOTSTRAP_SCRIPT="+run.env.scriptName,
	)

	for _, env := range run.environmentVars {
		args = append(args, "-e", env)
	}

	args = append(args,
		run.env.image,
		"./test-with-docker-sandbox.sh",
	)

	args = append(args, run.commandline...)

	log.Printf("running docker with args: %v\n", args)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	if exitCode != run.expectedExitCode {
		t.Errorf("expected exit code %d, got %d\noutput:\n%s", run.expectedExitCode, exitCode, output)
	}

	for _, expectedOutput := range run.expectedOutput {
		if !strings.Contains(output, expectedOutput) {
			t.Errorf("expected error message %s not found in output:\n%s", expectedOutput, output)
		}
	}
}

func TestParseSH(t *testing.T) {
	runAndAssert(t, Run{
		env:             Env{"devrig", "ubuntu:18.04"},
		environmentVars: []string{"DEVRIG_DEBUG_YAML_DOWNLOAD=1", "DEVRIG_CONFIG=devrig-example.yaml", "DEVRIG_CPU=arm64"},
		commandline:     []string{},

		expectedExitCode: 44,
		expectedOutput: []string{
			"https://devrig.dev/download/v1.0.0/devrig-linux-arm64",
			"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
		},
	})
}

func TestParsePS1(t *testing.T) {
	runAndAssert(t, Run{
		env:              Env{"devrig.ps1", "mcr.microsoft.com/dotnet/sdk:8.0"},
		environmentVars:  []string{"DEVRIG_DEBUG_YAML_DOWNLOAD=1", "DEVRIG_CONFIG=devrig-example.yaml", "DEVRIG_OS=windows", "DEVRIG_CPU=arm64"},
		commandline:      []string{},
		expectedExitCode: 44,
		expectedOutput: []string{
			"https://devrig.dev/download/v1.0.0/devrig-windows-arm64.exe",
			"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
	})
}

func TestParsePS1_AutoDetectLinux(t *testing.T) {
	runAndAssert(t, Run{
		env:              Env{"devrig.ps1", "mcr.microsoft.com/dotnet/sdk:8.0"},
		environmentVars:  []string{"DEVRIG_DEBUG_YAML_DOWNLOAD=1", "DEVRIG_CONFIG=devrig-example.yaml", "DEVRIG_CPU=arm64"},
		commandline:      []string{},
		expectedExitCode: 44,
		expectedOutput: []string{
			"https://devrig.dev/download/v1.0.0/devrig-linux-arm64",
			"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
		},
	})
}

func TestHashMismatch_LocalFile(t *testing.T) {
	// Generate config with wrong hash
	configPath := setupTestConfig(t, "mismatch", "https://devrig.dev/", "badhash1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:18.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
		},
		commandline:      []string{},
		expectedExitCode: 7,
		expectedOutput: []string{
			"[ERROR] Downloaded binary checksum mismatch",
			"[ERROR] Expected:",
			"[ERROR] Actual:",
		},
	})
}

func TestDownload_Curl_Success(t *testing.T) {
	// Use a stable URL with unchanging content
	testURL := "https://raw.githubusercontent.com/github/gitignore/main/Python.gitignore"

	// Download file and calculate hash
	_, hash, err := downloadFile(testURL)
	if err != nil {
		t.Skipf("Skipping test, cannot download file: %v", err)
	}

	configPath := setupTestConfig(t, "download-curl", testURL, hash)

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:22.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
			"DEVRIG_TEST_CURL_ONLY=1",
		},
		commandline:      []string{},
		expectedExitCode: 45,
		expectedOutput: []string{
			"[INFO] Devrig binary not found, downloading...",
			"[INFO] Verifying downloaded binary checksum...",
			"[INFO] Installing devrig binary...",
			"[INFO] Devrig binary installed successfully",
		},
	})
}

func TestDownload_Wget_Success(t *testing.T) {
	// Use a stable URL with unchanging content
	testURL := "https://raw.githubusercontent.com/github/gitignore/main/Python.gitignore"

	// Download file and calculate hash
	_, hash, err := downloadFile(testURL)
	if err != nil {
		t.Skipf("Skipping test, cannot download file: %v", err)
	}

	configPath := setupTestConfig(t, "download-wget", testURL, hash)

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:22.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
			"DEVRIG_TEST_WGET_ONLY=1",
		},
		commandline:      []string{},
		expectedExitCode: 45,
		expectedOutput: []string{
			"[INFO] Devrig binary not found, downloading...",
			"[INFO] Verifying downloaded binary checksum...",
			"[INFO] Installing devrig binary...",
			"[INFO] Devrig binary installed successfully",
		},
	})
}

func TestDownload_IncorrectHash(t *testing.T) {
	// Use a stable URL but with incorrect hash
	testURL := "https://raw.githubusercontent.com/github/gitignore/main/Python.gitignore"
	configPath := setupTestConfig(t, "bad-hash", testURL, "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:22.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
			"DEVRIG_TEST_CURL_ONLY=1",
		},
		commandline:      []string{},
		expectedExitCode: 7,
		expectedOutput: []string{
			"[INFO] Devrig binary not found, downloading...",
			"[ERROR] Downloaded binary checksum mismatch",
		},
	})
}

func TestLocalBinary_ValidHash(t *testing.T) {
	// Create a test binary and get its hash
	testBinary := []byte("#!/bin/sh\necho 'test binary'\n")
	hash := sha512.Sum512(testBinary)
	hashStr := hex.EncodeToString(hash[:])

	// Create config with the correct hash
	configPath := setupTestConfig(t, "local-valid", "https://example.com/binary", hashStr)

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:18.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
			"DEVRIG_TEST_CREATE_LOCAL_BINARY=valid",
		},
		commandline:      []string{},
		expectedExitCode: 45,
		expectedOutput: []string{
			hashStr,
		},
	})
}

func TestLocalBinary_InvalidHash(t *testing.T) {
	// Use incorrect hash for the binary
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	// Create config with wrong hash
	configPath := setupTestConfig(t, "local-invalid", "https://example.com/binary", wrongHash)

	runAndAssert(t, Run{
		env: Env{"devrig", "ubuntu:18.04"},
		environmentVars: []string{
			"DEVRIG_DEBUG_NO_EXEC=1",
			"DEVRIG_CONFIG=" + configPath,
			"DEVRIG_CPU=x86_64",
			"DEVRIG_TEST_CREATE_LOCAL_BINARY=invalid",
		},
		commandline:      []string{},
		expectedExitCode: 7,
		expectedOutput: []string{
			"[ERROR] Downloaded binary checksum mismatch",
			"[ERROR] Expected: " + wrongHash,
		},
	})
}
