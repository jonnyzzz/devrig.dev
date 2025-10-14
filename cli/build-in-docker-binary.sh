#!/usr/bin/env bash
set -x -e -o

if [ "${BUILD_INSIDE_DOCKER:-not-set}" != "YES" ]; then
  echo "ERROR: This script is designed to run in Docker environment"
  exit 79
fi

# Compute default GOOS and GOARCH if not provided
if [ -z "${GOOS:-}" ]; then
  GOOS="$(go env GOOS)"
  echo "GOOS not provided, using default: ${GOOS}"
fi

if [ -z "${GOARCH:-}" ]; then
  GOARCH="$(go env GOARCH)"
  echo "GOARCH not provided, using default: ${GOARCH}"
fi

# Validate required parameters
if [ -z "${OUTPUT:-}" ]; then
  echo "ERROR: OUTPUT environment variable is required"
  exit 1
fi

if [ -z "${VERSION:-}" ]; then
  echo "ERROR: VERSION environment variable is required"
  exit 1
fi

echo "Building for GOOS=${GOOS} GOARCH=${GOARCH}"
echo "Output: ${OUTPUT}"
echo "Version: ${VERSION}"

GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
  go build -v \
  -ldflags="-X main.version=${VERSION}" \
  -o "${OUTPUT}" \
  .

