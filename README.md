# Auto Rename

Auto Rename scans a configured video folder, gives unrenamed files UUID names, finds files with identical content, and keeps an operation history in Cloudflare D1. The MVP runs in Docker on the NixOS VM. [Issue #4](https://github.com/nguyenhuy158/auto-rename/issues/4) tracks moving the service to Pipi while the files remain on NixOS.

## MVP behavior

- The dashboard shows scan progress. `/duplicates` starts a rename scan immediately, scans for duplicate content, displays the oldest file to keep, and lists records for rollback.
- A duplicate scan groups regular video files by size and then verifies matching content with SHA-256. Selected duplicates, one entire group, or all groups can be moved to quarantine while keeping the oldest file in each group. The UI lists exact paths and asks for confirmation before moving them.
- Quarantined files can be restored to their original path. A renamed file can be restored to its original name. An existing target is never overwritten.
- D1 records the planned operation before a file move. On startup, the app reconciles interrupted operations using the files present at each path. Ambiguous operations block further changes for manual review.
- Only extensions listed in `settings.json` are scanned. The default list contains video extensions; technical files such as `.frag`, `.m3u8`, and `.xml` are excluded. The service does not rename files on startup.

## Browse, preview, and search

The file browser shows JPEG thumbnails for images and videos and opens an image viewer or video player. Video formats that the browser cannot decode use temporary HLS segments produced by FFmpeg. At most two streams run at once; inactive streams and their temporary segments expire after three minutes. Streaming and thumbnails only read the source folder.

Ctrl+K searches pages, active settings, file names and paths, duplicate groups, D1 records, runs, quarantine history, media metadata, and extracted PDF text. The file index stays in memory and is rebuilt after a service restart. Settings shows indexing progress and lets you refresh it. PDF text is not stored in D1. Lists support search, sorting, and pagination; the browser's search applies within the current folder.

## Storage and runtime

The Go process runs on NixOS and is the only component that touches local files. A small Cloudflare Worker exposes authenticated operations to the shared D1 database named `db`. All app tables, indexes, and the migration table use the `auto_rename_` prefix. The Worker token lives in a Worker secret and the VM's ignored `.env.runtime` file; it is never sent to the browser.

The Docker UI is bound to `127.0.0.1:8768` on NixOS. It is not a public web service. Use an SSH tunnel to view it from another computer. Publishing the GitHub repository does not publish the UI or the files.

The default configuration enables `dry_run` and disables cron. A manual rename scan in this mode records a preview run without changing files. To apply renames, set `dry_run` to `false` in `settings.json` and restart the container. To schedule later scans, also set `cron` to `true` and choose `cron_expr`.

## Configuration

Copy `.env.runtime.example` to `.env.runtime` and set `AUTO_RENAME_API_URL` to the deployed Worker URL and `AUTO_RENAME_API_TOKEN` to the matching Worker secret. Never commit `.env.runtime` or paste its values into an issue.

Edit `settings.json` for the source directory, quarantine directory, video extensions, dry-run mode, cron, and web port. The settings page displays the active values and is read-only. The NixOS Compose file mounts the Stash video directory at `/app/data`, the repo's ignored `quarantine/` directory at `/app/quarantine`, and `settings.json` read-only. Quarantine and source must be on the same filesystem.

Set `STASH_DATA_DIR` in an ignored `.env` file using `.env.example` as a template. The default video extensions are `.mp4`, `.mov`, `.mkv`, `.avi`, `.webm`, `.m4v`, and `.wmv`. Directory symlinks are not traversed by scans. File browser paths are restricted to the configured source directory.

## D1 and Worker

The D1 database ID is configured in `wrangler.jsonc`. The table migration is isolated from other apps using the `auto_rename_migrations` table. Review the SQL before applying it to the shared D1 database:

```sh
cf d1 migrations apply 6eb9cfce-f6f3-4476-8e46-7c8e954d93c1 --dir migrations --table auto_rename_migrations
```

`wrangler.jsonc` describes the Worker entry point and D1 binding. Deploy with `wrangler deploy`, then set its `API_TOKEN` secret. The Worker only accepts authenticated POST requests to `/rpc`; operation names map to prepared SQL statements against `auto_rename_` tables.

The legacy SQLite file remains untouched. To copy its history into D1 after deployment, run `scripts/import_sqlite.py` with the SQLite file path, host source directory, and `/app/data` as the app root. The importer reads SQLite in read-only mode and uses a unique legacy ID so re-running it does not duplicate records. It marks unmatched files as historical entries that cannot be rolled back automatically.

## Build and checks

```sh
go test ./...
cd web && pnpm install --frozen-lockfile && pnpm build && cd ..
make build
docker compose up -d
```

The React UI uses the published `ui-kit` design tokens and project-owned components. [openapi.yaml](openapi.yaml) is the browser API contract. `pnpm build` generates `web/src/generated/api.d.ts` from the contract, checks TypeScript, and builds the UI. The running service serves the contract at `/openapi.yaml`. Update the spec when changing Go response or request JSON and regenerate the types. `make build` builds both the UI and Go server in a local Docker image; it does not push to a registry. GitHub Actions checks Go, the frontend contract and build, and the Docker image. Browser E2E belongs to the next milestone.

For a local D1 test, apply the migrations with `wrangler d1 migrations apply db --local`, start `wrangler dev` with a temporary `API_TOKEN`, and run the Go app against that local Worker with sample files outside the Stash directory.

## Safety and recovery

An operation can span D1 and a local filesystem, so there is no single transaction covering both. Each rename or quarantine move first records a planned operation in D1, moves the file without replacing an existing destination, then updates D1. On restart, an interrupted operation is reconciled from the source and destination paths. If both or neither path exists, the app blocks changes and requires manual review.

Quarantine moves files; it does not permanently delete them. The MVP has no purge operation. A duplicate scan result is kept in memory, so scan again after a restart or after changing files. Always review the listed source paths before confirming a real operation against Stash.

Current project status and acceptance evidence belong in [the MVP milestone](https://github.com/nguyenhuy158/auto-rename/milestone/1). Do not treat a passing build as proof that D1, the VM container, or the real Stash directory has been verified.
