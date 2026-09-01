# systemd setup: dump + litestream

Runs the `dump` binary as a dedicated unprivileged user, sharing the
existing box-wide `litestream.service` (already running unprivileged as
its own `litestream` user for `biodata`) via a group shared with dump's
data directory.

This replaces the Fly/Docker deployment (`Dockerfile`, `docker-entrypoint`,
`fly.toml`). Those files are left in place, inactive, and can be removed
once this deployment has run cleanly for a while and Fly is confirmed
decommissioned.

## 1. Build the binary

```bash
./dev-scripts/build-release-server.sh   # produces ./release/dump
```

## 2. Create the dump user and shared group

The `litestream` user already exists on this box (set up for `biodata`) —
no need to create it again, just add it to dump's new group.

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin dump

sudo groupadd dump-data
sudo usermod -aG dump-data dump
sudo usermod -aG dump-data litestream
```

## 3. Create directories and set permissions

```bash
sudo mkdir -p /opt/dump /etc/dump /var/lib/dump

sudo chown dump:dump-data /var/lib/dump
sudo chmod 2770 /var/lib/dump   # setgid: new files inherit the dump-data group
```

## 4. Install the dump binary and unit

```bash
sudo cp release/dump /opt/dump/dump
sudo cp systemd/dump.service /etc/systemd/system/dump.service

sudo cp systemd/dump.env.example /etc/dump/dump.env
sudo chown root:dump-data /etc/dump/dump.env
sudo chmod 640 /etc/dump/dump.env
sudo $EDITOR /etc/dump/dump.env   # fill DUMP_PASSPHRASE, DUMP_SESSION_ENV_KEY, VOYAGE_API_KEY
```

`/etc/dump/dump.env` should contain:

```
DUMP_DB_PATH=/var/lib/dump
DUMP_SIMILARITY_THRESHOLD=0.5
PORT=8081
DUMP_SESSION_ENV_KEY=<generated>
DUMP_PASSPHRASE=<generated>
VOYAGE_API_KEY=<generated>
```

## 5. litestream's packaged unit override — already in place

`litestream.service`'s drop-in at `/etc/systemd/system/litestream.service.d/override.conf`
(set up for `biodata`) already runs it as the unprivileged `litestream`
user with `After=`/`Wants=network-online.target`. Nothing to change here —
supplementary group membership (step 2) is enough for it to pick up access
to `/var/lib/dump`; the override doesn't need dump-specific edits.

## 6. Add dump's entry to the shared litestream.yml

`litestream.yml` in this repo defines dump's replica config as one `dbs:`
entry. **Merge it into the existing on-box `/etc/litestream.yml`** (which
already has biodata's entry) — don't overwrite the file, and don't hand-edit
the merged copy going forward; always regenerate it by combining the
current `dbs:` entries from each app's repo.

Dump's entry has no `access-key-id`/`secret-access-key` of its own — this
litestream instance uses one shared S3 identity across every bucket it
replicates (biodata's and dump's alike), so credentials live in exactly
one place in the merged file: the existing top-level global block already
set up for biodata. Don't add a second one for dump — litestream only
supports a single global credentials block per config file, and a second
one (even if scoped per-replica) would just be a redundant thing to keep
in sync for no benefit, since the identity is the same either way. If
buckets across apps on this box are ever meant to use *different* S3
identities, that's the point where per-replica credential overrides
would earn their place — not before.

```bash
sudo $EDITOR /etc/litestream.yml   # add dump's dbs: entry from this repo's litestream.yml
sudo chown root:litestream /etc/litestream.yml
sudo chmod 640 /etc/litestream.yml
```

## 7. Add dump's bucket vars to litestream's env file

Dump's `litestream.yml` pulls `${LITESTREAM_BUCKET}`,
`${LITESTREAM_ENDPOINT}`, and `${DUMP_DB_PATH}` — no credential vars, since
those come from the one shared global block covered in step 6. These names
don't collide with biodata's (`BIOTRAK_BUCKET`, `BIOTRAK_ENDPOINT`,
`BIOTRAK_PATH`) — but that's incidental, not a designed guarantee. If a
third app ever joins this shared litestream instance, check its var names
against both of these before merging.

Add dump's vars to the existing shared env file rather than creating a new
one — it's a plain `KEY=VALUE` file, safe to edit in place (unlike a
systemd unit drop-in, which has to be rewritten whole):

```bash
sudo $EDITOR /etc/litestream/litestream.env
```

Add:

```
DUMP_DB_PATH=/var/lib/dump
LITESTREAM_BUCKET=
LITESTREAM_ENDPOINT=
```

(Permissions on this file are already `root:litestream 640` from biodata's
setup — no change needed.)

## 8. Restore data (manual, first boot / disaster recovery only)

`litestream.service` only runs `litestream replicate` — no automatic
restore-on-start. Restore only matters once per box (initial provisioning
or actual disaster recovery), so it isn't wired into `ExecStartPre=`;
baking it in would make every ordinary restart/reboot pay the cost of
`litestream restore`'s generation/WAL enumeration, and — since
`dump.service` only soft-depends on litestream (`Wants=`, not `Requires=`)
— that cost wouldn't even protect dump from starting with an empty
database anyway. Run this by hand instead, once, before starting
`dump.service` for the first time (skip entirely for a genuinely fresh
deployment with no existing backup — dump will just create a new
database):

```bash
sudo -u litestream bash -c '
  set -a; source /etc/litestream/litestream.env; set +a
  litestream restore -v -if-replica-exists "${DUMP_DB_PATH}/dump.sqlite"
'
ls -la /var/lib/dump/
```

litestream writes restored files as `-rw-------` (owner `litestream` only)
regardless of umask — this is not the same gap `UMask=0007` on
`dump.service` fixes (that only covers files the `dump` process itself
creates, not litestream's). Left alone, dump can't open the db it just
restored. Fix the mode before starting `dump.service`, and clean up
litestream's leftover restore scratch files:

```bash
sudo chmod 660 /var/lib/dump/dump.sqlite
sudo rm -f /var/lib/dump/dump.sqlite.tmp-wal /var/lib/dump/dump.sqlite.tmp-shm
ls -la /var/lib/dump/
```

`systemd/litestream-restore-if-missing.sh` in this repo does the same
"only restore if the local file doesn't already exist" check plus the
`chmod`/cleanup above automatically — copy it to the box and run it
directly if you'd rather not do those steps by hand:

```bash
sudo cp systemd/litestream-restore-if-missing.sh /usr/local/bin/dump-litestream-restore-if-missing.sh
sudo chown root:root /usr/local/bin/dump-litestream-restore-if-missing.sh
sudo chmod 755 /usr/local/bin/dump-litestream-restore-if-missing.sh

sudo -u litestream bash -c '
  set -a; source /etc/litestream/litestream.env; set +a
  /usr/local/bin/dump-litestream-restore-if-missing.sh
'
```

If a restore ever seems to hang, time it directly rather than guessing —
`restore` looks up its replica config by matching its `DB_PATH` argument
against `litestream.yml`'s configured `dbs[].path`, so pass the real path
and use `-o` to redirect actual output elsewhere to test safely without
touching the real db file:

```bash
sudo -u litestream bash -c '
  set -a; source /etc/litestream/litestream.env; set +a
  time litestream restore -o /tmp/dump-restore-test.sqlite -if-replica-exists "${DUMP_DB_PATH}/dump.sqlite"
'
ls -la /tmp/dump-restore-test.sqlite
sudo rm -f /tmp/dump-restore-test.sqlite
```

## 9. Start everything

```bash
sudo systemctl daemon-reload

sudo systemctl enable --now dump.service
sudo systemctl restart litestream   # picks up dump's merged config + new env vars

sudo systemctl status dump.service litestream
sudo systemctl cat litestream       # confirm the override still shows User=litestream, Group=litestream
journalctl -u dump.service -u litestream -f
```

## 10. Verify

```bash
sudo -u litestream test -r /var/lib/dump/dump.sqlite && echo "litestream can read the db"
sudo -u litestream test -w /var/lib/dump/dump.sqlite && echo "litestream can write to the db"
sudo -u dump test -w /var/lib/dump && echo "dump can write to the data dir"

curl -i http://localhost:8081/
```

If the second check fails (or `journalctl -u litestream` shows "attempt to
write to a readonly database"), see the UMask note below — it's almost
always the db/WAL files being group-unwritable, not the directory.

Confirm graceful shutdown works (validates `KillSignal=SIGINT` in
`dump.service`, since the app only handles SIGINT, not SIGTERM):

```bash
sudo systemctl kill -s SIGINT dump
journalctl -u dump -n 20    # should show "Received termination signal. Shutting down"
```

## Fallback to root if litestream can't start

```bash
sudo rm -rf /etc/systemd/system/litestream.service.d
sudo systemctl daemon-reload
sudo systemctl restart litestream
```

Root already has full access to `/var/lib/dump`, so no group changes need
reverting — this affects biodata too, since it's the same shared litestream
instance. `dump.service`'s own hardening is a separate unit and unaffected
either way.

## Notes

- **Group-writable db files (UMask)**: the setgid bit on `/var/lib/dump`
  (`chmod 2770` in step 3) makes new files inherit the `dump-data` *group*,
  but doesn't change the *mode* bits — with systemd's default `umask 0022`,
  files dump creates come out `rw-r--r--`, so litestream (a group member,
  not the owner) can read `dump.sqlite` but not write to it, causing
  "attempt to write to a readonly database" in `journalctl -u litestream`.
  `dump.service` sets `UMask=0007` so any file dump creates (the db, and
  the `-wal`/`-shm` files SQLite creates alongside it in WAL mode) comes
  out `rw-rw----` instead. If you hit this on an existing install, fix the
  already-created files directly: `sudo chmod 660
  /var/lib/dump/dump.sqlite*`. Don't chmod the *directory* to `660` —
  directories need the execute bit to be traversable, so that would break
  both processes' access entirely; `2770` from step 3 is already correct.
- **litestream.yml has no credentials in it**: unlike biodata's
  `litestream.yml` (whose var names predate a `biotrak` → `biodata` rename
  and diverge from `.env.example`), dump's `litestream.yml` var names
  already match `.env.example` — no naming cleanup needed there. What *was*
  needed was dropping `access-key-id`/`secret-access-key` from dump's
  `dbs:` entry entirely: this litestream instance uses one shared S3
  identity for every bucket it replicates, so credentials belong in
  exactly one place — the existing global block in the merged
  `/etc/litestream.yml` — not duplicated (even correctly, per-replica) in
  each app's own entry.
- **SIGINT vs SIGTERM**: `cmd/server/main.go` only handles `os.Interrupt`
  (SIGINT), not SIGTERM. `dump.service` sets `KillSignal=SIGINT` so
  `systemctl stop`/`restart` still trigger the app's graceful
  `server.Shutdown()` instead of a hard kill.
- **Startup ordering**: `dump.service` sets `After=litestream.service` and
  `Wants=litestream.service` — soft dependency, so a litestream failure
  doesn't take dump down with it, and restore being manual (step 8) means
  there's no automatic-restore completion for a hard `Requires=` to
  actually wait on.
- **`snapshot-interval` and restore time**: litestream 0.3.13 (pinned in
  the `Dockerfile`, same major version biodata runs) has no compaction —
  WAL segments accumulate between snapshots rather than getting merged, so
  a long `snapshot-interval` with a short `sync-interval` can build up many
  small objects that slow down future restores. Dump's `litestream.yml`
  currently uses `snapshot-interval: 24h` / `sync-interval: 1s`, looser
  than biodata's `1h` (which was tuned down after biodata's write volume
  made that a real problem). Leaving dump at `24h` for now since its write
  volume hasn't been characterized — revisit if restores start feeling
  slow as usage grows, same as the reasoning behind biodata's value.
- **Port default unchanged in code**: `PORT` defaults to `8081` in
  `dump.env.example` only. `cmd/server/main.go`'s fallback (used only when
  `PORT` is unset) stays `8080`, deliberately not changed, to keep local
  dev behavior as-is — the unit always sets `PORT` explicitly via
  `EnvironmentFile=`, so the code fallback never applies in production.
