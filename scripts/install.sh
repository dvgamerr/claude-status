#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
PREFIX=${PREFIX:-"$HOME/.local"}
DESTINATION="$PREFIX/bin/claude-status"
TEMP_BINARY=$(mktemp "${TMPDIR:-/tmp}/claude-status.XXXXXX")
trap 'rm -f "$TEMP_BINARY"' EXIT HUP INT TERM

if [ "$#" -gt 1 ]; then
  echo "usage: $0 [path-to-claude-status-binary]" >&2
  exit 2
fi

if [ "$#" -eq 1 ]; then
  cp "$1" "$TEMP_BINARY"
else
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required when no prebuilt binary is supplied" >&2
    exit 1
  fi
  (
    cd "$REPO_DIR"
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$TEMP_BINARY" ./cmd/claude-status
  )
fi

mkdir -p "$PREFIX/bin"
chmod 0755 "$PREFIX/bin"
install -m 0755 "$TEMP_BINARY" "$DESTINATION"

echo "installed $DESTINATION"
echo "next: add configs/claude-settings.json to ~/.claude/settings.json, then run claude-status tui"
