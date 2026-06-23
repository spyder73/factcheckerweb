# Deployment guide

This is the runbook for putting Alethea on a single VPS with Caddy + Docker.
Target hardware: Hetzner CX22 (4GB RAM, €4.51/mo) for v1; jump to CCX13
(8GB, €15/mo) once Plus is live and the pipeline is processing real load.

**This guide assumes you've already**:

- Bought a domain and pointed an A record at the VPS public IP.
- Got root SSH access to the VPS.
- Set up Stripe (see `.env.example` for the variables) — or you're OK
  skipping billing for the first deploy and adding it later.

---

## 1. Provision the box

```sh
# As root on the VPS, fresh Debian 12 / Ubuntu 24.04:
apt-get update && apt-get upgrade -y
apt-get install -y curl git ufw rsync age openssh-client

# Firewall: only SSH + HTTP + HTTPS.
ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 443/udp  # HTTP/3
ufw enable

# Docker via the official convenience script (faster than packaging round-trips).
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

# Non-root user for compose ops.
useradd -m -s /bin/bash alethea
usermod -aG docker alethea
```

## 2. Get the code

```sh
mkdir -p /opt/alethea && chown alethea: /opt/alethea
sudo -u alethea git clone https://github.com/YOUR-ORG/alethea.git /opt/alethea
cd /opt/alethea
```

## 3. Secrets

`/etc/alethea/.env` holds production secrets. Mode 600, owned by root.

```sh
mkdir -p /etc/alethea && chmod 700 /etc/alethea
cp /opt/alethea/.env.example /etc/alethea/.env
chmod 600 /etc/alethea/.env
$EDITOR /etc/alethea/.env
```

Mandatory fields (in addition to anything you already filled for dev):

```
# Public surface
ALETHEA_DOMAIN=alethea.example
ALETHEA_ADMIN_EMAIL=ops@alethea.example   # for Let's Encrypt + abuse
BASE_URL=https://alethea.example
VITE_API_URL=https://alethea.example
ALLOWED_ORIGINS=https://alethea.example
COOKIE_SECURE=true
TRUSTED_PROXIES=127.0.0.1/32,::1/128,172.16.0.0/12

# Secrets — generate fresh ones for prod!
POSTGRES_PASSWORD=$(openssl rand -base64 32)
BYOK_MASTER_KEY=$(openssl rand -base64 32)

# AI provider
AI_PROVIDER=openrouter
OPENROUTER_API_KEY=sk-or-...
OPENROUTER_MODEL=anthropic/claude-4.6-sonnet

# Search providers
BRAVE_SEARCH_API_KEY=...
TAVILY_API_KEY=...           # fallback; can leave blank for v1

# Billing (or leave blank to ship without Plus tier)
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_PLUS_MONTHLY=price_...
STRIPE_PRICE_PLUS_YEARLY=price_...
```

> **NEVER commit /etc/alethea/.env or any production secret. The
> repository's .env.example has placeholder values only.**

## 4. First-time boot

```sh
cd /opt/alethea
docker compose -f infra/docker-compose.prod.yml --env-file /etc/alethea/.env build
docker compose -f infra/docker-compose.prod.yml --env-file /etc/alethea/.env up -d
docker compose -f infra/docker-compose.prod.yml --env-file /etc/alethea/.env logs -f caddy
```

Watch the Caddy logs. Within ~30 seconds you should see Let's Encrypt
issue a cert. If you see `cannot solve challenge` errors, DNS isn't
pointing here yet — fix and `docker compose restart caddy`.

Verify:

```sh
curl -sS https://alethea.example/health      # → {"status":"healthy"}
curl -sS https://alethea.example/version     # → {"name":"alethea-api",...}
curl -sSI https://alethea.example/           # check security headers
```

## 5. Make it survive reboots

```sh
cp /opt/alethea/infra/systemd/alethea.service /etc/systemd/system/
cp /opt/alethea/infra/systemd/alethea-backup.service /etc/systemd/system/
cp /opt/alethea/infra/systemd/alethea-backup.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now alethea.service
systemctl enable --now alethea-backup.timer
```

## 6. Backups

`/etc/alethea/backup.env` holds the destination + age recipients:

```
AGE_RECIPIENTS=age1q9...,age1zr...  # PUBLIC keys; PRIVATE keys stay off-box
BACKUP_DEST=sb
STORAGE_BOX_USER=u12345-sub1
STORAGE_BOX_HOST=u12345.your-storagebox.de
STORAGE_BOX_PATH=./alethea-backups
POSTGRES_PASSWORD=...               # same value as the main .env
```

Generate the age key pair OFF-box:

```sh
age-keygen -o ~/.age/alethea-backup.txt
# Move ~/.age/alethea-backup.txt OFF this machine (1Password / hardware key).
# The public line goes in AGE_RECIPIENTS above.
```

Test it manually:

```sh
sudo -u alethea-backup /opt/alethea/infra/backup/backup.sh
```

Then schedule via the timer (already done in step 5).

> Read `infra/backup/RETENTION.md` for rotation + monthly restore drills.
> An untested backup is not a backup.

## 7. Stripe webhook

In the Stripe dashboard → Developers → Webhooks → Add endpoint:

- URL: `https://alethea.example/api/billing/webhook`
- Events to send:
  - `checkout.session.completed`
  - `customer.subscription.updated`
  - `customer.subscription.deleted`
  - `invoice.payment_failed`
- Copy the signing secret (`whsec_...`) into `STRIPE_WEBHOOK_SECRET` in
  `/etc/alethea/.env`, then `systemctl restart alethea`.

Test with the Stripe CLI:

```sh
stripe trigger checkout.session.completed
# Check the audit_log table for a "billing.plan_change" row.
```

## 8. Monitoring

Either:

- **Off-box** (recommended for a one-person team): point Better Stack /
  Pingdom / etc. at `https://alethea.example/health` with a 60s interval.
- **On-box**: `docker compose -f infra/monitoring/uptime-kuma.yml up -d`.

Wire at least one notification channel (Telegram is the lowest-friction).

## 9. Day-2 operations

| Task | Command |
| --- | --- |
| Tail API logs | `docker compose -f infra/docker-compose.prod.yml logs -f api` |
| Rebuild after `git pull` | `systemctl restart alethea` (it pulls + recreates) |
| psql shell | `docker compose exec postgres psql -U alethea alethea` |
| Migration ad hoc | (already runs on API boot) — just restart the API |
| Caddy reload only | `docker compose exec caddy caddy reload --config /etc/caddy/Caddyfile` |
| Disk / inode check | `df -h && df -i` (postgres-data volume is the one to watch) |
| Image cleanup | `docker image prune -a -f` (weekly cron is fine) |

## 10. Updating

```sh
cd /opt/alethea
git pull
systemctl restart alethea
```

If the update changes the Caddyfile or any volumes, `systemctl restart`
recreates only what changed — Postgres data is preserved via the named
volume.

Roll back: `git checkout PREV_SHA && systemctl restart alethea`.

## 11. Disaster recovery dry-run

Once a quarter:

1. Boot a throwaway VPS.
2. `scp` the latest `alethea-*.dump.age` over.
3. `age -d -i alethea-backup.txt alethea-*.dump.age > restore.dump`.
4. `pg_restore --create --dbname=postgres restore.dump`.
5. Smoke-test by booting the API against the restored DB.
6. Document the restore wall-clock time.

If step 6 is > 30 minutes, the backup format is wrong (likely the dump is
plain-SQL and not custom-format — fix `backup.sh`).
