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
	data, err := c.downloader.DownloadLatestJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to download update info: %w", err)
	}

	// Download signature
	signature, err := c.downloader.DownloadLatestJSONSig()
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

// FetchLatestUpdateInfoUnsafe downloads and parses the latest update information without signature verification
// WARNING: Only use this for testing or when signature verification is not required
func (c *Client) FetchLatestUpdateInfoUnsafe() (*UpdateInfo, error) {
	// Download latest.json
	data, err := c.downloader.DownloadLatestJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to download update info: %w", err)
	}

	// Parse JSON without verification
	var updateInfo UpdateInfo
	if err := json.Unmarshal(data, &updateInfo); err != nil {
		return nil, fmt.Errorf("failed to parse update info: %w", err)
	}

	return &updateInfo, nil
}

// FindBinaryForCurrentSystem finds a binary matching the current OS and architecture
func (updateInfo *UpdateInfo) FindBinaryForCurrentSystem() *Binary {
	sys := CurrentSystem{}
	return updateInfo.FindBinary(sys.OS(), sys.Arch())
}

// FindBinary finds a binary matching the given OS and architecture
func (updateInfo *UpdateInfo) FindBinary(os, arch string) *Binary {
	for i := range updateInfo.Binaries {
		binary := &updateInfo.Binaries[i]
		if binary.OS == os && binary.Arch == arch {
			return binary
		}
	}
	return nil
}
