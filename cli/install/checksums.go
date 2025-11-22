package install

// KnownChecksums stores verified SHA-512 checksums for JetBrains Mono font releases.
// These checksums are calculated from official GitHub releases and serve as the source of truth.
//
// When a new version is released:
// 1. Download the font from: https://github.com/JetBrains/JetBrainsMono/releases
// 2. Calculate SHA-512: sha512sum JetBrainsMono-*.zip
// 3. Verify the checksum matches the official release
// 4. Update this map with the new version and checksum
var KnownChecksums = map[string]string{
	// Version: SHA-512 checksum
	"v2.304": "1889354a5ab1b20a523eccd67686dd6c5aea550a7e9b84d0301b1dac9193c4dde4b6bdac3892bf10603dc0c5f13f2e68363c70c294cc123b91196901f793bdab",
}

// GetKnownChecksum returns the known-good SHA-512 checksum for a given version.
// Returns empty string if the version is not in the known checksums.
func GetKnownChecksum(version string) string {
	return KnownChecksums[version]
}
