// QailServiceProvider — central read+write surface for the whole UI.
// Wraps the four list hooks (workspaces/repos/settings/tmux) and exposes
// mutate functions that refresh the right list on success.
//
// Mirrors qail_ui's QailServiceProvider so pages port over with minimal
// edits; the only adapter is that workspace-creating mutations also
// open the progress drawer (Wails event stream).

import { createContext, useContext } from "react";
import type { ReactNode } from "react";
import { Spin } from "antd";
import { Clipboard } from "@wailsio/runtime";

import {
  mutateAddEditor,
  mutateAddRepo,
  mutateCdWorkspace,
  mutateCreateWorkspace,
  mutateEditWorkspace,
  mutateMuxWorkspace,
  mutateOpenWorkspace,
  mutateRemoveEditor,
  mutateRemoveRepo,
  mutateRemoveTmux,
  mutateRemoveWorkspace,
  mutateSaveRoot,
  mutateSetDefaultEditor,
  mutateSetWorkspaceEditor,
  useListRepos,
  useListSettings,
  useListTmux,
  useListWorkspaces,
} from "../api/qail";
import type { RepoMap, Settings, WorkspaceMap } from "../types";
import { useNotification } from "./notification";
import { useProgress } from "./progress";

export type QailServiceShape = {
  list: {
    workspaces: WorkspaceMap;
    repos: RepoMap;
    settings: Settings;
    tmux: string[];
  };
  create: {
    workspace: (name: string, repos: string[]) => void;
    repo: (name: string, repoUrl: string) => void;
  };
  edit: {
    workspace: (name: string, repos: string[]) => void;
  };
  remove: {
    workspace: (name: string) => void;
    repo: (name: string) => void;
    tmux: (name: string) => void;
  };
  open: {
    workspace: (name: string, editor?: string) => void;
  };
  mux: {
    attach: (name: string) => void;
    refresh: () => void;
  };
  cd: {
    workspace: (name: string) => void;
  };
  save: {
    root: (root: string) => void;
  };
  editors: {
    add: (name: string, command: string) => void;
    remove: (name: string) => void;
    setDefault: (name: string) => void;
    setWorkspace: (workspace: string, name: string) => void;
  };
};

const QailServiceContext = createContext<QailServiceShape | null>(null);

// Wails v3 routes Clipboard.SetText through the native runtime, which is
// the only reliable path inside the WKWebView — navigator.clipboard
// silently fails because the webview origin isn't a secure context.
async function copyToClipboard(text: string): Promise<void> {
  try {
    await Clipboard.SetText(text);
  } catch {
    /* swallow — UI already toasted success; surfacing a copy error is
       worse than a missed clipboard write in this flow. */
  }
}

export const QailServiceProvider = ({ children }: { children: ReactNode }) => {
  const { toast } = useNotification();
  const progress = useProgress();

  const ws = useListWorkspaces();
  const repos = useListRepos();
  const settings = useListSettings();
  const tmux = useListTmux();

  const value: QailServiceShape = {
    list: {
      workspaces: ws.result,
      repos: repos.result,
      settings: settings.result,
      tmux: tmux.result,
    },
    create: {
      workspace: (name, selected) => {
        progress.start(`Create workspace: ${name}`);
        mutateCreateWorkspace(
          name,
          selected,
          () => {
            ws.refresh();
            toast.success(`Workspace "${name}" created`);
          },
          (e) => toast.error(`Create failed: ${e.message}`)
        );
      },
      repo: (name, repoUrl) => {
        mutateAddRepo(
          name,
          repoUrl,
          () => {
            repos.refresh();
            toast.success(`Repo "${name}" added`);
          },
          (e) => toast.error(`Add repo failed: ${e.message}`)
        );
      },
    },
    edit: {
      workspace: (name, selected) => {
        progress.start(`Edit workspace: ${name}`);
        mutateEditWorkspace(
          name,
          selected,
          () => {
            ws.refresh();
            toast.success(`Workspace "${name}" updated`);
          },
          (e) => toast.error(`Edit failed: ${e.message}`)
        );
      },
    },
    remove: {
      workspace: (name) =>
        mutateRemoveWorkspace(
          name,
          () => {
            ws.refresh();
            toast.success(`Workspace "${name}" removed`);
          },
          (e) => toast.error(`Remove failed: ${e.message}`)
        ),
      repo: (name) =>
        mutateRemoveRepo(
          name,
          () => {
            repos.refresh();
            toast.success(`Repo "${name}" removed`);
          },
          (e) => toast.error(`Remove failed: ${e.message}`)
        ),
      tmux: (name) =>
        mutateRemoveTmux(
          name,
          () => {
            tmux.refresh();
            toast.success(`Session "${name}" killed`);
          },
          (e) => toast.error(`Kill failed: ${e.message}`)
        ),
    },
    open: {
      workspace: (name, editor) =>
        mutateOpenWorkspace(
          name,
          () =>
            toast.success(
              editor
                ? `Opening "${name}" in ${editor}`
                : `Opening "${name}" in editor`
            ),
          (e) => toast.error(`Open failed: ${e.message}`),
          editor
        ),
    },
    mux: {
      attach: (name) =>
        mutateMuxWorkspace(
          name,
          (cmd) => {
            void copyToClipboard(cmd);
            toast.success("Tmux attach command copied to clipboard");
            tmux.refresh();
          },
          (e) => toast.error(`Mux failed: ${e.message}`)
        ),
      refresh: () => tmux.refresh(),
    },
    cd: {
      workspace: (name) =>
        mutateCdWorkspace(
          name,
          (path) => {
            void copyToClipboard(`cd ${path}`);
            toast.success("cd command copied to clipboard");
          },
          (e) => toast.error(`Cd failed: ${e.message}`)
        ),
    },
    save: {
      root: (root) =>
        mutateSaveRoot(
          root,
          () => {
            settings.refresh();
            toast.success("Root saved");
          },
          (e) => toast.error(`Save failed: ${e.message}`)
        ),
    },
    editors: {
      add: (name, command) =>
        mutateAddEditor(
          name,
          command,
          () => {
            settings.refresh();
            toast.success(`Editor "${name}" added`);
          },
          (e) => toast.error(`Add editor failed: ${e.message}`)
        ),
      remove: (name) =>
        mutateRemoveEditor(
          name,
          () => {
            settings.refresh();
            ws.refresh();
            toast.success(`Editor "${name}" removed`);
          },
          (e) => toast.error(`Remove editor failed: ${e.message}`)
        ),
      setDefault: (name) =>
        mutateSetDefaultEditor(
          name,
          () => {
            settings.refresh();
            toast.success(`Default editor set to "${name}"`);
          },
          (e) => toast.error(`Set default failed: ${e.message}`)
        ),
      setWorkspace: (workspace, name) =>
        mutateSetWorkspaceEditor(
          workspace,
          name,
          () => {
            ws.refresh();
            toast.success(
              name
                ? `${workspace} → editor "${name}"`
                : `${workspace} → inherits default`
            );
          },
          (e) => toast.error(`Update failed: ${e.message}`)
        ),
    },
  };

  const loading =
    ws.loading || repos.loading || settings.loading || tmux.loading;
  const firstError =
    ws.error || repos.error || settings.error || tmux.error;

  if (firstError && !loading) {
    // Surface the error once; the UI is still usable with empty lists.
    // Don't toast here (would fire on every re-render); pages render
    // an empty state instead.
  }

  return (
    <QailServiceContext.Provider value={value}>
      {loading ? (
        <div className="flex items-center justify-center h-full">
          <Spin />
        </div>
      ) : (
        children
      )}
    </QailServiceContext.Provider>
  );
};

export const useQailService = (): QailServiceShape => {
  const ctx = useContext(QailServiceContext);
  if (!ctx) {
    throw new Error("useQailService outside QailServiceProvider");
  }
  return ctx;
};
