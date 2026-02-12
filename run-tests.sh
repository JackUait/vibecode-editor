#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"
go test ./test/bash/... ./test/internal/... ./internal/... "$@"
