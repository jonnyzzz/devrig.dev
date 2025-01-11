package feed

import (
	"cli/config"
	"context"
	"fmt"
)

type RemoteIDE interface {
}

func ResolveRemoteIdeByConfig(ideRequest config.IDEConfig) (RemoteIDE, error) {
	entries, err := downloadAndProcessFeedImpl(context.Background(), getFeedUrls())
	if err != nil {
		return nil, err
	}

	var result *feedEntry
	result = nil

	for _, p := range entries {
		entry := p
		if entry.Name != ideRequest.Name() {
			continue
		}

		if len(ideRequest.Version()) > 0 && ideRequest.Version() != entry.Version {
			continue
		}

		if len(ideRequest.Build()) > 0 && ideRequest.Build() != entry.Build {
			continue
		}

		if result == nil || result.OrderEntry < entry.OrderEntry {
			result = &entry
		}
	}

	if result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("IDE is not found in feed. Name: %s", ideRequest.Name())
}
