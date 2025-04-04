#!/bin/bash
# Script to run tests for FileKeeper

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

echo "Running tests for FileKeeper..."
go test -v ./...

# Check if tests passed
if [ $? -eq 0 ]; then
    echo "All tests passed successfully!"
    exit 0
else
    echo "Tests failed."
    exit 1
fi