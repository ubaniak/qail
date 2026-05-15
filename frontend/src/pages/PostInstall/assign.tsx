// Bulk-attach a script to one or more targets. Workspace-scope scripts
// attach to workspaces; repo-scope scripts attach to repos. The form
// computes the inverse change: for every target whose checkbox state
// flipped, dispatch a set-post-install with the new list (existing
// attachments minus this script, or existing + this script).

import { useMemo, useState } from "react";
import { Alert, Checkbox, Tag } from "antd";
import { FolderOpenOutlined, GithubOutlined } from "@ant-design/icons";

import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import type { Scope } from "../../types";

export type AssignScriptProps = {
  name: string;
  scope: Scope;
  onClose: () => void;
};

export const AssignScript = ({ name, scope, onClose }: AssignScriptProps) => {
  const { list, postInstall } = useQailService();
  const [query, setQuery] = useState("");

  const targets = useMemo(() => {
    if (scope === "workspace") {
      return Object.entries(list.workspaces).map(([n, ws]) => ({
        id: n,
        secondary: `${(ws.repos || []).length} repos`,
        attached: (ws.postInstall ?? []).includes(name),
        currentList: ws.postInstall ?? [],
      }));
    }
    return Object.entries(list.repos).map(([n, r]) => ({
      id: n,
      secondary: r.url,
      attached: (r.postInstall ?? []).includes(name),
      currentList: r.postInstall ?? [],
    }));
  }, [list, scope, name]);

  // Track per-target "next attached" state in a Set keyed by target id.
  const [nextAttached, setNextAttached] = useState<Set<string>>(
    () => new Set(targets.filter((t) => t.attached).map((t) => t.id))
  );

  const toggle = (id: string) =>
    setNextAttached((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const filtered = targets.filter((t) =>
    t.id.toLowerCase().includes(query.toLowerCase())
  );

  const changed = targets.some(
    (t) => t.attached !== nextAttached.has(t.id)
  );

  const save = () => {
    for (const t of targets) {
      const wanted = nextAttached.has(t.id);
      if (wanted === t.attached) continue;
      const updated = wanted
        ? [...t.currentList, name]
        : t.currentList.filter((s) => s !== name);
      if (scope === "workspace") postInstall.setWorkspace(t.id, updated);
      else postInstall.setRepo(t.id, updated);
    }
    onClose();
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-end gap-2 px-3 py-2 border-b border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton variant="accent" disabled={!changed} onClick={save}>
          Save
        </QButton>
      </div>

      <div className="px-3 py-2">
        <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
          <span>Assign {name}</span>
          <Tag color={scope === "workspace" ? "blue" : "purple"}>{scope}</Tag>
        </div>
        <div className="text-zinc-400 text-xs">
          Pick {scope === "workspace" ? "workspaces" : "repos"} that should run this script after clone.
        </div>
      </div>

      <div className="flex-1 min-h-0">
        <IndexableLayout
          placeholder={`Search ${scope === "workspace" ? "workspaces" : "repos"}…`}
          onSearch={setQuery}
        >
          <ScrollableLayout>
            {filtered.length === 0 ? (
              <Alert
                type="info"
                showIcon
                message="Nothing to assign"
                description={
                  scope === "workspace"
                    ? "Create a workspace first."
                    : "Register a repo first."
                }
              />
            ) : (
              filtered.map((t) => {
                const checked = nextAttached.has(t.id);
                return (
                  <ToolboxListItem
                    key={t.id}
                    id={t.id}
                    primary={t.id}
                    secondary={t.secondary}
                    icon={
                      scope === "workspace" ? (
                        <FolderOpenOutlined />
                      ) : (
                        <GithubOutlined />
                      )
                    }
                    selected={checked}
                    onClick={() => toggle(t.id)}
                  >
                    <Checkbox checked={checked} />
                  </ToolboxListItem>
                );
              })
            )}
          </ScrollableLayout>
        </IndexableLayout>
      </div>

    </div>
  );
};
