#!/bin/bash
# Script to generate code coverage report for FileKeeper

echo "Generating code coverage report for FileKeeper..."

# Run tests with coverage profiling
go test -v ./... -coverprofile=coverage.txt -covermode=atomic

# Check if test passed
if [ $? -ne 0 ]; then
    echo "Tests failed. Coverage report may be incomplete."
    exit 1
fi

# Display coverage statistics in the terminal
go tool cover -func=coverage.txt

# Generate HTML coverage report
go tool cover -html=coverage.txt -o coverage.html

echo ""
echo "HTML coverage report generated: coverage.html"
echo "Open this file in a browser to view detailed coverage information."