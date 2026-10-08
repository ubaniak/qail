// Typed access to the `window.qail` bridge exposed by
// electron/preload.cjs. This is the only place the renderer touches
// Electron; pages import these helpers instead.

type QailBridge = {
  call: (
    method: string,
    args: unknown[]
  ) => Promise<{ result?: unknown; error?: string }>;
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

function bridge(): QailBridge {
  if (!window.qail) {
    throw new Error("qail desktop bridge not available (run inside Electron)");
  }
  return window.qail;
}

// call invokes a Go Bindings method by name. The main process returns
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
