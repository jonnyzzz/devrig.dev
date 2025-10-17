#!/bin/bash
set -e -x
set -o pipefail

# Script to sync GitHub release to website
# This script downloads, validates, signs, and uploads a GitHub release

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log_error() {
    echo -e "[ERROR] $*" >&2
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Sync GitHub release to website by downloading, validating, signing, and uploading artifacts.

Options:
  -t, --tag TAG         Specify release tag (default: fetch latest)
  -k, --key-id ID       SSH key identifier for signing

Examples:
  $0                              # Sync latest release
  $0 --tag v1.0.0                 # Sync specific tag
  $0 --key-id "devrig key"        # Use specific SSH key

EOF
    exit 0
}

# Parse arguments
TAG=""
SSH_KEY_ID=""
WORK_DIR="${SCRIPT_DIR}/downloads"

rm -rf "${WORK_DIR}" || true
mkdir -p "${WORK_DIR}" || true

while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -k|--key-id)
            SSH_KEY_ID="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            ;;
    esac
done

cd "$WORK_DIR"

# Step 1: Fetch release info from GitHub
echo "Fetching release information from GitHub..."

if [ -z "$TAG" ]; then
    echo "Fetching latest release..."
    RELEASE_JSON=$(curl -sL https://api.github.com/repos/jonnyzzz/devrig.dev/releases/latest)
    TAG=$(echo "$RELEASE_JSON" | jq -r '.tag_name')
    echo "Latest release tag: $TAG"
else
    echo "Fetching release: $TAG"
    RELEASE_JSON=$(curl -sL "https://api.github.com/repos/jonnyzzz/devrig.dev/releases/tags/$TAG")
fi

if [ -z "$RELEASE_JSON" ] || [ "$RELEASE_JSON" = "null" ]; then
    log_error "Failed to fetch release information"
    exit 1
fi

echo "$RELEASE_JSON" > release.github.json
echo "Release tag: $TAG"
echo "Downloading all artifacts from GitHub..."

echo "$RELEASE_JSON" | jq -r '.assets[] | @json' | while IFS= read -r asset_json; do
    FILENAME=$(echo "$asset_json" | jq -r '.name')
    DOWNLOAD_URL=$(echo "$asset_json" | jq -r '.browser_download_url')

    echo "Downloading: $FILENAME"
    curl -sL "$DOWNLOAD_URL" -o "$FILENAME"

    # Save GitHub metadata for this file
    echo "$asset_json" > "${FILENAME}.github"

    # Extract and save URL
    echo "$DOWNLOAD_URL" > "${FILENAME}.url"

    echo "✓ Downloaded: $FILENAME"
done

echo "✓ All artifacts downloaded"

# Step 3: Validate signatures and extract SHA512 files
echo "Extracting and validating SHA512 files..."

for shafile in *.sha512; do
    if [ -f "$shafile" ]; then
        BINARY_FILE="${shafile%.sha512}"

        if [ -f "$BINARY_FILE" ]; then
            echo "Validating: $BINARY_FILE"

            EXPECTED_SHA512=$(cat "$shafile")

            if command -v sha512sum >/dev/null 2>&1; then
                ACTUAL_SHA512=$(sha512sum "$BINARY_FILE" | awk '{print $1}')
            elif command -v shasum >/dev/null 2>&1; then
                ACTUAL_SHA512=$(shasum -a 512 "$BINARY_FILE" | awk '{print $1}')
            else
                log_error "No SHA512 tool found"
                exit 1
            fi

            if [ "$ACTUAL_SHA512" != "$EXPECTED_SHA512" ]; then
                log_error "SHA512 mismatch for $BINARY_FILE"
                log_error "Expected: $EXPECTED_SHA512"
                log_error "Actual:   $ACTUAL_SHA512"
                exit 1
            fi

            echo "✓ $BINARY_FILE validated"
        fi
    fi
done

echo "✓ All SHA512 hashes validated"

# Step 4: Read latest.json and build final version with URLs
echo "Building final latest.json with URLs..."

if [ ! -f "latest.json" ]; then
    log_error "latest.json not found in artifacts"
    exit 1
fi

NEW_BASE_URL="https://devrig.dev/download"

# Process latest.json: for each binary, add the url field
jq -c '(.binaries // .releases)[]' latest.json | while IFS= read -r binary_json; do
    FILENAME=$(echo "$binary_json" | jq -r '(.filename // .url) | split("/") | last')
    SHA512=$(echo "$binary_json" | jq -r '.sha512')

    if [ ! -f "${FILENAME}.sha512" ]; then
        log_error "SHA512 file not found for: $FILENAME"
        exit 1
    fi

    # Validate SHA512 matches
    DISK_SHA512=$(cat "${FILENAME}.sha512")
    if [ "$SHA512" != "$DISK_SHA512" ]; then
        log_error "SHA512 mismatch in latest.json vs .sha512 file for $FILENAME"
        exit 1
    fi

    # Add URL to the binary entry
    echo "$binary_json" | jq --arg url "$(cat "${FILENAME}.url")" '. + {url: $url}' >> binaries.jsonl
done

# Build final JSON
jq -s '{binaries: .}' binaries.jsonl > latest.final.json

echo "✓ Generated latest.final.json"

# Step 5: Sign latest.final.json
echo "Signing latest.final.json..."

SSH_SIGN_SCRIPT="$SCRIPT_DIR/ssh-sign.sh"

if [ ! -x "$SSH_SIGN_SCRIPT" ]; then
    log_error "ssh-sign.sh not found or not executable: $SSH_SIGN_SCRIPT"
    exit 1
fi

"$SSH_SIGN_SCRIPT" --sign "$(cat latest.final.json)" "$SSH_KEY_ID" > latest.final.json.sig

if [ ! -s latest.final.json.sig ]; then
    log_error "Failed to create signature"
    exit 1
fi

echo "✓ Created latest.json.sig"

echo ""
echo "✓ Release sync completed successfully!"
echo "Release tag: $TAG"
echo "Work directory: $WORK_DIR"
echo "Output files: latest.final.json, latest.final.json.sig"
