// Live log drawer for long-running workspace ops. Subscribes to the
// ProgressProvider context; renders one row per `workspace:progress`
// event with auto-scroll-to-bottom. Footer button closes the drawer
// once status flips to done/error.

import { CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined } from "@ant-design/icons";
import { Button, Drawer } from "antd";
import { useEffect, useRef } from "react";
import { useProgress } from "../../providers/progress";

export default function ProgressDrawer() {
  const { open, title, status, lines, errorMsg, clear } = useProgress();
  const logRef = useRef<HTMLDivElement | null>(null);

  // Auto-scroll log on every new line.
  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [lines]);

  const statusBadge = () => {
    if (status === "running") {
      return (
        <span className="text-amber-400 inline-flex items-center gap-1">
          <LoadingOutlined /> running
        </span>
      );
    }
    if (status === "done") {
      return (
        <span className="text-emerald-400 inline-flex items-center gap-1">
          <CheckCircleOutlined /> done
        </span>
      );
    }
    if (status === "error") {
      return (
        <span className="text-rose-400 inline-flex items-center gap-1">
          <CloseCircleOutlined /> error
        </span>
      );
    }
    return null;
  };

  return (
    <Drawer
      title={
        <div className="flex items-center justify-between gap-3">
          <span className="text-zinc-100">{title || "Workspace"}</span>
          <span className="text-xs">{statusBadge()}</span>
        </div>
      }
      placement="bottom"
      height={320}
      open={open}
      onClose={clear}
      closable={status !== "running"}
      maskClosable={status !== "running"}
      footer={
        <div className="flex justify-end">
          <Button
            onClick={clear}
            disabled={status === "running"}
            type={status === "error" ? "default" : "primary"}
          >
            Close
          </Button>
        </div>
      }
      styles={{
        body: { padding: 0, background: "#0b0b0e" },
        header: { background: "#18181b", borderBottom: "1px solid #27272a" },
        footer: { background: "#18181b", borderTop: "1px solid #27272a" },
      }}
    >
      <div
        ref={logRef}
        className="h-full overflow-auto px-3 py-2 text-xs font-mono text-zinc-300"
      >
        {lines.length === 0 && status === "running" && (
          <div className="text-zinc-500 italic">waiting for output…</div>
        )}
        {lines.map((l, i) => (
          <div key={i} className="whitespace-pre-wrap">
            {l}
          </div>
        ))}
        {status === "error" && errorMsg && (
          <div className="mt-2 text-rose-400 whitespace-pre-wrap">
            {errorMsg}
          </div>
        )}
      </div>
    </Drawer>
  );
}
