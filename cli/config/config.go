package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sync"
)

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

// ideConfigImpl is the internal implementation of IDEConfig
type ideConfigImpl struct {
	NameV    string `yaml:"name"`
	VersionV string `yaml:"version"`
	BuildV   string `yaml:"build,omitempty"`
}

func (i *ideConfigImpl) Name() string    { return i.NameV }
func (i *ideConfigImpl) Version() string { return i.VersionV }
func (i *ideConfigImpl) Build() string   { return i.BuildV }

// configImpl is the internal implementation of Config
type configImpl struct {
	configPath string
	cacheDir   string
	ide        IDEConfig
}

func (c *configImpl) CacheDir() string {
	return c.cacheDir
}

func (c *configImpl) ConfigPath() string {
	return c.configPath
}

func (c *configImpl) GetIDE() IDEConfig {
	return c.ide
}

// Add String method to configImpl
func (c *configImpl) String() string {
	return fmt.Sprintf("ConfigPath: %s, CacheDir: %s", c.configPath, c.cacheDir)
}

var (
	instance Config
	once     sync.Once
)

func ResolveConfig() (Config, error) {
	return ResolveConfigFromDirectory(".")
}

func ResolveConfigFromDirectory(cwd string) (Config, error) {
	var err error
	once.Do(func() {
		var configPath string
		configPath, err = FindConfigFile(cwd)
		if err != nil {
			return
		}

		// Create cache directory next to config file
		cacheDir := filepath.Join(filepath.Dir(configPath), ".idew", "cache")
		_, err := os.Stat(cacheDir)

		// Parse .ides.yaml file
		ide, err := parseConfigFile(configPath)
		if err != nil {
			return
		}

		instance = &configImpl{
			configPath: configPath,
			cacheDir:   cacheDir,
			ide:        ide,
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to resolve config: %w", err)
	}

	return instance, nil
}

func parseConfigFile(configPath string) (*ideConfigImpl, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var configData struct {
		IDE *ideConfigImpl `yaml:"ide"`
	}

	err = yaml.Unmarshal(data, &configData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if configData.IDE == nil {
		return nil, fmt.Errorf("missing ide configuration")
	}

	if configData.IDE.NameV == "" {
		return nil, fmt.Errorf("IDE name is required in config file")
	}
	if configData.IDE.VersionV == "" {
		return nil, fmt.Errorf("IDE version is required in config file")
	}

	return configData.IDE, nil
}
