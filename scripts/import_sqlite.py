import argparse
import json
import os
import sqlite3
import urllib.request
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--database", required=True)
    parser.add_argument("--host-root", required=True)
    parser.add_argument("--app-root", default="/app/data")
    args = parser.parse_args()
    endpoint = os.environ["AUTO_RENAME_API_URL"].rstrip("/") + "/rpc"
    token = os.environ["AUTO_RENAME_API_TOKEN"]
    host_root = Path(args.host_root).resolve()
    paths = {}
    for directory, _, names in os.walk(host_root):
        for name in names:
            paths.setdefault(name, []).append(Path(directory) / name)
    database = sqlite3.connect(f"file:{args.database}?mode=ro", uri=True)
    database.row_factory = sqlite3.Row
    rows = database.execute("SELECT id, original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at FROM file_records ORDER BY id")
    total = matched = 0
    for row in rows:
        candidates = paths.get(row["new_name"], [])
        status = "renamed" if row["success"] and len(candidates) == 1 else "legacy_missing" if row["success"] else "failed"
        directory = str(Path(args.app_root) / candidates[0].parent.relative_to(host_root)) if len(candidates) == 1 else args.app_root
        record = {
            "legacy_id": row["id"],
            "original_name": row["original_name"] or "",
            "new_name": row["new_name"] or "",
            "file_path": directory,
            "file_size": row["file_size"] or 0,
            "file_mode": row["file_mode"] or "",
            "mod_time": row["mod_time"] or "",
            "success": bool(row["success"]),
            "error_msg": row["error_msg"] or "",
            "renamed_at": row["renamed_at"] or "",
            "status": status,
        }
        request = urllib.request.Request(endpoint, json.dumps({"op": "record.import", "input": record}).encode(), {"Authorization": "Bearer " + token, "Content-Type": "application/json", "User-Agent": "AutoRename-Migrator/1.0"})
        with urllib.request.urlopen(request, timeout=20) as response:
            if response.status != 200:
                raise RuntimeError("D1 import failed")
        total += 1
        matched += status == "renamed"
    print(f"Imported {total} historical records; {matched} matched one current file")


if __name__ == "__main__":
    main()
