// Typed surface of the Go Bindings (internal/app/bindings.go). The rest
// of the app imports `requireApi()` and calls methods on the returned
// object. Each method is forwarded by name over the Electron bridge to
// the `qail desktop-backend` process, which dispatches it to the Go
// method of the same name with the same positional arguments.
//
// If a method is added on the Go side, add it to the Bindings type here
// so the type-check covers the call site.

import { call } from "../desktop";
import type { models, WorkspaceMap, RepoMap, Scope } from "../types";

export type Bindings = {
  GetConfig: () => Promise<models.ConfigDTO>;
  SetRoot: (value: string) => Promise<void>;
  AddEditor: (name: string, command: string) => Promise<void>;
  RemoveEditor: (name: string) => Promise<void>;
  SetDefaultEditor: (name: string) => Promise<void>;
  SetWorkspaceEditor: (workspace: string, name: string) => Promise<void>;
  AddAI: (name: string, command: string) => Promise<void>;
  RemoveAI: (name: string) => Promise<void>;
  SetDefaultAI: (name: string) => Promise<void>;
  SetWorkspaceAI: (workspace: string, name: string) => Promise<void>;

  ListRepos: () => Promise<RepoMap>;
  AddRepo: (name: string, url: string) => Promise<void>;
  RemoveRepos: (names: string[]) => Promise<void>;
  SetRepoPostInstall: (repo: string, scripts: string[]) => Promise<void>;

  ListWorkspaces: () => Promise<WorkspaceMap>;
  AddWorkspace: (name: string, packages: string[], postInstall: string[]) => Promise<void>;
  EditWorkspace: (name: string, packages: string[]) => Promise<void>;
  CloneWorkspace: (dst: string, packages: string[]) => Promise<void>;
  CreateWorkspaceOnDisk: (name: string) => Promise<void>;
  RemoveWorkspace: (name: string) => Promise<void>;
  ListOrphanWorkspaces: () => Promise<string[]>;
  RemoveOrphanWorkspaces: (names: string[]) => Promise<void>;
  OrphanPath: (name: string) => Promise<string>;
  OpenOrphanEditor: (name: string) => Promise<void>;
  OpenOrphanAI: (name: string) => Promise<void>;
  InspectOrphan: (name: string) => Promise<models.OrphanInspectionDTO>;
  RestoreWorkspace: (name: string, repos: string[]) => Promise<void>;
  SetWorkspacePostInstall: (name: string, scripts: string[]) => Promise<void>;
  CdWorkspace: (name: string) => Promise<string>;
  AttachCommand: (name: string) => Promise<string>;
  OpenCommand: (name: string) => Promise<models.OpenCommandDTO>;
  OpenCommandWith: (name: string, editor: string) => Promise<models.OpenCommandDTO>;
  OpenEditor: (name: string) => Promise<void>;
  OpenEditorWith: (name: string, editor: string) => Promise<void>;
  OpenAICommand: (name: string) => Promise<models.OpenAICommandDTO>;
  OpenAICommandWith: (name: string, ai: string) => Promise<models.OpenAICommandDTO>;
  OpenAI: (name: string) => Promise<void>;
  OpenAIWith: (name: string, ai: string) => Promise<void>;
  ExplorePath: (name: string) => Promise<string>;

  ListScripts: (scope: Scope) => Promise<string[]>;
  AddScript: (name: string, scope: Scope) => Promise<void>;
  RemoveScript: (name: string, scope: Scope) => Promise<void>;
  ReadScript: (name: string, scope: Scope) => Promise<string>;
  WriteScript: (name: string, scope: Scope, contents: string) => Promise<void>;
  ScriptsDir: () => Promise<string>;
  RunWorkspaceScript: (workspace: string, script: string) => Promise<void>;
  RunRepoScript: (workspace: string, repo: string, script: string) => Promise<void>;

  ListMuxSessions: () => Promise<string[]>;
  RemoveMuxSession: (name: string) => Promise<void>;
};

// Every property access yields a function that forwards its arguments to
// the Go method of the same name. The Bindings type above is what keeps
// call sites honest.
export const api = new Proxy({} as Bindings, {
  get: (_target, method) =>
    typeof method === "string"
      ? (...args: unknown[]) => call(method, args)
      : undefined,
});

// requireApi is the single accessor every page uses.
export function requireApi(): Bindings {
  return api;
}
