#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

workdir=$(mktemp -d)
worker_pid=
app_pid=
container_name=
cleanup() {
  if [[ -n "$container_name" ]]; then docker stop "$container_name" >/dev/null 2>&1 || true; fi
  if [[ -n "$app_pid" ]]; then kill "$app_pid" 2>/dev/null || true; fi
  if [[ -n "$worker_pid" ]]; then kill "$worker_pid" 2>/dev/null || true; fi
  wait "$app_pid" "$worker_pid" 2>/dev/null || true
  rm -r "$workdir"
}
trap cleanup EXIT

get_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}

worker_port=$(get_port)
app_port=$(get_port)
while [[ "$app_port" == "$worker_port" ]]; do app_port=$(get_port); done
token=$(openssl rand -hex 16)
source_dir="$workdir/stash/data"
quarantine_dir="$workdir/stash/quarantine"
mkdir -p "$source_dir" "$quarantine_dir" "$workdir/d1"
printf 'first video\n' > "$source_dir/first.mp4"
printf 'duplicate video\n' > "$source_dir/older.mp4"
printf 'duplicate video\n' > "$source_dir/newer.mp4"
python3 - "$source_dir/older.mp4" "$source_dir/newer.mp4" <<'PY'
import os
import sys
import time

now = time.time()
os.utime(sys.argv[1], (now - 120, now - 120))
os.utime(sys.argv[2], (now - 60, now - 60))
PY

app_source="$source_dir"
app_quarantine="$quarantine_dir"
if [[ "${1:-}" == "--docker" ]]; then
  app_source=/app/data
  app_quarantine=/app/quarantine
fi
cat > "$workdir/settings.json" <<EOF
{"dir":"$app_source","quarantine_dir":"$app_quarantine","dry_run":false,"web_port":"$app_port","web_only":false,"extensions":[".mp4"],"cron":false,"cron_expr":"*/30 * * * *","rename_subfolder":true}
EOF

CI=1 npx --yes wrangler@4.152.0 d1 migrations apply db --local --persist-to "$workdir/d1" > "$workdir/migration.log" 2>&1
npx --yes wrangler@4.152.0 dev --local --persist-to "$workdir/d1" --ip 127.0.0.1 --port "$worker_port" --var "API_TOKEN:$token" --show-interactive-dev-session=false > "$workdir/worker.log" 2>&1 &
worker_pid=$!

for _ in {1..60}; do
  code=$(curl -sS -X POST -o /dev/null -w '%{http_code}' --max-time 1 "http://127.0.0.1:$worker_port/rpc" 2>/dev/null || true)
  if [[ "$code" == "401" ]]; then break; fi
  sleep 1
done
if [[ "$code" != "401" ]]; then printf 'Local Worker did not become ready\n' >&2; exit 1; fi

if [[ "${1:-}" == "--docker" ]]; then
  container_name="auto-rename-e2e-$$"
else
  go build -o "$workdir/auto-rename" ./cmd/auto-rename
fi

start_app() {
  if [[ "${1:-}" == "--docker" ]]; then
    docker run --rm --name "$container_name" --network host --user "$(id -u):$(id -g)" --read-only --mount "type=bind,src=$workdir/stash,dst=/app" --mount "type=bind,src=$workdir/settings.json,dst=/srv/app/settings.json,readonly" --tmpfs /tmp --tmpfs /app/config --tmpfs /app/blobs --tmpfs /app/cache --tmpfs /app/generated --tmpfs /app/metadata -e "AUTO_RENAME_API_URL=http://127.0.0.1:$worker_port" -e "AUTO_RENAME_API_TOKEN=$token" auto-rename:latest -config=/srv/app/settings.json >> "$workdir/app.log" 2>&1 &
    app_pid=$!
  else
    AUTO_RENAME_API_URL="http://127.0.0.1:$worker_port" AUTO_RENAME_API_TOKEN="$token" "$workdir/auto-rename" -config="$workdir/settings.json" >> "$workdir/app.log" 2>&1 &
    app_pid=$!
  fi
  for _ in {1..60}; do
    code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 1 "http://127.0.0.1:$app_port/api/health" 2>/dev/null || true)
    if [[ "$code" == "200" ]]; then break; fi
    sleep 1
  done
  if [[ "$code" != "200" ]]; then printf 'Local app did not become ready\n' >&2; exit 1; fi
}

stop_app() {
  if [[ -n "$container_name" ]]; then docker stop "$container_name" >/dev/null; else kill "$app_pid"; fi
  wait "$app_pid" 2>/dev/null || true
  app_pid=
}

run_phase() {
  AUTO_RENAME_E2E_PHASE="$1" AUTO_RENAME_E2E_URL="http://127.0.0.1:$app_port" AUTO_RENAME_E2E_SOURCE="$source_dir" AUTO_RENAME_E2E_QUARANTINE="$quarantine_dir" AUTO_RENAME_E2E_APP_SOURCE="$app_source" AUTO_RENAME_E2E_APP_QUARANTINE="$app_quarantine" AUTO_RENAME_E2E_WORKER="http://127.0.0.1:$worker_port" AUTO_RENAME_E2E_TOKEN="$token" go test ./e2e -count=1 -v
}

start_app "${1:-}"
run_phase flow
run_phase plan_applied
stop_app
start_app "${1:-}"
run_phase reconciled
run_phase plan_ambiguous
stop_app
start_app "${1:-}"
run_phase blocked
