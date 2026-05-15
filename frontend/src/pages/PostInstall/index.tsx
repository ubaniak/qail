// Post-install tab — two stacked panes (Workspace scripts / Repo
// scripts). Each row shows a script name, lets the user view its
// contents, attach it to one or more workspaces/repos, run it on
// demand, or remove it. The scope is a hard partition in the data
// model: a workspace-scope script cannot be attached to a repo and
// vice versa.

import { useState } from "react";
import { CodeOutlined, PlusOutlined } from "@ant-design/icons";
import { Alert, Segmented, Tag } from "antd";

import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import type { MenuAction } from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import type { Scope } from "../../types";
import { AddScript } from "./add";
import { AssignScript } from "./assign";
import { RunScript } from "./run";
import { ViewScript } from "./view";

type Sub =
  | {
      kind: "view";
      scope: "view";
      name: string;
      scriptScope: Scope;
      mode?: "view" | "edit";
    }
  | { kind: "add"; scope: Scope }
  | { kind: "assign"; scope: Scope; name: string }
  | { kind: "run"; scope: Scope; name: string }
  | null;

export const PostInstallIndex = () => {
  const { list, scripts: svc } = useQailService();
  const [scope, setScope] = useState<Scope>("workspace");
  const [query, setQuery] = useState("");
  const [sub, setSub] = useState<Sub>(null);

  if (sub?.kind === "add")
    return <AddScript scope={sub.scope} onClose={() => setSub(null)} />;
  if (sub?.kind === "view")
    return (
      <ViewScript
        name={sub.name}
        scope={sub.scriptScope}
        initialMode={sub.mode ?? "view"}
        onClose={() => setSub(null)}
      />
    );
  if (sub?.kind === "assign")
    return (
      <AssignScript
        name={sub.name}
        scope={sub.scope}
        onClose={() => setSub(null)}
      />
    );
  if (sub?.kind === "run")
    return (
      <RunScript
        name={sub.name}
        scope={sub.scope}
        onClose={() => setSub(null)}
      />
    );

  const all = scope === "workspace" ? list.scripts.workspace : list.scripts.repo;
  const filtered = all.filter((n) =>
    n.toLowerCase().includes(query.toLowerCase())
  );

  const attachedTo = (scriptName: string): string[] => {
    if (scope === "workspace") {
      return Object.entries(list.workspaces)
        .filter(([, ws]) => (ws.postInstall ?? []).includes(scriptName))
        .map(([n]) => n);
    }
    return Object.entries(list.repos)
      .filter(([, r]) => (r.postInstall ?? []).includes(scriptName))
      .map(([n]) => n);
  };

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between gap-2">
        <Segmented
          value={scope}
          onChange={(v) => setScope(v as Scope)}
          options={[
            { label: "Workspace", value: "workspace" },
            { label: "Repo", value: "repo" },
          ]}
          size="small"
        />
        <QButton
          variant="accent"
          icon={<PlusOutlined />}
          onClick={() => setSub({ kind: "add", scope })}
        >
          Add
        </QButton>
      </div>

      <div className="flex-1 min-h-0">
        <IndexableLayout
          placeholder={`Search ${scope} scripts…`}
          onSearch={setQuery}
        >
          <ScrollableLayout>
            {filtered.length === 0 ? (
              <Alert
                type="info"
                showIcon
                message={query ? "No matches" : `No ${scope} scripts yet`}
                description={
                  query
                    ? `No script name matches "${query}".`
                    : "Use + to create one."
                }
              />
            ) : (
              filtered.map((name) => {
                const attached = attachedTo(name);
                const menu: MenuAction[] = [
                  {
                    label: "View contents",
                    onClick: () =>
                      setSub({
                        kind: "view",
                        scope: "view",
                        name,
                        scriptScope: scope,
                        mode: "view",
                      }),
                  },
                  {
                    label: "Edit",
                    onClick: () =>
                      setSub({
                        kind: "view",
                        scope: "view",
                        name,
                        scriptScope: scope,
                        mode: "edit",
                      }),
                  },
                  {
                    label: "Assign to…",
                    onClick: () => setSub({ kind: "assign", scope, name }),
                  },
                  ...(attached.length > 0
                    ? [
                        {
                          label: "Run now",
                          onClick: () => setSub({ kind: "run", scope, name }),
                        },
                      ]
                    : []),
                  {
                    label: "Remove",
                    danger: true,
                    onClick: () => svc.remove(name, scope),
                  },
                ];
                return (
                  <ToolboxListItem
                    key={name}
                    id={name}
                    primary={name}
                    secondary={
                      attached.length > 0 ? (
                        <div className="flex flex-wrap gap-1 mt-1">
                          {attached.map((t) => (
                            <Tag
                              key={t}
                              bordered
                              color="default"
                              className="!text-[10px] !leading-4 !m-0"
                            >
                              {t}
                            </Tag>
                          ))}
                        </div>
                      ) : (
                        <span className="text-xs text-zinc-500">
                          Not attached
                        </span>
                      )
                    }
                    icon={<CodeOutlined />}
                    menuItems={menu}
                    onClick={() =>
                      setSub({
                        kind: "view",
                        scope: "view",
                        name,
                        scriptScope: scope,
                        mode: "view",
                      })
                    }
                  />
                );
              })
            )}
          </ScrollableLayout>
        </IndexableLayout>
      </div>
    </div>
  );
};
