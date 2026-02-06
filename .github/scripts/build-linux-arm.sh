#!/bin/bash
set -e

# Build script for Linux ARM (arm64 and arm/v7) in Amazon Linux 2 containers
# This ensures GLIBC 2.26 compatibility (AL2, RHEL8, CentOS 7/8, Debian Bullseye)
# Uses QEMU emulation when running on x86_64 hosts

WORKSPACE_DIR="${1:-$(pwd)}"
VERSION="${2:-dev}"
GO_VERSION="${3:-1.25.6}"
TIMEOUT="${4:-30m}"

echo "Building ARM binaries in Amazon Linux 2 containers..."
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

# Build arm64 natively in arm64 container
echo ""
echo "=========================================="
echo "Building Linux arm64 in Amazon Linux 2 arm64 container..."
echo "=========================================="

$TIMEOUT_CMD docker run --rm \
  --platform linux/arm64 \
  -v "$WORKSPACE_DIR:/workspace" \
  -w /workspace \
  -e VERSION="$VERSION" \
  -e GO_VERSION="$GO_VERSION" \
  amazonlinux:2 \
  bash -c '
    set -ex
    
    # Install build dependencies
    echo "Installing build dependencies..."
    echo "sslverify=0" >> /etc/yum.conf
    yum install -y git make gcc gcc-c++ diffutils wget tar glibc-devel && yum clean all
    
    # Install Go for arm64
    echo "Installing Go $GO_VERSION for arm64..."
    cd /tmp
    wget -q --tries=3 --timeout=60 https://go.dev/dl/go$GO_VERSION.linux-arm64.tar.gz
    
    if [ ! -f go$GO_VERSION.linux-arm64.tar.gz ]; then
      echo "ERROR: Go download failed"
      exit 1
    fi
    
    tar -C /usr/local -xzf go$GO_VERSION.linux-arm64.tar.gz
    export PATH=/usr/local/go/bin:$PATH
    
    # Verify
    echo "Verifying Go installation..."
    go version
    uname -m
    
    # Build arm64 natively (no cross-compiler needed)
    cd /workspace
    echo "Building Linux arm64 binary..."
    
    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
      go build -buildvcs=false -buildmode=c-shared \
      -o out_newrelic-linux-arm64-${VERSION}.so .
    
    # Show build artifact
    echo "Build artifact details:"
    ls -lh out_newrelic-linux-arm64-*.so
    file out_newrelic-linux-arm64-*.so
    
    # Check GLIBC requirement
    echo "GLIBC requirements:"
    objdump -T out_newrelic-linux-arm64-*.so 2>/dev/null | grep GLIBC | sed "s/.*GLIBC_/GLIBC_/" | sort -u || true
    
    echo "arm64 build completed successfully!"
  '

# Build arm/v7 natively in arm32 container
echo ""
echo "=========================================="
echo "Building Linux arm/v7 in arm32v7 Debian container..."
echo "=========================================="

# Note: Amazon Linux 2 doesn't have official arm32 images
# Using Debian Buster (GLIBC 2.28) which is compatible with RHEL8
$TIMEOUT_CMD docker run --rm \
  --platform linux/arm/v7 \
  -v "$WORKSPACE_DIR:/workspace" \
  -w /workspace \
  -e VERSION="$VERSION" \
  -e GO_VERSION="$GO_VERSION" \
  arm32v7/debian:buster \
  bash -c '
    set -ex
    
    # Install build dependencies
    echo "Installing build dependencies..."
    apt-get update
    apt-get install -y git make gcc g++ wget tar ca-certificates
    apt-get clean
    
    # Install Go for arm
    echo "Installing Go $GO_VERSION for arm..."
    cd /tmp
    wget -q --tries=3 --timeout=60 https://go.dev/dl/go$GO_VERSION.linux-armv6l.tar.gz
    
    if [ ! -f go$GO_VERSION.linux-armv6l.tar.gz ]; then
      echo "ERROR: Go download failed"
      exit 1
    fi
    
    tar -C /usr/local -xzf go$GO_VERSION.linux-armv6l.tar.gz
    export PATH=/usr/local/go/bin:$PATH
    
    # Verify
    echo "Verifying Go installation..."
    go version
    uname -m
    
    # Build arm natively (no cross-compiler needed)
    cd /workspace
    echo "Building Linux arm binary..."
    
    CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 \
      go build -buildvcs=false -buildmode=c-shared \
      -o out_newrelic-linux-arm-${VERSION}.so .
    
    # Show build artifact
    echo "Build artifact details:"
    ls -lh out_newrelic-linux-arm-*.so
    file out_newrelic-linux-arm-*.so
    
    # Check GLIBC requirement
    echo "GLIBC requirements:"
    objdump -T out_newrelic-linux-arm-*.so 2>/dev/null | grep GLIBC | sed "s/.*GLIBC_/GLIBC_/" | sort -u || true
    
    echo "arm build completed successfully!"
  '

echo ""
echo "=========================================="
echo "Verifying build artifacts..."
echo "=========================================="
ls -lh "$WORKSPACE_DIR"/out_newrelic-linux-arm64-*.so
ls -lh "$WORKSPACE_DIR"/out_newrelic-linux-arm-*.so

echo ""
echo "ARM builds completed successfully!"

