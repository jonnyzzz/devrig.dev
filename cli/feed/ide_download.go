package feed

import (
	"cli/config"
	"cli/feed_api"
	"cli/layout"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadFeedEntry(ctx context.Context, entry feed_api.RemoteIDE, config config.Config) error {
	feedEntry, ok := entry.(*feedEntry)
	if !ok {
		log.Panicln("Failed to cast entry to feedEntry")
	}

	url := feedEntry.Package.URL
	fmt.Println("Downloading ", url, " for ", feedEntry, "...")

	packageSha256 := ""
	for _, checksum := range feedEntry.Package.Checksums {
		if checksum.Algorithm == "sha-256" {
			packageSha256 = checksum.Value
			break
		}
	}

	if len(packageSha256) == 0 {
		log.Panicln("Failed to resolve packageSha256 checksum for ", url)
	}

	size := feedEntry.Package.Size

	if size <= 1000 {
		log.Panicln("Failed to resolve size for ", url)
	}

	targetFile := layout.ResolveLocalDownloadFileName(config, feedEntry)

	pros := downloadRequest{
		url,
		size,
		packageSha256,

		targetFile,
	}

	err := downloadIdeBinaryIfNeeded(ctx, pros)

	if err != nil {
		return err
	}

	return nil
}

type downloadRequest struct {
	Url    string
	Size   int64
	Sha256 string

	TargetFile string
}

func downloadIdeBinaryIfNeeded(ctx context.Context, request downloadRequest) error {
	err := validateDownloadedFile(request)
	if err == nil {
		fmt.Printf("File %s already exists for %s\n", request.TargetFile, request.Url)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", request.Url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w for %s", err, request.Url)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w for %s", err, request.Url)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d for %s", resp.StatusCode, request.Url)
	}

	err = saveResponseToFile(request.Url, request.TargetFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save response to file %s: %w", request.TargetFile, err)
	}

	return nil
}

func saveResponseToFile(url string, targetFile string, body io.ReadCloser) error {
	// Ensure the parent directory of targetFile exists
	if err := os.MkdirAll(filepath.Dir(targetFile), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create parent directories for %s: %w", targetFile, err)
	}

	out, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w for %s", targetFile, err, url)
	}

	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("failed to close file %s: %v for %s", targetFile, err, url)
		}
	}()

	//TODO: implement progress
	// Write the response to the file
	if _, err := io.Copy(out, body); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", targetFile, err)
	}

	fmt.Printf("Downloaded %s to %s\n", url, targetFile)
	return nil
}

func validateDownloadedFile(request downloadRequest) error {
	targetFileInfo, err := os.Stat(request.TargetFile)
	if err != nil {
		return fmt.Errorf("failed to read download file: %w for %s for %s", err, request.TargetFile, request.Url)
	}

	if targetFileInfo.Size() != request.Size {
		return fmt.Errorf("actual file size %d does not match expected size %d for %s", targetFileInfo.Size(), request.Size, request.Url)
	}

	computedHash, err := computeSha256(request)
	if err != nil {
		return fmt.Errorf("failed to compute hash for %s: %w", request.TargetFile, err)
	}

	if computedHash != request.Sha256 {
		return fmt.Errorf("computed hash %s does not match expected hash %s for %s", computedHash, request.Sha256, request.Url)
	}

	return nil
}

func computeSha256(request downloadRequest) (string, error) {
	// Compute SHA-256 hash of the downloaded file
	file, err := os.Open(request.TargetFile)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w for %s", err, request.TargetFile)
	}

	defer func() {
		err := file.Close()
		if err != nil {
			log.Printf("failed to close file %s: %v for %s", request.TargetFile, err, request.Url)
		}
	}()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute SHA-256 hash for %s: %w", request.TargetFile, err)
	}

	hash := hasher.Sum(nil)
	computedHash := fmt.Sprintf("%x", hash)
	return computedHash, nil
}
