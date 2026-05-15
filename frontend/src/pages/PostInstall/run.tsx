// On-demand run sub-page. Lists every target the script is currently
// attached to (workspaces for workspace-scope scripts, workspace+repo
// pairs for repo-scope scripts) and fires the corresponding bindings
// method. Progress streams through the existing ProgressDrawer because
// the Go-side action funnels through the same custom-event channel.

import { useMemo } from "react";
import { Alert, Tag } from "antd";
import { FolderOpenOutlined, ThunderboltOutlined } from "@ant-design/icons";

import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import type { Scope } from "../../types";

export type RunScriptProps = {
  name: string;
  scope: Scope;
  onClose: () => void;
};

type Target = {
  id: string;
  workspace: string;
  repo?: string;
  label: string;
  secondary: string;
};

export const RunScript = ({ name, scope, onClose }: RunScriptProps) => {
  const { list, scripts: svc } = useQailService();

  const targets = useMemo<Target[]>(() => {
    if (scope === "workspace") {
      return Object.entries(list.workspaces)
        .filter(([, ws]) => (ws.postInstall ?? []).includes(name))
        .map(([n, ws]) => ({
          id: n,
          workspace: n,
          label: n,
          secondary: `${(ws.repos || []).length} repos`,
        }));
    }
    // Repo-scope: one target per (workspace, repo) pair where the
    // workspace includes the repo AND the repo has the script attached.
    const repoNames = Object.entries(list.repos)
      .filter(([, r]) => (r.postInstall ?? []).includes(name))
      .map(([n]) => n);
    const out: Target[] = [];
    for (const [wsName, ws] of Object.entries(list.workspaces)) {
      for (const r of ws.repos || []) {
        if (!repoNames.includes(r)) continue;
        out.push({
          id: `${wsName}/${r}`,
          workspace: wsName,
          repo: r,
          label: `${wsName} / ${r}`,
          secondary: `Runs in ${wsName}/${r}`,
        });
      }
    }
    return out;
  }, [list, scope, name]);

  const run = (t: Target) => {
    if (scope === "workspace") {
      svc.runWorkspace(t.workspace, name);
    } else if (t.repo) {
      svc.runRepo(t.workspace, t.repo, name);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
            <span>Run {name}</span>
            <Tag color={scope === "workspace" ? "blue" : "purple"}>{scope}</Tag>
          </div>
          <div className="text-zinc-500 text-xs">
            Pick a target to execute on demand.
          </div>
        </div>
        <QButton variant="cancel" onClick={onClose}>
          Back
        </QButton>
      </div>

      <div className="flex-1 min-h-0">
        <ScrollableLayout>
          {targets.length === 0 ? (
            <Alert
              type="info"
              showIcon
              message="No targets"
              description={`Attach "${name}" to a ${scope === "workspace" ? "workspace" : "repo"} first.`}
            />
          ) : (
            targets.map((t) => (
              <ToolboxListItem
                key={t.id}
                id={t.id}
                primary={t.label}
                secondary={t.secondary}
                icon={
                  scope === "workspace" ? (
                    <FolderOpenOutlined />
                  ) : (
                    <ThunderboltOutlined />
                  )
                }
                menuItems={[
                  {
                    label: "Run",
                    onClick: () => run(t),
                  },
                ]}
                onClick={() => run(t)}
              />
            ))
          )}
        </ScrollableLayout>
      </div>
    </div>
  );
};
