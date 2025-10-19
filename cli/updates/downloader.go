package updates

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	LatestJSONURL    = "https://devrig.dev/download/latest.json"
	LatestJSONSigURL = "https://devrig.dev/download/latest.json.sig"
)

// Downloader handles downloading update information
type Downloader struct {
	HTTPClient *http.Client
}

// NewDownloader creates a new Downloader with default settings
func NewDownloader() *Downloader {
	return &Downloader{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// download is a helper method that performs the actual HTTP download
func (d *Downloader) download(url, name string) ([]byte, error) {
	resp, err := d.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", name, err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: status %d", name, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}

	return data, nil
}
