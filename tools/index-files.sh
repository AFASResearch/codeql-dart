#!/usr/bin/env bash
set -euo pipefail

LIST="$1"
EXE="$(dirname "$0")/../extractor/dart-extractor"

echo "[index-files] CWD: $(pwd)"
echo "[index-files] Using extractor: $EXE"

if [ ! -f "$EXE" ]; then
  echo "ERROR: extractor binary not found: $EXE"
  exit 1
fi

if [ ! -f "$LIST" ]; then
  echo "ERROR: file list not found: $LIST"
  exit 1
fi

while IFS= read -r FILE || [ -n "$FILE" ]; do
  # skip empty lines
  [ -z "$FILE" ] && continue

  "$EXE" --index "$FILE"
done < "$LIST"
