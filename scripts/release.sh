#!/bin/bash
set -euo pipefail

# --- Preflight check functions ---

check_clean_tree() {
  if [[ -n "$(git status --porcelain)" ]]; then
    echo "Error: Working tree is not clean. Commit or stash changes first." >&2
    return 1
  fi
}

check_main_branch() {
  local branch
  branch="$(git rev-parse --abbrev-ref HEAD)"
  if [[ "$branch" != "main" ]]; then
    echo "Error: Must be on main branch (currently on '$branch')." >&2
    return 1
  fi
}

read_version() {
  local version_file="$1"
  if [[ ! -f "$version_file" ]]; then
    echo "Error: VERSION file not found at $version_file" >&2
    return 1
  fi
  local version
  version="$(tr -d '[:space:]' < "$version_file")"
  if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: VERSION '$version' is not valid semver (expected X.Y.Z)" >&2
    return 1
  fi
  echo "$version"
}

check_tag_not_exists() {
  local tag="$1"
  if git rev-parse "$tag" &>/dev/null; then
    echo "Error: Tag $tag already exists." >&2
    return 1
  fi
}

check_gh_auth() {
  if ! command -v gh &>/dev/null; then
    echo "Error: gh CLI is not installed. Install with: brew install gh" >&2
    return 1
  fi
  if ! gh auth status &>/dev/null; then
    echo "Error: gh CLI is not authenticated. Run: gh auth login" >&2
    return 1
  fi
}

check_formula_exists() {
  local formula_path="$1"
  if [[ ! -f "$formula_path" ]]; then
    echo "Error: Homebrew formula not found at $formula_path" >&2
    return 1
  fi
}

# --- Formula update function ---

update_formula() {
  local formula_path="$1"
  local version="$2"
  local sha="$3"

  sed -i '' \
    -e "s|archive/refs/tags/v[0-9]*\.[0-9]*\.[0-9]*\.tar\.gz|archive/refs/tags/v${version}.tar.gz|" \
    -e "s|sha256 \"[a-zA-Z0-9]*\"|sha256 \"${sha}\"|" \
    "$formula_path"
}

# --- Main orchestration ---

main() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local project_dir="$script_dir/.."
  local version_file="${RELEASE_VERSION_FILE:-$project_dir/VERSION}"
  local formula_path="${RELEASE_FORMULA_PATH:-$project_dir/../homebrew-ghost-tab/Formula/ghost-tab.rb}"
  local yes_flag=false

  # Parse args
  for arg in "$@"; do
    case "$arg" in
      --yes|-y) yes_flag=true ;;
    esac
  done

  # Preflight checks
  echo "Running preflight checks..."

  check_clean_tree
  echo "  ✓ Working tree is clean"

  check_main_branch
  echo "  ✓ On main branch"

  local version
  version="$(read_version "$version_file")"
  echo "  ✓ Version: $version"

  local tag="v$version"
  check_tag_not_exists "$tag"
  echo "  ✓ Tag $tag does not exist"

  check_gh_auth
  echo "  ✓ gh CLI authenticated"

  check_formula_exists "$formula_path"
  echo "  ✓ Homebrew formula found"

  echo ""

  # Confirmation
  if [[ "$yes_flag" != true ]]; then
    echo "Release $tag?"
    echo "  - Create annotated tag $tag"
    echo "  - Push to origin"
    echo "  - Create GitHub release with auto-generated notes"
    echo "  - Update Homebrew formula"
    echo ""
    read -rp "Proceed? [y/N] " confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
      echo "Aborted."
      exit 0
    fi
  fi

  # Tag and push
  echo ""
  echo "Creating tag $tag..."
  git tag -a "$tag" -m "Release $tag"
  echo "Pushing to origin..."
  git push origin main --tags

  # Create GitHub release
  echo "Creating GitHub release..."
  gh release create "$tag" --generate-notes

  # Download tarball and compute SHA256
  echo "Computing SHA256..."
  local tarball_url="https://github.com/JackUait/ghost-tab/archive/refs/tags/${tag}.tar.gz"
  local tmp_tarball
  tmp_tarball="$(mktemp)"
  trap 'rm -f "$tmp_tarball"' EXIT

  if ! curl -fsSL -o "$tmp_tarball" "$tarball_url"; then
    echo "Error: Failed to download tarball from $tarball_url" >&2
    echo "GitHub release was created. Update formula manually." >&2
    exit 1
  fi

  local sha256
  sha256="$(shasum -a 256 "$tmp_tarball" | awk '{print $1}')"
  echo "  SHA256: $sha256"

  # Update Homebrew formula
  echo "Updating Homebrew formula..."
  update_formula "$formula_path" "$version" "$sha256"

  # Commit and push formula
  echo "Committing formula update..."
  (
    cd "$(dirname "$formula_path")/.."
    git add Formula/ghost-tab.rb
    git commit -m "Bump ghost-tab to $tag"
    git push origin main
  )

  echo ""
  echo "✓ Release $tag complete!"
  echo "  GitHub: https://github.com/JackUait/ghost-tab/releases/tag/$tag"
  echo "  Formula: updated and pushed"
}

# Only run main when executed directly (not sourced for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
