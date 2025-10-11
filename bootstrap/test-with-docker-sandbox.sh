#!/bin/sh

set -e -x -u

# Install dependencies based on distro
if [ -f /etc/alpine-release ]; then
  if ! command -v wget >/dev/null 2>&1; then
    apk add --no-cache wget coreutils bash
  fi
elif command -v apk >/dev/null 2>&1; then
  # curlimages/curl - needs bash and coreutils
  apk add --no-cache bash coreutils
fi

# For testing specific download tools
if [ "${DEVRIG_TEST_CURL_ONLY:-}" = "1" ]; then
  # Install curl if not present
  if ! command -v curl >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y -qq curl
  fi
  # Remove wget to force curl usage
  rm -f /usr/bin/wget || true
fi

if [ "${DEVRIG_TEST_WGET_ONLY:-}" = "1" ]; then
  # Remove curl to force wget usage
  rm -f /usr/bin/curl || true
  # Install wget if not present
  if ! command -v wget >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y -qq wget
  fi
fi

DIR="/dir name/"
mkdir -p "$DIR"
cd "$DIR"

cp -av /image/ ./
cd image

ls -lah .

chmod +x "$BOOTSTRAP_SCRIPT"

# For hash mismatch test, pre-create a binary with wrong content
case "${DEVRIG_CONFIG:-}" in
  *test-config-mismatch.yaml)
    mkdir -p .devrig
    echo "wrong content" > ".devrig/devrig-linux-x86_64-badhash1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
    ;;
esac

case "$BOOTSTRAP_SCRIPT" in
  *.ps1)
    exec pwsh "./$BOOTSTRAP_SCRIPT" "$@"
    ;;
  *)
    exec "./$BOOTSTRAP_SCRIPT" "$@"
    ;;
esac
