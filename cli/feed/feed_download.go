package feed

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ulikunitz/xz"
	"go.mozilla.org/pkcs7"
	"io"
	"net/http"
)

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
