---
name: release
description: Pre-release checklist to ensure CHANGELOG, version numbers, tests, and documentation are updated before cutting a release.
---

# Release Skill - Pre-Release Validation

You are an expert at preparing software releases and ensuring no critical steps are missed.

## When to Use

Use this skill **before** cutting any release to validate:
- CHANGELOG is updated with all changes
- Version numbers are consistent across files
- All tests pass
- Documentation reflects new features
- No uncommitted changes exist

## The Problem

Releases often ship with:
- ❌ Missing CHANGELOG entries
- ❌ Inconsistent version numbers
- ❌ Outdated documentation
- ❌ Failing tests
- ❌ Uncommitted changes

This skill provides a systematic checklist to prevent these issues.

## Pre-Release Checklist

### Step 1: Determine Release Version

```bash
# Check current version
git describe --tags --abbrev=0

# Check recent commits for release type
git log --oneline $(git describe --tags --abbrev=0)..HEAD
```

**Semantic Versioning:**
- `MAJOR.MINOR.PATCH` (e.g., 1.2.0)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backwards compatible)
- **PATCH**: Bug fixes only

**Next version for this release:**
- Breaking changes? → Bump MAJOR (2.0.0)
- New features? → Bump MINOR (1.3.0)
- Bug fixes only? → Bump PATCH (1.2.1)

### Step 2: Update CHANGELOG.md

**CRITICAL: This must be done BEFORE creating the release.**

```bash
# 1. Read CHANGELOG.md
cat CHANGELOG.md

# 2. Check what's changed since last release
git log --oneline $(git describe --tags --abbrev=0)..HEAD --pretty=format:"%s"

# 3. Categorize commits
#    - feat: → Added section
#    - fix: → Fixed section
#    - docs: → Changed section
#    - refactor/perf: → Changed section
#    - BREAKING: → mention in description
```

**CHANGELOG Format:**

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features go here
- Each feature as a bullet point
- Use present tense ("Add feature" not "Added feature")

### Changed
- Breaking changes
- Improvements to existing features
- Documentation updates

### Fixed
- Bug fixes
- Security patches

### Removed
- Deprecated features removed

[X.Y.Z]: https://github.com/USER/REPO/compare/vX.Y.Z-1...vX.Y.Z
```

**What to Include:**
- ✅ All user-facing changes
- ✅ New commands, flags, features
- ✅ Bug fixes that users would notice
- ✅ Breaking changes (ALWAYS mention these)
- ✅ New skills or major documentation
- ❌ Internal refactoring (unless it improves performance)
- ❌ Code cleanup that doesn't affect users
- ❌ Test-only changes

### Step 3: Update Version Numbers

Check and update version in all files:

```bash
# 1. Homebrew formula
grep "version" Formula/linear-cli.rb

# 2. Any version constants in code
grep -r "version.*=.*\"" --include="*.go" .

# 3. Package files (if applicable)
grep "version" package.json 2>/dev/null
```

**Files to check:**
- `Formula/linear-cli.rb` - version field
- Any version constants in Go code
- README badges (if version is shown)

### Step 4: Verify Documentation

**README.md:**
```bash
# Check README mentions new features
grep -i "search" README.md  # Example: check new feature is documented
```

**CLAUDE.md:**
```bash
# Check AI agent documentation is current
cat CLAUDE.md | head -100
```

**Questions to ask:**
- [ ] Does README showcase new features?
- [ ] Are new commands in the help output?
- [ ] Do examples use the latest syntax?
- [ ] Are new skills listed?

### Step 5: Run All Tests

```bash
# Run full test suite
make test

# Check for test failures
echo $?  # Should be 0
```

**If tests fail:**
- ❌ DO NOT proceed with release
- Fix the failing tests first
- Commit the fixes
- Run tests again

### Step 6: Check for Uncommitted Changes

```bash
git status
```

**Must show:**
```
On branch main
nothing to commit, working tree clean
```

**If there are uncommitted changes:**
- Review them carefully
- Commit if they should be in the release
- Discard if they're experimental

### Step 7: Build and Smoke Test

```bash
# Build the binary
make build

# Test critical commands
./bin/linear --help
./bin/linear search --help
./bin/linear skills list

# Test a real command (if auth is configured)
./bin/linear issues list --limit 1
```

### Step 8: Create Release

Only after ALL previous steps pass:

```bash
# 1. Create annotated tag
git tag -a vX.Y.Z -m "Release vX.Y.Z - <Brief summary>

<Copy key points from CHANGELOG>
"

# 2. Push tag
git push origin vX.Y.Z

# 3. Run goreleaser
export GITHUB_TOKEN=$(gh auth token)
goreleaser release --clean

# 4. Update Homebrew formula (after goreleaser completes)
# - Update version in Formula/linear-cli.rb
# - Update SHA256 checksums from dist/checksums.txt
# - Commit and push

git add Formula/linear-cli.rb
git commit -m "chore: Update Homebrew formula to vX.Y.Z"
git push origin main
```

### Step 9: Verify Release

```bash
# 1. Check GitHub release was created
open "https://github.com/USER/REPO/releases/tag/vX.Y.Z"

# 2. Uninstall old version
brew uninstall linear-cli

# 3. Install new version from Homebrew
brew update
brew install linear-cli

# 4. Verify version
linear --version

# 5. Test critical functionality
linear search --help
linear skills list
```

## Release Checklist Template

Copy this checklist before each release:

```markdown
## Release vX.Y.Z Checklist

### Pre-Release
- [ ] Determined version number (MAJOR.MINOR.PATCH)
- [ ] Updated CHANGELOG.md with all changes
  - [ ] Added section
  - [ ] Changed section
  - [ ] Fixed section
  - [ ] Version link at bottom
- [ ] Verified version numbers consistent
  - [ ] Formula/linear-cli.rb
  - [ ] Code constants (if any)
- [ ] Documentation updated
  - [ ] README.md includes new features
  - [ ] CLAUDE.md current
  - [ ] Help text accurate
- [ ] All tests pass (`make test`)
- [ ] No uncommitted changes (`git status`)
- [ ] Built and smoke tested (`make build`)

### Release
- [ ] Created annotated tag (`git tag -a vX.Y.Z`)
- [ ] Pushed tag (`git push origin vX.Y.Z`)
- [ ] Ran goreleaser successfully
- [ ] Updated Homebrew formula
  - [ ] Version number
  - [ ] SHA256 checksums
  - [ ] Committed and pushed

### Post-Release
- [ ] Verified GitHub release exists
- [ ] Tested Homebrew install
- [ ] Verified `linear --version` shows correct version
- [ ] Smoke tested critical commands
```

## Common Mistakes to Avoid

### ❌ Forgetting CHANGELOG

**Problem:** Users don't know what changed.

**Solution:** ALWAYS update CHANGELOG BEFORE creating tag.

### ❌ Inconsistent Versions

**Problem:** Formula says v1.2.0 but tag is v1.3.0.

**Solution:** Grep for all version references before tagging.

### ❌ Releasing with Failing Tests

**Problem:** Broken release shipped to users.

**Solution:** `make test` must pass before release.

### ❌ Missing Documentation

**Problem:** New feature exists but isn't documented.

**Solution:** Update README and skills before release.

### ❌ Wrong Semantic Version

**Problem:** Bug fix released as MINOR version.

**Solution:** Review commit log to classify changes:
- New features = MINOR
- Bug fixes = PATCH
- Breaking changes = MAJOR

## Rollback Procedure

If you need to rollback a bad release:

```bash
# 1. Delete the GitHub release
gh release delete vX.Y.Z --yes

# 2. Delete the tag locally and remotely
git tag -d vX.Y.Z
git push origin :refs/tags/vX.Y.Z

# 3. Fix the issues
# ... make fixes ...

# 4. Start release process again with same version (or bump to vX.Y.Z+1)
```

## Automation Notes

**What could be automated:**
- ✅ Version bump in files (script)
- ✅ CHANGELOG generation from commits (partially - needs human review)
- ✅ Running tests (CI/CD)
- ❌ Writing good CHANGELOG descriptions (needs human judgment)
- ❌ Deciding what's user-facing vs internal (needs context)

**This skill focuses on the human judgment parts that can't be automated.**

## Example Release Session

```
User: I want to cut a new release for the search feature we just added.
Assistant (using /release skill):
  Let me guide you through the pre-release checklist for cutting a release.

  **Step 1: Determine Version**
  Current version: v1.0.1
  Commits since last release:
  - feat: Add unified search command
  - feat: Add dependency filtering
  - feat: Add /link-deps skill
  - docs: Update documentation
  - fix: Token sanitization

  This contains new features (search, link-deps skill), so we should bump MINOR version.
  → Next version: v1.2.0

  **Step 2: CHANGELOG Check**
  Let me read CHANGELOG.md...
  
  ❌ CHANGELOG.md is missing v1.2.0 section!
  
  I need to add a section for v1.2.0. Let me review commits to categorize:
  
  Added:
  - Unified search command
  - Dependency filters (--blocked-by, --blocks, etc.)
  - /link-deps skill
  - Enhanced documentation
  
  Changed:
  - All skills updated with search references
  
  Fixed:
  - Token sanitization
  
  Let me update CHANGELOG.md now...
  [Updates CHANGELOG]
  
  ✅ CHANGELOG.md now includes v1.2.0
  
  **Step 3: Version Numbers**
  Checking Formula/linear-cli.rb... shows v1.1.1
  ❌ Needs update to v1.2.0 (but this happens AFTER goreleaser)
  
  **Step 4: Documentation**
  Checking README.md... ✅ Search feature documented
  Checking CLAUDE.md... ✅ Current
  Checking --help output... ✅ Up to date
  
  **Step 5: Tests**
  Running: make test
  ✅ All tests pass
  
  **Step 6: Uncommitted Changes**
  Running: git status
  
  Modified files:
  - CHANGELOG.md
  
  ❌ We have uncommitted CHANGELOG changes
  → Commit these before proceeding
  
  [User commits CHANGELOG]
  
  **Step 7: Build and Smoke Test**
  Running: make build
  Testing: ./bin/linear search --help
  ✅ Build successful, commands working
  
  **Step 8: Ready to Release**
  ✅ All checks passed! You can now:
  
  1. Create tag: git tag -a v1.2.0 -m "Release v1.2.0"
  2. Push tag: git push origin v1.2.0
  3. Run goreleaser: export GITHUB_TOKEN=$(gh auth token) && goreleaser release --clean
  4. Update Formula/linear-cli.rb with new SHA256 checksums
  5. Commit and push formula
```

This systematic approach caught the missing CHANGELOG entry!

## Quick Reference

**Before EVERY release, run through this:**

```bash
# 1. What version?
git log --oneline $(git describe --tags --abbrev=0)..HEAD

# 2. CHANGELOG updated?
grep -A 20 "## \[.*\]" CHANGELOG.md | head -30

# 3. Tests pass?
make test

# 4. Clean git state?
git status

# 5. Build works?
make build && ./bin/linear --help
```

**If ANY check fails, stop and fix it before creating the tag.**

## Best Practices

1. **Update CHANGELOG first** - Don't wait until after tagging
2. **Review every commit** - Understand what's actually changing
3. **Test the build** - Don't assume it works
4. **Document new features** - Users need to know what's new
5. **Use semantic versioning** - Breaking changes = MAJOR bump

## Success Criteria

After using this skill, you should have:
- ✅ Complete CHANGELOG entry for the new version
- ✅ All version numbers updated
- ✅ All tests passing
- ✅ Documentation current
- ✅ Clean git state
- ✅ Confidence that the release is ready

**Never skip the checklist. Ever.**
