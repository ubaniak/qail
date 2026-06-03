// DTOs mirror internal/app/bindings.go. Names match the JSON tags so a
// raw fetch from the bound methods deserializes straight into these.
// When the Go side grows, regenerate via `wails generate module` and
// replace by hand.

export namespace models {
  export interface WorkspaceDTO {
    repos: string[];
    lastUsed: string; // ISO-8601 from time.Time JSON marshal
    editor?: string;
    postInstall?: string[];
  }

  export interface EditorDTO {
    name: string;
    command: string;
  }

  export interface ConfigDTO {
    root: string;
    editors: EditorDTO[];
    defaultEditor: string;
    workspaces: Record<string, WorkspaceDTO>;
  }

  export interface RepoDTO {
    url: string;
    postInstall?: string[];
  }

  export interface OpenCommandDTO {
    editor: string;
    path: string;
    command: string;
  }

  export interface DetectedRepoDTO {
    dir: string;
    remoteUrl: string;
    match: string;
  }

  export interface OrphanInspectionDTO {
    path: string;
    repos: DetectedRepoDTO[];
    preservedScripts: string[];
  }
}

export type WorkspaceMap = Record<string, models.WorkspaceDTO>;
export type RepoMap = Record<string, models.RepoDTO>;

export type Settings = {
  root: string;
  editors: models.EditorDTO[];
  defaultEditor: string;
};

// Scope mirrors scripts.Scope in the Go backend. Every scripts binding
// takes the string form of this; the constants exist so callers never
// have to remember the literal.
export type Scope = "workspace" | "repo";
export const SCOPE_WORKSPACE: Scope = "workspace";
export const SCOPE_REPO: Scope = "repo";
