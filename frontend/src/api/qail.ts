// Typed wrappers around Wails Bindings for use with callService /
// mutateService. Maps qail_ui's QailService.* naming to qail Wails v2
// binding names so the rest of the UI stays close to the reference
// implementation.

import { requireApi } from "./bindings";
import { callService, mutateService } from "./hooks";
import type { CallServiceResult } from "./hooks";
import type {
  WorkspaceMap,
  RepoMap,
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
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().AddWorkspace(name, repos),
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
export const mutateOpenWorkspace = (
  name: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => requireApi().OpenEditor(name),
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

// ---------- settings --------------------------------------------------------

export const useListSettings = (): CallServiceResult<Settings> =>
  callService<Settings>(
    {
      call: async () => {
        const cfg = await requireApi().GetConfig();
        return { root: cfg.root, editor: cfg.editor };
      },
    },
    { root: "", editor: "" }
  );

export const mutateSaveSettings = (
  root: string,
  editor: string,
  onSuccess: () => void,
  onError?: (e: Error) => void
): void =>
  mutateService({
    call: async () => {
      await requireApi().SetRoot(root);
      await requireApi().SetEditor(editor);
    },
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
