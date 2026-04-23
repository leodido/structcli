.PHONY: release test generate

# Release a new version.
#
# Usage:
#   make release VERSION=0.17.0
#
# Before running:
#   1. Fill in the CHANGELOG.md [Unreleased] section
#   2. Commit the changelog: git commit -m "docs(changelog): add vX.Y.Z release notes"
#
# This target then:
#   1. Bumps the Version constant in version.go
#   2. Bumps the mcp-command-factory example go.mod require
#   3. Regenerates files that depend on the version (SKILL.md, etc.)
#   4. Commits, tags, and pushes
#
release:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=0.17.0)
endif
	@# Guard: clean working tree
	@git diff --quiet && git diff --cached --quiet || (echo "error: working tree is dirty" && exit 1)
	@# Guard: on main
	@test "$$(git branch --show-current)" = "main" || (echo "error: not on main branch" && exit 1)
	@echo "==> Releasing v$(VERSION)"
	@# 1. Bump version.go
	@sed -i 's/const Version = ".*"/const Version = "$(VERSION)"/' version.go
	@# 2. Regenerate (before bumping the example go.mod, so the workspace
	@#    doesn't try to resolve a version that doesn't exist on the proxy yet)
	@(cd examples/full && go generate ./...)
	@# 3. Bump mcp-command-factory example
	@sed -i 's|github.com/leodido/structcli v[0-9.]*|github.com/leodido/structcli v$(VERSION)|' examples/mcp-command-factory/go.mod
	@# 4. Commit, tag, push
	@git add version.go examples/mcp-command-factory/go.mod examples/full/ go.work.sum
	@git diff --quiet || (echo "error: unstaged changes remain after regeneration — stage them and retry" && git diff --stat && exit 1)
	@git commit -m "chore: bump Version constant to $(VERSION)" \
		-m "Also bump the mcp-command-factory example go.mod require to v$(VERSION)." \
		--trailer "Co-authored-by: Ona <no-reply@ona.com>"
	@git tag v$(VERSION)
	@git push origin main
	@git push origin v$(VERSION)
	@echo "==> v$(VERSION) released. Watch CI at https://github.com/leodido/structcli/actions"

test:
	go test -count=1 ./...

generate:
	(cd examples/full && go generate ./...)
