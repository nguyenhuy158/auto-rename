import { useCallback, useEffect, useMemo, useState } from "react";
import { Activity, ArchiveRestore, Check, ChevronLeft, ChevronRight, Clock3, Ellipsis, Files, Folder, FolderOpen, LayoutDashboard, ListFilter, Moon, Play, RefreshCw, Search, Settings, ShieldCheck, Sun, Trash2, X } from "lucide-react";
import { api, unwrap } from "./api";
import { Badge, Button, Card, Metric } from "./ui";
import type { components } from "./generated/api";

type ScanRun = components["schemas"]["ScanRun"];
type DuplicateGroup = components["schemas"]["DuplicateGroup"];
type FileRecord = components["schemas"]["FileRecord"];
type QuarantineAction = components["schemas"]["QuarantineAction"];
type SettingsData = components["schemas"]["Settings"];
type Stats = components["schemas"]["Stats"];
type BrowserData = { current_path: string; entries: components["schemas"]["FileEntry"][]; breadcrumb: components["schemas"]["Breadcrumb"][]; base_dir: string };
type ConfirmAction = { title: string; description: string; paths: string[]; execute: () => Promise<unknown>; danger?: boolean };

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
const statusTone = (status: string) => status === "completed" || status === "restored" || status === "renamed" || status === "quarantined" || status === "success" ? "success" : status === "running" || status === "planned" ? "warning" : status === "failed" || status === "error" ? "danger" : "neutral";

function useLocation() {
  const [path, setPath] = useState(window.location.pathname);
  useEffect(() => {
    const change = () => setPath(window.location.pathname);
    window.addEventListener("popstate", change);
    return () => window.removeEventListener("popstate", change);
  }, []);
  return [path, (next: string) => { window.history.pushState({}, "", next); setPath(next); window.scrollTo(0, 0); }] as const;
}

export function App() {
  const [path, navigate] = useLocation();
  const [dark, setDark] = useState(localStorage.getItem("auto-rename-theme") === "dark");
  const [settings, setSettings] = useState<SettingsData | null>(null);
  const [run, setRun] = useState<ScanRun | null>(null);
  const [notice, setNotice] = useState("");
  const [confirm, setConfirm] = useState<ConfirmAction | null>(null);
  const [busy, setBusy] = useState(false);
  const [revision, setRevision] = useState(0);
  const [moreOpen, setMoreOpen] = useState(false);

  const refresh = useCallback(() => setRevision(value => value + 1), []);
  const loadRun = useCallback(() => api.GET("/api/cron/status").then(({ data }) => data && setRun(data)).catch(() => {}), []);
  useEffect(() => { document.documentElement.classList.toggle("dark", dark); localStorage.setItem("auto-rename-theme", dark ? "dark" : "light"); }, [dark]);
  useEffect(() => { api.GET("/api/settings").then(({ data }) => data && setSettings(data)).catch(() => {}); }, []);
  useEffect(() => { void loadRun(); const timer = window.setInterval(loadRun, 4000); return () => window.clearInterval(timer); }, [loadRun]);

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
      <header className="topbar"><div className="min-w-0"><p className="text-xs text-fg-muted">Auto Rename / {title}</p><h1 className="mt-1 truncate text-xl font-semibold tracking-tight">{title}</h1></div><div className="flex items-center gap-2"><Badge tone={settings?.dry_run ? "warning" : "success"}>{settings?.dry_run ? "Chế độ xem trước" : "Áp dụng thật"}</Badge><Button variant="ghost" size="sm" aria-label={dark ? "Bật giao diện sáng" : "Bật giao diện tối"} onClick={() => setDark(value => !value)}>{dark ? <Sun size={18} /> : <Moon size={18} />}</Button></div></header>
      <main className="page-content">{notice && <div role="status" className="notice"><span>{notice}</span><button onClick={() => setNotice("")} aria-label="Đóng thông báo">×</button></div>}
        {path === "/" && <Dashboard revision={revision} run={run} settings={settings} busy={busy} perform={perform} navigate={navigate} />}
        {path === "/duplicates" && <Duplicates revision={revision} run={run} busy={busy} perform={perform} setConfirm={setConfirm} />}
        {path === "/records" && <Records revision={revision} busy={busy} setConfirm={setConfirm} />}
        {path === "/logs" && <Logs revision={revision} run={run} />}
        {path === "/browse" && <Browser />}
        {path === "/settings" && <SettingsPage settings={settings} />}
      </main>
    </div>
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
  const [actions, setActions] = useState<QuarantineAction[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  useEffect(() => { api.GET("/api/duplicates").then(({ data }) => data && setGroups(data.groups || [])); api.GET("/api/quarantine").then(({ data }) => data && setActions(data.actions || [])); }, [revision, run?.status]);
  const visible = useMemo(() => groups.filter(group => group.files.some(file => file.path.toLowerCase().includes(search.toLowerCase()))), [groups, search]);
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
    <label className="search-field mt-4"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm theo đường dẫn file..." aria-label="Tìm file trùng" /></label>
    <div className="mt-4 grid gap-3">{displayed.length ? displayed.map(group => <Card key={group.sha256}><div className="flex flex-wrap items-start justify-between gap-3"><div><div className="flex items-center gap-2"><Badge tone="primary">{group.files.length} bản</Badge><span className="text-xs text-fg-muted">{size(group.size)} mỗi file</span></div><p className="mt-2 text-sm font-medium" title={group.files[0]?.path}>{basename(group.files[0]?.path || "")}</p><p className="mt-1 text-xs text-fg-muted">SHA-256: {group.sha256.slice(0, 18)}…</p></div><Button variant="outline" size="sm" disabled={busy} onClick={() => confirmGroup(group, group.files.slice(1).map(file => file.path))}>Cách ly nhóm</Button></div><div className="mt-4 divide-y divide-border-subtle rounded-ui border border-border-subtle">{group.files.map((file, index) => <div key={file.path} className="flex min-w-0 items-center gap-3 px-3 py-2.5"><input type="checkbox" checked={index > 0 && selected.includes(file.path)} disabled={index === 0} onChange={() => toggle(file.path)} aria-label={`Chọn ${file.path}`} /><div className="min-w-0 flex-1"><p className="truncate text-sm" title={file.path}>{file.path}</p><p className="text-xs text-fg-muted">{time(file.modified_at)}</p></div>{index === 0 ? <Badge tone="success">Giữ lại</Badge> : <Badge tone="warning">Bản trùng</Badge>}</div>)}</div></Card>) : <Card><Empty text={search ? "Không tìm thấy file phù hợp." : "Chưa có dữ liệu trùng. Chọn Quét lại để bắt đầu."} /></Card>}</div><Pager page={page} total={visible.length} pageSize={10} setPage={setPage} />
    <div className="mt-10"><SectionTitle title="File trong cách ly" detail="Khôi phục về đúng đường dẫn gốc nếu vị trí đó còn trống." /><Card className="mt-4">{actions.length ? <div className="table-scroll"><table><thead><tr><th>File gốc</th><th>Trạng thái</th><th>Thời gian</th><th></th></tr></thead><tbody>{actions.map(action => <tr key={action.id}><td title={action.original_path}>{action.original_path}</td><td><Badge tone={statusTone(action.status)}>{action.status}</Badge></td><td>{time(action.created_at)}</td><td>{action.status !== "restored" && <Button variant="outline" size="sm" disabled={busy} onClick={() => setConfirm({ title: "Khôi phục file?", description: "File được chuyển về vị trí ban đầu nếu đường dẫn chưa bị chiếm.", paths: [action.original_path], execute: () => unwrap(api.POST("/api/quarantine/restore", { body: { id: action.id } })) })}><ArchiveRestore size={15} /> Khôi phục</Button>}</td></tr>)}</tbody></table></div> : <Empty text="Chưa có file trong cách ly." />}</Card></div></>;
}

function Records({ revision, busy, setConfirm }: { revision: number; busy: boolean; setConfirm: (value: ConfirmAction | null) => void }) {
  const [records, setRecords] = useState<FileRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  useEffect(() => { api.GET("/api/records", { params: { query: { page, pageSize: 20, search, status } } }).then(({ data }) => { if (data) { setRecords(data.records || []); setTotal(data.total); } }); }, [revision, page, search, status]);
  return <><SectionTitle title="Lịch sử đổi tên" detail="Tra cứu từng lần đổi tên và hoàn tác về tên gốc." /><div className="mt-5 flex flex-wrap gap-3"><label className="search-field flex-1"><Search size={17} /><input value={search} onChange={event => { setSearch(event.target.value); setPage(1); }} placeholder="Tìm tên file..." aria-label="Tìm bản ghi" /></label><select className="select-field" value={status} onChange={event => { setStatus(event.target.value); setPage(1); }} aria-label="Lọc trạng thái"><option value="">Mọi trạng thái</option><option value="renamed">Đã đổi tên</option><option value="restored">Đã khôi phục</option><option value="failed">Lỗi</option></select></div><Card className="mt-4">{records.length ? <div className="table-scroll"><table><thead><tr><th>Tên gốc</th><th>Tên mới</th><th>Trạng thái</th><th>Thời gian</th><th></th></tr></thead><tbody>{records.map(record => <tr key={record.id}><td title={record.original_name}>{record.original_name}</td><td title={record.new_name}>{record.new_name}</td><td><Badge tone={statusTone(record.status)}>{record.status}</Badge></td><td>{time(record.renamed_at)}</td><td>{record.success && record.status !== "restored" && <Button variant="outline" size="sm" disabled={busy} onClick={() => setConfirm({ title: "Khôi phục tên gốc?", description: `Đổi tên ${record.new_name} về ${record.original_name}. Đường dẫn đích phải còn trống.`, paths: [record.file_path], execute: () => unwrap(api.POST("/api/records/restore", { body: { id: record.id } })) })}><ArchiveRestore size={15} /> Hoàn tác</Button>}</td></tr>)}</tbody></table></div> : <Empty text="Không có bản ghi phù hợp." />}</Card><Pager page={page} total={total} pageSize={20} setPage={setPage} /></>;
}

function Logs({ revision, run }: { revision: number; run: ScanRun | null }) {
  const [runs, setRuns] = useState<ScanRun[]>([]);
  useEffect(() => { api.GET("/api/runs").then(({ data }) => data && setRuns(data.recent || [])); }, [revision, run?.status]);
  return <><SectionTitle title="Nhật ký chạy" detail="Tiến trình và kết quả các lần quét gần đây." /><Card className="mt-5"><Progress run={run} /></Card><Card className="mt-5">{runs.length ? <div className="table-scroll"><table><thead><tr><th>#</th><th>Loại</th><th>Trạng thái</th><th>Đã xử lý</th><th>Đổi tên</th><th>Trùng</th><th>Bắt đầu</th></tr></thead><tbody>{runs.map(item => <tr key={item.id}><td>{item.id}</td><td>{item.source}</td><td><Badge tone={statusTone(item.status)}>{item.status}</Badge></td><td>{number(item.processed_files)}</td><td>{number(item.renamed_files)}</td><td>{number(item.duplicate_files)}</td><td>{time(item.started_at)}</td></tr>)}</tbody></table></div> : <Empty text="Chưa có nhật ký chạy." />}</Card></>;
}

function Browser() {
  const [current, setCurrent] = useState(".");
  const [data, setData] = useState<BrowserData | null>(null);
  const [error, setError] = useState("");
  useEffect(() => { api.GET("/api/browse", { params: { query: { path: current } } }).then(({ data, error }) => { if (data) { setData(data); setError(""); } else if (error) setError("Không mở được thư mục."); }); }, [current]);
  return <><SectionTitle title="Duyệt thư mục nguồn" detail="Chỉ xem nội dung trong thư mục được cấu hình." /><Card className="mt-5"><div className="flex flex-wrap items-center gap-2 text-sm"><button className="text-primary-text hover:underline" onClick={() => setCurrent(".")}>Gốc</button>{data?.breadcrumb.map(item => <span key={item.path} className="flex items-center gap-2"><ChevronRight size={14} className="text-fg-muted" /><button className="text-primary-text hover:underline" onClick={() => setCurrent(item.path)}>{item.name}</button></span>)}</div><p className="mt-2 break-all text-xs text-fg-muted">{data?.current_path || current}</p>{error && <p className="mt-4 text-sm text-danger-text">{error}</p>}<div className="mt-5 divide-y divide-border-subtle">{data?.entries.map(entry => <div key={entry.path} className="flex min-w-0 items-center gap-3 py-3"><div className="text-primary-text">{entry.is_dir ? <Folder size={18} /> : <Files size={18} />}</div><div className="min-w-0 flex-1">{entry.is_dir ? <button className="max-w-full truncate text-left text-sm font-medium hover:text-primary-text" onClick={() => setCurrent(entry.path)}>{entry.name}</button> : <p className="truncate text-sm" title={entry.name}>{entry.name}</p>}<p className="text-xs text-fg-muted">{entry.is_dir ? "Thư mục" : size(entry.size)} · {time(entry.mod_time)}</p></div>{!entry.is_dir && (entry.is_video || entry.is_image) && <a className="text-xs font-medium text-primary-text hover:underline" href={`/api/file?path=${encodeURIComponent(entry.path)}`} target="_blank" rel="noopener noreferrer">Mở</a>}</div>)}{data?.entries.length === 0 && <Empty text="Thư mục trống." />}</div></Card></>;
}

function SettingsPage({ settings }: { settings: SettingsData | null }) {
  const rows = settings ? [["Thư mục nguồn", settings.dir], ["Thư mục cách ly", settings.quarantine_dir], ["Định dạng video", settings.extensions.join(", ")], ["Xem trước", settings.dry_run ? "Bật" : "Tắt"], ["Lịch tự động", settings.cron ? `Bật · ${settings.cron_expr}` : "Tắt"], ["Đổi tên thư mục con", settings.rename_subfolder ? "Bật" : "Tắt"], ["Cổng web", settings.web_port]] : [];
  return <><SectionTitle title="Cấu hình đang dùng" detail="Giá trị được đọc từ settings.json khi dịch vụ khởi động." /><Card className="mt-5"><div className="divide-y divide-border-subtle">{rows.map(([label, value]) => <div key={label} className="grid gap-1 py-3 sm:grid-cols-[12rem_1fr]"><span className="text-sm text-fg-muted">{label}</span><span className="break-all text-sm font-medium">{value}</span></div>)}</div>{!settings && <Empty text="Đang tải cấu hình..." />}</Card><p className="mt-4 text-sm text-fg-muted">Để thay đổi, sửa settings.json trên máy chủ rồi khởi động lại dịch vụ.</p><a href="/openapi.yaml" target="_blank" rel="noreferrer" className="mt-5 inline-flex text-sm font-medium text-primary-text hover:underline">Xem hợp đồng API (OpenAPI) ↗</a></>;
}

function Empty({ text }: { text: string }) { return <div className="py-9 text-center text-sm text-fg-muted">{text}</div>; }

function Pager({ page, total, pageSize, setPage }: { page: number; total: number; pageSize: number; setPage: (page: number) => void }) {
  if (total <= pageSize) return null;
  const pages = Math.ceil(total / pageSize);
  return <div className="mt-4 flex items-center justify-between"><span className="text-sm text-fg-muted">Trang {page} / {pages} · {number(total)} mục</span><div className="flex gap-2"><Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)} aria-label="Trang trước"><ChevronLeft size={16} /></Button><Button variant="outline" size="sm" disabled={page >= pages} onClick={() => setPage(page + 1)} aria-label="Trang sau"><ChevronRight size={16} /></Button></div></div>;
}
