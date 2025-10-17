package updates

import "runtime"

// UpdateInfo represents the current update information
type UpdateInfo struct {
	Binaries []Binary `json:"binaries"`
}

// Binary represents a single binary distribution
type Binary struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	SHA512   string `json:"sha512"`
	URL      string `json:"url"`
}

// SystemInfo provides information about the current system
type SystemInfo interface {
	OS() string
	Arch() string
}

// CurrentSystem represents the current operating system and architecture
type CurrentSystem struct{}

// OS returns the operating system name
func (s CurrentSystem) OS() string {
	return runtime.GOOS
}

// Arch returns the architecture name
func (s CurrentSystem) Arch() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		return "x86_64"
	}
	return arch
}
