import { useState } from "react";
import { FileImage, FileVideo, Folder } from "lucide-react";

export function MediaThumbnail({ path, type, className = "" }: { path: string; type: "image" | "video" | "folder" | "file"; className?: string }) {
  const [failed, setFailed] = useState(false);
  const Icon = type === "folder" ? Folder : type === "video" ? FileVideo : FileImage;
  return <div className={`media-thumbnail ${className}`}>
    {(type === "image" || type === "video") && !failed ? <img src={`/api/thumbnail?path=${encodeURIComponent(path)}`} loading="lazy" alt="" onError={() => setFailed(true)} /> : <Icon size={24} aria-hidden="true" />}
    {type === "video" && !failed && <span className="media-play" aria-hidden="true">▶</span>}
  </div>;
}
