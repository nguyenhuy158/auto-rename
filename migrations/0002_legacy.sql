ALTER TABLE auto_rename_records ADD COLUMN legacy_id INTEGER;
CREATE UNIQUE INDEX IF NOT EXISTS auto_rename_records_legacy_idx ON auto_rename_records(legacy_id);
