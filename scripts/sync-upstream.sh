#!/usr/bin/env bash
#
# sync-upstream.sh — re-vendor libodzip's C sources from upstream odzip.
#
# Go modules cannot use git submodules (the module proxy ships only a module's
# own tracked files, so a submodule directory would arrive empty for `go get`).
# So the C sources are vendored. This script refreshes them from a pinned ref.
#
# Usage:
#   scripts/sync-upstream.sh                # use the pinned ref in UPSTREAM_REF
#   scripts/sync-upstream.sh v1.0.4         # vendor a specific tag/branch/sha
#
# After running, review the diff, run `go test ./...`, and if the public API in
# libodzip.h changed, update shim.c / shim.h / libodzip.go to match. Commit the
# updated sources together with a bumped UPSTREAM_REF.
set -euo pipefail

REPO="${ODZIP_REPO:-https://github.com/odpay/odzip.git}"
HERE="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$HERE/internal/libodzip"
REF="${1:-$(cat "$DEST/UPSTREAM_REF")}"

# The library translation units + headers, matching upstream CMake LIB_SOURCES.
FILES=(
  odz_util.c bitstream.c huffman.c lz_hashchain.c compress.c decompress.c
  libodzip.h odz.h odz_thread.h bitstream.h huffman.h lz_matcher.h lz_tables.h
)

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Cloning $REPO ..."
git clone --quiet "$REPO" "$tmp/odzip"
git -C "$tmp/odzip" checkout --quiet "$REF"
resolved="$(git -C "$tmp/odzip" rev-parse HEAD)"

echo "Vendoring at $resolved ($(git -C "$tmp/odzip" describe --tags --always "$resolved"))"
for f in "${FILES[@]}"; do
  cp "$tmp/odzip/$f" "$DEST/$f"
done
printf '%s\n' "$resolved" > "$DEST/UPSTREAM_REF"

echo "Done. Pinned ref written to internal/libodzip/UPSTREAM_REF."
echo "Next: review 'git diff', then run 'go test ./...'."
