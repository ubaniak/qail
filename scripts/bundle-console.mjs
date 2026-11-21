#!/usr/bin/env node
// bundle-console — stages the Agent Console (ADR-0018) for one installer
// target, so the app runs it with no `make console-install` and no system
// Node (ADR-0019).
//
//   node scripts/bundle-console.mjs [--platform darwin|linux|win32] [--arch x64|arm64]
//
// Defaults to the host. Output, picked up by electron-builder's
// extraResources as resources/console. The extra console/ level matters:
// electron-builder always drops a node_modules at the root of a FileSet's
// "from", but keeps nested ones.
//
//   build/console/<os>-<arch>/console/  os = mac | linux | win (electron-builder's ${os})
//     server.mjs, public/, package.json, package-lock.json
//     node_modules/                     production deps, plus the Claude Agent SDK's
//                                       native CLI package for this target only
//     runtime/node[.exe], runtime/LICENSE   pinned Node.js, checksum-verified
//
// Cross-targets work from any host: npm's --os/--cpu pick the SDK's
// platform package, and the console's other deps are pure JS. Node
// downloads are cached in build/cache/.

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Pinned runtime. Bump deliberately; the checksum is checked against
// nodejs.org's SHASUMS256.txt for this exact version.
const NODE_VERSION = "22.23.5";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const src = path.join(root, "console");

function arg(name, fallback) {
  const i = process.argv.indexOf(`--${name}`);
  return i > 0 && process.argv[i + 1] ? process.argv[i + 1] : fallback;
}
const platform = arg("platform", process.platform);
const arch = arg("arch", process.arch);

const ebOS = { darwin: "mac", linux: "linux", win32: "win" }[platform];
if (!ebOS) throw new Error(`unsupported platform ${platform}`);
if (!["x64", "arm64"].includes(arch)) throw new Error(`unsupported arch ${arch}`);

const out = path.join(root, "build", "console", `${ebOS}-${arch}`, "console");
const cache = path.join(root, "build", "cache");
const sdkPkg = `claude-agent-sdk-${platform}-${arch}`;
const npm = process.platform === "win32" ? "npm.cmd" : "npm";

console.log(`bundle-console: ${platform}-${arch} → ${path.relative(root, out)}`);

// 1. Console sources. Never copy a dev's node_modules or chat data.
fs.rmSync(out, { recursive: true, force: true });
fs.mkdirSync(out, { recursive: true });
for (const f of ["server.mjs", "package.json", "package-lock.json", "public"]) {
  fs.cpSync(path.join(src, f), path.join(out, f), { recursive: true });
}

// 2. Production deps for the target. --os/--cpu make npm install the SDK's
// native CLI package for the target instead of the host's.
execFileSync(
  npm,
  ["ci", "--omit=dev", "--no-audit", "--no-fund", `--os=${platform}`, `--cpu=${arch}`],
  { cwd: out, stdio: "inherit", shell: process.platform === "win32" }
);

// npm also installs the musl twin on Linux (and leaves room for others);
// keep only this target's ~350 MB CLI binary.
const scope = path.join(out, "node_modules", "@anthropic-ai");
for (const name of fs.readdirSync(scope)) {
  if (name.startsWith("claude-agent-sdk-") && name !== sdkPkg) {
    fs.rmSync(path.join(scope, name), { recursive: true, force: true });
  }
}
const cli = path.join(scope, sdkPkg, platform === "win32" ? "claude.exe" : "claude");
if (!fs.existsSync(cli)) throw new Error(`missing Claude CLI for ${platform}-${arch}: ${cli}`);

// 3. Node runtime.
const runtime = path.join(out, "runtime");
fs.mkdirSync(runtime, { recursive: true });
const dist = `https://nodejs.org/dist/v${NODE_VERSION}`;
const base = `node-v${NODE_VERSION}-${platform === "win32" ? "win" : platform}-${arch}`;

async function download(file) {
  const dest = path.join(cache, `v${NODE_VERSION}`, file);
  if (fs.existsSync(dest)) return dest;
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  const res = await fetch(`${dist}/${file}`);
  if (!res.ok) throw new Error(`GET ${dist}/${file}: ${res.status}`);
  fs.writeFileSync(dest + ".part", Buffer.from(await res.arrayBuffer()));
  fs.renameSync(dest + ".part", dest);
  return dest;
}

async function verify(file, local) {
  const sums = fs.readFileSync(await download("SHASUMS256.txt"), "utf8");
  const want = sums.split("\n").find((l) => l.trim().endsWith(`  ${file}`))?.split(/\s+/)[0];
  const got = createHash("sha256").update(fs.readFileSync(local)).digest("hex");
  if (!want || want !== got) {
    fs.rmSync(local, { force: true });
    throw new Error(`checksum mismatch for ${file}: got ${got}, want ${want}`);
  }
}

if (platform === "win32") {
  const zip = `${base}.zip`;
  const local = await download(zip);
  await verify(zip, local);
  // bsdtar (macOS, Windows 10+) reads zips; GNU tar doesn't, so fall back to unzip.
  const staging = path.join(cache, "win-extract");
  fs.rmSync(staging, { recursive: true, force: true });
  fs.mkdirSync(staging, { recursive: true });
  const members = [`${base}/node.exe`, `${base}/LICENSE`];
  try {
    execFileSync("tar", ["-xf", local, "-C", staging, ...members], { stdio: "ignore" });
  } catch {
    execFileSync("unzip", ["-q", "-o", local, ...members, "-d", staging]);
  }
  fs.copyFileSync(path.join(staging, base, "node.exe"), path.join(runtime, "node.exe"));
  fs.copyFileSync(path.join(staging, base, "LICENSE"), path.join(runtime, "LICENSE"));
  fs.rmSync(staging, { recursive: true, force: true });
} else {
  const tgz = `${base}.tar.gz`;
  const local = await download(tgz);
  await verify(tgz, local);
  execFileSync("tar", ["-xzf", local, "-C", runtime, "--strip-components=1", `${base}/LICENSE`]);
  execFileSync("tar", ["-xzf", local, "-C", runtime, "--strip-components=2", `${base}/bin/node`]);
  fs.chmodSync(path.join(runtime, "node"), 0o755);
}

console.log(`bundle-console: done (${sdkPkg}, node v${NODE_VERSION})`);
