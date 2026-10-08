// Electron main process for the qail menubar app.
//
// - Spawns the Go backend (`qail desktop-backend`) and bridges the
//   renderer's calls to it over IPC (see backend.cjs, preload.cjs).
// - Shows a small frameless window anchored under a tray icon; left-click
//   toggles it, right-click shows a menu. Closing only hides it.
// - Global hotkey Ctrl+Alt+Q toggles the window.
//
// Dev: set QAIL_DEV_SERVER_URL (e.g. http://localhost:5173) to load the
// Vite dev server instead of dist/, and QAIL_BIN to point at a qail
// binary (defaults to ../bin/qail from the repo root).

const path = require("node:path");
const fs = require("node:fs");
const {
  app,
  BrowserWindow,
  Menu,
  Tray,
  clipboard,
  dialog,
  globalShortcut,
  ipcMain,
  nativeImage,
  screen,
  shell,
} = require("electron");

const { Backend } = require("./backend.cjs");

const DEV_URL = process.env.QAIL_DEV_SERVER_URL;
const exe = process.platform === "win32" ? "qail.exe" : "qail";

// Packaged builds ship the Go binary and tray icon in resources/ via
// electron-builder's extraResources (see package.json "build").
function resourcePath(...parts) {
  return app.isPackaged
    ? path.join(process.resourcesPath, ...parts)
    : path.join(__dirname, "..", "..", ...parts);
}

function backendPath() {
  if (process.env.QAIL_BIN) return process.env.QAIL_BIN;
  return resourcePath("bin", exe);
}

let win = null;
let tray = null;
let backend = null;
let quitting = false;
let dialogOpen = false;

function createWindow() {
  win = new BrowserWindow({
    width: 480,
    height: 640,
    minWidth: 400,
    minHeight: 500,
    show: false,
    frame: false,
    resizable: true,
    skipTaskbar: true,
    fullscreenable: false,
    backgroundColor: "#09090b", // zinc-950, matches body bg
    webPreferences: {
      preload: path.join(__dirname, "preload.cjs"),
      contextIsolation: true,
      sandbox: true,
      nodeIntegration: false,
    },
  });

  if (DEV_URL) win.loadURL(DEV_URL);
  else win.loadFile(path.join(__dirname, "..", "dist", "index.html"));

  // The UI never navigates; anything that tries goes to the real browser.
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:/.test(url)) shell.openExternal(url);
    return { action: "deny" };
  });
  win.webContents.on("will-navigate", (e) => e.preventDefault());

  // Closing only hides; the tray brings it back.
  win.on("close", (e) => {
    if (!quitting) {
      e.preventDefault();
      win.hide();
    }
  });

  // Behave like a menubar popover: dismiss on focus loss. Skipped while
  // devtools are open, otherwise inspecting the UI hides it.
  win.on("blur", () => {
    if (!dialogOpen && !win.webContents.isDevToolsOpened()) win.hide();
  });
}

// Position the window centred under the tray icon (or above it when the
// tray sits at the bottom of the screen, as on Windows), clamped to the
// work area of the display the tray is on.
function positionWindow() {
  const trayBounds = tray?.getBounds();
  if (!trayBounds || trayBounds.width === 0) {
    win.center();
    return;
  }
  const { width, height } = win.getBounds();
  const display = screen.getDisplayNearestPoint({ x: trayBounds.x, y: trayBounds.y });
  const area = display.workArea;
  const gap = 5;

  let x = Math.round(trayBounds.x + trayBounds.width / 2 - width / 2);
  let y =
    trayBounds.y < area.y + area.height / 2
      ? trayBounds.y + trayBounds.height + gap
      : trayBounds.y - height - gap;

  x = Math.max(area.x, Math.min(x, area.x + area.width - width));
  y = Math.max(area.y, Math.min(y, area.y + area.height - height));
  win.setPosition(x, y, false);
}

function showWindow() {
  positionWindow();
  win.show();
  win.focus();
}

function toggleWindow() {
  if (win.isVisible()) win.hide();
  else showWindow();
}

function createTray() {
  const iconFile = resourcePath("icon.png");
  const icon = fs.existsSync(iconFile)
    ? nativeImage.createFromPath(iconFile)
    : nativeImage.createEmpty();
  tray = new Tray(icon);
  if (icon.isEmpty()) tray.setTitle?.("Q");
  tray.setToolTip("qail");

  const menu = Menu.buildFromTemplate([
    { label: "Show qail", click: showWindow },
    { type: "separator" },
    { label: "Quit", click: () => app.quit() },
  ]);
  // On macOS/Windows left-click toggles and right-click opens the menu.
  // Linux trays generally don't deliver click events, so the menu is the
  // only entry point there.
  if (process.platform === "linux") {
    tray.setContextMenu(menu);
  } else {
    tray.on("click", toggleWindow);
    tray.on("right-click", () => tray.popUpContextMenu(menu));
  }
}

function registerIpc() {
  // Only our own window may drive the backend.
  const fromOurWindow = (e) => win && e.sender === win.webContents;

  // Errors come back as data rather than a rejected invoke: Electron
  // logs every rejected handler with a stack trace and mangles the
  // message, and a failed action is an ordinary outcome here.
  ipcMain.handle("qail:call", async (e, method, args) => {
    if (!fromOurWindow(e)) return { error: "forbidden" };
    if (typeof method !== "string" || !Array.isArray(args)) {
      return { error: "invalid call" };
    }
    try {
      return { result: await backend.call(method, args) };
    } catch (err) {
      return { error: err instanceof Error ? err.message : String(err) };
    }
  });

  ipcMain.handle("qail:clipboard", (e, text) => {
    if (!fromOurWindow(e)) return;
    clipboard.writeText(String(text));
  });

  ipcMain.handle("qail:pickDirectory", async (e, title) => {
    if (!fromOurWindow(e)) return "";
    // The dialog steals focus; don't let the blur handler hide the window.
    dialogOpen = true;
    try {
      const res = await dialog.showOpenDialog(win, {
        title: typeof title === "string" ? title : undefined,
        properties: ["openDirectory", "createDirectory"],
      });
      return res.canceled ? "" : res.filePaths[0] ?? "";
    } finally {
      dialogOpen = false;
      win.focus();
    }
  });

  ipcMain.handle("qail:hideWindow", (e) => {
    if (fromOurWindow(e)) win.hide();
  });
}

if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  app.on("second-instance", () => win && showWindow());

  app.whenReady().then(() => {
    // Menubar app: no Dock icon or Cmd-Tab entry.
    if (process.platform === "darwin") app.dock?.hide();

    backend = new Backend(backendPath());
    backend.start();
    backend.onEvent((name, data) => {
      if (win && !win.isDestroyed()) win.webContents.send("qail:event", name, data);
    });

    registerIpc();
    createWindow();
    createTray();

    if (!globalShortcut.register("Control+Alt+Q", toggleWindow)) {
      console.error("global hotkey Ctrl+Alt+Q unavailable");
    }

    // In dev there's no tray-click habit yet; show the window straight away.
    if (DEV_URL) showWindow();
  });

  app.on("before-quit", () => {
    quitting = true;
  });

  app.on("will-quit", () => {
    globalShortcut.unregisterAll();
    backend?.stop();
  });

  // Keep running with no visible windows; the tray owns the lifecycle.
  app.on("window-all-closed", () => {});
}
