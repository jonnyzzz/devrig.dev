package feed

import (
	"log"
	"runtime"
)

func resolveOsAndArch() (os string, arch string) {
	// Detect OS
	os = runtime.GOOS
	// Detect CPU architecture
	arch = runtime.GOARCH

	if os == "darwin" {
		os = "mac"
	}

	if arch == "amd64" {
		arch = "x64"
	}

	switch os {
	case "windows":
	case "linux":
	case "mac":
	default:
		log.Fatalln("Unknown operating system: ", os)
	}

	switch arch {
	case "arm64":
	case "x64":
	default:
		log.Fatalln("Unknown arch: ", arch)
	}

	return
}

func filterEntriesByOsAndArch(slice []feedEntry) []feedEntry {
	targetOS, targetArch := resolveOsAndArch()

	var result []feedEntry

	if slice == nil {
		return result
	}

	for _, entry := range slice {
		if entry.Package.OS != targetOS {
			continue
		}

		if entry.Package.Requirements.CPUArch.Equals != targetArch {
			continue
		}

		result = append(result, entry) // Append items that satisfy the predicate
	}

	return result
}
