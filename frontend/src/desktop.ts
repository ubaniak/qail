// The UI's only link to the backend. Two transports, same surface:
//
//   - Electron: the `window.qail` bridge exposed by electron/preload.cjs,
//     which forwards to `qail desktop-backend` over stdio.
//   - Browser (`qail tray`, ADR-0021): the qail binary serves this UI and
//     the same Bindings over HTTP on 127.0.0.1 (internal/app/http.go).
//
// Pages import these helpers and never touch either transport directly.

type CallResult = { result?: unknown; error?: string };

type QailBridge = {
  call: (method: string, args: unknown[]) => Promise<CallResult>;
  onEvent: (name: string, fn: (data: unknown) => void) => () => void;
  copyText: (text: string) => Promise<void>;
  pickDirectory: (title?: string) => Promise<string>;
  hideWindow: () => Promise<void>;
};

declare global {
  interface Window {
    qail?: QailBridge;
  }
}

// isDesktop is true inside the Electron window, false in a browser tab.
export const isDesktop = typeof window !== "undefined" && !!window.qail;

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const data = (await res.json()) as { error?: string };
      if (data.error) msg = data.error;
    } catch {
      /* not JSON */
    }
    throw new Error(msg);
  }
  return (await res.json()) as T;
}

// One EventSource for the whole page, fanned out by event name. The
// browser reconnects it on its own if the tray restarts.
type Listener = (data: unknown) => void;
const listeners = new Map<string, Set<Listener>>();
let source: EventSource | null = null;

function ensureEventSource() {
  if (source) return;
  source = new EventSource("api/events");
  source.onmessage = (e) => {
    let msg: { event?: string; data?: unknown };
    try {
      msg = JSON.parse(e.data);
    } catch {
      return;
    }
    if (!msg.event) return;
    for (const fn of listeners.get(msg.event) ?? []) fn(msg.data);
  };
}

const httpBridge: QailBridge = {
  call: (method, args) => postJSON<CallResult>("api/call", { method, args }),

  onEvent: (name, fn) => {
    ensureEventSource();
    let set = listeners.get(name);
    if (!set) listeners.set(name, (set = new Set()));
    set.add(fn);
    return () => set!.delete(fn);
  },

  // 127.0.0.1 is a secure context, so the async clipboard API works.
  copyText: (text) => navigator.clipboard.writeText(text),

  pickDirectory: async (title) => {
    const res = await postJSON<{ path?: string; error?: string }>(
      "api/pick-directory",
      { title: title ?? "" }
    );
    if (res.error) throw new Error(res.error);
    return res.path ?? "";
  },

  // A browser tab has no window of ours to hide.
  hideWindow: async () => {},
};

function bridge(): QailBridge {
  return window.qail ?? httpBridge;
}

// call invokes a Go Bindings method by name. The backend returns
// `{ result }` or `{ error }` (the Go error string), which becomes a
// rejected promise here so callers can just await.
export async function call<T>(method: string, args: unknown[]): Promise<T> {
  const res = await bridge().call(method, args);
  if (res.error) throw new Error(res.error);
  return res.result as T;
}

export function onEvent(name: string, fn: (data: unknown) => void): () => void {
  return bridge().onEvent(name, fn);
}

export const copyText = (text: string) => bridge().copyText(text);
export const pickDirectory = (title?: string) => bridge().pickDirectory(title);
export const hideWindow = () => bridge().hideWindow();
