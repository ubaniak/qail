// cli — puts the bundled `qail` binary on the user's PATH, so one install
// from the .dmg gives both the menubar app and the `qail` command.
//
// macOS only: links /usr/local/bin/qail → <app>/Contents/Resources/bin/qail
// (the same binary the app runs as its backend, so the two never drift).
// /usr/local/bin is on the default PATH for every shell. When it isn't
// writable by the user, macOS asks for an admin password once.

const fs = require("node:fs");
const path = require("node:path");
const { execFile } = require("node:child_process");
const { app, dialog } = require("electron");

const LINK = "/usr/local/bin/qail";

// A menubar app with no Dock icon is never frontmost on its own, so its
// dialogs would open behind other windows. Pull it forward first.
function showMessageBox(opts) {
  app.focus({ steal: true });
  return dialog.showMessageBox(opts);
}

const supported = () => process.platform === "darwin" && app.isPackaged;

// "installed" when the link points at this app's binary, "other" when
// something else already owns the path (another qail, a stale link to a
// moved app), "missing" otherwise.
function status(target) {
  let st;
  try {
    st = fs.lstatSync(LINK);
  } catch {
    return "missing";
  }
  if (st.isSymbolicLink() && fs.readlinkSync(LINK) === target) return "installed";
  return "other";
}

function linkAsUser(target) {
  fs.mkdirSync(path.dirname(LINK), { recursive: true });
  fs.rmSync(LINK, { force: true });
  fs.symlinkSync(target, LINK);
}

// Same as linkAsUser, through the macOS admin prompt.
function linkAsAdmin(target) {
  if (/["'\\]/.test(target)) {
    return Promise.reject(new Error(`unsupported characters in app path: ${target}`));
  }
  const sh = `mkdir -p '${path.dirname(LINK)}' && ln -sf '${target}' '${LINK}'`;
  const script = `do shell script "${sh}" with administrator privileges`;
  return new Promise((resolve, reject) =>
    execFile("osascript", ["-e", script], (err, _out, stderr) =>
      err ? reject(new Error(stderr.trim() || err.message)) : resolve()
    )
  );
}

// Running straight from the mounted .dmg, or from a quarantined copy that
// macOS has moved to a random path (App Translocation), the binary's path
// won't survive; a link to it would break on eject or relaunch. Both live
// on read-only volumes, which catches a .dmg mounted outside /Volumes too.
function temporaryLocation(target) {
  if (target.startsWith("/Volumes/") || target.includes("/AppTranslocation/")) return true;
  try {
    fs.accessSync(path.dirname(target), fs.constants.W_OK);
  } catch (err) {
    return err.code === "EROFS";
  }
  return false;
}

// install links the CLI, asking before replacing a qail it didn't put
// there. quiet skips the "already installed" message (used by the
// first-launch offer, which only asks when it isn't).
async function install(target, { quiet = false } = {}) {
  if (temporaryLocation(target)) {
    showMessageBox({
      message: "Move qail to Applications first.",
      detail: "The qail command links to the app, so the app needs to stay in one place. Drag qail into Applications, open it from there, and try again.",
    });
    return;
  }
  const current = status(target);
  if (current === "installed") {
    if (!quiet) {
      showMessageBox({
        message: "The qail command is already installed.",
        detail: `${LINK} points to this app.`,
      });
    }
    return;
  }
  if (current === "other") {
    const { response } = await showMessageBox({
      type: "question",
      message: `Replace the existing ${LINK}?`,
      detail: "Another qail is installed there. It will be replaced with a link to this app's qail.",
      buttons: ["Replace", "Cancel"],
      defaultId: 0,
      cancelId: 1,
    });
    if (response !== 0) return;
  }

  try {
    try {
      linkAsUser(target);
    } catch (err) {
      if (err.code !== "EACCES" && err.code !== "EPERM") throw err;
      await linkAsAdmin(target);
    }
  } catch (err) {
    // Cancelling the password prompt lands here too.
    showMessageBox({
      type: "error",
      message: "Couldn't install the qail command.",
      detail: err.message,
    });
    return;
  }
  showMessageBox({
    message: "The qail command is installed.",
    detail:
      "Open a new terminal window and run `qail`. If an older qail runs instead, " +
      "`which -a qail` shows which copy comes first on your PATH.",
  });
}

// offerOnFirstLaunch asks once whether to install the CLI. The answer is
// remembered in the app's userData dir, so later launches stay quiet; the
// tray menu item stays available either way.
async function offerOnFirstLaunch(target) {
  if (!supported() || temporaryLocation(target) || status(target) === "installed") return;
  const marker = path.join(app.getPath("userData"), "cli-offered");
  if (fs.existsSync(marker)) return;
  fs.mkdirSync(path.dirname(marker), { recursive: true });
  fs.writeFileSync(marker, "");

  const { response } = await showMessageBox({
    type: "question",
    message: "Install the qail command line tool?",
    detail:
      `This links ${LINK} to the qail inside this app, so you can run \`qail\` from any terminal. ` +
      "You can do this later from the tray menu.",
    buttons: ["Install", "Not Now"],
    defaultId: 0,
    cancelId: 1,
  });
  if (response === 0) await install(target, { quiet: true });
}

module.exports = { supported, install, offerOnFirstLaunch };
