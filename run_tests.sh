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

echo "Running tests for FileKeeper..."
# Use the current directory instead of ./...
go test -v .

# Check if tests passed
if [ $? -eq 0 ]; then
    echo "All tests passed successfully!"
    exit 0
else
    echo "Tests failed."
    exit 1
fi