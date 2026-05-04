---
name: release
description: Release a new version of structcli. Use when asked to release, cut a release, bump the version, or prepare a new tag. Triggers on "release", "cut a release", "new version", "tag and release", "prepare release".
---

# Release structcli

## Prerequisites

- You are on `main` with a clean working tree.
- Git user email is set to `120051+leodido@users.noreply.github.com` (GitHub blocks pushes with the real email due to email privacy settings).
- All CI workflows (`testing`, `wasm`) are green on `main`.

## Steps

### 1. Determine the version

Identify the next version from `git tag --sort=-v:refname | head -1` and the nature of changes since the last tag (`git log <last-tag>..HEAD --oneline`). Look for `feat!:` or `refactor!:` prefixes indicating breaking changes.

### 2. Write the changelog

Gather merged PRs since the last tag. Each commit on `main` corresponds to a rebase-merged PR. Use the GitHub API to read PR titles, bodies, and labels.

Write the `[Unreleased]` section in `CHANGELOG.md` following Keep a Changelog format with these subsections as needed: Added, Changed, Removed, Fixed. Prefix breaking items in Changed with `**Breaking:**`.

Then:
- Rename `[Unreleased]` to `[X.Y.Z] - YYYY-MM-DD`
- Add a new empty `[Unreleased]` section above it
- Update comparison links at the bottom of the file
- Commit directly to `main`: `git commit -m "docs(changelog): add vX.Y.Z release notes"`
- Push to `main`

### 3. Check for pending release process improvements

If there are Makefile or CI changes needed for the release process itself, open a PR with a `chore:` prefix (excluded from release notes by `.github/release.yml`). Wait for merge before proceeding.

### 4. Release

```
git checkout main && git pull origin main
make release VERSION=X.Y.Z
```

The Makefile target:
1. Guards: clean tree, on `main`
2. Bumps `const Version` in `version.go`
3. Regenerates `examples/full/` (SKILL.md, llms.txt, AGENTS.md via `go generate`)
4. Bumps `github.com/leodido/structcli` version in all example `go.mod` files
5. Commits, tags `vX.Y.Z`, pushes `main` + tag

### 5. Verify

All three must pass:

1. **`releasing` CI workflow** — triggered by the tag push. Runs `version-check` (verifies `version.go` matches tag) then `goreleaser` (creates GitHub Release with auto-generated notes from PR labels).
2. **GitHub Release** — exists at `https://github.com/leodido/structcli/releases/tag/vX.Y.Z`. Auto-generated notes categorize PRs by label: `breaking-change` → "Breaking Changes", `enhancement` → "Features & Enhancements", `bug` → "Bug Fixes", `documentation` → "Documentation". The `chore` label is excluded.
3. **Go module proxy** — `go list -m github.com/leodido/structcli@vX.Y.Z` resolves. May take a few minutes.

## How labels work

The labeler workflow (`.github/workflows/labeler.yml`) auto-labels PRs from conventional commit prefixes in the title. `!:` in the title adds `breaking-change`. The release notes generator (`.github/release.yml`) uses these labels to categorize PRs in the GitHub Release body. This only works for PR-based workflows; direct pushes to `main` won't appear in auto-generated notes.
