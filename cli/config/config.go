package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
)

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
	instances = make(map[string]Config)
	mutex     sync.RWMutex
)

func ResolveConfig() (Config, error) {
	return ResolveConfigFromDirectory(".")
}

func ResolveConfigFromDirectory(cwd string) (Config, error) {
	// Convert to absolute path for consistent caching
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if we already have an instance for this directory
	mutex.RLock()
	if instance, exists := instances[absCwd]; exists {
		mutex.RUnlock()
		return instance, nil
	}
	mutex.RUnlock()

	// Create new instance
	mutex.Lock()
	defer mutex.Unlock()

	// Double-check after acquiring write lock
	if instance, exists := instances[absCwd]; exists {
		return instance, nil
	}

	var instance Config
	var configErr error
	var configPath string
	configPath, configErr = FindConfigFile(cwd)
	if configErr != nil {
		return nil, fmt.Errorf("failed to resolve config: %w", configErr)
	}

	// Create cache directory next to config file
	cacheDir := filepath.Join(filepath.Dir(configPath), ".idew", "cache")

	// Ensure cache directory exists
	if configErr = os.MkdirAll(cacheDir, 0755); configErr != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", configErr)
	}

	// Parse config file
	ide, configErr := parseConfigFile(configPath)
	if configErr != nil {
		return nil, fmt.Errorf("failed to parse config: %w", configErr)
	}

	instance = &configImpl{
		configPath: configPath,
		cacheDir:   cacheDir,
		ide:        ide,
	}

	// Cache the instance
	instances[absCwd] = instance

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
