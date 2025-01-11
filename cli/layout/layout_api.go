package layout

import (
	"cli/config"
	"cli/ide"
	"os"
	"path"
	"regexp"
	"strings"
)

// SanitizePath sanitizes a single path or filename component
// to avoid invalid or unsafe characters.
func sanitizePath(input string) string {
	// Define allowed characters: alphanumeric, underscore (_), dash (-), and dot (.)
	// Replace any sequence of disallowed characters with an underscore (_)
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	sanitized := re.ReplaceAllString(input, "_")

	// Prevent filenames with dots like ".." or empty paths
	return strings.Trim(sanitized, ".")
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

func resolveTargetIdeHome(config config.Config) string {
	ide := config.GetIDE()
	return path.Join(config.CacheDir(), "ide", sanitizePath(ide.Name()+"-"+ide.Build()))
}

type ResolveLocallyAvailableIdeNotFound struct{}

func (e *ResolveLocallyAvailableIdeNotFound) Error() string {
	return "IDE is not available locally"
}

func ResolveLocallyAvailableIde(config config.Config) (ide.LocalIDE, error) {
	targetIdeHome := resolveTargetIdeHome(config)

	existsAndNotEmpty, err := isDirectoryExistsAndNotEmpty(targetIdeHome)
	if err != nil {
		return nil, err
	}

	if !existsAndNotEmpty {
		return nil, &ResolveLocallyAvailableIdeNotFound{}
	}

	//TODO: to finger print check to double-check the IDE is not modified

	return nil, nil
}
