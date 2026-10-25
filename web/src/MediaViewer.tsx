import { useEffect, useMemo, useRef, useState } from "react";
import Hls from "hls.js";
import Lightbox from "yet-another-react-lightbox";
import "yet-another-react-lightbox/styles.css";
import { api } from "./api";

export type ViewableMedia = { path: string; name: string; is_image: boolean; is_video: boolean; mime_type: string };

function VideoPlayback({ item }: { item: ViewableMedia }) {
  const ref = useRef<HTMLVideoElement>(null);
  const [hlsURL, setHlsURL] = useState("");
  const [error, setError] = useState("");
  const [starting, setStarting] = useState(false);
  const nativeURL = `/api/file?path=${encodeURIComponent(item.path)}`;
  const likelyNative = /\.(mp4|m4v|webm)$/i.test(item.path);
  useEffect(() => { setHlsURL(""); setError(""); setStarting(false); }, [item.path]);
  useEffect(() => {
    if (!hlsURL || !ref.current) return;
    const video = ref.current;
    if (Hls.isSupported()) {
      const hls = new Hls({ maxBufferLength: 18, maxMaxBufferLength: 30, startFragPrefetch: true });
      hls.on(Hls.Events.MANIFEST_PARSED, () => hls.startLoad());
      hls.on(Hls.Events.LEVEL_LOADED, () => hls.resumeBuffering());
      hls.loadSource(hlsURL);
      hls.attachMedia(video);
      hls.on(Hls.Events.ERROR, (_, data) => { if (data.fatal) setError("Không phát được video này."); });
      return () => hls.destroy();
    }
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = hlsURL;
      return () => { video.removeAttribute("src"); video.load(); };
    }
    setError("Trình duyệt không hỗ trợ HLS.");
  }, [hlsURL]);
  async function startHLS() {
    if (starting || hlsURL) return;
    setStarting(true);
    setError("");
    try {
      const { data, error: issue } = await api.POST("/api/hls/start", { body: { path: item.path } });
      if (issue || !data?.url) throw new Error("Không tạo được luồng phát.");
      setHlsURL(data.url);
    } catch {
      setError("Không tạo được luồng phát. Thử lại sau.");
      setStarting(false);
    }
  }
  useEffect(() => { if (!likelyNative) void startHLS(); }, [item.path, likelyNative]);
  return <div className="viewer-video"><video ref={ref} controls playsInline preload="auto" poster={`/api/thumbnail?path=${encodeURIComponent(item.path)}`} src={hlsURL ? undefined : likelyNative ? nativeURL : undefined} onError={() => { if (!hlsURL) void startHLS(); else setError("Không phát được video này."); }} />{starting && !hlsURL && <p>Đang chuẩn bị video...</p>}{error && <p role="alert">{error}</p>}</div>;
}

export function MediaViewer({ items, index, onClose }: { items: ViewableMedia[]; index: number; onClose: () => void }) {
  const [current, setCurrent] = useState(index);
  const slides = useMemo(() => items.map(item => ({ src: item.is_video ? `video:${item.path}` : `/api/file?path=${encodeURIComponent(item.path)}`, alt: item.name })), [items]);
  const item = items[current];
  return <Lightbox open close={onClose} slides={slides} index={index} carousel={{ preload: 0 }} render={{ slide: ({ slide }) => { const video = items.find(item => slide.src === `video:${item.path}`); return video ? <VideoPlayback key={video.path} item={video} /> : undefined; } }} on={{ view: ({ index: next }) => setCurrent(next) }} toolbar={{ buttons: [item && <a key="open" className="viewer-open" href={`/api/file?path=${encodeURIComponent(item.path)}`} target="_blank" rel="noopener noreferrer" aria-label="Mở file gốc">Mở file</a>, "close"] }} labels={{ Close: "Đóng", Next: "Tiếp", Previous: "Trước" }} />;
}
