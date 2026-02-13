#!/bin/bash
set -euo pipefail

generate_release_notes() {
  local _from_tag="$1"
  local _to_tag="$2"
  # Stub: suppress unused variable warnings
  : "$_from_tag" "$_to_tag"
  echo ""
}

# Only run when executed directly (not sourced for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  generate_release_notes "${1:-}" "${2:-}"
fi
