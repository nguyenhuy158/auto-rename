#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [[ "$(id -u)" != "0" || "$(hostname -s)" != "pipi" ]]; then
  printf 'Run as root on Pipi\n' >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  printf 'Missing .env\n' >&2
  exit 1
fi

if ! command -v mount.nfs >/dev/null; then
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nfs-common
fi

install -d -m 0755 /mnt/stash
python3 - <<'PY'
from pathlib import Path
import os
import re

values = {}
for line in Path(".env").read_text().splitlines():
    if "=" in line:
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip()

server = values.get("STASH_NFS_SERVER", "")
export = values.get("STASH_NFS_EXPORT", "")
if not re.fullmatch(r"[A-Za-z0-9.-]+", server) or not export.startswith("/") or ".." in Path(export).parts or values.get("STASH_PARENT_DIR") != "/mnt/stash":
    raise SystemExit("Invalid NFS mount configuration in .env")

template = Path("deploy/pipi/mnt-stash.mount").read_text()
destination = Path("/etc/systemd/system/mnt-stash.mount")
destination.write_text(template.replace("@NFS_SERVER@", server).replace("@NFS_EXPORT@", export))
os.chmod(destination, 0o644)
PY
systemctl daemon-reload
systemctl enable --now mnt-stash.mount
mountpoint -q /mnt/stash
test -d /mnt/stash/data
test -d /mnt/stash/quarantine

if [[ "${1:-}" == "--mount-only" ]]; then
  exit 0
fi

for file in .env.runtime settings.json; do
  if [[ ! -f "$file" ]]; then
    printf 'Missing %s\n' "$file" >&2
    exit 1
  fi
done

docker image inspect auto-rename:latest >/dev/null
docker compose config --quiet

install -m 0644 deploy/pipi/auto-rename.service /etc/systemd/system/auto-rename.service
systemctl daemon-reload
systemctl enable --now auto-rename.service
