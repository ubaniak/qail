# Context — qail

Domain glossary for `qail` (pronounced "kyle"). Use these terms exactly. Add new terms here when sharpened during design discussions.

## Core domain

**Workspace**
A named group of repos cloned together into a single directory under the user's `Root`. The unit of "open this set of services in my editor / tmux." A workspace has a name, a list of packages (selected subset of registered repos), and last-used timestamp.

**Repo**
A registered git repository: name + clone URL. Lives in the global registry; referenced by name from workspaces. Does not imply a clone exists on disk — a repo is just the registration. Cloning happens during workspace creation.

**Package**
A repo *as included in a particular workspace*. Same repo can be a package of multiple workspaces. The slot, not the thing.

**Root**
The directory under which workspace directories are created. Configured once via `init` or `config root`. All cloned repos live at `<Root>/<workspace-name>/<repo-name>`.

**Post-install script**
A bash script run after a clone completes. Two scopes:
- *Repo-scoped*: runs once per repo clone, in that repo's directory.
- *Workspace-scoped*: runs once per workspace creation, in the workspace directory after all repos are cloned.

Stored as files in `~/.qail/scripts/`; attached to repos or workspaces by name.

**Editor**
The configured external command used to open a workspace directory (e.g. `code`, `cursor`, `nvim`). Stored as a single string.

## Persistence

**Store**
The interface for reading and writing the entire `Config` snapshot. Two adapters: `SQLiteStore` (production, GORM-backed at `~/.qail/qail.db`) and `MemoryStore` (tests).

**Config snapshot**
The full state read from / written to the `Store` in one go: root, editor, repos, workspaces, post-install script attachments. Writes replace the entire snapshot in a transaction (delete-all + re-insert). The `Config` struct is the in-memory shape of one snapshot.

**qailhome**
The well-known paths module. Single source of truth for `~/.qail/` layout: DB path, scripts dir. Honors `$QAIL_HOME` override.

## Command semantics

**Action**
An atomic command-level operation that wraps read-modify-write through a `Store` (e.g. `AddRepo`, `RemoveRepos`, `AddWorkspace`, `TouchWorkspace`, `SetEditor`). Lives in `internal/actions`. The full set of cmd handlers route through actions; the older `HandleConfig` callback pattern has been retired.

**Touch (a workspace)**
Update a workspace's `LastUsed` timestamp. Used by `open`, `cd`, `mux` — every flow that does a side effect on a workspace also touches it. `actions.TouchWorkspace` is the atomic write; the caller performs the side effect using the resolved profile/path.

## Subprocess and TUI

**Runner**
The subprocess seam. Interface that owns "execute a command line, return result." `OS` adapter shells out via `os/exec`; `Recorder` adapter captures invocations for tests. Used by `git`, `tmux`, `scripts`, and any module that runs an external binary.

**Layout (tmux)**
The pane/window arrangement for a workspace's tmux session. Computed by the pure function `tmux.Plan(sessionName, root, subfolders) → []runner.Command`. Layout policy lives in `Plan`; execution lives in `Tmux.Launch`.

**Form**
A charmbracelet/huh TUI prompt. Lives in `internal/forms`. Forms return values (selection result, confirmation bool, error) and never mutate caller state. The caller (handler or action) applies the change.

**Installer**
The per-package install orchestrator. Given a `PackageSpec` (name, repo URL, dest, post-install scripts), runs clone then post-install. Wired with `GitClient` and `ScriptsClient` interfaces. Injected into `Workspace` via the consumer-defined `workspace.Installer` interface.

**FS (filesystem seam)**
The narrow filesystem interface workspace methods use to mkdir, remove, stat, and list entries. `workspace.OSFS{}` is the production adapter wrapping the os package; tests substitute a fake. Defined in the consumer (`workspace`) so only the operations workspace performs are exposed.
