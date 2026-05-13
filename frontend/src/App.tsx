// Root shell. TopBar fixed at the top (drag region), then either the
// tab pane (Workspaces / Repos / Tmux) or the Settings overlay. Ctrl/
// Cmd + 1/2/3 jumps between tabs; the active tab persists across
// reloads via localStorage so reopening the app lands where the user
// left off.

import { useEffect, useState } from "react";
import ProgressDrawer from "./component/Toolbox/ProgressDrawer";
import TopBar from "./component/Toolbox/TopBar";
import ToolboxTabs from "./component/Toolbox/Tabs";
import type { ToolboxTab } from "./component/Toolbox/Tabs";
import { RepoIndex } from "./pages/Repo";
import { SettingsIndex } from "./pages/Settings";
import { TmuxIndex } from "./pages/Tmux";
import { WorkspaceIndex } from "./pages/Workspace";

const STORAGE_KEY = "qail.tabKey";

const tabs: ToolboxTab[] = [
  {
    key: "workspaces",
    label: "Workspaces",
    component: <WorkspaceIndex />,
    shortcut: "⌘1",
  },
  { key: "repos", label: "Repos", component: <RepoIndex />, shortcut: "⌘2" },
  { key: "tmux", label: "Tmux", component: <TmuxIndex />, shortcut: "⌘3" },
];

function App() {
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
      <main className="flex-1 mt-14 min-h-0 overflow-hidden">
        {showSettings ? (
          <SettingsIndex />
        ) : (
          <ToolboxTabs
            tabs={tabs}
            activeKey={activeKey}
            onChange={onTabChange}
          />
        )}
      </main>
      <ProgressDrawer />
    </div>
  );
}

export default App;
