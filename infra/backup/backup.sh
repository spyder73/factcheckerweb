#!/usr/bin/env bash
# Nightly encrypted Postgres backup → off-host storage.
#
# What it does:
#   1. pg_dump --format=custom from the running postgres container
#   2. encrypt with `age` using a public key (so the box itself can't decrypt)
#   3. upload to Hetzner Storage Box (rsync over SSH) or S3 (s3cmd put)
#   4. rotate: keep daily backups for 14 days, weekly for 8 weeks
#
# Required env (sourced from /etc/alethea/backup.env, mode 600):
#   AGE_RECIPIENTS         comma-separated age public keys (offline-stored private keys)
#   BACKUP_DEST            "sb" (storage box) or "s3"
#   STORAGE_BOX_USER       e.g. u12345-sub1  (for sb)
#   STORAGE_BOX_HOST       e.g. u12345.your-storagebox.de  (for sb)
#   STORAGE_BOX_PATH       e.g. ./alethea-backups  (for sb)
#   AWS_S3_BUCKET          (for s3)
#   AWS_S3_PREFIX          (for s3)
# Optional:
#   PGSERVICE_CONTAINER    docker compose project name + service (default: alethea-postgres-1)
#   BACKUP_TMPDIR          default /var/tmp/alethea-backup
#
# Set up as a systemd timer (see infra/systemd/alethea-backup.{service,timer}).

set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/alethea/backup.env}"
if [[ ! -f "$ENV_FILE" ]]; then
    echo "FATAL: $ENV_FILE not found" >&2
    exit 1
fi
# shellcheck disable=SC1090
source "$ENV_FILE"

: "${AGE_RECIPIENTS:?must be set in $ENV_FILE}"
: "${BACKUP_DEST:?must be set in $ENV_FILE (sb | s3)}"

CONTAINER="${PGSERVICE_CONTAINER:-alethea-postgres-1}"
TMPDIR="${BACKUP_TMPDIR:-/var/tmp/alethea-backup}"
mkdir -p "$TMPDIR"

TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
DUMP="$TMPDIR/alethea-$TIMESTAMP.dump"
SEALED="$DUMP.age"

cleanup() { rm -f "$DUMP" "$SEALED"; }
trap cleanup EXIT

echo "[$(date -u -Iseconds)] starting backup → $SEALED"

# 1. Dump
docker exec -e PGPASSWORD="${POSTGRES_PASSWORD:-}" "$CONTAINER" \
    pg_dump --format=custom --no-owner --no-privileges \
            --dbname="${POSTGRES_DB:-alethea}" --username="${POSTGRES_USER:-alethea}" \
    > "$DUMP"

DUMP_SIZE=$(stat -c%s "$DUMP" 2>/dev/null || stat -f%z "$DUMP")
echo "  dumped $DUMP_SIZE bytes"

# 2. Encrypt — age accepts multiple -r flags from comma-separated input.
IFS=',' read -ra RECIPIENTS_ARR <<< "$AGE_RECIPIENTS"
AGE_ARGS=()
for r in "${RECIPIENTS_ARR[@]}"; do
    AGE_ARGS+=(-r "$r")
done
age "${AGE_ARGS[@]}" -o "$SEALED" "$DUMP"
echo "  encrypted to $SEALED"

# 3. Upload
case "$BACKUP_DEST" in
    sb)
        : "${STORAGE_BOX_USER:?}" "${STORAGE_BOX_HOST:?}" "${STORAGE_BOX_PATH:?}"
        rsync -avz --partial \
            -e "ssh -o StrictHostKeyChecking=accept-new -p 23" \
            "$SEALED" \
            "${STORAGE_BOX_USER}@${STORAGE_BOX_HOST}:${STORAGE_BOX_PATH}/"
        echo "  uploaded to storage box"
        ;;
    s3)
        : "${AWS_S3_BUCKET:?}" "${AWS_S3_PREFIX:?}"
        aws s3 cp "$SEALED" "s3://${AWS_S3_BUCKET}/${AWS_S3_PREFIX}/$(basename "$SEALED")"
        echo "  uploaded to s3"
        ;;
    *)
        echo "FATAL: unknown BACKUP_DEST=$BACKUP_DEST" >&2
        exit 1
        ;;
esac

echo "[$(date -u -Iseconds)] backup complete"

# 4. Rotation — done remotely, since 'this script' has no list permissions
# beyond its own writes. Set up a separate retention job on the storage box
# OR enable S3 lifecycle rules. See infra/backup/RETENTION.md for both.
