# ADR-0012: `qailhome` exposes only purpose-specific paths

**Status**: Accepted (refines ADR-0005)

## Context

ADR-0005 established `qailhome` as the single source of truth for the `~/.qail/` layout. The original `Home` struct exposed:

- `Root() string` — the home directory
- `DBPath() string`
- `ScriptsDir() string`

`Root()` was used in exactly one production caller, `internal/config/convert.go`, which computed `filepath.Join(defaultRootDir, "config.json")` for the legacy JSON migration. Everywhere else, callers used the purpose-specific accessors.

Exposing `Root()` was a quiet leak: it tempted future callers to bypass `qailhome` and compute their own paths under the home directory, defeating the "single source of truth" discipline.

## Decision

`Home.Root()` is unexported. The internal field `root` is still used by `Default()` for the bootstrap `MkdirAll`, but no external caller can ask for it.

A new purpose-specific accessor replaces the only legitimate use:

- `LegacyJSONPath() string` returns `<root>/config.json` for the `qail config convert` migration.

When a new well-known path appears, the answer is to add a new accessor on `Home`, not to expose the root and let callers compute paths themselves.

## Consequences

- `qailhome`'s public interface lists exactly the paths qail uses today. Adding a new file under `~/.qail/` is a deliberate one-line edit to `Home`, not an ad-hoc `filepath.Join` somewhere else.
- Tests had to drop `h.Root()` assertions; they assert on the purpose-specific accessors instead.
- The deepening test from LANGUAGE.md applies here: smaller interface, same body → more depth.

## Alternatives considered

- *Keep `Root()` but document it's for legacy use*: rejected — comments don't survive code review pressure. An unexported field can't be misused.
- *Move `LegacyJSONPath` to `convert.go`*: rejected — it's a path under `qailhome`'s root, so `qailhome` owns it.
