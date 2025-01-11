package unpack

import (
	"cli/config"
	"cli/feed_api"
	"cli/layout"
	"fmt"
	"log"
	"os"
	"strings"
)

type UnpackedDownloadedRemoteIde interface {
	fmt.Stringer

	UnpackedHome() string
	RemoteIde() feed_api.RemoteIDE
}

func UnpackIde(localConfig config.Config, request feed_api.DownloadedRemoteIde) (UnpackedDownloadedRemoteIde, error) {
	targetDir := layout.ResolveLocalHome(localConfig, request.RemoteIde())
	fmt.Println("Unpacking ", request.TargetFile(), " to ", targetDir, "...")

	if request.RemoteIde().PackageType() == "dmg" {
		if !strings.HasSuffix(targetDir, ".app") {
			log.Fatalln("Target directory must end with .app: ", targetDir)
		}

		targetApp, err := unpackDmg(localConfig, request, targetDir)
		if err != nil {
			return nil, err
		}

		fmt.Println("Unpacked ", request.TargetFile(), " to ", targetApp, "...")
		return targetApp, nil
	}

	return nil, fmt.Errorf("unsupported package type: %s", request.RemoteIde().PackageType())
}

func isDirectoryExistsAndNotEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Directory does not exist
		}
		return false, err // Unexpected error
	}

	// Check if directory is empty
	return len(entries) > 0, nil
}
