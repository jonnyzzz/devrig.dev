package config

// Config represents the configuration interface
type Config interface {
	// CacheDir returns the path to the cache directory
	CacheDir() string

	// ConfigPath returns the path to the config file
	ConfigPath() string

	// GetIDE returns the IDE configuration
	GetIDE() IDEConfig
}

// IDEConfig represents the IDE configuration interface
type IDEConfig interface {
	// Name returns the IDE name
	Name() string
	// Version returns the IDE version
	Version() string
	// Build returns the optional build number
	Build() string
}
