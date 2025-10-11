package bootstrap

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Env struct {
	scriptName string
	image      string
}

var ENV_SH Env = Env{"devrig", "ubuntu:18.04"}
var ENV_PS1 Env = Env{"devrig.ps1", "mcr.microsoft.com/dotnet/sdk:8.0"}

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
		env:             ENV_SH,
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
		env:              ENV_PS1,
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
		env:              ENV_PS1,
		environmentVars:  []string{"DEVRIG_DEBUG_YAML_DOWNLOAD=1", "DEVRIG_CONFIG=devrig-example.yaml", "DEVRIG_CPU=arm64"},
		commandline:      []string{},
		expectedExitCode: 44,
		expectedOutput: []string{
			"https://devrig.dev/download/v1.0.0/devrig-linux-arm64",
			"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
		},
	})
}
