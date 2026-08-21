# Migrating `dump` off Fly.io to systemd

This replaces the Fly/Docker deployment (`Dockerfile`, `docker-entrypoint`,
`fly.toml`) with a directly-run Go binary managed by systemd. Litestream
replication is handled by the existing `litestream.service` already running
on the target host (see step 5) rather than a dump-specific litestream
unit. `dump.service` soft-depends on it (`After=`/`Wants=`, not
`Requires=`) so a litestream hiccup doesn't take dump down with it.

The old Fly/Docker files are left in place and untouched — they're inactive
but not deleted. Their env var naming (e.g. `fly.toml`'s `[env]` block) can
drift from `dump.env.example` below without that being a bug, since only
the systemd path is live going forward. Remove them once this deployment
has run cleanly for a while and Fly is confirmed decommissioned.

## 1. User and group

```
groupadd --system dump
useradd --system --no-create-home --shell /usr/sbin/nologin -g dump dump

# Give your existing litestream user access to dump's data directory
usermod -aG dump <your-litestream-username>
```

## 2. Directories

```
install -d -o root -g root -m 755 /opt/dump
install -d -o dump -g dump -m 2770 /var/lib/dump   # setgid: new files inherit group `dump`
install -d -o root -g dump -m 750 /etc/dump
```

## 3. Install the binaries

Build with `./dev-scripts/build-release-server.sh` (produces
`./release/dump`), then copy to the host:

```
install -o root -g root -m 755 release/dump /opt/dump/dump
```

`cmd/backfill-embeddings` is a one-off CLI tool, not a long-running
service — build and drop it in `/opt/dump/` too (`go build -o
/opt/dump/backfill-embeddings cmd/backfill-embeddings/main.go`) for manual
runs; it doesn't get a unit.

## 4. Secrets

Copy `systemd/dump.env.example` to `/etc/dump/dump.env` on the host, fill
in real values, and lock it down:

```
chown root:dump /etc/dump/dump.env
chmod 640 /etc/dump/dump.env
```

`DUMP_SESSION_ENV_KEY` is generated with `make secrets/hmac`.

## 5. Litestream integration

`litestream.yml` in this repo defines dump's replica config and expects
`DUMP_DB_PATH` and the `LITESTREAM_*` vars in its process environment (it
interpolates `${DUMP_DB_PATH}/dump.sqlite` for the DB path and
`${LITESTREAM_ACCESS_KEY_ID}` etc. for the S3-compatible replica).

Since `litestream.service` is already running on the target host under its
own user:

- [ ] Add this repo's `dbs:` entry (from `litestream.yml`) into whatever
      config your existing litestream instance reads — merge, don't
      overwrite, if it's already replicating other apps' databases.
- [ ] Make sure `DUMP_DB_PATH=/var/lib/dump` and the `LITESTREAM_ACCESS_KEY_ID`
      / `LITESTREAM_SECRET_ACCESS_KEY` / `LITESTREAM_BUCKET` /
      `LITESTREAM_ENDPOINT` values for this DB are available in whatever
      env file that litestream service already reads.
- [ ] Confirm the litestream user (now in group `dump`) can read/write
      `/var/lib/dump` — the directory is `2770 dump:dump`.

Redeploy `litestream.yml` from this repo to wherever your litestream setup
reads its config from on every change — don't hand-edit the on-host copy,
or the two will drift.

## 6. Restore (first provision / disaster recovery only — not automatic)

Do this once, before starting `dump.service` for the first time on a fresh
host, using your existing litestream user:

```
sudo -u <litestream-user> litestream restore -if-replica-exists /var/lib/dump/dump.sqlite
```

Litestream restores commonly come back owner-only regardless of umask —
fix ownership/mode afterward so `dump` can actually use the file:

```
chown dump:dump /var/lib/dump/dump.sqlite*
chmod 660 /var/lib/dump/dump.sqlite*
```

This is deliberately not wired into `ExecStartPre=` — restore only matters
on first provisioning or real disaster recovery, so it shouldn't be on the
path of every ordinary restart/reboot.

## 7. Install and start the unit

```
cp systemd/dump.service /etc/systemd/system/dump.service
systemctl daemon-reload
systemctl enable --now dump.service
```

## 8. Verify

```
journalctl -u dump -f
curl -i http://localhost:8081/
```

Confirm graceful shutdown actually happens (validates `KillSignal=SIGINT`
in `dump.service`, since the app only handles SIGINT, not SIGTERM):

```
systemctl kill -s SIGINT dump
journalctl -u dump -n 20    # should show "Received termination signal. Shutting down"
```

## Notes / things that were guessed, not read from the box

- Install paths (`/opt/dump`, `/var/lib/dump`, `/etc/dump`) — no existing
  box convention was given; adjust if your host uses something else.
- `TimeoutStopSec=10s` / `RestartSec=2s` in `dump.service` match the
  convention from your existing `biodata.service`; the app itself exits
  within ~5s of SIGINT.
- The default `PORT` is `8081` in `dump.env.example` only — deliberately
  *not* changed in `cmd/server/main.go`'s fallback (still `8080`), to keep
  local dev behavior unchanged. The unit always sets `PORT` explicitly via
  `EnvironmentFile=`, so the code fallback never actually applies in
  production.
