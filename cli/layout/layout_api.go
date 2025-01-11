package layout

import (
	"cli/config"
	"cli/feed_api"
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
	ideConfig := config.GetIDE()
	//TODO: resolve here the right directory for the IDE, current Version() or Build may not be known
	return path.Join(config.CacheDir(), "ide", sanitizePath(ideConfig.Name()+"-"+ideConfig.Version()))
}

func ResolveLocalDownloadFileName(localConfig config.Config, remoteIde feed_api.RemoteIDE) string {
	ideDir := sanitizePath(remoteIde.Name()+"-"+remoteIde.Build()) + "." + remoteIde.PackageType()
	return path.Join(localConfig.CacheDir(), "download", ideDir)
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
