package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyBootstrapScripts(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Copy bootstrap scripts
	err := CopyBootstrapScripts(tempDir)
	if err != nil {
		t.Fatalf("CopyBootstrapScripts failed: %v", err)
	}

	// Verify all three files exist
	files := []string{"devrig", "devrig.bat", "devrig.ps1"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}

	// Verify devrig has executable permissions
	devrigPath := filepath.Join(tempDir, "devrig")
	info, err := os.Stat(devrigPath)
	if err != nil {
		t.Fatalf("Failed to stat devrig: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("devrig is not executable, mode: %v", info.Mode())
	}

	// Verify content is not empty
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
		}
		if len(content) == 0 {
			t.Errorf("File %s is empty", file)
		}
	}

	// Verify devrig script starts with shebang
	devrigContent, _ := os.ReadFile(devrigPath)
	if !bytes.HasPrefix(devrigContent, []byte("#!/bin/sh")) {
		t.Errorf("devrig does not start with correct shebang")
	}

	// Verify devrig.ps1 contains PowerShell content
	ps1Path := filepath.Join(tempDir, "devrig.ps1")
	ps1Content, _ := os.ReadFile(ps1Path)
	if !bytes.Contains(ps1Content, []byte("param(")) {
		t.Errorf("devrig.ps1 does not contain expected PowerShell content")
	}

	// Verify devrig.bat contains batch content
	batPath := filepath.Join(tempDir, "devrig.bat")
	batContent, _ := os.ReadFile(batPath)
	if !bytes.Contains(batContent, []byte("@echo off")) {
		t.Errorf("devrig.bat does not contain expected batch content")
	}
}

func TestCopyBootstrapScripts_NonExistentParent(t *testing.T) {
	// Test that it creates parent directories
	tempDir := t.TempDir()
	targetDir := filepath.Join(tempDir, "nested", "directory", "structure")

	err := CopyBootstrapScripts(targetDir)
	if err != nil {
		t.Fatalf("CopyBootstrapScripts failed to create nested directories: %v", err)
	}

	// Verify files exist in nested directory
	devrigPath := filepath.Join(targetDir, "devrig")
	if _, err := os.Stat(devrigPath); os.IsNotExist(err) {
		t.Errorf("devrig does not exist in nested directory")
	}
}
