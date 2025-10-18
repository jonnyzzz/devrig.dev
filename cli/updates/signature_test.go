package updates

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

//go:embed test-payload.txt
var testPayload string

//go:embed key1.txt.sig
var key1Signature []byte

//go:embed key2.txt.sig
var key2Signature []byte

func TestVerifySignature_Key1(t *testing.T) {
	data := []byte(testPayload)

	err := VerifySignature(data, key1Signature)
	if err != nil {
		t.Fatalf("Failed to verify signature with key1: %v", err)
	}
}

func TestVerifySignature_Key2(t *testing.T) {
	data := []byte(testPayload)

	err := VerifySignature(data, key2Signature)
	if err != nil {
		t.Fatalf("Failed to verify signature with key2: %v", err)
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	data := []byte(testPayload)
	invalidSig := []byte("invalid signature")

	err := VerifySignature(data, invalidSig)
	if err == nil {
		t.Fatal("Expected error for invalid signature, got nil")
	}
}

func TestVerifySignature_WrongData(t *testing.T) {
	wrongData := []byte("wrong data")

	err := VerifySignature(wrongData, key1Signature)
	if err == nil {
		t.Fatal("Expected error for wrong data, got nil")
	}
}

func TestVerifySignatureOfLocalLatestRelease(t *testing.T) {
	bytes, err1 := os.ReadFile("../../website/static/download/latest.json")
	sigBytes, err2 := os.ReadFile("../../website/static/download/latest.json.sig")

	if err1 != nil || err2 != nil {
		t.Fatal("Failed to read latest.json or latest.json.sig", err1, err2)
	}

	err := VerifySignature(bytes, sigBytes)
	if err != nil {
		t.Fatal("Expected signatures match, but got: ", err)
	}
}

func TestVerifySignature_OnlySHA512Allowed(t *testing.T) {
	// Test that the code properly parses and verifies sha512 signatures
	sig, err := parseSSHSignature(key1Signature)
	if err != nil {
		t.Fatalf("Failed to parse key1 signature: %v", err)
	}

	// Verify that the signature uses sha512
	if strings.ToLower(sig.hashAlgorithm) != "sha512" {
		t.Fatalf("Expected key1 signature to use sha512, got: %s", sig.hashAlgorithm)
	}

	// Test that sha256 would be rejected by modifying the hash algorithm field
	// Create a copy of the signature with sha256 instead of sha512
	sig.hashAlgorithm = "sha256"

	// Try to verify with the modified signature - this should fail
	data := []byte(testPayload)
	for _, keyStr := range TrustedPublicKeys {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyStr))
		if err != nil {
			continue
		}

		err = verifySSHSignature(pubKey, data, sig)
		if err == nil {
			t.Fatal("Expected error for sha256 hash algorithm, but verification succeeded")
		}

		// Check that the error message mentions the hash algorithm restriction
		if !strings.Contains(err.Error(), "sha512") {
			t.Fatalf("Expected error message to mention sha512 restriction, got: %v", err)
		}

		// Found at least one key that properly rejected sha256
		return
	}

	t.Fatal("No keys were tested for sha256 rejection")
}

func TestTrustedPublicKeys_LoadedFromFiles(t *testing.T) {
	// Verify that TrustedPublicKeys contains exactly 2 keys
	if len(TrustedPublicKeys) != 2 {
		t.Fatalf("Expected 2 trusted public keys, got %d", len(TrustedPublicKeys))
	}

	// Verify key1 is loaded correctly
	if !strings.HasPrefix(TrustedPublicKeys[0], "ssh-ed25519") {
		t.Errorf("Expected key1 to be ssh-ed25519, got: %s", TrustedPublicKeys[0][:20])
	}

	if !strings.Contains(TrustedPublicKeys[0], "devrig key 1") {
		t.Error("Expected key1 to contain 'devrig key 1'")
	}

	// Verify key2 is loaded correctly
	if !strings.HasPrefix(TrustedPublicKeys[1], "ssh-rsa") {
		t.Errorf("Expected key2 to be ssh-rsa, got: %s", TrustedPublicKeys[1][:20])
	}

	if !strings.Contains(TrustedPublicKeys[1], "devrig key 2") {
		t.Error("Expected key2 to contain 'devrig key 2'")
	}
}
