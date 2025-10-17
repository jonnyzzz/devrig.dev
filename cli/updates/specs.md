# Updates Module Specification

## Overview

Create a `cli/updates` module in the Go project that handles downloading, verifying, and parsing DevRig update information.

## Requirements

### 1. Download Functionality

The module must download two files from the DevRig website:
- `https://devrig.dev/download/latest.json` - Contains information about available binaries
- `https://devrig.dev/download/latest.json.sig` - SSH signature for the JSON file

### 2. Signature Validation

The module must validate the SSH signature of `latest.json` using hardcoded trusted public keys.

**Trusted Public Keys:**
```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDIPpXgnYpUQnJaaGkVfqLtoZVGjsmnphxI9EZB/P0Fq devrig key 1
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDPFE5IPqPHxFimyrw+Xr6xK8clkhWMtEP61yM0fMuP/24PpE0hd8zTSdgZ1K1UrdnyFaZZqmm0/zxW0Yrj39m69YoxD1GzC5gcF43nmlCaLpcqXU130oTsYzdmvrMGZiZhazLP30mTSjFg8EC9gz5ZJA10xR7p4Bp4syLdRp6bYq3r4b70bHDoTRgxwgsbJZLYJI6/9wkYcSkUpuQmRM7tknwXwFbC5uFoIyaG8chjlJm76HcidSoAOYhpUgE6yC3S1N0DTdi/Rv/5fgr4IQJfFglp8zRyTuKKh5LFjlpsGvt1jnlM7FwQS8VcEJEvkk8nJDSi0J0AB9NB6EiZBvlfIaoRFJbgvhgzookHfxxLd36LOO0Ck+ExfkptW5JUQmS0UiW9PpZrIZG8d4KEZgC86k0OUcnXDP5gTvqC+kwUFnNJjrv/67OVXb1dzXtCLX8BXjn+CXRzWo0d9t+1YJOp3BGlnJfuIwF+UK8V98Hm3mUFW2C0ky6kfEZoCnEd67BI2yasiEpg1/CWv2oPxEflQWQhAhm0NNKKUJGt/oXP1Z54NMHYiM66jcY/6EmaMJ5OZrhxgXtlip2GC+17riD5CPaKaMlDdT41I8OR9lZoiEfjnliiXNoGdao+avzZvZGOSINzMLWtr3VeaX3JooQ6ZRyYlARkzooxdoynJXsvkQ== devrig key 2
```

The signature verification must:
- Use Go's native `golang.org/x/crypto/ssh` library for validation
- Parse SSH signature format (armored base64 format with PEM-like markers)
- Support both Ed25519 and RSA signature types
- Match signature type with key type before verification
- Accept signatures from either of the two trusted public keys
- Fail if the signature is invalid or cannot be verified

### 3. JSON Parsing

Parse the `latest.json` file into a structured format:

```go
type UpdateInfo struct {
    Binaries []Binary `json:"binaries"`
}

type Binary struct {
    Filename string `json:"filename"`
    OS       string `json:"os"`
    Arch     string `json:"arch"`
    SHA512   string `json:"sha512"`
    URL      string `json:"url"`
}
```

### 4. System Information Interface

Provide an interface to query the current operating system and architecture:

```go
type SystemInfo interface {
    OS() string
    Arch() string
}
```

Implementation must:
- Return values from `runtime.GOOS` and `runtime.GOARCH`
- Return values compatible with the binary distribution naming scheme
- No OS/Arch normalization is performed

### 5. Tests

Include comprehensive tests that:
- Parse JSON successfully
- Verify signature parsing logic
- Load and parse test files from `website/static/download/`
- Test invalid signature rejection
- Find binaries for the current system
- Test the system information interface

## Implementation Details

### Download Component

- Use HTTP client with 30-second timeout
- Return errors for non-200 status codes
- Read full response body into memory

### Signature Verification

The signature verification uses Go's native SSH library and implements the OpenSSH signature format:

1. Parse armored signature (PEM-like format with `-----BEGIN SSH SIGNATURE-----` markers)
2. Decode base64 content
3. Parse binary structure:
   - Magic bytes: `SSHSIG`
   - Version (uint32)
   - Public key (length-prefixed)
   - Namespace (length-prefixed string, e.g., "file")
   - Reserved field (empty)
   - Hash algorithm (e.g., "sha512")
   - Signature blob (contains format and signature data)
4. Reconstruct signed message:
   - Magic: `SSHSIG`
   - Namespace
   - Reserved (empty)
   - Hash algorithm
   - Hash of the data
5. Verify using `ssh.PublicKey.Verify()`

No external `ssh-keygen` command is required.

### Error Handling

All functions should return descriptive errors using `fmt.Errorf` with `%w` for error wrapping.

### Security Considerations

- Public keys are hardcoded and cannot be changed at runtime
- All downloads use HTTPS
- Signature verification must succeed before trusting downloaded data
- Signature type must match key type to prevent type confusion attacks

## Module Structure

The module is organized into the following files:

- **`updates.go`** - High-level client API for fetching and parsing updates
- **`downloader.go`** - HTTP download functionality
- **`signature.go`** - SSH signature verification and cryptographic functions
- **`types.go`** - Data structures (UpdateInfo, Binary, SystemInfo)
- **`updates_test.go`** - Comprehensive test suite
- **`example_test.go`** - Usage examples

## Usage Examples

### High-Level API (Recommended)

The simplest way to fetch and verify update information:

```go
package main

import (
    "fmt"
    "log"
    "jonnyzzz.com/devrig.dev/updates"
)

func main() {
    // Create a client
    client := updates.NewClient()

    // Fetch, verify, and parse in one call
    updateInfo, err := client.FetchLatestUpdateInfo()
    if err != nil {
        log.Fatalf("Failed to fetch updates: %v", err)
    }

    // Find binary for current system
    binary := updateInfo.FindBinaryForCurrentSystem()
    if binary == nil {
        log.Fatal("No binary found for current system")
    }

    fmt.Printf("Download URL: %s\n", binary.URL)
    fmt.Printf("SHA512: %s\n", binary.SHA512)
}
```

### Finding Binaries for Specific Platforms

```go
// Find binary for a specific OS/arch
binary := updateInfo.FindBinary("darwin", "arm64")
if binary != nil {
    fmt.Printf("Darwin ARM64: %s\n", binary.URL)
}
```

### Low-Level API

For advanced use cases where you need more control:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "jonnyzzz.com/devrig.dev/updates"
)

func main() {
    // Create downloader
    downloader := updates.NewDownloader()

    // Download files separately
    data, err := downloader.DownloadLatestJSON()
    if err != nil {
        log.Fatalf("Failed to download: %v", err)
    }

    signature, err := downloader.DownloadLatestJSONSig()
    if err != nil {
        log.Fatalf("Failed to download signature: %v", err)
    }

    // Verify signature manually
    if err := updates.VerifySignature(data, signature); err != nil {
        log.Fatalf("Signature verification failed: %v", err)
    }

    // Parse JSON manually
    var updateInfo updates.UpdateInfo
    if err := json.Unmarshal(data, &updateInfo); err != nil {
        log.Fatalf("Failed to parse: %v", err)
    }

    fmt.Printf("Successfully verified and parsed %d binaries\n", len(updateInfo.Binaries))
}
```

## Testing

Run unit tests:
```bash
go test ./cli/updates
```

Run tests with verbose output:
```bash
go test -v ./cli/updates
```

## Dependencies

- `golang.org/x/crypto/ssh` - SSH signature parsing and verification
- Standard library - HTTP client, JSON parsing, hashing

No external commands required.
