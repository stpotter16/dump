#!/usr/bin/env bash
set -euo pipefail

DB_FILE="${DUMP_DB_PATH}/dump.sqlite"

if [[ -f "$DB_FILE" ]]; then
  echo "Existing database found at ${DB_FILE} ($(stat -c %s "$DB_FILE") bytes) — skipping restore"
  exit 0
fi

echo "No existing database found — restoring from replica"
litestream restore -v -if-replica-exists "$DB_FILE"

if [[ -f "$DB_FILE" ]]; then
  chmod 660 "$DB_FILE"
  rm -f "${DB_FILE}.tmp-wal" "${DB_FILE}.tmp-shm"
  echo "Restore complete, permissions fixed"
else
  echo "No replica found to restore from — starting fresh"
fi
