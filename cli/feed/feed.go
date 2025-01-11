package feed

import (
	"bytes"
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

func downloadAndProcessFeed(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w for %s", err, url)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download feed: %w for %s", err, url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d for %s", resp.StatusCode, url)
	}

	// Read PKCS7 data
	signedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read signed data: %w for %s", err, url)
	}

	// Parse PKCS7
	p7, err := pkcs7.Parse(signedData)
	if err != nil {
		return fmt.Errorf("failed to parse signed data: %w fopr %s", err, url)
	}

	//TODO: implement signature verification

	// Get content from PKCS7
	content := p7.Content

	// Setup XZ decoder
	xzReader, err := xz.NewReader(bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w for %s", err, url)
	}

	// Read all decompressed content
	decompressed, err := io.ReadAll(xzReader)
	if err != nil {
		return fmt.Errorf("failed to decompress content: %w for %s", err, url)
	}

	// Try to decode as a feed list first
	var feedList struct {
		Feeds []nestedFeed `json:"feeds"`
	}

	if err := json.Unmarshal(decompressed, &feedList); err != nil {
		return fmt.Errorf("failed to parse nested feeds: %w for %s", err, url)
	}

	// Process nested feeds
	for _, nestedFeed := range feedList.Feeds {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Printf("Processing nested feed: %s for %s\n", nestedFeed.URL, url)
			if err := downloadAndProcessFeed(ctx, nestedFeed.URL); err != nil {
				return fmt.Errorf("failed to process nested feed %s: %w for %s", nestedFeed.URL, err, url)
			}
		}
	}

	// Try to decode as entries
	var entriesList struct {
		Entries []feedEntry `json:"entries"`
	}

	if err := json.Unmarshal(decompressed, &entriesList); err != nil {
		return fmt.Errorf("failed to decode entries: %w for %s", err, url)
	}

	targetOS, targetArch := resolveOsAndArch()

	// Process entries if any exist
	for _, entry := range entriesList.Entries {
		if entry.Package.OS != targetOS {
			continue
		}

		if entry.Package.Requirements.CPUArch.Equals != targetArch {
			continue
		}

		logFeedItem(entry)
	}

	return nil
}

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
