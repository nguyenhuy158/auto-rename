CREATE TABLE IF NOT EXISTS auto_rename_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  original_name TEXT NOT NULL,
  new_name TEXT NOT NULL,
  file_path TEXT NOT NULL,
  file_size INTEGER NOT NULL DEFAULT 0,
  file_mode TEXT NOT NULL DEFAULT '',
  mod_time TEXT NOT NULL DEFAULT '',
  success INTEGER NOT NULL DEFAULT 0,
  error_msg TEXT NOT NULL DEFAULT '',
  renamed_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'renamed',
  restored_at TEXT
);

CREATE INDEX IF NOT EXISTS auto_rename_records_path_idx ON auto_rename_records(file_path, new_name);

CREATE TABLE IF NOT EXISTS auto_rename_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  total_files INTEGER NOT NULL DEFAULT 0,
  processed_files INTEGER NOT NULL DEFAULT 0,
  renamed_files INTEGER NOT NULL DEFAULT 0,
  duplicate_files INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS auto_rename_quarantine (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  original_path TEXT NOT NULL,
  quarantine_path TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  kept_path TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'quarantined',
  created_at TEXT NOT NULL,
  restored_at TEXT
);

CREATE TABLE IF NOT EXISTS auto_rename_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
