#!/bin/bash
set -euo pipefail

NEW_VERSION="$1"

if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <new-version>"
    echo "Example: $0 0.1.2"
    exit 1
fi

# Validate version format (basic check)
if ! [[ "$NEW_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Invalid version format. Use: MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-prerelease"
    echo "Examples: 0.1.2, 0.2.0, 1.0.0, 0.1.2-rc.1"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION_FILE="$REPO_ROOT/VERSION"

CURRENT_VERSION=$(cat "$VERSION_FILE" 2>/dev/null || echo "unknown")

echo "Bumping version: $CURRENT_VERSION → $NEW_VERSION"
echo ""

# Update VERSION file
echo "$NEW_VERSION" > "$VERSION_FILE"
echo "✅ Updated $VERSION_FILE"

# Git operations
if git rev-parse --git-dir > /dev/null 2>&1; then
    echo ""
    echo "Git operations:"

    # Stage the file
    git add "$VERSION_FILE"
    echo "✅ Staged VERSION file"

    # Commit
    git commit -m "Bump version to $NEW_VERSION"
    echo "✅ Committed version bump"

    # Create tag
    git tag -a "v$NEW_VERSION" -m "Wild Cloud Central v$NEW_VERSION"
    echo "✅ Created tag v$NEW_VERSION"

    echo ""
    echo "📋 Next steps:"
    echo "  1. Review the commit: git show"
    echo "  2. Build packages: cd dist && make package-all"
    echo "  3. Create release: cd dist && make release"
    echo "  4. Push to remote: git push origin main v$NEW_VERSION"
else
    echo ""
    echo "⚠️  Not a git repository, skipping git operations"
fi

echo ""
echo "✨ Version bump complete!"
