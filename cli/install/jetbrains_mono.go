package install

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const (
	jetBrainsMonoRepo   = "JetBrains/JetBrainsMono"
	jetBrainsMonoAPIURL = "https://api.github.com/repos/" + jetBrainsMonoRepo + "/releases/latest"
)

// JetBrainsMonoInstaller handles installation of JetBrains Mono font
type JetBrainsMonoInstaller struct {
	devrigVersion string
	fontVersion   string
	downloadURL   string
	tempDir       string
	userAgent     string
}

// GitHubRelease represents a GitHub release response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// NewJetBrainsMonoInstaller creates a new JetBrains Mono installer
func NewJetBrainsMonoInstaller(devrigVersion string) (*JetBrainsMonoInstaller, error) {
	// TODO: Validate SHA-sum of the downloaded font
	// Issue: Add checksum validation for downloaded fonts
	// Reference: https://github.com/jonnyzzz/devrig.dev/issues/TBD

	installer := &JetBrainsMonoInstaller{
		devrigVersion: devrigVersion,
		userAgent:     fmt.Sprintf("devrig/%s", devrigVersion),
	}

	// Fetch latest release info
	if err := installer.fetchLatestRelease(); err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	return installer, nil
}

// fetchLatestRelease fetches the latest JetBrains Mono release from GitHub
func (j *JetBrainsMonoInstaller) fetchLatestRelease() error {
	req, err := http.NewRequest("GET", jetBrainsMonoAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", j.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode release info: %w", err)
	}

	j.fontVersion = release.TagName

	// Find the zip asset
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".zip") && strings.Contains(asset.Name, "JetBrainsMono") {
			j.downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if j.downloadURL == "" {
		return fmt.Errorf("could not find font zip in release %s", j.fontVersion)
	}

	return nil
}

// Install downloads and installs JetBrains Mono font
func (j *JetBrainsMonoInstaller) Install(cmd *cobra.Command) error {
	cmd.Printf("Downloading JetBrains Mono %s...\n", j.fontVersion)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "devrig-jetbrains-mono-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	j.tempDir = tempDir
	defer os.RemoveAll(tempDir)

	// Download font
	zipPath := filepath.Join(tempDir, "JetBrainsMono.zip")
	if err := j.downloadFile(zipPath); err != nil {
		return fmt.Errorf("failed to download font: %w", err)
	}

	cmd.Println("Extracting fonts...")

	// Extract fonts
	fontsDir := filepath.Join(tempDir, "fonts")
	if err := j.extractFonts(zipPath, fontsDir); err != nil {
		return fmt.Errorf("failed to extract fonts: %w", err)
	}

	cmd.Println("Installing fonts...")

	// Install fonts based on OS
	if err := j.installFontsForOS(fontsDir); err != nil {
		return fmt.Errorf("failed to install fonts: %w", err)
	}

	return nil
}

// downloadFile downloads a file from URL to destPath
func (j *JetBrainsMonoInstaller) downloadFile(destPath string) error {
	// TODO: Validate SHA-sum after download
	// Issue: Add checksum validation for downloaded font archive
	// Reference: https://github.com/jonnyzzz/devrig.dev/issues/TBD

	req, err := http.NewRequest("GET", j.downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", j.userAgent)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// extractFonts extracts TTF fonts from the zip archive
func (j *JetBrainsMonoInstaller) extractFonts(zipPath, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create fonts directory: %w", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	// Extract only TTF files from the fonts/ttf directory
	for _, f := range r.File {
		if !strings.Contains(f.Name, "fonts/ttf/") {
			continue
		}

		if !strings.HasSuffix(strings.ToLower(f.Name), ".ttf") {
			continue
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		fileName := filepath.Base(f.Name)
		destPath := filepath.Join(destDir, fileName)

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

// installFontsForOS installs fonts based on the current operating system
func (j *JetBrainsMonoInstaller) installFontsForOS(fontsDir string) error {
	switch runtime.GOOS {
	case "windows":
		return j.installFontsWindows(fontsDir)
	case "darwin":
		return j.installFontsMacOS(fontsDir)
	case "linux":
		return j.installFontsLinux(fontsDir)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// installFontsWindows installs fonts on Windows
func (j *JetBrainsMonoInstaller) installFontsWindows(fontsDir string) error {
	// Windows font installation directory
	fontsPath := filepath.Join(os.Getenv("WINDIR"), "Fonts")

	files, err := os.ReadDir(fontsDir)
	if err != nil {
		return fmt.Errorf("failed to read fonts directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".ttf") {
			continue
		}

		srcPath := filepath.Join(fontsDir, file.Name())
		destPath := filepath.Join(fontsPath, file.Name())

		// Copy font file
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy font %s: %w", file.Name(), err)
		}
	}

	// Note: On Windows, fonts need to be registered in the registry
	// This requires admin privileges. For now, we just copy the files.
	// Users may need to double-click fonts to install them or restart.
	fmt.Println("Note: You may need to restart your applications to see the new fonts.")

	return nil
}

// installFontsMacOS installs fonts on macOS
func (j *JetBrainsMonoInstaller) installFontsMacOS(fontsDir string) error {
	// macOS user fonts directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fontsPath := filepath.Join(homeDir, "Library", "Fonts")
	if err := os.MkdirAll(fontsPath, 0755); err != nil {
		return fmt.Errorf("failed to create fonts directory: %w", err)
	}

	files, err := os.ReadDir(fontsDir)
	if err != nil {
		return fmt.Errorf("failed to read fonts directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".ttf") {
			continue
		}

		srcPath := filepath.Join(fontsDir, file.Name())
		destPath := filepath.Join(fontsPath, file.Name())

		// Copy font file
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy font %s: %w", file.Name(), err)
		}
	}

	return nil
}

// installFontsLinux installs fonts on Linux
func (j *JetBrainsMonoInstaller) installFontsLinux(fontsDir string) error {
	// Linux user fonts directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fontsPath := filepath.Join(homeDir, ".local", "share", "fonts", "JetBrainsMono")
	if err := os.MkdirAll(fontsPath, 0755); err != nil {
		return fmt.Errorf("failed to create fonts directory: %w", err)
	}

	files, err := os.ReadDir(fontsDir)
	if err != nil {
		return fmt.Errorf("failed to read fonts directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".ttf") {
			continue
		}

		srcPath := filepath.Join(fontsDir, file.Name())
		destPath := filepath.Join(fontsPath, file.Name())

		// Copy font file
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy font %s: %w", file.Name(), err)
		}
	}

	// Refresh font cache on Linux
	fmt.Println("Refreshing font cache...")
	// Attempts to run fc-cache -f to refresh the font cache
	// This is not critical and won't fail if fc-cache is not installed
	_ = refreshFontCacheLinux()

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// refreshFontCacheLinux refreshes the font cache on Linux
func refreshFontCacheLinux() error {
	// Try to run fc-cache to refresh font cache
	// This is not critical, so we don't return errors if it fails
	cmd := exec.Command("fc-cache", "-f")

	// Run the command, but ignore any errors
	// fc-cache might not be installed or might fail for various reasons
	if err := cmd.Run(); err != nil {
		// Log that we tried but failed (not critical)
		// In a production system, you might want to use a proper logger here
		_ = err // Ignore the error
	}

	return nil
}
