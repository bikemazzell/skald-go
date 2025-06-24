#!/bin/bash

# Version bump script for Skald-Go
# Usage: ./scripts/bump-version.sh [patch|minor|major]

set -e

# Ensure we are in the project root
if [ ! -f "VERSION" ]; then
    echo "Error: VERSION file not found. Please run this script from the project root."
    exit 1
fi

VERSION_FILE="VERSION"
CURRENT_VERSION=$(cat "$VERSION_FILE")
echo "Current version: $CURRENT_VERSION"

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Determine bump type
BUMP_TYPE=${1:-patch}

case $BUMP_TYPE in
    patch)
        PATCH=$((PATCH + 1))
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    *)
        echo "Usage: $0 [patch|minor|major]"
        exit 1
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"
echo "New version: $NEW_VERSION"

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"

# Update README.md version badge
if grep -q "version-.*-blue.svg" README.md; then
    sed -i.bak "s/version-$CURRENT_VERSION-blue.svg/version-$NEW_VERSION-blue.svg/g" README.md && rm README.md.bak
    echo "Updated README.md"
else
    echo "Warning: Version badge not found in README.md"
fi

echo ""
echo "Version bumped to $NEW_VERSION"
echo ""
echo "Next steps:"
echo "1. Commit changes: git add VERSION README.md && git commit -m 'chore: bump version to v$NEW_VERSION'"
echo "2. Create tag: git tag -a v$NEW_VERSION -m 'Release v$NEW_VERSION'"
echo "3. Push changes: git push && git push --tags" 