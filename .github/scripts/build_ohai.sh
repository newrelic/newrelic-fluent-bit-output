#!/bin/bash
set -euo pipefail

# Check if VERSION is provided
if [[ -z "${VERSION:-}" ]]; then
  echo "Error: VERSION environment variable is not set."
  exit 1
fi

echo "Starting Cross-Compilation with Zig..."
echo "Target Version: $VERSION"

# Parametrize glibc version for Linux targets
GLIBC_VERSION="2.17"

# Array of targets: "GOOS GOARCH ZIG_TARGET OUTPUT_PREFIX EXTENSION"
# We use a specific glibc version for Linux to ensure compatibility with Amazon Linux 2
# Format: "GOOS GOARCH ZIG_TARGET OUTPUT_PREFIX EXTENSION"
targets=(
  "linux amd64 x86_64-linux-gnu.${GLIBC_VERSION} out_newrelic-linux-amd64 .so"
  "linux arm64 aarch64-linux-gnu.${GLIBC_VERSION} out_newrelic-linux-arm64 .so"
  "linux arm arm-linux-gnueabihf.${GLIBC_VERSION} out_newrelic-linux-arm .so"
  "windows amd64 x86_64-windows-gnu out_newrelic-windows-amd64 .dll"
  "windows 386 x86-windows-gnu out_newrelic-windows-386 .dll"
)

pids=()

for target in "${targets[@]}"; do
  # Run in a subshell in the background
  (
    read -r goos goarch zigtarget prefix ext <<< "$target"
    echo "Starting build for $goos/$goarch..."
    
    # Run the build command
    CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch \
    CC="zig cc -target $zigtarget" \
    CXX="zig c++ -target $zigtarget" \
    go build \
      -buildmode=c-shared \
      -ldflags="-s -w" \
      -o "${prefix}-${VERSION}${ext}" \
      .
  ) &
  
  # Store the Process ID (PID) to wait for it later
  pids+=($!)
done

# Wait for all background processes and check exit status
failed=0
for i in "${!pids[@]}"; do
  pid="${pids[$i]}"
  # Retrieve the original target string using the same index
  target_info="${targets[$i]}"
  
  # Parse just the OS/Arch for cleaner logging
  read -r goos goarch _ <<< "$target_info"
  
  # Wait for the PID. If it fails, capture the error.
  if ! wait "$pid"; then
    status=$?
    echo " ERROR: Build failed for $goos/$goarch"
    echo "   Details: PID $pid exited with code $status"
    echo "   Target config: $target_info"
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  echo "Build Summary: One or more builds failed."
  exit 1
fi

echo "All builds finished. Verifying artifacts..."

# Verify artifacts exist
missing=0
for target in "${targets[@]}"; do
  read -r goos goarch zigtarget prefix ext <<< "$target"
  file="${prefix}-${VERSION}${ext}"
  
  if [[ ! -f "$file" ]]; then
    echo "Error: Missing artifact: $file"
    missing=1
  else
    echo "Verified: $file"
  fi
done

if [ "$missing" -ne 0 ]; then
  exit 1
fi

echo "Build complete. Artifacts:"
ls -lh out_newrelic-*