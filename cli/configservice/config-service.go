package configservice

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// ConfigService provides validation of devrig.yaml configuration
type ConfigService interface {
	// EnsureValidConfig checks that devrig.yaml exists and is valid
	// Returns detailed diagnostic errors if validation fails
	EnsureValidConfig() error

	// Binaries returns the DevrigBinariesService interface for managing binary configurations
	Binaries() DevrigBinariesService
}

// configServiceImpl is the default implementation of ConfigService
type configServiceImpl struct {
	configPath string
}

// NewConfigService creates a new ConfigService instance with the given devrig.yaml path
func NewConfigService(configPath string) ConfigService {
	return &configServiceImpl{
		configPath: configPath,
	}
}

// Binaries returns the DevrigBinariesService interface for managing binary configurations
func (s *configServiceImpl) Binaries() DevrigBinariesService {
	return s
}

// ReadDevrigSection reads and parses the devrig section from devrig.yaml
func (s *configServiceImpl) ReadDevrigSection() (*DevrigSection, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s", s.configPath)
		}
		return nil, fmt.Errorf("failed to read configuration file %s: %w", s.configPath, err)
	}

	// Parse into a map to extract just the devrig section
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", s.configPath, err)
	}

	// Extract the devrig section
	devrigData, ok := yamlData["devrig"]
	if !ok {
		return nil, fmt.Errorf("devrig section not found in %s", s.configPath)
	}

	// Marshal the devrig section back to YAML and unmarshal into struct
	devrigBytes, err := yaml.Marshal(devrigData)
	if err != nil {
		return nil, fmt.Errorf("failed to process devrig section from %s: %w", s.configPath, err)
	}

	var section DevrigSection
	if err := yaml.Unmarshal(devrigBytes, &section); err != nil {
		return nil, fmt.Errorf("failed to parse devrig section from %s: %w", s.configPath, err)
	}

	// Validate the section
	if err := validateDevrigSection(&section); err != nil {
		return nil, fmt.Errorf("validation failed for %s: %w", s.configPath, err)
	}

	return &section, nil
}

// EnsureValidConfig checks that devrig.yaml exists and is valid
func (s *configServiceImpl) EnsureValidConfig() error {
	// Check if file exists
	info, err := os.Stat(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("devrig.yaml not found at: %s\n\nPlease run 'devrig init' to create it", s.configPath)
		}
		return fmt.Errorf("cannot access devrig.yaml at %s: %w", s.configPath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected devrig.yaml to be a file, but %s is a directory", s.configPath)
	}

	// Try to read and validate
	_, err = s.Binaries().ReadDevrigSection()
	if err != nil {
		return fmt.Errorf("devrig.yaml is invalid:\n  %w\n\nPlease fix the configuration or run 'devrig init' to recreate it", err)
	}

	return nil
}

// validateDevrigSection validates the devrig section structure and required fields
func validateDevrigSection(section *DevrigSection) error {
	if section == nil {
		return fmt.Errorf("devrig section is empty")
	}

	if section.Binaries == nil || len(section.Binaries) == 0 {
		return fmt.Errorf("no binaries configured in devrig section")
	}

	// Validate each binary entry
	for platform, binary := range section.Binaries {
		if binary.URL == "" {
			return fmt.Errorf("missing URL for platform: %s", platform)
		}
		if binary.SHA512 == "" {
			return fmt.Errorf("missing SHA512 hash for platform: %s", platform)
		}
		// Validate SHA512 format (should be 128 hex characters)
		if len(binary.SHA512) != 128 {
			return fmt.Errorf("invalid SHA512 hash length for platform %s: expected 128 characters, got %d", platform, len(binary.SHA512))
		}
		// Validate hash contains only hex characters
		for _, c := range binary.SHA512 {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return fmt.Errorf("invalid SHA512 hash for platform %s: contains non-hexadecimal character '%c'", platform, c)
			}
		}
	}

	return nil
}
