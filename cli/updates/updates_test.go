package updates

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentSystem_OS(t *testing.T) {
	sys := CurrentSystem{}
	oo := sys.OS()

	// Verify we get an OS name
	if oo != "darwin" && oo != "linux" && oo != "windows" {
		t.Errorf("unexpected OS: %s", oo)
	}
}

func TestCurrentSystem_Arch(t *testing.T) {
	sys := CurrentSystem{}
	arch := sys.Arch()

	// Verify we get an architecture name
	if arch != "x86_64" && arch != "arm64" {
		t.Errorf("unexpected architecture: %s", arch)
	}
}

func TestUpdateInfo_JSONParsing(t *testing.T) {
	jsonData := `{
		"binaries": [
			{
				"filename": "devrig-darwin-arm64",
				"os": "darwin",
				"arch": "arm64",
				"sha512": "abc123",
				"url": "https://devrig.dev/download/devrig-darwin-arm64"
			},
			{
				"filename": "devrig-linux-arm64",
				"os": "linux",
				"arch": "arm64",
				"sha512": "def456",
				"url": "https://devrig.dev/download/devrig-linux-arm64"
			}
		]
	}`

	var updateInfo UpdateInfo
	err := json.Unmarshal([]byte(jsonData), &updateInfo)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(updateInfo.Binaries) != 2 {
		t.Errorf("expected 2 binaries, got %d", len(updateInfo.Binaries))
	}

	// Verify first binary
	if updateInfo.Binaries[0].Filename != "devrig-darwin-arm64" {
		t.Errorf("expected filename 'devrig-darwin-arm64', got '%s'", updateInfo.Binaries[0].Filename)
	}
	if updateInfo.Binaries[0].OS != "darwin" {
		t.Errorf("expected os 'darwin', got '%s'", updateInfo.Binaries[0].OS)
	}
	if updateInfo.Binaries[0].Arch != "arm64" {
		t.Errorf("expected arch 'arm64', got '%s'", updateInfo.Binaries[0].Arch)
	}

	// Verify second binary
	if updateInfo.Binaries[1].OS != "linux" {
		t.Errorf("expected os 'linux', got '%s'", updateInfo.Binaries[1].OS)
	}
	if updateInfo.Binaries[1].Arch != "arm64" {
		t.Errorf("expected arch 'arm64', got '%s'", updateInfo.Binaries[1].Arch)
	}
}

func TestCalculateFingerprint(t *testing.T) {
	data := []byte("test data")
	fingerprint := CalculateFingerprint(data)

	if fingerprint == "" {
		t.Error("fingerprint should not be empty")
	}

	// Verify fingerprint is deterministic
	fingerprint2 := CalculateFingerprint(data)
	if fingerprint != fingerprint2 {
		t.Error("fingerprint should be deterministic")
	}

	// Verify different data produces different fingerprint
	differentData := []byte("different test data")
	differentFingerprint := CalculateFingerprint(differentData)
	if fingerprint == differentFingerprint {
		t.Error("different data should produce different fingerprint")
	}
}

// CalculateFingerprint calculates a SHA256 fingerprint for caching purposes
func CalculateFingerprint(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.URLEncoding.EncodeToString(hash[:])
}

func TestVerifySignature_ParsingWorks(t *testing.T) {
	// Load test data from website/static/download
	repoRoot := filepath.Join("..", "..")
	sigPath := filepath.Join(repoRoot, "website", "static", "download", "latest.json.sig")

	// Read signature
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatalf("Could not read test file %s: %v", sigPath, err)
	}

	// Test that signature parsing works (we can parse the SSH signature format)
	sig, err := parseSSHSignature(sigData)
	if err != nil {
		t.Fatalf("Failed to parse signature: %v", err)
	}

	if sig.namespace != "file" {
		t.Errorf("Expected namespace 'file', got '%s'", sig.namespace)
	}

	if sig.hashAlgorithm != "sha512" {
		t.Errorf("Expected hash algorithm 'sha512', got '%s'", sig.hashAlgorithm)
	}

	t.Logf("Successfully parsed signature with namespace=%s, hashAlg=%s", sig.namespace, sig.hashAlgorithm)
}

func TestVerifySignature_WithInvalidSignature(t *testing.T) {
	data := []byte(`{"binaries": []}`)
	invalidSignature := []byte("invalid signature data")

	err := VerifySignature(data, invalidSignature)
	if err == nil {
		t.Fatal("expected signature verification to fail with invalid signature")
	}

	t.Logf("Correctly rejected invalid signature: %v", err)
}

func TestParseRealJSONFile(t *testing.T) {
	// Load test data from website/static/download
	repoRoot := filepath.Join("..", "..")
	jsonPath := filepath.Join(repoRoot, "website", "static", "download", "latest.json")

	// Read JSON data
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Could not read test file %s: %v", jsonPath, err)
	}

	// Parse JSON
	var updateInfo UpdateInfo
	err = json.Unmarshal(jsonData, &updateInfo)
	if err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(updateInfo.Binaries) == 0 {
		t.Error("expected at least one binary in latest.json")
	}

	t.Logf("Successfully parsed %d binaries from latest.json", len(updateInfo.Binaries))

	// Verify structure
	for i, binary := range updateInfo.Binaries {
		if binary.Filename == "" {
			t.Errorf("Binary %d has empty filename", i)
		}
		if binary.OS == "" {
			t.Errorf("Binary %d has empty OS", i)
		}
		if binary.Arch == "" {
			t.Errorf("Binary %d has empty Arch", i)
		}
		if binary.SHA512 == "" {
			t.Errorf("Binary %d has empty SHA512", i)
		}
		if binary.URL == "" {
			t.Errorf("Binary %d has empty URL", i)
		}
	}
}

func TestClient_FetchLatestUpdateInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := NewClient()
	updateInfo, err := client.FetchLatestUpdateInfo()
	if err != nil {
		// Signature verification may fail if server signature is created with different key
		t.Fatalf("FetchLatestUpdateInfo failed (signature may not match test keys): %v", err)
	}

	if len(updateInfo.Binaries) == 0 {
		t.Error("expected at least one binary")
	}

	t.Logf("Successfully fetched and verified %d binaries", len(updateInfo.Binaries))
}

func TestUpdateInfo_FindBinaryForCurrentSystem(t *testing.T) {
	updateInfo := &UpdateInfo{
		Binaries: []Binary{
			{
				Filename: "devrig-darwin-arm64",
				OS:       "darwin",
				Arch:     "arm64",
				SHA512:   "test",
				URL:      "https://example.com/devrig-darwin-arm64",
			},
			{
				Filename: "devrig-linux-x86_64",
				OS:       "linux",
				Arch:     "x86_64",
				SHA512:   "test",
				URL:      "https://example.com/devrig-linux-x86_64",
			},
		},
	}

	// Test finding by current system
	binary := updateInfo.FindBinaryForCurrentSystem()
	if binary != nil {
		t.Logf("Found binary for current system: %s", binary.Filename)
	} else {
		t.Logf("No binary found for current system (OS: %s, Arch: %s)", CurrentSystem{}.OS(), CurrentSystem{}.Arch())
	}
}

func TestUpdateInfo_FindBinary(t *testing.T) {
	updateInfo := &UpdateInfo{
		Binaries: []Binary{
			{
				Filename: "devrig-darwin-arm64",
				OS:       "darwin",
				Arch:     "arm64",
				SHA512:   "test",
				URL:      "https://example.com/devrig-darwin-arm64",
			},
			{
				Filename: "devrig-linux-x86_64",
				OS:       "linux",
				Arch:     "x86_64",
				SHA512:   "test",
				URL:      "https://example.com/devrig-linux-x86_64",
			},
		},
	}

	// Test finding specific binary
	binary := updateInfo.FindBinary("darwin", "arm64")
	if binary == nil {
		t.Error("expected to find darwin/arm64 binary")
	} else if binary.Filename != "devrig-darwin-arm64" {
		t.Errorf("expected filename 'devrig-darwin-arm64', got '%s'", binary.Filename)
	}

	// Test finding non-existent binary
	binary = updateInfo.FindBinary("windows", "arm64")
	if binary != nil {
		t.Error("expected nil for non-existent binary")
	}
}
