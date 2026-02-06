#!/bin/bash
# Build script for ARM (arm64 and arm/v7) binaries
# Uses native ARM containers via QEMU emulation to ensure GLIBC compatibility
# with older distributions like Amazon Linux 2, RHEL 8, CentOS 7/8

set -e

# Configuration
WORKSPACE_DIR="${1:-$(pwd)}"
VERSION="${2:-0.0.0}"
GO_VERSION="${3:-1.25.6}"

echo "=== ARM Build Configuration ==="
echo "Workspace: $WORKSPACE_DIR"
echo "Version: $VERSION"
echo "Go Version: $GO_VERSION"

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    exit 1
fi

# Set up QEMU for multi-platform builds
echo "=== Setting up QEMU for ARM emulation ==="
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# Build arm64 binary using Amazon Linux 2 (GLIBC 2.26)
echo "=== Building arm64 binary in Amazon Linux 2 container ==="
docker run --rm --platform linux/arm64 \
    -v "$WORKSPACE_DIR:/workspace" \
    -w /workspace \
    -e VERSION="$VERSION" \
    -e CGO_ENABLED=1 \
    amazonlinux:2 bash -c "
        set -e
        echo 'Installing build dependencies...'
        yum install -y gcc make tar gzip git
        
        echo 'Installing Go $GO_VERSION...'
        curl -sL https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz -o go.tar.gz
        tar -C /usr/local -xzf go.tar.gz
        rm go.tar.gz
        export PATH=\$PATH:/usr/local/go/bin
        export GOPATH=/root/go
        export GOCACHE=/tmp/go-cache
        
        echo 'Go version:'
        go version
        
        echo 'Building arm64 plugin...'
        go build -buildmode=c-shared -ldflags \"-X main.VERSION=\$VERSION\" -o out_newrelic-linux-arm64-\${VERSION}.so .
        
        echo 'Verifying build...'
        ls -la out_newrelic-linux-arm64-*.so
        file out_newrelic-linux-arm64-*.so
    "

# Build arm/v7 binary using Debian Buster (GLIBC 2.28)
echo "=== Building arm/v7 binary in Debian Buster container ==="
docker run --rm --platform linux/arm/v7 \
    -v "$WORKSPACE_DIR:/workspace" \
    -w /workspace \
    -e VERSION="$VERSION" \
    -e CGO_ENABLED=1 \
    arm32v7/debian:buster bash -c "
        set -e
        echo 'Installing build dependencies...'
        apt-get update
        apt-get install -y gcc make curl git ca-certificates
        
        echo 'Installing Go $GO_VERSION...'
        curl -sL https://go.dev/dl/go${GO_VERSION}.linux-armv6l.tar.gz -o go.tar.gz
        tar -C /usr/local -xzf go.tar.gz
        rm go.tar.gz
        export PATH=\$PATH:/usr/local/go/bin
        export GOPATH=/root/go
        export GOCACHE=/tmp/go-cache
        
        echo 'Go version:'
        go version
        
        echo 'Building arm (32-bit) plugin...'
        go build -buildmode=c-shared -ldflags \"-X main.VERSION=\$VERSION\" -o out_newrelic-linux-arm-\${VERSION}.so .
        
        echo 'Verifying build...'
        ls -la out_newrelic-linux-arm-*.so
        file out_newrelic-linux-arm-*.so
    "

echo "=== ARM Build Complete ==="
ls -la "$WORKSPACE_DIR"/out_newrelic-linux-arm*.so

echo "=== Verifying GLIBC requirements ==="
# Note: objdump may not be available for cross-architecture, but file command shows basic info
file "$WORKSPACE_DIR"/out_newrelic-linux-arm64-*.so
file "$WORKSPACE_DIR"/out_newrelic-linux-arm-*.so
