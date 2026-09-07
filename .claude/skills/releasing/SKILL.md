---
name: releasing
description: Cut a wisp-deck release. Use when asked to release, tag, publish, or ship a version, or when bumping VERSION. Covers the mandatory `make release` flow, the binary/packaging rules a release must satisfy, and post-release verification.
---

# Releasing wisp-deck

**EVERY release MUST include fresh `wisp-deck-tui` binaries. NO EXCEPTIONS.**
- The installer downloads binaries from release assets — missing binaries = 404 on install
- The developer's local binary MUST be rebuilt — stale local binary = developer sees old UI
- **NEVER create releases manually via GitHub UI or bare `gh release create`**
- **ALWAYS use `make release`** — it handles binaries (GitHub + local), tagging, and release creation

Run `make release` to automate the full release process. Before running:

1. Bump the version in `VERSION` (semver format: `X.Y.Z`)
2. Commit and push all changes — working tree must be clean
3. Must be on `main` branch
4. `gh` CLI must be installed and authenticated (`brew install gh && gh auth login`)

The script will:
- Run preflight checks (clean tree, main branch, valid version, tag doesn't exist, gh auth, **install verification**)
- Show a confirmation prompt (skip with `--yes` flag)
- Build `wisp-deck-tui` binaries for darwin/arm64 and darwin/amd64
- Create annotated git tag `vX.Y.Z` and push
- Create a GitHub release with binaries attached as assets
- Rebuild the local `~/.local/bin/wisp-deck-tui` binary so the developer sees changes immediately

```bash
make release              # Interactive (with confirmation prompt)
bash scripts/release.sh --yes  # Non-interactive (skip confirmation)
```

**Gotcha:** `gh release create FILE#LABEL` uses the file's **basename** as the download name (not the label). If you build to a mktemp path, users get assets named `tmp.XXXX`. The release script builds to a temp directory with correct filenames to avoid this.

**Post-release verification (MANDATORY):**
```bash
# Verify binaries are downloadable (users get 404 if this fails)
gh release view v$(cat VERSION) --json assets --jq '.assets[].name'
# Must show: wisp-deck-tui-darwin-arm64, wisp-deck-tui-darwin-amd64
```
