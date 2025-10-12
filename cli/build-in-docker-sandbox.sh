#!/usr/bin/env bash
set -x -e -o

if [ "${BUILD_INSIDE_DOCKER:-not-set}" != "YES" ]; then
  echo "ERROR: This script is designed to run in Docker environment, see build.sh for details"
  exit 79
fi

VERSION="$(cat ./VERSION).${BUILD_NUMBER:-SNAPSHOT}"
echo "Target build number is $VERSION"

mkdir -p "/devrig-build-$VERSION"
cp -av ./ "/devrig-build-$VERSION"
cd "/devrig-build-$VERSION"

ls -lah .

OUTPUT_DIR="/devrig-build-$VERSION-output"

echo "Building devrig v${VERSION} for all platforms..."
echo "Output directory: ${OUTPUT_DIR}"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Platform matrix: OS/ARCH combinations as per specs.md
# Linux: x86_64, ARM64
# macOS: ARM64 (Apple Silicon only, no Intel Macs)
# Windows: x86_64, ARM64

PLATFORMS=(
  "linux/amd64/devrig-linux-x86_64"
  "linux/arm64/devrig-linux-arm64"
  "darwin/arm64/devrig-darwin-arm64"
  "windows/amd64/devrig-windows-x86_64.exe"
  "windows/arm64/devrig-windows-arm64.exe"
)

for platform in "${PLATFORMS[@]}"; do
  IFS='/' read -r GOOS GOARCH OUTPUT <<< "${platform}"

  echo ""
  echo "Building ${OUTPUT}..."

  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build -v \
    -ldflags="-X main.version=${VERSION}" \
    -o "${OUTPUT_DIR}/${OUTPUT}" \
    .

  # Calculate SHA-512 checksum
  if command -v sha512sum &> /dev/null; then
    sha512sum "${OUTPUT_DIR}/${OUTPUT}" | awk '{print $1}' > "${OUTPUT_DIR}/${OUTPUT}.sha512"
  elif command -v shasum &> /dev/null; then
    shasum -a 512 "${OUTPUT_DIR}/${OUTPUT}" | awk '{print $1}' > "${OUTPUT_DIR}/${OUTPUT}.sha512"
  else
    echo "Warning: No SHA-512 tool found, skipping checksum for ${OUTPUT}"
  fi

  CHECKSUM=$(cat "${OUTPUT_DIR}/${OUTPUT}.sha512" 2>/dev/null || echo "N/A")
  echo "  SHA-512: ${CHECKSUM}"
done

echo ""
echo "Build completed successfully!"
echo "Binaries are in: ${OUTPUT_DIR}"
echo ""
echo "Generated files:"
ls -lh "${OUTPUT_DIR}"

DOWNLOAD_URL_BASE="https://github.com/jonnyzzz/devrig/releases/download/v${VERSION}"

# Generate JSON array of releases

for file in "${OUTPUT_DIR}"/devrig-*; do
    [[ "$file" == *.sha512 ]] && continue
    [[ ! -f "$file" ]] && continue

    # Extract just the filename without directory path
    name=$(basename "$file")
    name="${name#devrig-}"      # Remove 'devrig-' prefix
    name="${name%.exe}"             # Remove '.exe' suffix if present
    os="${name%%-*}"                # Everything before first '-'
    arch="${name#*-}"               # Everything after first '-'

    sha512=$(cat "${file}.sha512" || exit 133)

    jq -n \
        --indent 2 \
        --arg os "$os" \
        --arg arch "$arch" \
        --arg url "${DOWNLOAD_URL_BASE}/${file}" \
        --arg sha512 "$sha512" \
        '{os: $os, arch: $arch, url: $url, sha512: $sha512}' \
        >> "${OUTPUT_DIR}/latest-tmp.json"
done

jq -n \
    --indent 2 \
    --arg version "${VERSION}" \
    --arg date "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --argjson releases "$(jq -s '.' < "${OUTPUT_DIR}/latest-tmp.json")" \
    '{version: $version, releaseDate: $date, releases: $releases}' \
    > "${OUTPUT_DIR}/latest.json"

rm "${OUTPUT_DIR}/latest-tmp.json"
cat "${OUTPUT_DIR}/latest.json"

cp -av "${OUTPUT_DIR}/." "/devrig-build/"
ls -lah "/devrig-build"

