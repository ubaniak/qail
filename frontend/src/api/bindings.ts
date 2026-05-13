// Thin typed wrapper around the Wails v3 generated bindings under
// frontend/bindings/. The rest of the app imports `requireApi()` and
// calls methods on the returned object, exactly as it did under v2 with
// window.go.app.Bindings.<Method>. The wrapper:
//
//   - turns CancellablePromise return values into plain Promise so the
//     existing `await` callers don't need to change,
//   - lets api/qail.ts keep its hand-typed shape independent of the
//     generated d.ts files (regenerated bindings can change without
//     rippling through every callsite).
//
// If a method is added on the Go side: run `wails3 generate bindings`,
// then add it here so the type-check covers the call site.

import * as B from "../../bindings/github.com/ubaniak/qail/internal/app/bindings.js";
import type { models, WorkspaceMap, RepoMap } from "../types";

export type Bindings = {
  GetConfig: () => Promise<models.ConfigDTO>;
  SetRoot: (value: string) => Promise<void>;
  SetEditor: (value: string) => Promise<void>;

  ListRepos: () => Promise<RepoMap>;
  AddRepo: (name: string, url: string) => Promise<void>;
  RemoveRepos: (names: string[]) => Promise<void>;
  SetRepoPostInstall: (repo: string, scripts: string[]) => Promise<void>;

  ListWorkspaces: () => Promise<WorkspaceMap>;
  AddWorkspace: (name: string, packages: string[]) => Promise<void>;
  EditWorkspace: (name: string, packages: string[]) => Promise<void>;
  CloneWorkspace: (dst: string, packages: string[]) => Promise<void>;
  CreateWorkspaceOnDisk: (name: string) => Promise<void>;
  RemoveWorkspace: (name: string) => Promise<void>;
  SetWorkspacePostInstall: (name: string, scripts: string[]) => Promise<void>;
  CdWorkspace: (name: string) => Promise<string>;
  AttachCommand: (name: string) => Promise<string>;
  OpenCommand: (name: string) => Promise<models.OpenCommandDTO>;
  OpenEditor: (name: string) => Promise<void>;
  ExplorePath: (name: string) => Promise<string>;

  ListScripts: () => Promise<string[]>;
  AddScript: (name: string) => Promise<void>;
  RemoveScript: (name: string) => Promise<void>;
  ScriptsDir: () => Promise<string>;

  ListMuxSessions: () => Promise<string[]>;
  RemoveMuxSession: (name: string) => Promise<void>;
};

// Hash maps generated CancellablePromise<T> → Promise<T> by wrapping in
// async. We don't need cancellation anywhere in the UI yet.
const wrap = <T,>(p: PromiseLike<T>): Promise<T> => Promise.resolve(p);

export const api: Bindings = {
  GetConfig: () => wrap(B.GetConfig()) as Promise<models.ConfigDTO>,
  SetRoot: (v) => wrap(B.SetRoot(v)),
  SetEditor: (v) => wrap(B.SetEditor(v)),

  ListRepos: () => wrap(B.ListRepos()) as Promise<RepoMap>,
  AddRepo: (name, url) => wrap(B.AddRepo(name, url)),
  RemoveRepos: (names) => wrap(B.RemoveRepos(names)),
  SetRepoPostInstall: (repo, s) => wrap(B.SetRepoPostInstall(repo, s)),

  ListWorkspaces: () => wrap(B.ListWorkspaces()) as Promise<WorkspaceMap>,
  AddWorkspace: (name, pkgs) => wrap(B.AddWorkspace(name, pkgs)),
  EditWorkspace: (name, pkgs) => wrap(B.EditWorkspace(name, pkgs)),
  CloneWorkspace: (dst, pkgs) => wrap(B.CloneWorkspace(dst, pkgs)),
  CreateWorkspaceOnDisk: (name) => wrap(B.CreateWorkspaceOnDisk(name)),
  RemoveWorkspace: (name) => wrap(B.RemoveWorkspace(name)),
  SetWorkspacePostInstall: (name, s) => wrap(B.SetWorkspacePostInstall(name, s)),
  CdWorkspace: (name) => wrap(B.CdWorkspace(name)),
  AttachCommand: (name) => wrap(B.AttachCommand(name)),
  OpenCommand: (name) => wrap(B.OpenCommand(name)) as Promise<models.OpenCommandDTO>,
  OpenEditor: (name) => wrap(B.OpenEditor(name)),
  ExplorePath: (name) => wrap(B.ExplorePath(name)),

  ListScripts: () => wrap(B.ListScripts()),
  AddScript: (name) => wrap(B.AddScript(name)),
  RemoveScript: (name) => wrap(B.RemoveScript(name)),
  ScriptsDir: () => wrap(B.ScriptsDir()),

  ListMuxSessions: () => wrap(B.ListMuxSessions()),
  RemoveMuxSession: (name) => wrap(B.RemoveMuxSession(name)),
};

// requireApi mirrors the v2 contract: a single call site for every
// page. v3 doesn't need a runtime presence probe (the generated $Call
// throws on its own if not in Wails) so this is now a static accessor.
export function requireApi(): Bindings {
  return api;
}
