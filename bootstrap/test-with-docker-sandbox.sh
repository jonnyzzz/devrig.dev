#!/bin/bash

set -e -x -u

DIR="/dir name/"
mkdir -p "$DIR"
cd "$DIR"

cp -av /image/ ./
cd image

ls -lah .

chmod +x "$BOOTSTRAP_SCRIPT"

if [[ "$BOOTSTRAP_SCRIPT" == *.ps1 ]]; then
  exec pwsh "./$BOOTSTRAP_SCRIPT" "$@"
else
  exec "./$BOOTSTRAP_SCRIPT" "$@"
fi
