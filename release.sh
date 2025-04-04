#!/bin/bash
# Script to prepare a new FileKeeper release

# Check arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.5"
    exit 1
fi

VERSION=$1

# Validate version format
if [[ ! $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Version must be in format X.Y.Z"
    exit 1
fi

# Check if the working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Commit all changes before releasing."
    exit 1
fi

echo "Preparing release v$VERSION..."

# Run tests to make sure everything works
echo "Running tests..."
./run_tests.sh
if [ $? -ne 0 ]; then
    echo "Error: Tests failed. Fix tests before releasing."
    exit 1
fi

# Update version in main.go
echo "Updating version in source code..."
sed -i "s/ProgramVersion = \"[0-9]\+\.[0-9]\+\.[0-9]\+\"/ProgramVersion = \"$VERSION\"/" main.go

# Commit the version change
echo "Committing version change..."
git add main.go
git commit -m "Version bump to $VERSION"

# Create a tag for this release
echo "Creating release tag..."
git tag -a "v$VERSION" -m "Release v$VERSION"

echo ""
echo "Release v$VERSION prepared successfully!"
echo ""
echo "To push the release to GitHub, run:"
echo "  git push origin main && git push origin v$VERSION"
echo ""
echo "GitHub Actions will automatically build and publish the release."