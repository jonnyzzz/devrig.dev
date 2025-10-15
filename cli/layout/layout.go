package layout

import (
	"path"
	"regexp"
	"strings"

	"jonnyzzz.com/devrig.dev/config"
	"jonnyzzz.com/devrig.dev/feed_api"
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

func ResolveLocalDownloadFileName(localConfig config.Config, remoteIde feed_api.RemoteIDE) string {
	ideDir := sanitizePath(remoteIde.Name()+"-"+remoteIde.Build()) + "." + remoteIde.PackageType()
	return path.Join(localConfig.CacheDir(), "download", ideDir)
}

func ResolveLocalHome(localConfig config.Config, remoteIde feed_api.RemoteIDE) string {
	ideDir := sanitizePath(remoteIde.Name() + "-" + remoteIde.Build())
	if remoteIde.PackageType() == "dmg" {
		ideDir += ".app"
	}
	return path.Join(localConfig.CacheDir(), "ide", ideDir)
}
