#!/usr/bin/env bash
#
# Downloads and extracts JetBrains Toolbox's signed XZ feed on macOS or Linux.
# Requires: curl, openssl, xz

set -euo pipefail

# Adjust if desired
URL="https://download.jetbrains.com/toolbox/feeds/v1/release.feed.xz.signed"
SIGNED_FILE="release.feed.xz.signed"
XZ_FILE="release.feed.xz"
FEED_FILE="release.feed"

rm $SIGNED_FILE || true
rm $XZ_FILE || true
rm $FEED_FILE || true

echo "1) Downloading signed feed from JetBrains..."
curl -L -o "$SIGNED_FILE" "$URL"

echo
echo "2) Extracting XZ from the CMS/PKCS#7 signature container via OpenSSL..."
openssl cms -verify \
  -in "$SIGNED_FILE" \
  -inform DER \
  -noverify \
  -out "$XZ_FILE"

echo
echo "3) Checking file type..."
file "$XZ_FILE"

echo
echo "5) Decompressing fully to $FEED_FILE..."
xz -d "$XZ_FILE"

echo
echo "All done! The feed is now available in \"$FEED_FILE\"."
echo "You can inspect it with 'less $FEED_FILE' or parse it with your tooling."


