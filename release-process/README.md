# Release Process Scripts

Scripts for managing devrig releases.

## ssh-sign.sh

See [README-SSH-SIGN.md](./README-SSH-SIGN.md) for complete documentation on signing files and verifying signatures using SSH keys.

---

## sync-release.sh

Syncs a GitHub release to the website by downloading, validating, signing, and uploading artifacts.

### Features

- **Fetches** `latest.json` from GitHub releases
- **Downloads** all release artifacts locally
- **Validates** GitHub checksums (SHA256)
- **Validates** SHA512 hashes against `latest.json`
- **Updates** download URLs in `latest.json` to point to devrig.dev
- **Signs** `latest.json` using SSH agent (via `ssh-sign.sh`)
- **Uploads** `latest.json` and `latest.json.sign` to website

### Usage

```bash
# Sync latest release
./sync-release.sh

# Sync specific tag
./sync-release.sh --tag v1.0.0

# Use custom work directory
./sync-release.sh --work-dir ./release-tmp

# Use specific SSH key for signing
./sync-release.sh --key-id "devrig key"

# Skip certain steps
./sync-release.sh --skip-download  # Use existing downloads
./sync-release.sh --skip-validation  # Skip hash checks
./sync-release.sh --skip-upload  # Don't upload (test mode)
```

### Options

- `-t, --tag TAG` - Specify release tag (default: latest)
- `-w, --work-dir DIR` - Working directory for downloads (default: temp)
- `-k, --key-id ID` - SSH key identifier for signing
- `--skip-download` - Skip downloading artifacts
- `--skip-validation` - Skip hash validation
- `--skip-upload` - Skip uploading to website
- `-h, --help` - Show help

### Requirements

- `curl` - For downloading files
- `jq` - For JSON parsing
- `sha256sum` or `shasum` - For checksum validation
- `sha512sum` or `shasum` - For SHA512 validation
- `ssh-keygen` with SSH agent - For signing
- Access to the website repository

### Workflow

1. **Fetch Release** - Gets latest release info from GitHub API
2. **Download latest.json** - Downloads from release assets
3. **Parse Binaries** - Extracts binary information
4. **Download Artifacts** - Downloads all binaries from release
5. **Validate GitHub Checksums** - Verifies SHA256 checksums
6. **Validate SHA512** - Verifies SHA512 hashes from latest.json
7. **Generate New latest.json** - Updates URLs to point to devrig.dev
8. **Sign** - Creates latest.json.sign using SSH key
9. **Upload** - Copies files to website/static/download

### Example Output

```
[INFO] Using temporary work directory: /tmp/tmp.XXXXXX
[INFO] Fetching release information from GitHub...
[INFO] Latest release tag: v1.0.0
[INFO] Release ID: 123456789
[INFO] Looking for latest.json in release assets...
[INFO] Downloading latest.json from: https://github.com/...
[INFO] ✓ Downloaded latest.json
[INFO] Parsing latest.json...
[INFO] Downloading release artifacts...
[INFO] Downloading: devrig-linux-x86_64
[INFO] ✓ Downloaded: devrig-linux-x86_64
[INFO] Downloading: devrig-linux-arm64
[INFO] ✓ Downloaded: devrig-linux-arm64
[INFO] ✓ All artifacts downloaded
[INFO] Validating GitHub checksums...
[INFO] ✓ GitHub checksums validated
[INFO] Validating SHA512 hashes from latest.json...
[INFO] ✓ devrig-linux-x86_64 validated
[INFO] ✓ devrig-linux-arm64 validated
[INFO] ✓ All SHA512 hashes validated
[INFO] Generating new latest.json with updated URLs...
[INFO] ✓ Generated new latest.json
[INFO] Signing latest.json...
Using key: devrig release key
[INFO] ✓ Created latest.json.sign
[INFO] Uploading to website...
[INFO] ✓ Files uploaded to /path/to/website/static/download
[INFO]
[INFO] ✓ Release sync completed successfully!
[INFO] Release tag: v1.0.0
[INFO] Work directory: /tmp/tmp.XXXXXX
[INFO] Files uploaded to: /path/to/website/static/download
```

### Error Handling

The script will exit with an error if:
- GitHub release cannot be fetched
- `latest.json` is not found in release assets
- Any artifact download fails
- SHA512 hash validation fails
- SSH signing fails
- Website directory is not accessible

### Testing

Test without uploading:

```bash
./sync-release.sh --skip-upload --work-dir ./test-release
```

This will download, validate, and sign but not upload, allowing you to inspect the results.

### Integration with CI/CD

The script can be integrated into CI/CD pipelines:

```yaml
- name: Sync Release
  run: |
    cd release-process/scripts
    ./sync-release.sh --tag ${{ github.event.release.tag_name }}
```

### Notes

- The script uses a temporary directory by default (cleaned up on exit)
- URL mapping is saved to `url_mapping.txt` for reference
- Original `latest.json` is saved as `latest.json.original`
- GitHub checksums are optional (warning if not found)
- Requires SSH agent to be running with appropriate keys loaded
