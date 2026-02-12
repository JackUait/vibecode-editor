# Homebrew Formula Update for v2.0.0

This document describes the changes needed in the Homebrew tap formula for the v2.0.0 hybrid architecture release.

## Location

The formula is in the separate tap repository:
- Repository: `https://github.com/JackUait/homebrew-ghost-tab`
- Formula: `Formula/ghost-tab.rb`

## Required Changes

### 1. Update Version and URL

```ruby
version "2.0.0"
url "https://github.com/JackUait/ghost-tab/archive/v2.0.0.tar.gz"
sha256 "..." # Generate with: shasum -a 256 v2.0.0.tar.gz
```

### 2. Add Build Dependencies

```ruby
depends_on "go" => :build  # Required to build ghost-tab-tui binary
```

### 3. Add Runtime Dependencies

```ruby
depends_on "jq"  # Required for JSON parsing in bash scripts
```

### 4. Add Go Build Step

```ruby
def install
  # Build Go TUI binary
  system "go", "build", "-o", "bin/ghost-tab-tui", "./cmd/ghost-tab-tui"

  # Install binaries
  bin.install "bin/ghost-tab"
  bin.install "bin/ghost-tab-tui"

  # Install support files
  prefix.install "lib", "ghostty"
end
```

### 5. Update Test

```ruby
test do
  # Test both binaries
  system "#{bin}/ghost-tab-tui", "help"
  # ghost-tab requires interactive setup, so just check it exists
  assert_predicate bin/"ghost-tab", :exist?
end
```

## Complete Example Formula

```ruby
class GhostTab < Formula
  desc "Ghostty + tmux wrapper with AI tools"
  homepage "https://github.com/JackUait/ghost-tab"
  url "https://github.com/JackUait/ghost-tab/archive/v2.0.0.tar.gz"
  sha256 "..."
  version "2.0.0"
  license "MIT"

  depends_on "go" => :build
  depends_on "jq"
  depends_on "tmux"
  depends_on "ghostty"

  def install
    # Build Go TUI binary
    system "go", "build", "-o", "bin/ghost-tab-tui", "./cmd/ghost-tab-tui"

    # Install binaries
    bin.install "bin/ghost-tab"
    bin.install "bin/ghost-tab-tui"

    # Install support files
    prefix.install "lib", "ghostty"
  end

  test do
    system "#{bin}/ghost-tab-tui", "help"
    assert_predicate bin/"ghost-tab", :exist?
  end
end
```

## Testing the Formula

Before submitting to the tap:

```bash
# Install from local formula
brew install --build-from-source ./Formula/ghost-tab.rb

# Run formula test
brew test ghost-tab

# Test the installed binaries
ghost-tab-tui help
ghost-tab-tui show-logo

# Uninstall
brew uninstall ghost-tab
```

## Migration Notes

Users upgrading from v1.x to v2.0.0 will get:
- The new `ghost-tab-tui` binary automatically
- `jq` dependency installed automatically
- Go is only used during build, not installed for end users (`:build` dependency)

No manual migration steps required for users.

## Checklist

- [ ] Update version to 2.0.0
- [ ] Update URL to v2.0.0 tarball
- [ ] Generate and update sha256
- [ ] Add `depends_on "go" => :build`
- [ ] Add `depends_on "jq"`
- [ ] Add Go build step
- [ ] Install both binaries
- [ ] Update test section
- [ ] Test formula locally
- [ ] Commit to tap repository
- [ ] Tag release in tap repository
