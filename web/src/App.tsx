import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { Activity, ArchiveRestore, Check, ChevronLeft, ChevronRight, Clock3, Ellipsis, Files, FolderOpen, LayoutDashboard, ListFilter, Moon, Play, RefreshCw, Search, Settings, ShieldCheck, Sun, Trash2, X } from "lucide-react";
import { api, unwrap } from "./api";
import { Badge, Button, Card, Metric } from "./ui";
import { MediaThumbnail } from "./MediaThumbnail";
import { CommandPalette } from "./CommandPalette";
import type { ViewableMedia } from "./MediaViewer";
import type { components, paths } from "./generated/api";

type ScanRun = components["schemas"]["ScanRun"];
type DuplicateGroup = components["schemas"]["DuplicateGroup"];
type FileRecord = components["schemas"]["FileRecord"];
type QuarantineAction = components["schemas"]["QuarantineAction"];
type SettingsData = components["schemas"]["Settings"];
type Stats = components["schemas"]["Stats"];
type BrowserData = { current_path: string; entries: components["schemas"]["FileEntry"][]; breadcrumb: components["schemas"]["Breadcrumb"][]; base_dir: string; total: number; page: number; pageSize: number; sort: string; order: string };
type ConfirmAction = { title: string; description: string; paths: string[]; execute: () => Promise<unknown>; danger?: boolean };
const MediaViewer = lazy(() => import("./MediaViewer").then(module => ({ default: module.MediaViewer })));

const navigation = [
  { path: "/", label: "Tổng quan", icon: LayoutDashboard },
  { path: "/duplicates", label: "Trùng lặp", icon: Files },
  { path: "/records", label: "Lịch sử đổi tên", icon: Clock3 },
  { path: "/logs", label: "Nhật ký chạy", icon: Activity },
  { path: "/browse", label: "Duyệt thư mục", icon: FolderOpen },
  { path: "/settings", label: "Cấu hình", icon: Settings },
];
const mobileNavigation = [navigation[0], navigation[1], navigation[4], navigation[2]];

const number = (value: number) => new Intl.NumberFormat("vi-VN").format(value);
const size = (bytes: number) => bytes < 1024 ? `${bytes} B` : bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB`;
const time = (value: string) => value ? new Date(value).toLocaleString("vi-VN") : "—";
const basename = (path: string) => path.split("/").filter(Boolean).at(-1) || path;
const videoMime = (path: string) => ({ mp4: "video/mp4", m4v: "video/mp4", webm: "video/webm", mov: "video/quicktime", mkv: "video/x-matroska", avi: "video/x-msvideo", wmv: "video/x-ms-wmv" })[path.split(".").at(-1)?.toLowerCase() as "mp4" | "m4v" | "webm" | "mov" | "mkv" | "avi" | "wmv"] || "video/mp4";
const statusTone = (status: string) => status === "completed" || status === "restored" || status === "renamed" || status === "quarantined" || status === "success" ? "success" : status === "running" || status === "planned" ? "warning" : status === "failed" || status === "error" ? "danger" : "neutral";

function useLocation() {
  const [location, setLocation] = useState(window.location.pathname + window.location.search);
  useEffect(() => {
    const change = () => setLocation(window.location.pathname + window.location.search);
    window.addEventListener("popstate", change);
    return () => window.removeEventListener("popstate", change);
  }, []);
  return [location.split("?")[0], (next: string) => { const target = new URL(next, window.location.origin); const location = target.pathname + target.search; window.history.pushState({}, "", location); setLocation(location); window.scrollTo(0, 0); }, location] as const;
}

export function App() {
  const [path, navigate, location] = useLocation();
  const [dark, setDark] = useState(localStorage.getItem("auto-rename-theme") === "dark");
  const [settings, setSettings] = useState<SettingsData | null>(null);
  const [run, setRun] = useState<ScanRun | null>(null);
  const [notice, setNotice] = useState("");
  const [confirm, setConfirm] = useState<ConfirmAction | null>(null);
  const [busy, setBusy] = useState(false);
  const [revision, setRevision] = useState(0);
  const [moreOpen, setMoreOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);

  const refresh = useCallback(() => setRevision(value => value + 1), []);
  const loadRun = useCallback(() => api.GET("/api/cron/status").then(({ data }) => data && setRun(data)).catch(() => {}), []);
  useEffect(() => { document.documentElement.classList.toggle("dark", dark); localStorage.setItem("auto-rename-theme", dark ? "dark" : "light"); }, [dark]);
  useEffect(() => { api.GET("/api/settings").then(({ data }) => data && setSettings(data)).catch(() => {}); }, []);
  useEffect(() => { void loadRun(); const timer = window.setInterval(loadRun, 4000); return () => window.clearInterval(timer); }, [loadRun]);
  useEffect(() => {
    const keydown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") { event.preventDefault(); setPaletteOpen(value => !value); }
      else if (event.key === "Escape") setPaletteOpen(false);
    };
    window.addEventListener("keydown", keydown);
    return () => window.removeEventListener("keydown", keydown);
  }, []);

  async function perform(action: () => Promise<unknown>, success: string) {
    setBusy(true);
    setNotice("");
    try { await action(); setNotice(success); refresh(); void loadRun(); }
    catch (error) { setNotice(error instanceof Error ? error.message : "Thao tác thất bại"); }
    finally { setBusy(false); setConfirm(null); }
  }

  const title = navigation.find(item => item.path === path)?.label || "Tổng quan";
  return <div className="app-shell">
    <aside className="sidebar">
      <a className="brand" href="/" onClick={event => { event.preventDefault(); navigate("/"); }}><span className="brand-icon"><Files size={21} /></span><span><strong>Auto Rename</strong><small>File operations</small></span></a>
      <p className="sidebar-label">KHÔNG GIAN LÀM VIỆC</p>
      <nav className="desktop-nav" aria-label="Điều hướng chính">{navigation.map(item => <a key={item.path} href={item.path} className={path === item.path ? "nav-link active" : "nav-link"} onClick={event => { event.preventDefault(); navigate(item.path); }}><item.icon size={18} /><span>{item.label}</span></a>)}</nav>
      <nav className="mobile-nav" aria-label="Điều hướng mobile">{mobileNavigation.map(item => <a key={item.path} href={item.path} className={path === item.path ? "nav-link active" : "nav-link"} onClick={event => { event.preventDefault(); setMoreOpen(false); navigate(item.path); }} aria-label={item.label}><item.icon size={20} /><span>{item.path === "/" ? "Tổng quan" : item.path === "/browse" ? "File" : item.path === "/records" ? "Lịch sử" : "Trùng lặp"}</span></a>)}<button className={moreOpen || path === "/logs" || path === "/settings" ? "nav-link active" : "nav-link"} onClick={() => setMoreOpen(value => !value)} aria-label="Thêm mục" aria-expanded={moreOpen}><Ellipsis size={20} /><span>Thêm</span></button></nav>
      <div className="sidebar-bottom"><div className="flex items-center gap-2"><ShieldCheck size={17} className="text-success-text" /><span className="text-sm font-medium">Không gian an toàn</span></div><p className="mt-2 text-xs leading-relaxed text-fg-muted">File trùng được chuyển vào cách ly và có thể khôi phục.</p></div>
    </aside>
    <div className="main-wrap">
      <header className="topbar"><div className="min-w-0"><p className="text-xs text-fg-muted">Auto Rename / {title}</p><h1 className="mt-1 truncate text-xl font-semibold tracking-tight">{title}</h1></div><div className="flex items-center gap-2"><Badge tone={settings?.dry_run ? "warning" : "success"}>{settings?.dry_run ? "Chế độ xem trước" : "Áp dụng thật"}</Badge><Button variant="ghost" size="sm" aria-label="Tìm mọi thứ (Ctrl+K)" title="Tìm mọi thứ · Ctrl+K" onClick={() => setPaletteOpen(true)}><Search size={18} /></Button><Button variant="ghost" size="sm" aria-label={dark ? "Bật giao diện sáng" : "Bật giao diện tối"} onClick={() => setDark(value => !value)}>{dark ? <Sun size={18} /> : <Moon size={18} />}</Button></div></header>
      <main className="page-content">{notice && <div role="status" className="notice"><span>{notice}</span><button onClick={() => setNotice("")} aria-label="Đóng thông báo">×</button></div>}
        {path === "/" && <Dashboard revision={revision} run={run} settings={settings} busy={busy} perform={perform} navigate={navigate} />}
        {path === "/duplicates" && <Duplicates key={location} revision={revision} run={run} busy={busy} perform={perform} setConfirm={setConfirm} />}
        {path === "/records" && <Records key={location} revision={revision} busy={busy} setConfirm={setConfirm} />}
        {path === "/logs" && <Logs key={location} revision={revision} run={run} />}
        {path === "/browse" && <Browser key={location} />}
        {path === "/settings" && <SettingsPage settings={settings} />}
      </main>
    </div>
    {paletteOpen && <CommandPalette onClose={() => setPaletteOpen(false)} navigate={navigate} />}
    {moreOpen && <div className="mobile-menu-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) setMoreOpen(false); }}><div role="dialog" aria-modal="true" aria-label="Điều hướng thêm" className="mobile-menu-panel"><div className="flex items-center justify-between"><strong>Thêm</strong><Button variant="ghost" size="sm" aria-label="Đóng menu" onClick={() => setMoreOpen(false)}><X size={20} /></Button></div>{navigation.filter(item => item.path === "/logs" || item.path === "/settings").map(item => <a key={item.path} href={item.path} className="mobile-menu-link" onClick={event => { event.preventDefault(); setMoreOpen(false); navigate(item.path); }}><item.icon size={20} /><span>{item.label}</span><ChevronRight size={16} /></a>)}</div></div>}
    {confirm && <div className="modal-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) setConfirm(null); }}><section role="dialog" aria-modal="true" aria-labelledby="confirm-title" className="modal-panel"><div className="flex items-start gap-3"><div className="rounded-ui bg-danger-soft p-2 text-danger-text"><Trash2 size={20} /></div><div><h2 id="confirm-title" className="text-lg font-semibold">{confirm.title}</h2><p className="mt-1 text-sm text-fg-muted">{confirm.description}</p></div></div>{confirm.paths.length > 0 && <div className="mt-4"><p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">Đường dẫn sẽ xử lý ({confirm.paths.length})</p><ul className="path-list mt-2">{confirm.paths.map(item => <li key={item} title={item}>{item}</li>)}</ul></div>}<div className="mt-6 flex justify-end gap-2"><Button variant="outline" onClick={() => setConfirm(null)}>Hủy</Button><Button variant={confirm.danger ? "danger" : "primary"} disabled={busy} onClick={() => void perform(confirm.execute, "Đã gửi yêu cầu. Theo dõi tiến trình ở Tổng quan.")}>Xác nhận</Button></div></section></div>}
  </div>;
}

function SectionTitle({ title, detail, action }: { title: string; detail?: string; action?: React.ReactNode }) {
  return <div className="section-title"><div><h2 className="text-lg font-semibold tracking-tight">{title}</h2>{detail && <p className="mt-1 text-sm text-fg-muted">{detail}</p>}</div>{action}</div>;
}

function Progress({ run }: { run: ScanRun | null }) {
  if (!run) return <p className="text-sm text-fg-muted">Chưa có lần quét nào.</p>;
  const percent = run.total_files ? Math.min(100, Math.round(run.processed_files / run.total_files * 100)) : 0;
  return <div><div className="flex flex-wrap items-center justify-between gap-2"><div className="flex items-center gap-2"><Badge tone={statusTone(run.status)}>{run.status}</Badge><span className="text-sm text-fg-muted">{run.source}</span></div><span className="text-sm tabular text-fg-muted">{number(run.processed_files)} / {number(run.total_files)}</span></div><div className="progress-track mt-4"><div className="progress-fill" style={{ width: `${percent}%` }} /></div><div className="mt-3 flex flex-wrap gap-x-5 gap-y-1 text-xs text-fg-muted"><span>Bắt đầu: {time(run.started_at)}</span><span>Đã đổi tên: {number(run.renamed_files)}</span><span>File trùng: {number(run.duplicate_files)}</span></div>{run.error && <p className="mt-3 text-sm text-danger-text">{run.error}</p>}</div>;
}

function Dashboard({ revision, run, settings, busy, perform, navigate }: { revision: number; run: ScanRun | null; settings: SettingsData | null; busy: boolean; perform: (action: () => Promise<unknown>, message: string) => Promise<void>; navigate: (path: string) => void }) {
  const [stats, setStats] = useState<Stats | null>(null);
  const [groups, setGroups] = useState<DuplicateGroup[]>([]);
  useEffect(() => { api.GET("/api/stats").then(({ data }) => data && setStats(data)); api.GET("/api/duplicates").then(({ data }) => data && setGroups(data.groups || [])); }, [revision, run?.status]);
  const duplicateCount = groups.reduce((sum, group) => sum + Math.max(0, group.files.length - 1), 0);
  return <><div className="hero"><div><div className="flex items-center gap-2"><span className="hero-dot" /> <span className="text-sm font-medium">Trung tâm quản lý file</span></div><h2 className="mt-3 text-2xl font-semibold tracking-tight sm:text-3xl">File gọn gàng, thao tác an toàn.</h2><p className="mt-2 max-w-2xl text-sm leading-relaxed text-fg-muted">Quét video, phát hiện nội dung trùng và theo dõi từng thay đổi. Dữ liệu gốc luôn có đường khôi phục.</p></div><Button disabled={busy || run?.status === "running"} onClick={() => void perform(() => unwrap(api.POST("/api/cron/trigger")), settings?.dry_run ? "Đã bắt đầu quét xem trước." : "Đã bắt đầu quét đổi tên.")}><Play size={16} /> Quét đổi tên ngay</Button></div>
    <div className="metric-grid"><Metric label="Bản ghi đổi tên" value={number(stats?.total_records || 0)} icon={<Files size={19} />} /><Metric label="Đổi tên thành công" value={number(stats?.successful_renames || 0)} icon={<Check size={19} />} /><Metric label="Nhóm trùng" value={number(groups.length)} icon={<ListFilter size={19} />} /><Metric label="File trùng có thể cách ly" value={number(duplicateCount)} icon={<ArchiveRestore size={19} />} /></div>
    <div className="content-grid"><Card><SectionTitle title="Tiến trình hiện tại" detail="Tự cập nhật mỗi 4 giây" /><div className="mt-5"><Progress run={run} /></div></Card><Card><SectionTitle title="Thao tác nhanh" detail="Bắt đầu từ một lần quét hoặc xem dữ liệu đã có" /><div className="mt-4 grid gap-2"><Button variant="outline" className="justify-start" onClick={() => navigate("/duplicates")}><Files size={17} /> Kiểm tra file trùng</Button><Button variant="outline" className="justify-start" onClick={() => navigate("/records")}><Clock3 size={17} /> Xem lịch sử đổi tên</Button><Button variant="outline" className="justify-start" onClick={() => navigate("/browse")}><FolderOpen size={17} /> Mở thư mục nguồn</Button></div></Card></div>
    <Card><SectionTitle title="Thao tác gần đây" action={<Button variant="ghost" size="sm" onClick={() => navigate("/records")}>Xem tất cả <ChevronRight size={16} /></Button>} /><div className="mt-4">{stats?.recent_operations?.length ? <div className="table-scroll"><table><thead><tr><th>File gốc</th><th>Tên mới</th><th>Trạng thái</th><th>Thời gian</th></tr></thead><tbody>{stats.recent_operations.slice(0, 5).map(record => <tr key={record.id}><td title={record.original_name}>{record.original_name}</td><td title={record.new_name}>{record.new_name}</td><td><Badge tone={statusTone(record.status)}>{record.status}</Badge></td><td>{time(record.renamed_at)}</td></tr>)}</tbody></table></div> : <Empty text="Chưa có thao tác đổi tên." />}</div></Card></>;
}

function Duplicates({ revision, run, busy, perform, setConfirm }: { revision: number; run: ScanRun | null; busy: boolean; perform: (action: () => Promise<unknown>, message: string) => Promise<void>; setConfirm: (value: ConfirmAction | null) => void }) {
  const [groups, setGroups] = useState<DuplicateGroup[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const initial = new URLSearchParams(window.location.search);
  const [search, setSearch] = useState(initial.get("search") || "");
  const [page, setPage] = useState(Math.max(1, Number(initial.get("page")) || 1));
  const [sort, setSort] = useState(initial.get("sort") || "oldest");
  const [order, setOrder] = useState<"asc" | "desc">(initial.get("order") === "desc" ? "desc" : "asc");
  const [viewer, setViewer] = useState<{ items: ViewableMedia[]; index: number } | null>(null);
  useEffect(() => { api.GET("/api/duplicates").then(({ data }) => data && setGroups(data.groups || [])); }, [revision, run?.status]);
  useEffect(() => {
    const query = new URLSearchParams(window.location.search);
    for (const key of ["search", "page", "sort", "order"]) query.delete(key);
    if (search) query.set("search", search);
    if (page > 1) query.set("page", String(page));
    if (sort !== "oldest") query.set("sort", sort);
    if (order !== "asc") query.set("order", order);
    window.history.replaceState({}, "", `/duplicates${query.size ? `?${query}` : ""}`);
  }, [search, page, sort, order]);
  const visible = useMemo(() => groups.filter(group => group.files.some(file => file.path.toLowerCase().includes(search.toLowerCase()))).sort((a, b) => {
    let value = 0;
    if (sort === "size") value = a.size - b.size;
    else if (sort === "files") value = a.files.length - b.files.length;
    else if (sort === "name") value = a.files[0]?.path.localeCompare(b.files[0]?.path || "") || 0;
    else value = (a.files[0]?.modified_at || "").localeCompare(b.files[0]?.modified_at || "");
    return order === "desc" ? -value : value;
  }), [groups, search, sort, order]);
  const displayed = visible.slice((page - 1) * 10, page * 10);
  const extra = groups.flatMap(group => group.files.slice(1).map(file => file.path));
  const toggle = (path: string) => setSelected(old => {
    if (old.includes(path)) return old.filter(item => item !== path);
    const group = groups.find(item => item.files.some(file => file.path === path));
    return [...old.filter(item => group?.files.some(file => file.path === item)), path];
  });
  function confirmGroup(group: DuplicateGroup, paths: string[]) {
    setConfirm({ title: "Chuyển file trùng vào cách ly?", description: `Giữ bản cũ nhất: ${group.files[0]?.path || "—"}. File được chuyển vào cách ly, có thể khôi phục.`, paths, danger: true, execute: () => unwrap(api.POST("/api/duplicates/quarantine", { body: { sha256: group.sha256, paths, all: false, confirm: true } })) });
  }
  return <><SectionTitle title="File trùng nội dung" detail="So sánh SHA-256; bản cũ nhất được giữ lại trong từng nhóm." action={<Button disabled={busy || run?.status === "running"} onClick={() => void perform(() => unwrap(api.POST("/api/duplicates/scan")), "Đã bắt đầu quét file trùng.")}><RefreshCw size={16} /> Quét lại</Button>} /><Card className="mt-5"><Progress run={run} /></Card><div className="mt-6 flex flex-wrap items-center justify-between gap-3"><div className="flex flex-wrap gap-2"><Badge tone="primary">{groups.length} nhóm</Badge><Badge tone="warning">{extra.length} file trùng</Badge><Badge>{selected.length} đã chọn</Badge></div><div className="flex flex-wrap gap-2">{selected.length > 0 && <Button variant="outline" disabled={busy} onClick={() => { const chosen = groups.find(group => group.files.slice(1).some(file => selected.includes(file.path))); if (chosen) confirmGroup(chosen, selected); }}><Trash2 size={16} /> Cách ly đã chọn</Button>}{extra.length > 0 && <Button variant="danger" disabled={busy} onClick={() => setConfirm({ title: "Cách ly tất cả file trùng?", description: "Giữ bản cũ nhất trong mỗi nhóm. Các file bên dưới được chuyển vào cách ly và có thể khôi phục.", paths: extra, danger: true, execute: () => unwrap(api.POST("/api/duplicates/quarantine-all", { body: { confirm: true } })) })}><Trash2 size={16} /> Cách ly tất cả</Button>}</div></div>
    <div className="browser-controls mt-4"><label className="search-field"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm file trùng..." aria-label="Tìm file trùng" /></label><select className="select-field" value={sort} onChange={event => { setSort(event.target.value); setPage(1); }} aria-label="Sắp xếp nhóm trùng"><option value="oldest">Cũ nhất</option><option value="size">Dung lượng</option><option value="files">Số bản</option><option value="name">Tên file</option></select><Button variant="outline" aria-label="Đổi chiều sắp xếp" onClick={() => setOrder(value => value === "asc" ? "desc" : "asc")}>{order === "asc" ? "↑" : "↓"}</Button></div>
    <div className="mt-4 grid gap-3">{displayed.length ? displayed.map(group => <Card key={group.sha256}><div className="flex flex-wrap items-start justify-between gap-3"><div><div className="flex items-center gap-2"><Badge tone="primary">{group.files.length} bản</Badge><span className="text-xs text-fg-muted">{size(group.size)} mỗi file</span></div><p className="mt-2 text-sm font-medium" title={group.files[0]?.path}>{basename(group.files[0]?.path || "")}</p><p className="mt-1 text-xs text-fg-muted">SHA-256: {group.sha256.slice(0, 18)}…</p></div><Button variant="outline" size="sm" disabled={busy} onClick={() => confirmGroup(group, group.files.slice(1).map(file => file.path))}>Cách ly nhóm</Button></div><div className="mt-4 divide-y divide-border-subtle rounded-ui border border-border-subtle">{group.files.map((file, index) => <div key={file.path} className="flex min-w-0 items-center gap-3 px-3 py-2.5"><input type="checkbox" checked={index > 0 && selected.includes(file.path)} disabled={index === 0} onChange={() => toggle(file.path)} aria-label={`Chọn ${file.path}`} /><button type="button" className="duplicate-preview" onClick={() => setViewer({ items: group.files.map(item => ({ path: item.path, name: basename(item.path), is_image: false, is_video: true, mime_type: videoMime(item.path) })), index })} aria-label={`Xem ${file.path}`}><MediaThumbnail path={file.path} type="video" /></button><div className="min-w-0 flex-1"><p className="truncate text-sm" title={file.path}>{file.path}</p><p className="text-xs text-fg-muted">{time(file.modified_at)}</p></div>{index === 0 ? <Badge tone="success">Giữ lại</Badge> : <Badge tone="warning">Bản trùng</Badge>}</div>)}</div></Card>) : <Card><Empty text={search ? "Không tìm thấy file phù hợp." : "Chưa có dữ liệu trùng. Chọn Quét lại để bắt đầu."} /></Card>}</div><Pager page={page} total={visible.length} pageSize={10} setPage={setPage} />
    <QuarantineList revision={revision} busy={busy} setConfirm={setConfirm} />{viewer && <Suspense fallback={null}><MediaViewer items={viewer.items} index={viewer.index} onClose={() => setViewer(null)} /></Suspense>}</>;
}

function QuarantineList({ revision, busy, setConfirm }: { revision: number; busy: boolean; setConfirm: (value: ConfirmAction | null) => void }) {
  const initial = new URLSearchParams(window.location.search);
  const [actions, setActions] = useState<QuarantineAction[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(Math.max(1, Number(initial.get("qPage")) || 1));
  const [search, setSearch] = useState(initial.get("qSearch") || "");
  const [sort, setSort] = useState(initial.get("qSort") || "id");
  const [order, setOrder] = useState<"asc" | "desc">(initial.get("qOrder") === "asc" ? "asc" : "desc");
  useEffect(() => {
    const query = new URLSearchParams(window.location.search);
    for (const key of ["qPage", "qSearch", "qSort", "qOrder"]) query.delete(key);
    if (page > 1) query.set("qPage", String(page));
    if (search) query.set("qSearch", search);
    if (sort !== "id") query.set("qSort", sort);
    if (order !== "desc") query.set("qOrder", order);
    window.history.replaceState({}, "", `/duplicates${query.size ? `?${query}` : ""}`);
    const timer = window.setTimeout(() => api.GET("/api/quarantine", { params: { query: { page, pageSize: 20, search, sort: sort as "id" | "date" | "path" | "status", order } } }).then(({ data }) => {
      if (data) { setActions(data.actions || []); setTotal(data.total); }
    }), search ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [revision, page, search, sort, order]);
  function restore(action: QuarantineAction) {
    setConfirm({ title: "Khôi phục file?", description: "File được chuyển về vị trí ban đầu nếu đường dẫn chưa bị chiếm.", paths: [action.original_path], execute: () => unwrap(api.POST("/api/quarantine/restore", { body: { id: action.id } })) });
  }
  return <div className="mt-10"><SectionTitle title="File trong cách ly" detail={`${number(total)} file · khôi phục về đường dẫn gốc khi còn trống`} /><div className="browser-controls mt-4"><label className="search-field"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm file cách ly..." aria-label="Tìm file cách ly" /></label><select className="select-field" value={sort} onChange={event => { setSort(event.target.value); setPage(1); }} aria-label="Sắp xếp cách ly"><option value="id">Mới nhất</option><option value="date">Thời gian</option><option value="path">Đường dẫn</option><option value="status">Trạng thái</option></select><Button variant="outline" aria-label="Đổi chiều sắp xếp" onClick={() => setOrder(value => value === "asc" ? "desc" : "asc")}>{order === "asc" ? "↑" : "↓"}</Button></div><Card className="mt-4"><div className="desktop-data-table">{actions.length ? <div className="table-scroll"><table><thead><tr><th>File gốc</th><th>Trạng thái</th><th>Thời gian</th><th></th></tr></thead><tbody>{actions.map(action => <tr key={action.id}><td title={action.original_path}>{action.original_path}</td><td><Badge tone={statusTone(action.status)}>{action.status}</Badge></td><td>{time(action.created_at)}</td><td>{action.status !== "restored" && <Button variant="outline" size="sm" disabled={busy} onClick={() => restore(action)}><ArchiveRestore size={15} /> Khôi phục</Button>}</td></tr>)}</tbody></table></div> : <Empty text="Chưa có file trong cách ly." />}</div><div className="mobile-data-list">{actions.map(action => <div className="mobile-data-item" key={action.id}><div className="flex items-center justify-between gap-2"><Badge tone={statusTone(action.status)}>{action.status}</Badge><span className="text-xs text-fg-muted">{time(action.created_at)}</span></div><p className="mt-2 break-all text-sm font-medium">{action.original_path}</p>{action.status !== "restored" && <Button variant="outline" className="mt-3" size="sm" disabled={busy} onClick={() => restore(action)}><ArchiveRestore size={15} /> Khôi phục</Button>}</div>)}{actions.length === 0 && <Empty text="Chưa có file trong cách ly." />}</div></Card><Pager page={page} total={total} pageSize={20} setPage={setPage} /></div>;
}

function Records({ revision, busy, setConfirm }: { revision: number; busy: boolean; setConfirm: (value: ConfirmAction | null) => void }) {
  const initial = new URLSearchParams(window.location.search);
  const [records, setRecords] = useState<FileRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(Math.max(1, Number(initial.get("page")) || 1));
  const [search, setSearch] = useState(initial.get("search") || "");
  const [status, setStatus] = useState(initial.get("status") || "");
  const [sort, setSort] = useState(initial.get("sort") || "id");
  const [order, setOrder] = useState<"asc" | "desc">(initial.get("order") === "asc" ? "asc" : "desc");
  useEffect(() => {
    const query = new URLSearchParams();
    if (page > 1) query.set("page", String(page));
    if (search) query.set("search", search);
    if (status) query.set("status", status);
    if (sort !== "id") query.set("sort", sort);
    if (order !== "desc") query.set("order", order);
    window.history.replaceState({}, "", `/records${query.size ? `?${query}` : ""}`);
    const timer = window.setTimeout(() => api.GET("/api/records", { params: { query: { page, pageSize: 20, search, status, sort: sort as "id" | "name" | "size" | "date" | "status", order } } }).then(({ data }) => {
      if (data) { setRecords(data.records || []); setTotal(data.total); }
    }), search ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [revision, page, search, status, sort, order]);
  function restore(record: FileRecord) {
    setConfirm({ title: "Khôi phục tên gốc?", description: `Đổi tên ${record.new_name} về ${record.original_name}. Đường dẫn đích phải còn trống.`, paths: [record.file_path], execute: () => unwrap(api.POST("/api/records/restore", { body: { id: record.id } })) });
  }
  return <>
    <SectionTitle title="Lịch sử đổi tên" detail={`${number(total)} bản ghi · có thể hoàn tác về tên gốc`} />
    <div className="list-controls mt-5"><label className="search-field"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm tên hoặc đường dẫn..." aria-label="Tìm bản ghi" /></label><select className="select-field" value={status} onChange={event => { setStatus(event.target.value); setPage(1); }} aria-label="Lọc trạng thái"><option value="">Mọi trạng thái</option><option value="renamed">Đã đổi tên</option><option value="restored">Đã khôi phục</option><option value="failed">Lỗi</option></select><select className="select-field" value={sort} onChange={event => { setSort(event.target.value); setPage(1); }} aria-label="Sắp xếp lịch sử"><option value="id">Mới nhất</option><option value="name">Tên</option><option value="size">Dung lượng</option><option value="date">Thời gian</option><option value="status">Trạng thái</option></select><Button variant="outline" aria-label="Đổi chiều sắp xếp" onClick={() => setOrder(value => value === "asc" ? "desc" : "asc")}>{order === "asc" ? "↑" : "↓"}</Button></div>
    <Card className="mt-4"><div className="desktop-data-table">{records.length ? <div className="table-scroll"><table><thead><tr><th>Tên gốc</th><th>Tên mới</th><th>Trạng thái</th><th>Thời gian</th><th></th></tr></thead><tbody>{records.map(record => <tr key={record.id}><td title={record.original_name}>{record.original_name}</td><td title={record.new_name}>{record.new_name}</td><td><Badge tone={statusTone(record.status)}>{record.status}</Badge></td><td>{time(record.renamed_at)}</td><td>{record.success && record.status === "renamed" && <Button variant="outline" size="sm" disabled={busy} onClick={() => restore(record)}><ArchiveRestore size={15} /> Hoàn tác</Button>}</td></tr>)}</tbody></table></div> : <Empty text="Không có bản ghi phù hợp." />}</div><div className="mobile-data-list">{records.map(record => <div className="mobile-data-item" key={record.id}><div className="flex items-center justify-between gap-2"><Badge tone={statusTone(record.status)}>{record.status}</Badge><span className="text-xs text-fg-muted">{time(record.renamed_at)}</span></div><p className="mt-2 break-all text-sm font-semibold">{record.original_name}</p><p className="mt-1 break-all text-xs text-fg-muted">→ {record.new_name}</p>{record.success && record.status === "renamed" && <Button className="mt-3" variant="outline" size="sm" disabled={busy} onClick={() => restore(record)}><ArchiveRestore size={15} /> Hoàn tác</Button>}</div>)}{records.length === 0 && <Empty text="Không có bản ghi phù hợp." />}</div></Card>
    <Pager page={page} total={total} pageSize={20} setPage={setPage} />
  </>;
}

function Logs({ revision, run }: { revision: number; run: ScanRun | null }) {
  const initial = new URLSearchParams(window.location.search);
  const [runs, setRuns] = useState<ScanRun[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(Math.max(1, Number(initial.get("page")) || 1));
  const [search, setSearch] = useState(initial.get("search") || "");
  const [sort, setSort] = useState(initial.get("sort") || "id");
  const [order, setOrder] = useState<"asc" | "desc">(initial.get("order") === "asc" ? "asc" : "desc");
  useEffect(() => {
    const query = new URLSearchParams();
    if (page > 1) query.set("page", String(page));
    if (search) query.set("search", search);
    if (sort !== "id") query.set("sort", sort);
    if (order !== "desc") query.set("order", order);
    window.history.replaceState({}, "", `/logs${query.size ? `?${query}` : ""}`);
    const timer = window.setTimeout(() => api.GET("/api/runs", { params: { query: { page, pageSize: 20, search, sort: sort as "id" | "date" | "files" | "status", order } } }).then(({ data }) => {
      if (data) { setRuns(data.recent || []); setTotal(data.total); }
    }), search ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [revision, run?.status, page, search, sort, order]);
  return <><SectionTitle title="Nhật ký chạy" detail={`${number(total)} lần quét`} /><Card className="mt-5"><Progress run={run} /></Card><div className="browser-controls mt-5"><label className="search-field"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm lần quét..." aria-label="Tìm nhật ký" /></label><select className="select-field" value={sort} onChange={event => { setSort(event.target.value); setPage(1); }} aria-label="Sắp xếp nhật ký"><option value="id">Mới nhất</option><option value="date">Thời gian</option><option value="files">Số file</option><option value="status">Trạng thái</option></select><Button variant="outline" aria-label="Đổi chiều sắp xếp" onClick={() => setOrder(value => value === "asc" ? "desc" : "asc")}>{order === "asc" ? "↑" : "↓"}</Button></div><Card className="mt-4"><div className="desktop-data-table">{runs.length ? <div className="table-scroll"><table><thead><tr><th>#</th><th>Loại</th><th>Trạng thái</th><th>Đã xử lý</th><th>Đổi tên</th><th>Trùng</th><th>Bắt đầu</th></tr></thead><tbody>{runs.map(item => <tr key={item.id}><td>{item.id}</td><td>{item.source}</td><td><Badge tone={statusTone(item.status)}>{item.status}</Badge></td><td>{number(item.processed_files)}</td><td>{number(item.renamed_files)}</td><td>{number(item.duplicate_files)}</td><td>{time(item.started_at)}</td></tr>)}</tbody></table></div> : <Empty text="Chưa có nhật ký chạy." />}</div><div className="mobile-data-list">{runs.map(item => <div className="mobile-data-item" key={item.id}><div className="flex justify-between gap-2"><strong>#{item.id} · {item.source}</strong><Badge tone={statusTone(item.status)}>{item.status}</Badge></div><p className="mt-2 text-xs text-fg-muted">{time(item.started_at)}</p><p className="mt-2 text-sm">{number(item.processed_files)} xử lý · {number(item.renamed_files)} đổi tên · {number(item.duplicate_files)} trùng</p></div>)}{runs.length === 0 && <Empty text="Chưa có nhật ký chạy." />}</div></Card><Pager page={page} total={total} pageSize={20} setPage={setPage} /></>;
}

function Browser() {
  const initial = new URLSearchParams(window.location.search);
  const [current, setCurrent] = useState(initial.get("path") || ".");
  const [page, setPage] = useState(Math.max(1, Number(initial.get("page")) || 1));
  const [search, setSearch] = useState(initial.get("search") || "");
  const [sort, setSort] = useState(initial.get("sort") || "name");
  const [order, setOrder] = useState<"asc" | "desc">(initial.get("order") === "desc" ? "desc" : "asc");
  const [data, setData] = useState<BrowserData | null>(null);
  const [error, setError] = useState("");
  const [viewer, setViewer] = useState<number | null>(null);
  useEffect(() => {
    const query = new URLSearchParams();
    if (current !== ".") query.set("path", current);
    if (page > 1) query.set("page", String(page));
    if (search) query.set("search", search);
    if (sort !== "name") query.set("sort", sort);
    if (order !== "asc") query.set("order", order);
    window.history.replaceState({}, "", `/browse${query.size ? `?${query}` : ""}`);
    const timer = window.setTimeout(() => {
      api.GET("/api/browse", { params: { query: { path: current, page, pageSize: 24, search, sort: sort as "name" | "size" | "mod_time" | "type", order } } }).then(({ data: result, error: issue }) => {
        if (result) { setData(result); setError(""); }
        else if (issue) setError("Không mở được thư mục.");
      });
    }, search ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [current, page, search, sort, order]);
  const media = useMemo<ViewableMedia[]>(() => (data?.entries || []).filter(entry => entry.is_image || entry.is_video).map(entry => ({ path: entry.path, name: entry.name, is_image: entry.is_image, is_video: entry.is_video, mime_type: entry.mime_type })), [data]);
  function openDirectory(path: string) { setCurrent(path || "."); setPage(1); setViewer(null); }
  return <>
    <SectionTitle title="Duyệt thư mục nguồn" detail={`${number(data?.total || 0)} mục trong thư mục này`} />
    <div className="browser-controls mt-5">
      <label className="search-field"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm file trong thư mục..." aria-label="Tìm file trong thư mục" /></label>
      <select className="select-field" value={sort} onChange={event => { setSort(event.target.value); setPage(1); }} aria-label="Sắp xếp file"><option value="name">Tên</option><option value="mod_time">Ngày sửa</option><option value="size">Dung lượng</option><option value="type">Loại file</option></select>
      <Button variant="outline" aria-label={order === "asc" ? "Sắp xếp giảm dần" : "Sắp xếp tăng dần"} title={order === "asc" ? "Tăng dần" : "Giảm dần"} onClick={() => { setOrder(value => value === "asc" ? "desc" : "asc"); setPage(1); }}>{order === "asc" ? "↑" : "↓"}</Button>
    </div>
    <Card className="mt-4">
      <div className="flex flex-wrap items-center gap-2 text-sm"><button className="text-primary-text hover:underline" onClick={() => openDirectory(".")}>Gốc</button>{data?.breadcrumb.filter(item => item.path).map(item => <span key={item.path} className="flex items-center gap-2"><ChevronRight size={14} className="text-fg-muted" /><button className="text-primary-text hover:underline" onClick={() => openDirectory(item.path)}>{item.name}</button></span>)}</div>
      {error && <p className="mt-4 text-sm text-danger-text">{error}</p>}
      <div className="browser-grid mt-5">{data?.entries.map(entry => {
        const type = entry.is_dir ? "folder" : entry.is_video ? "video" : entry.is_image ? "image" : "file";
        const mediaIndex = media.findIndex(item => item.path === entry.path);
        return <button key={entry.path} className="browser-card" onClick={() => entry.is_dir ? openDirectory(entry.path) : mediaIndex >= 0 ? setViewer(mediaIndex) : entry.name.toLowerCase().endsWith(".pdf") ? window.open(`/api/file?path=${encodeURIComponent(entry.path)}`, "_blank", "noopener,noreferrer") : undefined} disabled={!entry.is_dir && mediaIndex < 0 && !entry.name.toLowerCase().endsWith(".pdf")} title={entry.path}>
          <MediaThumbnail path={entry.path} type={type} className="browser-thumb" />
          <span className="browser-card-name">{entry.name}</span>
          <span className="browser-card-meta">{entry.is_dir ? "Thư mục" : size(entry.size)}</span>
        </button>;
      })}</div>
      {data?.entries.length === 0 && <Empty text={search ? "Không tìm thấy file phù hợp." : "Thư mục trống."} />}
      <Pager page={page} total={data?.total || 0} pageSize={24} setPage={setPage} />
    </Card>
    {viewer !== null && <Suspense fallback={null}><MediaViewer items={media} index={viewer} onClose={() => setViewer(null)} /></Suspense>}
  </>;
}

function SettingsPage({ settings }: { settings: SettingsData | null }) {
  type IndexStatus = paths["/api/index/status"]["get"]["responses"][200]["content"]["application/json"];
  const [index, setIndex] = useState<IndexStatus | null>(null);
  const [indexError, setIndexError] = useState("");
  useEffect(() => {
    let active = true;
    const load = () => api.GET("/api/index/status").then(({ data }) => { if (active && data) setIndex(data); }).catch(() => {});
    void load();
    const timer = window.setInterval(load, 2500);
    return () => { active = false; window.clearInterval(timer); };
  }, []);
  async function refreshIndex() {
    setIndexError("");
    const { data, error } = await api.POST("/api/index/trigger");
    if (error) setIndexError("Không thể cập nhật chỉ mục.");
    else if (!data?.started) setIndexError("Chỉ mục đang được cập nhật.");
  }
  const rows = settings ? [["Thư mục nguồn", settings.dir], ["Thư mục cách ly", settings.quarantine_dir], ["Định dạng video", settings.extensions.join(", ")], ["Xem trước", settings.dry_run ? "Bật" : "Tắt"], ["Lịch tự động", settings.cron ? `Bật · ${settings.cron_expr}` : "Tắt"], ["Đổi tên thư mục con", settings.rename_subfolder ? "Bật" : "Tắt"], ["Cổng web", settings.web_port]] : [];
  return <><SectionTitle title="Cấu hình đang dùng" detail="Giá trị được đọc từ settings.json khi dịch vụ khởi động." /><Card className="mt-5"><div className="divide-y divide-border-subtle">{rows.map(([label, value]) => <div key={label} className="grid gap-1 py-3 sm:grid-cols-[12rem_1fr]"><span className="text-sm text-fg-muted">{label}</span><span className="break-all text-sm font-medium">{value}</span></div>)}</div>{!settings && <Empty text="Đang tải cấu hình..." />}</Card><div className="mt-5"><SectionTitle title="Chỉ mục tìm kiếm" detail="Tên file, metadata media và nội dung PDF." action={<Button variant="outline" size="sm" disabled={index?.running} onClick={() => void refreshIndex()}><RefreshCw size={16} /> Cập nhật</Button>} /><Card className="mt-3"><p className="text-sm">{index?.running ? `Đang xử lý ${number(index.processed_files)} / ${number(index.total_files)}` : index?.updated_at ? `Đã lập chỉ mục ${number(index.total_files)} mục` : "Đang chuẩn bị chỉ mục..."}</p>{index?.running && <div className="progress-track mt-3"><div className="progress-fill" style={{ width: `${index.total_files ? Math.round(index.processed_files / index.total_files * 100) : 0}%` }} /></div>}<p className="mt-2 text-xs text-fg-muted">Media: {number(index?.media_indexed || 0)} · PDF: {number(index?.pdf_indexed || 0)} · Lỗi: {number(index?.failed_files || 0)}</p>{indexError && <p className="mt-2 text-xs text-danger-text">{indexError}</p>}</Card></div><p className="mt-4 text-sm text-fg-muted">Để thay đổi, sửa settings.json trên máy chủ rồi khởi động lại dịch vụ.</p><a href="/openapi.yaml" target="_blank" rel="noreferrer" className="mt-5 inline-flex text-sm font-medium text-primary-text hover:underline">Xem hợp đồng API (OpenAPI) ↗</a></>;
}

function Empty({ text }: { text: string }) { return <div className="py-9 text-center text-sm text-fg-muted">{text}</div>; }

function Pager({ page, total, pageSize, setPage }: { page: number; total: number; pageSize: number; setPage: (page: number) => void }) {
  if (total <= pageSize) return null;
  const pages = Math.ceil(total / pageSize);
  return <div className="mt-4 flex items-center justify-between"><span className="text-sm text-fg-muted">Trang {page} / {pages} · {number(total)} mục</span><div className="flex gap-2"><Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)} aria-label="Trang trước"><ChevronLeft size={16} /></Button><Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage(page + 1)} aria-label="Trang sau"><ChevronRight size={16} /></Button></div></div>;
}
