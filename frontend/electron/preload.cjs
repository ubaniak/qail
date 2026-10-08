// Preload — exposes a narrow `window.qail` bridge to the renderer. The
// renderer has no Node access; everything goes through these IPC calls,
// which main.cjs handles. Typed on the renderer side in src/desktop.ts.

const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("qail", {
  call: (method, args) => ipcRenderer.invoke("qail:call", method, args),

  onEvent: (name, fn) => {
    const listener = (_e, eventName, data) => {
      if (eventName === name) fn(data);
    };
    ipcRenderer.on("qail:event", listener);
    return () => ipcRenderer.removeListener("qail:event", listener);
  },

  copyText: (text) => ipcRenderer.invoke("qail:clipboard", text),
  pickDirectory: (title) => ipcRenderer.invoke("qail:pickDirectory", title),
  hideWindow: () => ipcRenderer.invoke("qail:hideWindow"),
});
