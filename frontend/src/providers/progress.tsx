// Progress provider — subscribes to the progress events fired by
// internal/app/bindings.go (workspace:progress / :done / :error) and
// exposes:
//   - start(title)           open the drawer in "running" state
//   - clear()                close + reset
//   - the current run's log lines + status
//
// The drawer itself (ProgressDrawer) consumes the same context and
// renders the log with auto-scroll.

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { ReactNode } from "react";
import { onEvent } from "../desktop";

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
    const offProgress = onEvent("workspace:progress", (data) => {
      const line = typeof data === "string" ? data : String(data ?? "");
      setLines((prev) => [...prev, line]);
    });
    const offDone = onEvent("workspace:done", () => {
      setStatus("done");
    });
    const offError = onEvent("workspace:error", (data) => {
      setStatus("error");
      const msg = typeof data === "string" ? data : String(data ?? "");
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
