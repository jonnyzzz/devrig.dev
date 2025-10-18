#!/usr/bin/env bash

set -e -x -o

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

SIGN=../../release-process/ssh-sign.sh

key_num=0
$SIGN -l | grep "devrig key" | while IFS= read -r line; do
  ((key_num++))
  FILE="key${key_num}.txt"
  SIG="${FILE}.sig"

  rm -f "$SIG" || true
  printf '%s' "$line" > "$FILE"

  $SIGN --sign "$(pwd)/test-payload.txt" "$SIG" "$line"
  $SIGN --verify "$(pwd)/test-payload.txt" "$SIG" "$line"
done

