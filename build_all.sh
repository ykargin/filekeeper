#!/bin/bash
# Script to build FileKeeper for multiple architectures

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

VERSION=$(grep -o 'ProgramVersion = "[^"]*"' main.go | cut -d '"' -f 2)
echo "Building FileKeeper version $VERSION for multiple architectures..."

# Create output directory
OUTDIR="build"
mkdir -p $OUTDIR

# Array of architectures to build
declare -A BUILDS=(
    ["amd64"]="filekeeper-linux-amd64"
    ["arm"]="filekeeper-linux-arm-v7" 
    ["arm64"]="filekeeper-linux-arm64"
)

# Build for each architecture
for arch in "${!BUILDS[@]}"; do
    output="${OUTDIR}/${BUILDS[$arch]}"
    echo "Building for $arch -> $output"
    
    # Set environment variables for cross-compilation
    export GOOS=linux
    export GOARCH=$arch
    
    # Set ARM version if needed
    if [ "$arch" == "arm" ]; then
        export GOARM=7
    else
        unset GOARM
    fi
    
    # Build binary
    go build -v -o "$output" .
    
    if [ $? -ne 0 ]; then
        echo "Failed to build for $arch"
        exit 1
    else
        echo "Successfully built for $arch"
    fi
done

echo -e "\nAll builds completed successfully! Binaries are in the $OUTDIR directory."
ls -la $OUTDIR