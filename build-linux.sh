#!/bin/bash

# Build script for tiny-crm Linux AMD64 binary
# This script uses Docker to cross-compile from macOS to Linux

set -e

echo "Building tiny-crm for Linux AMD64..."

# Build using Docker with glibc (compatible with most Linux distributions)
docker run --platform linux/amd64 --rm \
    -v "$PWD":/usr/src/app \
    -w /usr/src/app \
    golang:1.24 \
    bash -c "apt-get update && apt-get install -y gcc libc6-dev && go build -o tinycrm-linux"

echo "Build completed!"
echo "Binary: tinycrm-linux"

# Show file info
file tinycrm-linux
ls -lh tinycrm-linux

echo ""
echo "Upload tinycrm-linux to your server and run:"
echo "  chmod +x tinycrm-linux"
echo "  ./tinycrm-linux --port 9090"
