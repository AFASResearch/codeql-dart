#!/usr/bin/env bash
set -euo pipefail

LIST="$1"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
EXE="$SCRIPT_DIR/../extractor/dart-extractor"

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
  # Skip empty lines
  [ -z "$FILE" ] && continue
  echo "[index-files] Indexing file: $FILE"
  "$EXE" --index "$FILE" || exit $?
done < "$LIST"
