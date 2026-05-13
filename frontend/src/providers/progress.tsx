// Progress provider — subscribes to the Wails v3 custom events fired
// by internal/app/bindings.go (workspace:progress / :done / :error) and
// exposes:
//   - start(title)           open the drawer in "running" state
//   - clear()                close + reset
//   - the current run's log lines + status
//
// The drawer itself (ProgressDrawer) consumes the same context and
// renders the log with auto-scroll. v3 hands the listener a
// `WailsEvent` object with `name` + `data`; we pull `data` out and
// keep the rest of the provider identical to the v2 version.

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { ReactNode } from "react";
import { Events } from "@wailsio/runtime";

export type ProgressStatus = "idle" | "running" | "done" | "error";

type Ctx = {
  open: boolean;
  title: string;
  status: ProgressStatus;
  lines: string[];
  errorMsg: string;
  start: (title: string) => void;
  clear: () => void;
};

const ProgressContext = createContext<Ctx | null>(null);

type WailsEvent = { name: string; data: unknown };

export const ProgressProvider = ({ children }: { children: ReactNode }) => {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState<ProgressStatus>("idle");
  const [lines, setLines] = useState<string[]>([]);
  const [errorMsg, setErrorMsg] = useState("");

  // Hold the latest setter references so we register listeners once.
  const linesRef = useRef(lines);
  linesRef.current = lines;

  useEffect(() => {
    const offProgress = Events.On("workspace:progress", (e: WailsEvent) => {
      const line = typeof e?.data === "string" ? e.data : String(e?.data ?? "");
      setLines((prev) => [...prev, line]);
    });
    const offDone = Events.On("workspace:done", () => {
      setStatus("done");
    });
    const offError = Events.On("workspace:error", (e: WailsEvent) => {
      setStatus("error");
      const msg = typeof e?.data === "string" ? e.data : String(e?.data ?? "");
      setErrorMsg(msg);
    });
    return () => {
      offProgress();
      offDone();
      offError();
    };
  }, []);

  const value: Ctx = useMemo(
    () => ({
      open,
      title,
      status,
      lines,
      errorMsg,
      start: (t: string) => {
        setTitle(t);
        setStatus("running");
        setLines([]);
        setErrorMsg("");
        setOpen(true);
      },
      clear: () => {
        setOpen(false);
        setStatus("idle");
        setLines([]);
        setErrorMsg("");
        setTitle("");
      },
    }),
    [open, title, status, lines, errorMsg]
  );

  return (
    <ProgressContext.Provider value={value}>
      {children}
    </ProgressContext.Provider>
  );
};

export const useProgress = (): Ctx => {
  const ctx = useContext(ProgressContext);
  if (!ctx) throw new Error("useProgress outside ProgressProvider");
  return ctx;
};
