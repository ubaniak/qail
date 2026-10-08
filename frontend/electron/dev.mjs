// Dev loop: start the Vite dev server, then launch Electron against it
// with HMR. Expects a qail binary at ../bin/qail (`make build`), or set
// QAIL_BIN to point elsewhere.

import { spawn } from "node:child_process";
import { createRequire } from "node:module";
import { createServer } from "vite";

const require = createRequire(import.meta.url);
const electron = require("electron"); // resolves to the binary path

const server = await createServer();
await server.listen();
const url = server.resolvedUrls.local[0];

const child = spawn(electron, ["."], {
  stdio: "inherit",
  env: { ...process.env, QAIL_DEV_SERVER_URL: url },
});
child.on("exit", async (code) => {
  await server.close();
  process.exit(code ?? 0);
});
