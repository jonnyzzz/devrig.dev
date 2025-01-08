package feed

import (
	"bytes"
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
	Name         string        `json:"name"`
	Build        string        `json:"build"`
	MajorVersion *majorVersion `json:"major_version"`
	Version      string        `json:"version"`
	Released     string        `json:"released"`
	Package      *Package      `json:"package"`
	Quality      *quality      `json:"quality"`
}

type majorVersion struct {
	MajorVersion string `json:"name"`
}

type quality struct {
	QualityName string `json:"name"`
}

type Package struct {
	OS           string       `json:"os"`
	Type         string       `json:"type"`
	Requirements Requirements `json:"requirements"`
	URL          string       `json:"url"`
	Size         int64        `json:"size"`
	Checksums    []Checksum   `json:"checksums"`
}

type Requirements struct {
	CPUArch CPUArchRequirement `json:"cpu_arch"`
}

type CPUArchRequirement struct {
	Equals       string `json:"$eq"`
	ErrorMessage string `json:"error_message"`
}

type Checksum struct {
	Algorithm string `json:"alg"`
	Value     string `json:"value"`
}

func main2() {
	// Detect OS
	os := runtime.GOOS
	// Detect CPU architecture
	arch := runtime.GOARCH

	// Print the detected OS and CPU architecture
	fmt.Printf("Operating System: %s\n", os)
	fmt.Printf("CPU Architecture: %s\n", arch)

	// Handle specific use cases
	if os == "windows" && arch == "amd64" {
		fmt.Println("Running on a Windows machine with x64 architecture.")
	} else if os == "linux" && arch == "arm64" {
		fmt.Println("Running on a Linux machine with ARM64 architecture.")
	} else if os == "darwin" && arch == "arm64" {
		fmt.Println("Running on a macOS machine with Apple Silicon (ARM64).")
	} else {
		fmt.Println("Detected system may not meet some requirements.")
	}
}

func downloadAndProcessFeed(url string) error {
	resp, err := http.Get(url)
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
		log.Printf("Processing nested feed: %s for %s\n", nestedFeed.URL, url)
		if err := downloadAndProcessFeed(nestedFeed.URL); err != nil {
			return fmt.Errorf("failed to process nested feed %s: %w for %s", nestedFeed.URL, err, url)
		}
	}

	// Try to decode as entries
	var entriesList struct {
		Entries []feedEntry `json:"entries"`
	}
	if err := json.Unmarshal(decompressed, &entriesList); err != nil {
		return fmt.Errorf("failed to decode entries: %w for %s", err, url)
	}

	// Process entries if any exist
	for _, release := range entriesList.Entries {
		processRelease(release)
	}

	return nil
}

func processRelease(release feedEntry) {
	fmt.Printf("Product: %s\n", release.Name)
	fmt.Printf("  Version: %s (Build: %s)\n", release.Version, release.Build)
	fmt.Printf("  Released: %s\n", release.Released)

	if release.Package != nil {
		pkg := release.Package
		fmt.Printf("  Package:\n")
		fmt.Printf("    OS: %s\n", pkg.OS)
		fmt.Printf("    Type: %s\n", pkg.Type)
		fmt.Printf("    Size: %d bytes\n", pkg.Size)

		if len(pkg.Checksums) > 0 {
			fmt.Printf("    Checksums:\n")
			for _, checksum := range pkg.Checksums {
				fmt.Printf("      %s: %s\n", checksum.Algorithm, checksum.Value)
			}
		}

		if pkg.Requirements.CPUArch.Equals != "" {
			fmt.Printf("    CPU Architecture: %s\n", pkg.Requirements.CPUArch.Equals)
			if pkg.Requirements.CPUArch.ErrorMessage != "" {
				fmt.Printf("    Architecture Error: %s\n", pkg.Requirements.CPUArch.ErrorMessage)
			}
		}

		fmt.Printf("    URL: %s\n", pkg.URL)
	}
	fmt.Println()
}
