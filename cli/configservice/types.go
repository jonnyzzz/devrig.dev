package configservice

// DevrigSection contains the devrig configuration section
type DevrigSection struct {
	Version     string                `yaml:"version,omitempty"`
	ReleaseDate string                `yaml:"release_date,omitempty"`
	Binaries    map[string]BinaryInfo `yaml:"binaries"`
}

// BinaryInfo contains information about a platform-specific binary
type BinaryInfo struct {
	URL    string `yaml:"url"`
	SHA512 string `yaml:"sha512"`
}
