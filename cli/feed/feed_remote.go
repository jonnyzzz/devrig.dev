package feed

import (
	"context"
	"encoding/json"
	"fmt"
)

type feedList struct {
	Feeds   []nestedFeed `json:"feeds"`
	Entries []feedEntry  `json:"entries"`
}

type nestedFeed struct {
	URL string `json:"url"`
}

type feedEntry struct {
	Name         string                `json:"name"`
	Build        string                `json:"build"`
	MajorVersion *feedItemMajorVersion `json:"major_version"`
	Version      string                `json:"version"`
	Released     string                `json:"released"`
	Package      *feedItemPackage      `json:"package"`
	Quality      *feedItemQuality      `json:"feedItemQuality"`
	OrderEntry   int64                 `json:"order_value"`
	RawJSON      json.RawMessage       `json:"-"` // Store original JSON
}

type feedItemMajorVersion struct {
	MajorVersion string `json:"name"`
}

type feedItemQuality struct {
	QualityName string `json:"name"`
}

type feedItemPackage struct {
	OS           string               `json:"os"`
	Type         string               `json:"type"`
	Requirements feedItemRequirements `json:"requirements"`
	URL          string               `json:"url"`
	Size         int64                `json:"size"`
	Checksums    []feedItemChecksum   `json:"checksums"`
}

type feedItemRequirements struct {
	CPUArch feedItemCPUArchRequirement `json:"cpu_arch"`
}

type feedItemCPUArchRequirement struct {
	Equals       string `json:"$eq"`
	ErrorMessage string `json:"error_message"`
}

type feedItemChecksum struct {
	Algorithm string `json:"alg"`
	Value     string `json:"value"`
}

func downloadAndProcessFeedImpl(ctx context.Context, urlsToProcess []string) ([]feedEntry, error) {
	processed := map[string]bool{}
	queueOfUrls := []string{}
	entries := []feedEntry{}

	queueOfUrls = append(queueOfUrls, urlsToProcess...)

	for len(queueOfUrls) > 0 {
		url := queueOfUrls[0]
		queueOfUrls = queueOfUrls[1:]

		if processed[url] {
			continue
		}

		processed[url] = true

		select {
		case <-ctx.Done():
			return []feedEntry{}, ctx.Err()
		default:
		}

		decompressed, err := downloadAndValidateFeedUrl(ctx, url)
		if err != nil {
			return []feedEntry{}, fmt.Errorf("failed to download feed: %w for %s", err, url)
		}

		var list feedList
		err = json.Unmarshal(decompressed, &list)
		if err != nil {
			return []feedEntry{}, fmt.Errorf("failed to parse nested feeds: %w for %s", err, url)
		}

		for _, nestedFeed := range list.Feeds {
			queueOfUrls = append(queueOfUrls, nestedFeed.URL)
		}

		entries = append(entries, filterEntriesByOsAndArch(list.Entries)...)
	}

	return entries, nil
}
