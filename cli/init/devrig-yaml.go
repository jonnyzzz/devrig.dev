package init

import (
	"fmt"
	"strings"

	"jonnyzzz.com/devrig.dev/configservice"
)

func generateDevrigSection(currentPlatform string, currentHash string) *configservice.DevrigSection {
	url := fmt.Sprintf("https://devrig.dev/local-build-fake-url/%s", currentPlatform)
	if strings.Contains(currentPlatform, "windows") {
		url += ".exe"
	}

	return &configservice.DevrigSection{
		Binaries: map[string]configservice.BinaryInfo{
			currentPlatform: {
				URL:    url,
				SHA512: currentHash,
			},
		},
	}
}
