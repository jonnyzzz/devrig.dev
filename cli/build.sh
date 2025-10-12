#!/usr/bin/env bash
set -xeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building production website..."
BUILDER_IMAGE=devrig-builder


docker build -t $BUILDER_IMAGE .

rm -rf "$(pwd)/build-in-docker" || true
mkdir -p "$(pwd)/build-in-docker" || true

docker run -it --rm \
       -v "$(pwd):/devrig-base-cli:ro" \
       -v "$(pwd)/build-in-docker:/devrig-build:rw" \
       -e BUILD_INSIDE_DOCKER=YES \
       --workdir "/devrig-base-cli" \
       $BUILDER_IMAGE \
       /devrig-base-cli/build-in-docker-sandbox.sh

