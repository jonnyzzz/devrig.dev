package updates

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/ssh"
)

// TrustedPublicKeys contains the hardcoded trusted SSH public keys for signature verification
var TrustedPublicKeys = []string{
	"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIPpXgnYpUQnJaaGkVfqLtoZVGjsmnphxI9EZB/P0Fq devrig key 1",
	"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPFE5IPqPHxFimyrw+Xr6xK8clkhWMtEP61yM0fMuP/24PpE0hd8zTSdgZ1K1UrdnyFaZZqmm0/zxW0Yrj39m69YoxD1GzC5gcF43nmlCaLpcqXU130oTsYzdmvrMGZiZhazLP30mTSjFg8EC9gz5ZJA10xR7p4Bp4syLdRp6bYq3r4b70bHDoTRgxwgsbJZLYJI6/9wkYcSkUpuQmRM7tknwXwFbC5uFoIyaG8chjlJm76HcidSoAOYhpUgE6yC3S1N0DTdi/Rv/5fgr4IQJfFglp8zRyTuKKh5LFjlpsGvt1jnlM7FwQS8VcEJEvkk8nJDSi0J0AB9NB6EiZBvlfIaoRFJbgvhgzookHfxxLd36LOO0Ck+ExfkptW5JUQmS0UiW9PpZrIZG8d4KEZgC86k0OUcnXDP5gTvqC+kwUFnNJjrv/67OVXb1dzXtCLX8BXjn+CXRzWo0d9t+1YJOp3BGlnJfuIwF+UK8V98Hm3mUFW2C0ky6kfEZoCnEd67BI2yasiEpg1/CWv2oPxEflQWQhAhm0NNKKUJGt/oXP1Z54NMHYiM66jcY/6EmaMJ5OZrhxgXtlip2GC+17riD5CPaKaMlDdT41I8OR9lZoiEfjnliiXNoGdao+avzZvZGOSINzMLWtr3VeaX3JooQ6ZRyYlARkzooxdoynJXsvkQ== devrig key 2",
}

// VerifySignature verifies the SSH signature of the data using trusted public keys
func VerifySignature(data []byte, signatureData []byte) error {
	// Parse the SSH signature format
	sig, err := parseSSHSignature(signatureData)
	if err != nil {
		return fmt.Errorf("failed to parse SSH signature: %w", err)
	}

	// Try each trusted public key
	var lastErr error
	for i, keyStr := range TrustedPublicKeys {
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyStr))
		if err != nil {
			lastErr = fmt.Errorf("failed to parse public key %d: %w", i, err)
			continue
		}

		// Check if key type matches signature type
		if pubKey.Type() != sig.signature.Format {
			continue
		}

		// Verify the signature
		err = verifySSHSignature(pubKey, data, sig)
		if err == nil {
			// Signature verified successfully
			return nil
		}
		lastErr = fmt.Errorf("key %d verification failed: %w", i, err)
	}

	if lastErr != nil {
		return fmt.Errorf("signature verification failed with all keys: %w", lastErr)
	}
	return fmt.Errorf("no valid trusted public keys found")
}

// sshSignature represents a parsed SSH signature
type sshSignature struct {
	namespace     string
	hashAlgorithm string
	signature     *ssh.Signature
}

// parseSSHSignature parses an SSH signature in armored format
func parseSSHSignature(data []byte) (*sshSignature, error) {
	// Find the signature block
	beginMarker := []byte("-----BEGIN SSH SIGNATURE-----")
	endMarker := []byte("-----END SSH SIGNATURE-----")

	beginIdx := bytes.Index(data, beginMarker)
	endIdx := bytes.Index(data, endMarker)

	if beginIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("invalid SSH signature format: missing markers")
	}

	// Extract base64 data
	base64Data := data[beginIdx+len(beginMarker) : endIdx]
	base64Data = bytes.ReplaceAll(base64Data, []byte("\n"), []byte(""))
	base64Data = bytes.ReplaceAll(base64Data, []byte("\r"), []byte(""))
	base64Data = bytes.TrimSpace(base64Data)

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(string(base64Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 signature: %w", err)
	}

	// Parse the decoded data
	return parseSSHSignatureBlob(decoded)
}

// parseSSHSignatureBlob parses the decoded SSH signature blob
func parseSSHSignatureBlob(blob []byte) (*sshSignature, error) {
	buf := bytes.NewReader(blob)

	// Read magic bytes "SSHSIG"
	magic := make([]byte, 6)
	if _, err := io.ReadFull(buf, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(magic) != "SSHSIG" {
		return nil, fmt.Errorf("invalid magic bytes: %s", magic)
	}

	// Read version
	var version uint32
	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	// Read public key
	_, err := readString(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Read namespace
	namespace, err := readString(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}

	// Read reserved
	_, err = readString(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read reserved: %w", err)
	}

	// Read hash algorithm
	hashAlg, err := readString(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read hash algorithm: %w", err)
	}

	// Read signature blob
	sigBytes, err := readString(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Parse the signature blob - it's a standard SSH signature format
	sigReader := bytes.NewReader(sigBytes)
	sigFormat, err := readString(sigReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature format: %w", err)
	}

	sigData, err := readString(sigReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature data: %w", err)
	}

	sig := &ssh.Signature{
		Format: string(sigFormat),
		Blob:   sigData,
	}

	return &sshSignature{
		namespace:     string(namespace),
		hashAlgorithm: string(hashAlg),
		signature:     sig,
	}, nil
}

// readString reads a length-prefixed string from the reader
func readString(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return data, nil
}

// verifySSHSignature verifies an SSH signature against data
func verifySSHSignature(pubKey ssh.PublicKey, data []byte, sig *sshSignature) error {
	// Compute the hash of the data
	var hash []byte
	switch strings.ToLower(sig.hashAlgorithm) {
	case "sha256":
		h := sha256.Sum256(data)
		hash = h[:]
	case "sha512":
		h := sha512.Sum512(data)
		hash = h[:]
	default:
		return fmt.Errorf("unsupported hash algorithm: %s", sig.hashAlgorithm)
	}

	// Build the signed message (SSH signature format)
	var buf bytes.Buffer

	// Magic preamble
	buf.WriteString("SSHSIG")

	// Namespace
	_ = writeString(&buf, []byte(sig.namespace))

	// Reserved (empty)
	_ = writeString(&buf, []byte{})

	// Hash algorithm
	_ = writeString(&buf, []byte(sig.hashAlgorithm))

	// Hash of the data
	_ = writeString(&buf, hash)

	// Verify the signature
	return pubKey.Verify(buf.Bytes(), sig.signature)
}

// writeString writes a length-prefixed string
func writeString(w io.Writer, data []byte) error {
	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}
