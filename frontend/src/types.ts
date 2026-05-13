// DTOs mirror internal/app/bindings.go. Names match the JSON tags so a
// raw fetch from the bound methods deserializes straight into these.
// When the Go side grows, regenerate via `wails generate module` and
// replace by hand.

export namespace models {
  export interface WorkspaceDTO {
    repos: string[];
    lastUsed: string; // ISO-8601 from time.Time JSON marshal
    postInstall?: string[];
  }

  export interface ConfigDTO {
    root: string;
    editor: string;
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
}

export type WorkspaceMap = Record<string, models.WorkspaceDTO>;
export type RepoMap = Record<string, models.RepoDTO>;

export type Settings = {
  root: string;
  editor: string;
};
