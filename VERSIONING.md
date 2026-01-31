# Wild Cloud Versioning Strategy

## Current Implementation: Unified Versioning

Wild Cloud uses a **single source of truth** for versioning during the development phase (pre-v1.0.0). All components (API, CLI, Web, Package) share the same version number.

### Version File Location
```
wild-cloud/VERSION
```

Single line file containing the current version (e.g., `0.1.1`).

### Why Unified Versioning Now?

- **Simplicity**: One version to track during development
- **Tight Coupling**: Components are developed and released together
- **Development Phase**: Pre-1.0.0, breaking changes are expected
- **Easy Migration**: Can transition to independent versioning later

---

## Version Format

We follow [Semantic Versioning 2.0.0](https://semver.org/):

```
MAJOR.MINOR.PATCH
```

### Pre-1.0 Semantics (Current)
- **MAJOR** (0): Signals development phase
- **MINOR** (0.x): New features, may include breaking changes
- **PATCH** (0.1.x): Bug fixes, small improvements

### Post-1.0 Semantics (Future)
- **MAJOR**: Breaking changes, incompatible API changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes only

---

## Workflows

### Development Workflow (Iterating on Same Version)

During testing and iteration, update packages without bumping version:

```bash
# Make code changes
cd dist
make package-all

# Update existing release (keeps same version)
make release
```

**Result**: Existing release assets are replaced with new builds.

### Creating a New Release

When ready for a new version:

```bash
# Option 1: Use helper script
./scripts/bump-version.sh 0.1.2

# Option 2: Manual steps
echo "0.1.2" > VERSION
git add VERSION
git commit -m "Bump version to 0.1.2"
git tag -a v0.1.2 -m "Wild Cloud Central v0.1.2"

# Build and release
cd dist
make package-all
make release

# Push to remote
cd ..
git push origin main v0.1.2
```

**Result**: New release with tag `v0.1.2` is created.

### Quick Reference

```bash
# Check current version
cat VERSION

# Bump version (helper script)
./scripts/bump-version.sh 0.2.0

# Build packages
cd dist && make package-all

# Create/update release
cd dist && make release
```

---

## Release Behavior

The `make release` command is intelligent:

- **If release v0.1.1 exists**: Updates package assets only
  - Deletes old .deb files
  - Uploads new .deb files
  - Keeps release notes and metadata

- **If release v0.1.1 doesn't exist**: Creates new release
  - Creates Git tag
  - Creates Gitea/GitHub release
  - Uploads package assets
  - Generates release notes

This allows rapid iteration during testing without creating dozens of releases.

---

## Versioning Best Practices

### When to Bump MINOR (0.x)
- New features added
- API endpoints added/changed
- Breaking changes (pre-1.0)
- Significant improvements

### When to Bump PATCH (0.1.x)
- Bug fixes
- Security patches
- Performance improvements
- Documentation updates
- No user-facing changes

### Pre-release Versions
For testing before official release:

```bash
echo "0.2.0-rc.1" > VERSION  # Release candidate
echo "0.2.0-beta.1" > VERSION  # Beta testing
echo "0.2.0-alpha.1" > VERSION  # Alpha testing
```

---

## Version Information in Components

### API (Go)
Version injected at build time:
```go
var Version = "dev"  // Injected with -ldflags
```

Build command in Makefile:
```makefile
go build -ldflags="-X main.Version=$(VERSION)"
```

### CLI (Go)
Same pattern as API:
```go
var Version = "dev"
```

### Web (React)
Version in package.json (can stay at 0.0.0):
```json
{
  "version": "0.0.0"
}
```

Version available via API at runtime.

### Debian Package
Version inserted into control file during packaging:
```
Package: wild-cloud-central
Version: 0.1.1
```

---

## Future: Independent Component Versioning

When Wild Cloud matures (around v1.0.0), we may need independent versioning for components. This is documented in:

```
dist/future/independent-versioning.md
```

### Migration Triggers
- CLI needs independent releases for CI/CD use
- API stability requires different versioning from UI
- Different release cadences for components
- Need to signal breaking API changes independently

### Migration Path
The future design is already documented and can be implemented when needed:
1. Create component VERSION files (api/VERSION, cli/VERSION, web/VERSION)
2. Create MANIFEST.yaml for compatibility tracking
3. Update build scripts to read multiple versions
4. Add CLI version checking against API
5. Support component-specific releases

---

## Version Discovery

### For Users
```bash
# From command line (future)
wild --version
# Output: Wild CLI v0.1.1 (API v0.1.1)

# From package
dpkg -l wild-cloud-central
# Output: ii  wild-cloud-central  0.1.1  ...
```

### For Developers
```bash
# Current version
cat VERSION

# Version in Makefile
cd dist && make version

# Version info in help
cd dist && make help
```

### For Automation
```bash
# Read version programmatically
VERSION=$(cat VERSION)

# Validate version format
if [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Valid version: $VERSION"
fi
```

---

## Tools and Scripts

### Helper Scripts

**`scripts/bump-version.sh`**
- Validates version format
- Updates VERSION file
- Creates git commit and tag
- Provides next steps

```bash
./scripts/bump-version.sh 0.2.0
```

### Makefile Targets

**`make version`** - Show current version and build info
**`make help`** - Shows version management instructions
**`make package-all`** - Build packages with current version
**`make release`** - Create/update release with current version

---

## FAQ

### Q: Can I release without bumping the version?
**A**: Yes! `make release` will update the existing release assets. Perfect for testing.

### Q: How do I create a patch release?
**A**: Bump the PATCH number: `echo "0.1.2" > VERSION`

### Q: What about pre-release versions?
**A**: Use semver pre-release format: `0.2.0-rc.1`, `0.2.0-beta.1`

### Q: When should we switch to independent versioning?
**A**: When CLI needs separate releases or API needs stability guarantees (likely around v1.0.0)

### Q: Can I manually edit the VERSION file?
**A**: Yes! It's just a text file. Or use `./scripts/bump-version.sh` for automation.

### Q: How does this work with git tags?
**A**: Git tags follow the version: `v0.1.1`. Create them manually or use the bump script.

### Q: What if I forget to bump the version?
**A**: No problem! The release will update the existing version's assets.

---

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- Future design: `dist/future/independent-versioning.md`

---

## Summary

**Current State**: Single VERSION file, unified versioning, perfect for development
**Future State**: Independent component versioning when needed (documented in future/)
**Philosophy**: Keep it simple until complexity is justified
