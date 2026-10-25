import { useEffect, useMemo, useState } from "react";
import { ArchiveRestore, Clock3, FileImage, FileText, FileVideo, Files, Folder, LayoutDashboard, Search, Settings, X } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { api } from "./api";
import type { paths } from "./generated/api";

type SearchData = paths["/api/search"]["get"]["responses"][200]["content"]["application/json"];
type IndexStatus = SearchData["index"];
type Command = { key: string; label: string; detail: string; url: string; icon: LucideIcon };

const quickLinks: Command[] = [
  { key: "overview", label: "Tổng quan", detail: "Dashboard", url: "/", icon: LayoutDashboard },
  { key: "duplicates", label: "File trùng", detail: "So sánh nội dung", url: "/duplicates", icon: Files },
  { key: "records", label: "Lịch sử đổi tên", detail: "Bản ghi", url: "/records", icon: Clock3 },
  { key: "browse", label: "Duyệt file", detail: "Thư mục nguồn", url: "/browse", icon: Folder },
  { key: "settings", label: "Cấu hình", detail: "Settings", url: "/settings", icon: Settings },
];

function fileURL(path: string) {
  const parts = path.split(/[\\/]/);
  const name = parts.pop() || "";
  const folder = parts.join("/") || ".";
  return `/browse?path=${encodeURIComponent(folder)}&search=${encodeURIComponent(name)}`;
}

export function CommandPalette({ onClose, navigate }: { onClose: () => void; navigate: (path: string) => void }) {
  const [query, setQuery] = useState("");
  const [data, setData] = useState<SearchData | null>(null);
  const [status, setStatus] = useState<IndexStatus | null>(null);
  const [selected, setSelected] = useState(0);
  const [error, setError] = useState("");
  useEffect(() => {
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    api.GET("/api/index/status").then(({ data }) => data && setStatus(data));
    return () => { document.body.style.overflow = overflow; };
  }, []);
  useEffect(() => {
    setSelected(0);
    if (!query.trim()) { setData(null); setError(""); return; }
    const controller = new AbortController();
    const timer = window.setTimeout(() => api.GET("/api/search", { params: { query: { q: query.trim() } }, signal: controller.signal }).then(({ data: result, error: issue }) => {
      if (result) { setData(result); setStatus(result.index); setError(""); }
      else if (issue) setError("Tìm kiếm tạm thời không khả dụng.");
    }).catch(() => {}), 180);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [query]);
  const commands = useMemo(() => {
    if (!query.trim()) return quickLinks;
    if (!data) return [];
    const items: Command[] = [];
    for (const item of data.navigation) items.push({ key: `nav:${item.kind}:${item.title}`, label: item.title, detail: item.subtitle, url: item.url, icon: item.kind === "setting" ? Settings : LayoutDashboard });
    for (const file of data.files) {
      const icon = file.kind === "video" ? FileVideo : file.kind === "image" ? FileImage : file.kind === "pdf" ? FileText : file.kind === "folder" ? Folder : Files;
      const metadata = [file.codec, file.width && file.height ? `${file.width}×${file.height}` : "", file.duration_seconds ? `${Math.round(file.duration_seconds)}s` : ""].filter(Boolean).join(" · ");
      items.push({ key: `file:${file.path}`, label: file.name, detail: file.snippet || metadata || file.path, url: fileURL(file.path), icon });
    }
    for (const group of data.duplicates) items.push({ key: `group:${group.sha256}`, label: `Nhóm trùng · ${group.files} bản`, detail: group.path, url: `/duplicates?search=${encodeURIComponent(group.path)}`, icon: Files });
    for (const record of data.records) items.push({ key: `record:${record.id}`, label: record.original_name, detail: `Đổi tên → ${record.new_name}`, url: `/records?search=${encodeURIComponent(record.original_name)}`, icon: Clock3 });
    for (const run of data.runs) items.push({ key: `run:${run.id}`, label: `Lần quét #${run.id} · ${run.source}`, detail: run.status, url: `/logs?search=${encodeURIComponent(run.source)}`, icon: Clock3 });
    for (const action of data.actions) items.push({ key: `action:${action.id}`, label: "File cách ly", detail: action.original_path, url: `/duplicates?qSearch=${encodeURIComponent(action.original_path)}`, icon: ArchiveRestore });
    return items;
  }, [data, query]);
  function choose(command: Command) { onClose(); navigate(command.url); }
  return <div className="palette-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) onClose(); }}><section role="dialog" aria-modal="true" aria-label="Tìm mọi thứ" className="palette-panel"><div className="palette-input-row"><Search size={20} /><input autoFocus role="combobox" aria-expanded="true" aria-controls="palette-results" aria-activedescendant={commands[selected] ? `palette-${selected}` : undefined} value={query} onChange={event => setQuery(event.target.value)} onKeyDown={event => { if (event.key === "ArrowDown") { event.preventDefault(); setSelected(index => (index + 1) % Math.max(commands.length, 1)); } if (event.key === "ArrowUp") { event.preventDefault(); setSelected(index => (index - 1 + Math.max(commands.length, 1)) % Math.max(commands.length, 1)); } if (event.key === "Enter" && commands[selected]) { event.preventDefault(); choose(commands[selected]); } }} placeholder="Tìm file, PDF, run, cấu hình..." aria-label="Tìm mọi thứ" /><button onClick={onClose} aria-label="Đóng tìm kiếm"><X size={19} /></button></div><div id="palette-results" role="listbox" className="palette-results">{commands.map((command, index) => <button type="button" role="option" aria-selected={index === selected} id={`palette-${index}`} className={index === selected ? "palette-item selected" : "palette-item"} key={command.key} onMouseEnter={() => setSelected(index)} onClick={() => choose(command)}><command.icon size={19} /><span><strong>{command.label}</strong><small>{command.detail}</small></span></button>)}{query && commands.length === 0 && <p className="p-5 text-sm text-fg-muted">{error || "Không có kết quả."}</p>}</div><div className="palette-footer"><span>{status?.running ? `Đang lập chỉ mục ${status.processed_files}/${status.total_files}` : status?.updated_at ? `${status.total_files} mục đã lập chỉ mục` : "Tìm trong ứng dụng"}</span><span>↑↓ chọn · Enter mở · Esc đóng</span></div></section></div>;
}
