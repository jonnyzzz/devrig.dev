package unpack

import (
	"cli/config"
	"cli/feed_api"
	"cli/unpack_api"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type unpackedDownloadedRemoteIdeDmg struct {
	unpack_api.UnpackedDownloadedRemoteIde

	appHome   string
	remoteIde feed_api.RemoteIDE
}

func (u *unpackedDownloadedRemoteIdeDmg) RemoteIde() feed_api.RemoteIDE {
	return u.remoteIde
}

func (u *unpackedDownloadedRemoteIdeDmg) UnpackedHome() string {
	return u.appHome
}

func (u *unpackedDownloadedRemoteIdeDmg) String() string {
	return fmt.Sprintf("UnpackedDownloadedRemoteIdeDmg{appHome: %s, remoteIde: %s}", u.appHome, u.remoteIde)
}

func unpackDmg(localConfig config.Config, request feed_api.DownloadedRemoteIde, targetDir string) (*unpackedDownloadedRemoteIdeDmg, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("unpacking DMG is only supported on macOS")
	}

	exists, err := isDirectoryExistsAndNotEmpty(targetDir)
	if err == nil && exists {
		//TODO: implement checksum validation here
		//TODO: list files and resolve the only .app there
		return &unpackedDownloadedRemoteIdeDmg{remoteIde: request.RemoteIde(), appHome: targetDir}, nil
	}

	// Ensure the parent directory of targetFile exists
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create parent directories for %s: %w", targetDir, err)
	}

	_ = os.RemoveAll(targetDir)
	// Create a temporary mount point
	mountPoint, err := os.MkdirTemp(localConfig.CacheDir(), "jbcli-dmg-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	defer os.RemoveAll(mountPoint)

	// Mount the DMG
	attachCmd := exec.Command("hdiutil", "attach", "-nobrowse", "-mountpoint", mountPoint, request.TargetFile())
	if err := attachCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to mount DMG: %w", err)
	}
	defer exec.Command("hdiutil", "detach", mountPoint, "-force").Run()

	// Find and copy the .app directory
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return nil, fmt.Errorf("failed to read mount directory: %w for %s", err, request.TargetFile())
	}

	dstPath := ""
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".app" {
			fmt.Printf("Skipping %s from %s\n", entry.Name(), request.TargetFile())
			continue
		}

		if dstPath != "" {
			return nil, fmt.Errorf("multiple .app directories found in DMG file %s", request.TargetFile())
		}

		srcPath := filepath.Join(mountPoint, entry.Name())
		dstPath = filepath.Join(targetDir)

		cpCmd := exec.Command("cp", "-Rv", srcPath+"/", dstPath+"/")
		if err := cpCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to copy application: %w to %s for %s", err, targetDir, request.TargetFile())
		}

		// Remove quarantine attributes
		xattrCmd := exec.Command("xattr", "-rd", "com.apple.quarantine", dstPath)
		if err := xattrCmd.Run(); err != nil {
			fmt.Printf("failed to remove quarantine attributes: %s\n", err.Error())
		}
	}

	if dstPath == "" {
		return nil, fmt.Errorf("no .app directories found in DMG file %s", request.TargetFile())
	}

	return &unpackedDownloadedRemoteIdeDmg{remoteIde: request.RemoteIde(), appHome: targetDir}, nil
}
