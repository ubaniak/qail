// Root shell. TopBar fixed at the top (drag region), then either the
// tab pane (Workspaces / Repos / Tmux) or the Settings overlay. Ctrl/
// Cmd + 1/2/3 jumps between tabs; the active tab persists across
// reloads via localStorage so reopening the app lands where the user
// left off.

import { SettingOutlined } from "@ant-design/icons";
import { Alert } from "antd";
import { useEffect, useMemo, useState } from "react";
import ProgressDrawer from "./component/Toolbox/ProgressDrawer";
import TopBar from "./component/Toolbox/TopBar";
import ToolboxTabs from "./component/Toolbox/Tabs";
import type { ToolboxTab } from "./component/Toolbox/Tabs";
import { PostInstallIndex } from "./pages/PostInstall";
import { RepoIndex } from "./pages/Repo";
import { SettingsIndex } from "./pages/Settings";
import { TmuxIndex } from "./pages/Tmux";
import { WorkspaceIndex } from "./pages/Workspace";
import { useQailService } from "./providers/qailservice";

const STORAGE_KEY = "qail.tabKey";

function App() {
  const { list } = useQailService();
  const numWorkspaces = Object.keys(list.workspaces).length;
  const numRepos = Object.keys(list.repos).length;
  const numTmux = list.tmux.length;
  const needsSetup =
    !list.settings.root || list.settings.editors.length === 0;

  const tabs: ToolboxTab[] = useMemo(
    () => [
      {
        key: "workspaces",
        label: `Workspaces (${numWorkspaces})`,
        component: <WorkspaceIndex />,
        shortcut: "⌘1",
      },
      {
        key: "repos",
        label: `Repos (${numRepos})`,
        component: <RepoIndex />,
        shortcut: "⌘2",
      },
      {
        key: "tmux",
        label: `Tmux (${numTmux})`,
        component: <TmuxIndex />,
        shortcut: "⌘3",
      },
      {
        key: "postinstall",
        label: "Post-install",
        component: <PostInstallIndex />,
        shortcut: "⌘4",
      },
    ],
    [numWorkspaces, numRepos, numTmux]
  );

  const initial =
    (typeof window !== "undefined" && localStorage.getItem(STORAGE_KEY)) ||
    tabs[0].key;
  const [activeKey, setActiveKey] = useState<string>(initial);
  const [showSettings, setShowSettings] = useState(false);

  const onTabChange = (k: string) => {
    setActiveKey(k);
    localStorage.setItem(STORAGE_KEY, k);
  };

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (!(e.ctrlKey || e.metaKey)) return;
      const n = parseInt(e.key, 10);
      if (Number.isNaN(n) || n < 1 || n > tabs.length) return;
      e.preventDefault();
      onTabChange(tabs[n - 1].key);
      setShowSettings(false);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <div className="flex flex-col h-full">
      <TopBar
        onSettingsClick={() => setShowSettings((s) => !s)}
        settingsActive={showSettings}
      />
      <main className="flex-1 mt-14 min-h-0 overflow-hidden flex flex-col">
        {showSettings ? (
          <SettingsIndex />
        ) : (
          <>
            {needsSetup && (
              <Alert
                type="warning"
                showIcon
                icon={<SettingOutlined />}
                className="!rounded-none !border-x-0 !border-t-0"
                message="Configure workspace root and editor"
                description={
                  !list.settings.root && list.settings.editors.length === 0
                    ? "Set a workspace root and add at least one editor to get started."
                    : !list.settings.root
                      ? "Set a workspace root to get started."
                      : "Add at least one editor to open workspaces."
                }
                action={
                  <button
                    type="button"
                    className="px-3 py-1 text-xs font-medium rounded bg-amber-500/20 text-amber-200 hover:bg-amber-500/30 border border-amber-500/40"
                    onClick={() => setShowSettings(true)}
                  >
                    Open Settings
                  </button>
                }
              />
            )}
            <div className="flex-1 min-h-0">
              <ToolboxTabs
                tabs={tabs}
                activeKey={activeKey}
                onChange={onTabChange}
              />
            </div>
          </>
        )}
      </main>
      <ProgressDrawer />
    </div>
  );
}

export default App;
