const recordColumns = "id, original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at, status, restored_at";
const record = (value) => value && { ...value, success: Boolean(value.success), restored_at: value.restored_at || "" };
const run = (value) => ({ ...value, finished_at: value.finished_at || "" });
const quarantine = (value) => value && ({ ...value, restored_at: value.restored_at || "" });

function filter(input) {
  const clauses = [];
  const values = [];
  if (input.search) {
    clauses.push("(original_name LIKE ? OR new_name LIKE ? OR file_path LIKE ?)");
    values.push(`%${input.search}%`, `%${input.search}%`, `%${input.search}%`);
  }
  if (input.status === "success" || input.status === "failed") {
    clauses.push("success = ?");
    values.push(input.status === "success" ? 1 : 0);
  }
  if (["renamed", "restored", "planned", "restore_pending", "needs_review"].includes(input.status)) {
    clauses.push("status = ?");
    values.push(input.status);
  }
  if (input.dateFrom) {
    clauses.push("datetime(renamed_at) >= datetime(?)");
    values.push(input.dateFrom);
  }
  if (input.dateTo) {
    clauses.push("datetime(renamed_at) <= datetime(?)");
    values.push(input.dateTo);
  }
  return { where: clauses.length ? ` WHERE ${clauses.join(" AND ")}` : "", values };
}

async function execute(db, op, input) {
  switch (op) {
    case "record.insert": {
      const result = await db.prepare("INSERT INTO auto_rename_records (original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)").bind(input.original_name, input.new_name, input.file_path, input.file_size, input.file_mode, input.mod_time, input.success ? 1 : 0, input.error_msg, input.renamed_at, input.status || "renamed").run();
      return { id: result.meta.last_row_id };
    }
    case "record.import": {
      if (!Number.isInteger(input.legacy_id) || input.legacy_id <= 0) throw new Error("Invalid legacy record");
      await db.prepare("INSERT OR IGNORE INTO auto_rename_records (legacy_id, original_name, new_name, file_path, file_size, file_mode, mod_time, success, error_msg, renamed_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)").bind(input.legacy_id, input.original_name, input.new_name, input.file_path, input.file_size || 0, input.file_mode || "", input.mod_time || "", input.success ? 1 : 0, input.error_msg || "", input.renamed_at || "", input.status).run();
      return { ok: true };
    }
    case "record.get":
      return record(await db.prepare(`SELECT ${recordColumns} FROM auto_rename_records WHERE id = ?`).bind(input.id).first());
    case "record.pending": {
      const rows = await db.prepare(`SELECT ${recordColumns} FROM auto_rename_records WHERE status IN ('planned', 'restore_pending', 'needs_review') ORDER BY id`).all();
      return rows.results.map(record);
    }
    case "record.update":
      await db.prepare("UPDATE auto_rename_records SET status = ?, success = ?, error_msg = ?, restored_at = ? WHERE id = ?").bind(input.status, input.success ? 1 : 0, input.error_msg || "", input.restored_at || null, input.id).run();
      return { ok: true };
    case "record.hasOriginal": {
      const row = await db.prepare("SELECT COUNT(*) AS count FROM auto_rename_records WHERE original_name = ? AND status = 'renamed'").bind(input.name).first();
      return { count: row.count };
    }
    case "record.restored": {
      const rows = await db.prepare("SELECT file_path, original_name FROM auto_rename_records WHERE status = 'restored'").all();
      return rows.results;
    }
    case "record.list": {
      const { where, values } = filter(input);
      const limit = Math.min(Math.max(Number(input.limit) || 20, 1), 500);
      const offset = Math.max(Number(input.offset) || 0, 0);
      const columns = { id: "id", name: "original_name", size: "file_size", date: "renamed_at", status: "status" };
      const sort = columns[input.sort] || "id";
      const order = input.order === "asc" ? "ASC" : "DESC";
      const rows = await db.prepare(`SELECT ${recordColumns} FROM auto_rename_records${where} ORDER BY ${sort} ${order}, id DESC LIMIT ? OFFSET ?`).bind(...values, limit, offset).all();
      return rows.results.map(record);
    }
    case "record.count": {
      const { where, values } = filter(input);
      const row = await db.prepare(`SELECT COUNT(*) AS count FROM auto_rename_records${where}`).bind(...values).first();
      return { count: row.count };
    }
    case "record.stats": {
      const row = await db.prepare("SELECT COUNT(*) AS total, SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS successful, SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS failed, SUM(CASE WHEN datetime(renamed_at) > datetime('now', '-24 hours') THEN 1 ELSE 0 END) AS recent FROM auto_rename_records").first();
      return { total: row.total || 0, successful: row.successful || 0, failed: row.failed || 0, recent: row.recent || 0 };
    }
    case "settings.save":
      await db.prepare("INSERT INTO auto_rename_settings (key, value, updated_at) VALUES (?, ?, datetime('now')) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at").bind(input.key, input.value).run();
      return { ok: true };
    case "settings.get":
      return await db.prepare("SELECT value FROM auto_rename_settings WHERE key = ?").bind(input.key).first();
    case "settings.all": {
      const rows = await db.prepare("SELECT key, value FROM auto_rename_settings").all();
      return rows.results;
    }
    case "run.start": {
      const result = await db.prepare("INSERT INTO auto_rename_runs (source, status, started_at) VALUES (?, 'running', ?)").bind(input.source, input.started_at).run();
      return { id: result.meta.last_row_id };
    }
    case "run.update":
      await db.prepare("UPDATE auto_rename_runs SET status = ?, finished_at = ?, total_files = ?, processed_files = ?, renamed_files = ?, duplicate_files = ?, error = ? WHERE id = ?").bind(input.status, input.finished_at || null, input.total_files, input.processed_files, input.renamed_files, input.duplicate_files, input.error || "", input.id).run();
      return { ok: true };
    case "run.recent": {
      const rows = await db.prepare("SELECT * FROM auto_rename_runs ORDER BY id DESC LIMIT 20").all();
      return rows.results.map(run);
    }
    case "run.page": {
      const limit = Math.min(Math.max(Number(input.limit) || 20, 1), 100);
      const offset = Math.max(Number(input.offset) || 0, 0);
      const search = `%${input.search || ""}%`;
      const columns = { id: "id", date: "started_at", files: "processed_files", status: "status" };
      const sort = columns[input.sort] || "id";
      const order = input.order === "asc" ? "ASC" : "DESC";
      const where = input.search ? " WHERE source LIKE ? OR status LIKE ? OR error LIKE ?" : "";
      const values = input.search ? [search, search, search] : [];
      const rows = await db.prepare(`SELECT * FROM auto_rename_runs${where} ORDER BY ${sort} ${order}, id DESC LIMIT ? OFFSET ?`).bind(...values, limit, offset).all();
      const count = await db.prepare(`SELECT COUNT(*) AS count FROM auto_rename_runs${where}`).bind(...values).first();
      return { rows: rows.results.map(run), total: count.count };
    }
    case "quarantine.insert": {
      const result = await db.prepare("INSERT INTO auto_rename_quarantine (original_path, quarantine_path, sha256, kept_path, created_at, status) VALUES (?, ?, ?, ?, ?, ?)").bind(input.original_path, input.quarantine_path, input.sha256, input.kept_path, input.created_at, input.status || "planned").run();
      return { id: result.meta.last_row_id };
    }
    case "quarantine.get":
      return quarantine(await db.prepare("SELECT * FROM auto_rename_quarantine WHERE id = ?").bind(input.id).first());
    case "quarantine.pending": {
      const rows = await db.prepare("SELECT * FROM auto_rename_quarantine WHERE status IN ('planned', 'restore_pending', 'needs_review') ORDER BY id").all();
      return rows.results.map(quarantine);
    }
    case "quarantine.list": {
      const rows = await db.prepare("SELECT * FROM auto_rename_quarantine ORDER BY id DESC LIMIT 200").all();
      return rows.results.map(quarantine);
    }
    case "quarantine.page": {
      const limit = Math.min(Math.max(Number(input.limit) || 20, 1), 100);
      const offset = Math.max(Number(input.offset) || 0, 0);
      const search = `%${input.search || ""}%`;
      const columns = { id: "id", date: "created_at", path: "original_path", status: "status" };
      const sort = columns[input.sort] || "id";
      const order = input.order === "asc" ? "ASC" : "DESC";
      const where = input.search ? " WHERE original_path LIKE ? OR kept_path LIKE ? OR status LIKE ?" : "";
      const values = input.search ? [search, search, search] : [];
      const rows = await db.prepare(`SELECT * FROM auto_rename_quarantine${where} ORDER BY ${sort} ${order}, id DESC LIMIT ? OFFSET ?`).bind(...values, limit, offset).all();
      const count = await db.prepare(`SELECT COUNT(*) AS count FROM auto_rename_quarantine${where}`).bind(...values).first();
      return { rows: rows.results.map(quarantine), total: count.count };
    }
    case "search.all": {
      const q = `%${String(input.query || "").slice(0, 128)}%`;
      const records = await db.prepare(`SELECT ${recordColumns} FROM auto_rename_records WHERE original_name LIKE ? OR new_name LIKE ? OR file_path LIKE ? ORDER BY id DESC LIMIT 8`).bind(q, q, q).all();
      const runs = await db.prepare("SELECT * FROM auto_rename_runs WHERE source LIKE ? OR status LIKE ? OR error LIKE ? ORDER BY id DESC LIMIT 5").bind(q, q, q).all();
      const actions = await db.prepare("SELECT * FROM auto_rename_quarantine WHERE original_path LIKE ? OR kept_path LIKE ? OR status LIKE ? ORDER BY id DESC LIMIT 5").bind(q, q, q).all();
      return { records: records.results.map(record), runs: runs.results.map(run), actions: actions.results.map(quarantine) };
    }
    case "quarantine.restore":
      await db.prepare("UPDATE auto_rename_quarantine SET status = 'restored', restored_at = ? WHERE id = ? AND status = 'restore_pending'").bind(input.restored_at, input.id).run();
      return { ok: true };
    case "quarantine.update":
      await db.prepare("UPDATE auto_rename_quarantine SET status = ? WHERE id = ?").bind(input.status, input.id).run();
      return { ok: true };
    default:
      throw new Error("Unknown operation");
  }
}

export default {
  async fetch(request, env) {
    if (request.method !== "POST" || new URL(request.url).pathname !== "/rpc") {
      return new Response("Not found", { status: 404 });
    }
    if (!env.API_TOKEN || request.headers.get("Authorization") !== `Bearer ${env.API_TOKEN}`) {
      return new Response("Unauthorized", { status: 401 });
    }
    try {
      const { op, input = {} } = await request.json();
      const result = await execute(env.DB, op, input);
      return Response.json({ result });
    } catch (error) {
      return Response.json({ error: error instanceof Error ? error.message : "Request failed" }, { status: 400 });
    }
  }
};
