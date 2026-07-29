// Typed wrappers around Wails Bindings for use with callService /
// mutateService. Maps qail_ui's QailService.* naming to qail Wails v2
// binding names so the rest of the UI stays close to the reference
// implementation.

import { requireApi } from "./bindings";
import { callService, mutateService } from "./hooks";
import type { CallServiceResult } from "./hooks";
import type {
  models,
  WorkspaceMap,
  RepoMap,
  Scope,
  Settings,
} from "../types";

// ---------- workspaces ------------------------------------------------------

export const useListWorkspaces = (): CallServiceResult<WorkspaceMap> =>
  callService<WorkspaceMap>(
    { call: async () => requireApi().ListWorkspaces() },
    {}
  );

export const mutateCreateWorkspace = (
  name: string,
  repos: string[],
  postInstall: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddWorkspace(name, repos, postInstall),
    onSuccess,
    onError,
  });

export const mutateEditWorkspace = (
  name: string,
  repos: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().EditWorkspace(name, repos),
    onSuccess,
    onError,
  });

export const mutateRemoveWorkspace = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveWorkspace(name),
    onSuccess,
    onError,
  });

// mutateOpenWorkspace launches the editor on the Go side and resolves
// when the spawn returns. The success callback no longer receives the
// command DTO — the UI just toasts; if a future caller needs the
// editor+path tuple it can fall back to requireApi().OpenCommand().
// Passing an editor name overrides the workspace/global default for this
// launch only.
export const mutateOpenWorkspace = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void,
  editor?: string
): void =>
  mutateService({
    call: async () =>
      editor
        ? requireApi().OpenEditorWith(name, editor)
        : requireApi().OpenEditor(name),
    onSuccess,
    onError,
  });

export const mutateSetWorkspaceEditor = (
  workspace: string,
  editor: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetWorkspaceEditor(workspace, editor),
    onSuccess,
    onError,
  });

// mutateOpenWorkspaceAI spawns a new terminal running the AI tool at the
// workspace path. Passing an ai name overrides the workspace/global
// default for this launch only.
export const mutateOpenWorkspaceAI = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void,
  ai?: string
): void =>
  mutateService({
    call: async () =>
      ai ? requireApi().OpenAIWith(name, ai) : requireApi().OpenAI(name),
    onSuccess,
    onError,
  });

export const mutateSetWorkspaceAI = (
  workspace: string,
  ai: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetWorkspaceAI(workspace, ai),
    onSuccess,
    onError,
  });

export const mutateMuxWorkspace = (
  name: string,
  onSuccess: (cmd: string) => void,
  onError?: (e: Error) => void
): void => {
  (async () => {
    try {
      const cmd = await requireApi().AttachCommand(name);
      onSuccess(cmd);
    } catch (err) {
      onError?.(err as Error);
    }
  })();
};

export const mutateCdWorkspace = (
  name: string,
  onSuccess: (path: string) => void,
  onError?: (e: Error) => void
): void => {
  (async () => {
    try {
      const path = await requireApi().CdWorkspace(name);
      onSuccess(path);
    } catch (err) {
      onError?.(err as Error);
    }
  })();
};

// ---------- repos -----------------------------------------------------------

export const useListRepos = (): CallServiceResult<RepoMap> =>
  callService<RepoMap>(
    { call: async () => requireApi().ListRepos() },
    {}
  );

export const mutateAddRepo = (
  name: string,
  url: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddRepo(name, url),
    onSuccess,
    onError,
  });

export const mutateRemoveRepo = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveRepos([name]),
    onSuccess,
    onError,
  });

// ---------- cleanup ---------------------------------------------------------

export const useListOrphanWorkspaces = (): CallServiceResult<string[]> =>
  callService<string[]>(
    { call: async () => requireApi().ListOrphanWorkspaces() },
    []
  );

export const mutateRemoveOrphanWorkspaces = (
  names: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveOrphanWorkspaces(names),
    onSuccess,
    onError,
  });

export const mutateOpenOrphanEditor = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().OpenOrphanEditor(name),
    onSuccess,
    onError,
  });

export const mutateOpenOrphanAI = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().OpenOrphanAI(name),
    onSuccess,
    onError,
  });

export const fetchOrphanInspection = async (
  name: string
): Promise<models.OrphanInspectionDTO> => requireApi().InspectOrphan(name);

export const mutateRestoreWorkspace = (
  name: string,
  repos: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RestoreWorkspace(name, repos),
    onSuccess,
    onError,
  });

// ---------- settings --------------------------------------------------------

export const useListSettings = (): CallServiceResult<Settings> =>
  callService<Settings>(
    {
      call: async () => {
        const cfg = await requireApi().GetConfig();
        return {
          root: cfg.root,
          editors: cfg.editors ?? [],
          defaultEditor: cfg.defaultEditor,
          ais: cfg.ais ?? [],
          defaultAI: cfg.defaultAI,
        };
      },
    },
    { root: "", editors: [], defaultEditor: "", ais: [], defaultAI: "" }
  );

export const mutateSaveRoot = (
  root: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetRoot(root),
    onSuccess,
    onError,
  });

export const mutateAddEditor = (
  name: string,
  command: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddEditor(name, command),
    onSuccess,
    onError,
  });

export const mutateRemoveEditor = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveEditor(name),
    onSuccess,
    onError,
  });

export const mutateSetDefaultEditor = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetDefaultEditor(name),
    onSuccess,
    onError,
  });

export const mutateAddAI = (
  name: string,
  command: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddAI(name, command),
    onSuccess,
    onError,
  });

export const mutateRemoveAI = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveAI(name),
    onSuccess,
    onError,
  });

export const mutateSetDefaultAI = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetDefaultAI(name),
    onSuccess,
    onError,
  });

// ---------- tmux ------------------------------------------------------------

export const useListTmux = (): CallServiceResult<string[]> =>
  callService<string[]>(
    {
      call: async () => {
        const sessions = await requireApi().ListMuxSessions();
        // Belt-and-suspenders: Go side now returns [] for empty cases,
        // but historical builds could return [""] from strings.Split on
        // empty stdout. Filter blanks so the UI never renders a phantom
        // row keyed on "".
        return Array.isArray(sessions) ? sessions.filter(Boolean) : [];
      },
    },
    []
  );

export const mutateRemoveTmux = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveMuxSession(name),
    onSuccess,
    onError,
  });

// ---------- scripts ---------------------------------------------------------

// Scripts are scoped: workspace or repo. The Wails binding accepts the
// scope as a string so the wrapper passes through directly. callService
// memoises by JSON-stringifying its second arg, so the scope flows in as
// part of the dependency key and the hook re-fetches on toggle.
export const useListScripts = (scope: Scope): CallServiceResult<string[]> =>
  callService<string[]>(
    { call: async () => requireApi().ListScripts(scope) },
    [],
    { scope }
  );

export const mutateAddScript = (
  name: string,
  scope: Scope,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddScript(name, scope),
    onSuccess,
    onError,
  });

export const mutateRemoveScript = (
  name: string,
  scope: Scope,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RemoveScript(name, scope),
    onSuccess,
    onError,
  });

export const fetchScriptContents = async (
  name: string,
  scope: Scope
): Promise<string> => requireApi().ReadScript(name, scope);

export const mutateWriteScript = (
  name: string,
  scope: Scope,
  contents: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().WriteScript(name, scope, contents),
    onSuccess,
    onError,
  });

export const mutateSetWorkspacePostInstall = (
  workspace: string,
  scripts: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetWorkspacePostInstall(workspace, scripts),
    onSuccess,
    onError,
  });

export const mutateSetRepoPostInstall = (
  repo: string,
  scripts: string[],
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().SetRepoPostInstall(repo, scripts),
    onSuccess,
    onError,
  });

export const mutateRunWorkspaceScript = (
  workspace: string,
  script: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RunWorkspaceScript(workspace, script),
    onSuccess,
    onError,
  });

export const mutateRunRepoScript = (
  workspace: string,
  repo: string,
  script: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().RunRepoScript(workspace, repo, script),
    onSuccess,
    onError,
  });
