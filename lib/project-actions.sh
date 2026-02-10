#!/bin/bash
# Project file operations — add, delete, validate.

# Validate a new project path against existing projects.
# Sets _validated_path (absolute, no trailing slash) and _validated_name (basename).
# Returns 0 if valid, 1 if duplicate.
validate_new_project() {
  local input="$1" projects_file="$2"
  local expanded

  expanded="${input/#\~/$HOME}"
  # Normalize: resolve to absolute path, remove trailing slash
  expanded="$(cd "$expanded" 2>/dev/null && pwd)" || expanded="${input/#\~/$HOME}"
  expanded="${expanded%/}"

  # Resolve symlinks in the final path component only
  # If the path is a symlink, follow it; otherwise use the path as-is
  if [ -L "$expanded" ]; then
    expanded="$(readlink "$expanded")"
    expanded="${expanded%/}"
  fi

  _validated_path="$expanded"
  _validated_name="$(basename "$expanded")"

  # Check for duplicates (resolve symlinks in existing paths too)
  if [ -f "$projects_file" ]; then
    local line proj_path
    while IFS= read -r line; do
      [[ -z "$line" || "$line" == \#* ]] && continue
      proj_path="${line#*:}"
      proj_path="${proj_path%/}"
      # Resolve symlinks in the existing project path
      if [ -L "$proj_path" ]; then
        proj_path="$(readlink "$proj_path")"
        proj_path="${proj_path%/}"
      fi
      if [[ "$proj_path" == "$expanded" ]]; then
        return 1
      fi
    done < "$projects_file"
  fi

  return 0
}

# Append a project entry to the projects file (creates parent dirs).
add_project_to_file() {
  local name="$1" path="$2" projects_file="$3"
  mkdir -p "$(dirname "$projects_file")"
  echo "${name}:${path}" >> "$projects_file"
}

# Remove an exact line from the projects file.
delete_project_from_file() {
  local line="$1" projects_file="$2"
  grep -vxF "$line" "$projects_file" > "$projects_file.tmp" || true
  mv "$projects_file.tmp" "$projects_file"
}
