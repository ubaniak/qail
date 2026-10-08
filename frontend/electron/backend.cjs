// Backend — owns the `qail desktop-backend` child process and speaks its
// newline-delimited JSON protocol (see internal/app/rpc.go).
//
//   call(method, args) → Promise resolved with the Go method's result
//   onEvent(fn)        → fn(name, data) for every progress event

const { spawn } = require("node:child_process");
const readline = require("node:readline");

class Backend {
  constructor(binPath) {
    this.binPath = binPath;
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Set();
    this.child = null;
    this.stopping = false;
  }

  start() {
    const child = spawn(this.binPath, ["desktop-backend"], {
      stdio: ["pipe", "pipe", "pipe"],
      windowsHide: true,
    });
    this.child = child;

    readline.createInterface({ input: child.stdout }).on("line", (line) => {
      let msg;
      try {
        msg = JSON.parse(line);
      } catch {
        console.error("[qail-backend] unparseable line:", line);
        return;
      }
      if (msg.event) {
        for (const fn of this.listeners) fn(msg.event, msg.data);
        return;
      }
      const p = this.pending.get(msg.id);
      if (!p) return;
      this.pending.delete(msg.id);
      if (msg.error) p.reject(new Error(msg.error));
      else p.resolve(msg.result);
    });

    // Backend logs (and any stray subprocess output) land on stderr.
    readline.createInterface({ input: child.stderr }).on("line", (line) => {
      console.error("[qail-backend]", line);
    });

    // EPIPE when the child is gone; the exit/error handlers report it.
    child.stdin.on("error", () => {});

    const gone = (err) => {
      if (this.child !== child) return;
      this.child = null;
      if (!this.stopping) this.failAll(err);
    };
    // "error" covers spawn failures (e.g. ENOENT: binary missing), where
    // "exit" may never fire.
    child.on("error", gone);
    child.on("exit", (code, signal) =>
      gone(new Error(`qail backend exited (code ${code}, signal ${signal})`))
    );
  }

  failAll(err) {
    console.error("[qail-backend]", err.message);
    for (const p of this.pending.values()) p.reject(err);
    this.pending.clear();
  }

  call(method, args) {
    // Restart lazily if the backend died, so one crash doesn't brick the
    // tray app until the user quits it.
    if (!this.child && !this.stopping) this.start();
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.child.stdin.write(JSON.stringify({ id, method, args }) + "\n");
    });
  }

  onEvent(fn) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  stop() {
    this.stopping = true;
    if (this.child) {
      // Closing stdin lets the Go side drain in-flight calls and exit.
      this.child.stdin.end();
      const child = this.child;
      setTimeout(() => child.kill(), 3000).unref();
    }
  }
}

module.exports = { Backend };
