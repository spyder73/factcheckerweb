# Backup retention policy

`backup.sh` only uploads; it deliberately does NOT delete old files (least-
privilege — the box can't wipe its own backups). Rotation is configured on
the destination instead.

## Option A — Hetzner Storage Box

SSH to the Storage Box (port 23) and add this to `~/.bashrc` or run via
cron from your laptop:

```sh
# Keep daily backups for 14 days, weekly for 8 weeks.
# Storage Box doesn't expose a real shell — use the `purge` action via SFTP
# from a controller host (your laptop, NOT the production VPS).
ssh -p 23 u12345-sub1@u12345.your-storagebox.de << 'EOF'
  cd alethea-backups
  ls -1 alethea-*.dump.age | sort | head -n -14 | xargs -r rm -- 2>/dev/null
  # Then keep Sundays for 8 weeks (anything older than 56 days that isn't
  # already kept by the daily window above is fair game)
EOF
```

Simpler: just keep 30 days; storage is cheap.

## Option B — S3 lifecycle

Create a lifecycle rule on the bucket. AWS CLI:

```sh
aws s3api put-bucket-lifecycle-configuration --bucket alethea-backups --lifecycle-configuration '{
  "Rules": [
    {
      "Id": "alethea-rotation",
      "Status": "Enabled",
      "Filter": {"Prefix": "daily/"},
      "Transitions": [{"Days": 30, "StorageClass": "GLACIER"}],
      "Expiration": {"Days": 180}
    }
  ]
}'
```

## Verifying backups

The encryption key (age private key) must be stored OFFLINE — not on the
production VPS. To verify a backup actually decrypts + restores:

```sh
# On a workstation that holds the age private key:
scp alethea-20260622T000000Z.dump.age workstation:
age -d -i ~/.age/alethea-backup.txt alethea-20260622T000000Z.dump.age \
  > alethea-restore.dump
# Restore into a temporary DB to verify integrity (NEVER restore over prod):
docker run --rm -d --name pg-verify -e POSTGRES_PASSWORD=verify postgres:16-alpine
docker cp alethea-restore.dump pg-verify:/tmp/
docker exec pg-verify pg_restore --no-owner --no-privileges \
  -U postgres -d postgres /tmp/alethea-restore.dump
```

Schedule a verify-test at LEAST monthly. An untested backup is not a backup.
