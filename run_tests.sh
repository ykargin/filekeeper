#!/bin/bash
# Script to run tests for FileKeeper

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

# Ensure go.mod exists
if [ ! -f go.mod ]; then
    echo "Initializing Go module..."
    go mod init github.com/ykargin/filekeeper
    go mod tidy
fi

# Part 1: Run unit tests on the current architecture
echo "Running unit tests for FileKeeper..."
# Use the current directory
go test -v .

# Check if tests passed
if [ $? -ne 0 ]; then
    echo "Unit tests failed."
    exit 1
fi

# Part 2: Test cross-compilation for different architectures
echo -e "\nTesting cross-compilation for different architectures..."

# Array of architectures to test
ARCHS=("amd64" "arm" "arm64")

# Save original environment
ORIGINAL_GOOS=$GOOS
ORIGINAL_GOARCH=$GOARCH
ORIGINAL_GOARM=$GOARM

for arch in "${ARCHS[@]}"; do
    echo -e "\nTesting compilation for $arch..."
    
    # Set environment variables for cross-compilation
    export GOOS=linux
    export GOARCH=$arch
    
    # Set ARM version if needed
    if [ "$arch" == "arm" ]; then
        export GOARM=7
    else
        unset GOARM
    fi
    
    # Test compilation only (don't run tests for non-native architectures)
    echo "Building for $arch architecture..."
    go build -o filekeeper-$arch
    
    if [ $? -ne 0 ]; then
        echo "Failed to compile for $arch"
        # Restore original environment
        export GOOS=$ORIGINAL_GOOS
        export GOARCH=$ORIGINAL_GOARCH
        export GOARM=$ORIGINAL_GOARM
        exit 1
    else
        echo "Successfully compiled for $arch"
        # Clean up binary
        rm filekeeper-$arch
    fi
done

# Restore original environment
export GOOS=$ORIGINAL_GOOS
export GOARCH=$ORIGINAL_GOARCH
export GOARM=$ORIGINAL_GOARM

echo -e "\nAll tests and cross-compilation tests passed successfully!"
exit 0