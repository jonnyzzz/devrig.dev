package bootstrap

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//go:embed devrig
var devrigScript []byte

//go:embed devrig.bat
var devrigBat []byte

//go:embed devrig.ps1
var devrigPs1 []byte

// CopyBootstrapScripts copies all bootstrap scripts (devrig, devrig.bat, devrig.ps1)
// to the specified directory with appropriate permissions.
func CopyBootstrapScripts(targetDir string) error {
	log.Printf("Creating target directory: %s\n", targetDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	scripts := []struct {
		name    string
		content []byte
		mode    os.FileMode
	}{
		{"devrig", devrigScript, 0755},
		{"devrig.bat", devrigBat, 0755},
		{"devrig.ps1", devrigPs1, 0644},
	}

	for _, script := range scripts {
		path := filepath.Join(targetDir, script.name)
		log.Printf("Writing %s to %s with mode %o\n", script.name, path, script.mode)
		if err := os.WriteFile(path, script.content, script.mode); err != nil {
			return fmt.Errorf("failed to write %s: %w", script.name, err)
		}
	}

	log.Println("Bootstrap scripts created successfully!")
	return nil
}
