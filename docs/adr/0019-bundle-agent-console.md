# ADR-0019: Bundle the Agent Console and a Node runtime in the installers

**Status**: Accepted

## Context

ADR-0018 shipped the Agent Console as a Node sidecar that users install
from a source checkout with `make console-install`, running on whatever
Node is on the machine. Desktop users get a dmg/exe/deb/AppImage and have
neither a checkout nor necessarily Node, so the tray's *Agent Console*
menu said "Not installed". The console should work straight after
installing qail.

Two things have to travel with the app:

1. **The console's dependencies.** All pure JS except the Claude Agent
   SDK, which runs Claude Code as a native binary from a per-platform
   optional package (`@anthropic-ai/claude-agent-sdk-<os>-<arch>`,
   ~350 MB). The installer must carry the one matching its target.
2. **A JavaScript runtime** for `server.mjs`.

## Decision

1. **`scripts/bundle-console.mjs`** (`make console-bundle`) stages the
   console for one target into `build/console/<os>-<arch>/console`:
   sources, `npm ci --omit=dev --os=<os> --cpu=<arch>` (so cross-targets
   such as the arm64 dmg or the Windows exe built on an Intel Mac get the
   right SDK binary), other platforms' SDK packages pruned, and a pinned
   Node (`runtime/node[.exe]`, checksum-verified against nodejs.org's
   `SHASUMS256.txt`, cached in `build/cache/`). Every `installer-*` make
   target runs it for its own target.
2. **electron-builder** copies `build/console/${os}-${arch}` into
   `resources/`, giving `resources/console` beside `resources/bin/qail`.
   (The extra directory level is needed: electron-builder always drops a
   `node_modules` at the root of an extraResources `from`.)
3. **The Go side finds it** (`console.ResolveDir`): `$QAIL_CONSOLE_DIR`,
   else `../console` relative to the running binary's directory
   (symlinks resolved, so the macOS `/usr/local/bin/qail` link works),
   else `~/.qail/console`. The same rule covers `~/.qail/bin/qail` with a
   `make console-install` console and `bin/qail` with the repo's
   `console/` in development. Node is `QAIL_NODE`, else
   `<console>/runtime/node`, else the system Node (ADR-0018's lookup).
4. **Chat threads live in `~/.qail/console/data`** (`AGENT_CONSOLE_DATA`)
   whichever install runs: app resources are read-only and replaced on
   update, and this is where `make console-install` users already have
   them.
5. **The bundled Node stays off the agent's PATH.** Only a system Node's
   directory is prepended, as before; otherwise an agent running `npm
   test` in a user's project would get our Node instead of theirs.

## Consequences

- Installers grow by roughly 500 MB unpacked (Claude Code binary
  ~350 MB, Node ~90–125 MB, other deps ~65 MB); compressed installers
  grow less.
- The app's Electron fuses stay locked down (`runAsNode: false`).
- Building an installer needs network access to npm and nodejs.org (or a
  warm `build/cache/`), plus Node and npm on the build host — already
  true for the frontend build.
- The console's SDK version is whatever `console/package-lock.json`
  pins, so updating Claude Code in the app means bumping the lockfile
  and cutting a release.
- macOS signing: the bundled `node` and `claude` binaries are signed by
  their vendors; the app itself is still unsigned, as before.

## Alternatives considered

- **Run `server.mjs` with Electron's own Node (`ELECTRON_RUN_AS_NODE`).**
  Saves the ~100 MB Node download, but needs the `runAsNode` fuse turned
  back on, which lets any local program use the app binary as a
  general-purpose Node interpreter (undoing the v2.1.1 lockdown), and
  ties `qail console` from a terminal to finding the Electron executable.
- **Electron `utilityProcess`.** Works with the fuse off, but moves
  supervision from Go into Electron, so `qail console` would still need a
  system Node.
- **Use the user's installed `claude` CLI instead of bundling it.** Much
  smaller, but needs a separate Claude Code install and ties the console
  to whatever CLI version is on PATH, which is the extra step this ADR
  removes.
