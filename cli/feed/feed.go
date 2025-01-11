package feed

import (
	"bytes"
	"cli/config"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ulikunitz/xz"
	"go.mozilla.org/pkcs7"
	"io"
	"log"
	"net/http"
	"runtime"
)

var public_feed_urls = []string{
	"https://download.jetbrains.com/toolbox/feeds/v1/release.feed.xz.signed",
	"https://download.jetbrains.com/toolbox/feeds/v1/public-feed-arm.feed.xz.signed",
	"https://download.jetbrains.com/toolbox/feeds/v1/android-studio.feed.xz.signed",

	//https://download.jetbrains.com/toolbox/feeds/v1/enterprise.feed.xz.signed,
}

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

func resolveOsAndArch() (os string, arch string) {
	// Detect OS
	os = runtime.GOOS
	// Detect CPU architecture
	arch = runtime.GOARCH

	if os == "darwin" {
		os = "mac"
	}

	if arch == "amd64" {
		arch = "x64"
	}

	switch os {
	case "windows":
	case "linux":
	case "mac":
	default:
		log.Fatalln("Unknown operating system: ", os)
	}

	switch arch {
	case "arm64":
	case "x64":
	default:
		log.Fatalln("Unknown arch: ", arch)
	}

	return
}

func FindEntryByConfig(ide config.IDEConfig) error {
	entries, err := downloadAndProcessFeedImpl(context.Background(), public_feed_urls[0])
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name != ide.Name() {
			continue
		}

		return nil
	}

	return nil
}

func filterEntriesByOsAndArch(slice []feedEntry) []feedEntry {
	targetOS, targetArch := resolveOsAndArch()

	var result []feedEntry

	if slice == nil {
		return result
	}

	for _, entry := range slice {
		if entry.Package.OS != targetOS {
			continue
		}

		if entry.Package.Requirements.CPUArch.Equals != targetArch {
			continue
		}

		result = append(result, entry) // Append items that satisfy the predicate
	}

	return result
}

func downloadAndValidateFeedUrl(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w for %s", err, url)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download feed: %w for %s", err, url)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for %s", resp.StatusCode, url)
	}

	// Read PKCS7 data
	signedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read signed data: %w for %s", err, url)
	}

	// Parse PKCS7
	p7, err := pkcs7.Parse(signedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed data: %w fopr %s", err, url)
	}

	//TODO: implement signature verification

	// Get content from PKCS7
	content := p7.Content

	// Setup XZ decoder
	xzReader, err := xz.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to create xz reader: %w for %s", err, url)
	}

	// Read all decompressed content
	decompressed, err := io.ReadAll(xzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress content: %w for %s", err, url)
	}

	return decompressed, nil
}

func downloadAndProcessFeedImpl(ctx context.Context, url string) ([]feedEntry, error) {
	decompressed, err := downloadAndValidateFeedUrl(ctx, url)
	if err != nil {
		return []feedEntry{}, fmt.Errorf("failed to download feed: %w for %s", err, url)
	}

	var feedList feedList
	err = json.Unmarshal(decompressed, &feedList)
	if err != nil {
		return []feedEntry{}, fmt.Errorf("failed to parse nested feeds: %w for %s", err, url)
	}

	feedList.Entries = filterEntriesByOsAndArch(feedList.Entries)

	// Process nested feeds
	for _, nestedFeed := range feedList.Feeds {
		select {
		case <-ctx.Done():
			return []feedEntry{}, ctx.Err()
		default:
		}

		log.Printf("Processing nested feed: %s for %s\n", nestedFeed.URL, url)
		nestedEntries, err := downloadAndProcessFeedImpl(ctx, nestedFeed.URL)

		if err != nil {
			return []feedEntry{}, fmt.Errorf("failed to process nested feed %s: %w for %s", nestedFeed.URL, err, url)
		}

		feedList.Entries = append(feedList.Entries, nestedEntries...)
	}

	return feedList.Entries, nil
}

func downloadAndProcessFeed(ctx context.Context, url string) error {
	entries, err := downloadAndProcessFeedImpl(ctx, url)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		logFeedItem(entry)
	}

	return nil
}

// Filter function that takes a slice and a predicate function
func logFeedItem(item feedEntry) {
	fmt.Printf("Product: %s\n", item.Name)
	fmt.Printf("  Version: %s (Build: %s)\n", item.Version, item.Build)
	fmt.Printf("  Released: %s\n", item.Released)

	if item.Package != nil {
		pkg := item.Package
		fmt.Printf("  feedItemPackage:\n")
		fmt.Printf("    OS: %s\n", pkg.OS)
		fmt.Printf("    Type: %s\n", pkg.Type)
		fmt.Printf("    Size: %d mb\n", pkg.Size/1024/1024)

		if len(pkg.Checksums) > 0 {
			fmt.Printf("    Checksums:\n")
			for _, checksum := range pkg.Checksums {
				fmt.Printf("      %s: %s\n", checksum.Algorithm, checksum.Value)
			}
		}

		if pkg.Requirements.CPUArch.Equals != "" {
			fmt.Printf("    CPU Architecture: %s\n", pkg.Requirements.CPUArch.Equals)
		}

		fmt.Printf("    URL: %s\n", pkg.URL)
	}
	fmt.Println()
}
