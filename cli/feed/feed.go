package feed

import (
	"cli/config"
	"cli/feed_api"
	"context"
	"fmt"
)

func (entry *feedEntry) Name() string {
	return entry.NameV
}

func (entry *feedEntry) Build() string {
	return entry.BuildV
}

func (entry *feedEntry) PackageType() string {
	return entry.Package.Type
}

func (entry *feedEntry) IdeType() string {
	if entry.IntelliJ != nil {
		return "intellij"
	}
	return "unknown"
}

func ResolveRemoteIdeByConfig(ideRequest config.IDEConfig) (feed_api.RemoteIDE, error) {
	entries, err := downloadAndProcessFeedImpl(context.Background(), getFeedUrls())
	if err != nil {
		return nil, err
	}

	var result *feedEntry
	result = nil

	for _, p := range entries {
		entry := p
		if entry.NameV != ideRequest.Name() {
			continue
		}

		if len(ideRequest.Version()) > 0 && ideRequest.Version() != entry.Version {
			continue
		}

		if len(ideRequest.Build()) > 0 && ideRequest.Build() != entry.BuildV {
			continue
		}

		if result == nil || result.OrderEntry < entry.OrderEntry {
			result = &entry
		}
	}

	if result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("IDE is not found in feed. NameV: %s", ideRequest.Name())
}
