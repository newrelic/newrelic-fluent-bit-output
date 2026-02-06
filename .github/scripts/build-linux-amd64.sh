#!/bin/bash
set -e

# Build script for Linux amd64 in Amazon Linux 2 container
# This ensures GLIBC 2.26 compatibility (AL2 and RHEL8)

WORKSPACE_DIR="${1:-$(pwd)}"
VERSION="${2:-dev}"
GO_VERSION="${3:-1.25.6}"
TIMEOUT="${4:-10m}"

echo "Building in Amazon Linux 2 container..."
echo "Workspace: $WORKSPACE_DIR"
echo "Version: $VERSION"
echo "Go Version: $GO_VERSION"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
  echo "ERROR: Docker is not running. Please start Docker and try again."
  exit 1
fi

# Use timeout if available (Linux/CI), skip on macOS
if command -v timeout >/dev/null 2>&1; then
  TIMEOUT_CMD="timeout $TIMEOUT"
else
  TIMEOUT_CMD=""
fi

$TIMEOUT_CMD docker run --rm \
  --platform linux/amd64 \
  -v "$WORKSPACE_DIR:/workspace" \
  -w /workspace \
  -e VERSION="$VERSION" \
  -e GO_VERSION="$GO_VERSION" \
  amazonlinux:2 \
  bash -c '
    set -ex
    
    # Install build dependencies
    echo "Installing build dependencies..."
    # Disable SSL verification for yum (workaround for Docker Desktop on macOS)
    echo "sslverify=0" >> /etc/yum.conf
    yum install -y git make gcc gcc-c++ diffutils wget tar glibc-devel && yum clean all
    
    # Install Go
    echo "Installing Go $GO_VERSION..."
    cd /tmp
    wget -q --tries=3 --timeout=30 https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz
    
    # Verify download succeeded
    if [ ! -f go$GO_VERSION.linux-amd64.tar.gz ]; then
      echo "ERROR: Go download failed"
      exit 1
    fi
    
    tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
    export PATH=/usr/local/go/bin:$PATH
    
    # Verify Go installation
    echo "Verifying Go installation..."
    go version
    
    # Go back to workspace
    cd /workspace
    
    # Run tests
    echo "Running tests..."
    go test ./...
    
    # Build Linux amd64 (GLIBC 2.26 compatible)
    echo "Building Linux amd64 binary..."
    make linux/amd64
    
    # Show build artifact info
    echo "Build artifact details:"
    ls -lh out_newrelic-linux-amd64-*.so
    
    # Fix permissions (Docker creates files as root)
    echo "Fixing file permissions..."
    chown -R $(id -u):$(id -g) .
    
    echo "Build completed successfully!"
  '

echo "Verifying build artifacts..."
ls -lh "$WORKSPACE_DIR"/out_newrelic-linux-amd64-*.so
