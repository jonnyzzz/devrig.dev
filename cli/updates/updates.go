package updates

import (
	"encoding/json"
	"fmt"
)

// Client provides high-level API for fetching and parsing update information
type Client struct {
	downloader *Downloader
}

// NewClient creates a new update client
func NewClient() *Client {
	return &Client{
		downloader: NewDownloader(),
	}
}

// FetchLatestUpdateInfo downloads, verifies, and parses the latest update information
// This is the main entry point for getting update information
func (c *Client) FetchLatestUpdateInfo() (*UpdateInfo, error) {
	// Download latest.json
	data, err := c.downloader.download(LatestJSONURL, "latest.json")
	if err != nil {
		return nil, fmt.Errorf("failed to download update info: %w", err)
	}

	// Download signature
	signature, err := c.downloader.download(LatestJSONSigURL, "latest.json.sig")
	if err != nil {
		return nil, fmt.Errorf("failed to download signature: %w", err)
	}

	// Verify signature
	if err := VerifySignature(data, signature); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Parse JSON
	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		return nil, fmt.Errorf("failed to parse update info: %w", err)
	}

	return &updateInfo, nil
}

// FindBinaryForCurrentSystem finds a binary matching the current OS and architecture
func (updateInfo *UpdateInfo) FindBinaryForCurrentSystem() *BinaryInfo {
	sys := CurrentSystem{}
	return updateInfo.FindBinary(sys.OS(), sys.Arch())
}

// FindBinary finds a binary matching the given OS and architecture
func (updateInfo *UpdateInfo) FindBinary(os, arch string) *BinaryInfo {
	for i := range updateInfo.Binaries {
		binary := &updateInfo.Binaries[i]
		if binary.OS == os && binary.Arch == arch {
			return binary
		}
	}
	return nil
}
