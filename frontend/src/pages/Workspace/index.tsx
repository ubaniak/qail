// Workspace listing — top-level item under the Workspaces tab.
// Maintains its own search + sub-view state (add / edit / remove) so
// each tab feels like a small app; switching tabs doesn't unmount us
// because Antd Tabs keeps panes alive.

import { ClockCircleOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Alert, Tag } from "antd";
import { formatDistanceToNow } from "date-fns";
import { useState } from "react";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import type { MenuAction } from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useNotification } from "../../providers/notification";
import { useQailService } from "../../providers/qailservice";
import { AddWorkspace } from "./add";
import { EditWorkspace } from "./edit";
import { RemoveWorkspace } from "./remove";

export const WorkspaceIndex = () => {
  const { list, open, mux, cd, editors } = useQailService();
  const { toast } = useNotification();

  const [query, setQuery] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [toRemove, setToRemove] = useState<string | undefined>();
  const [toEdit, setToEdit] = useState<string | undefined>();

  if (showAdd) return <AddWorkspace onClose={() => setShowAdd(false)} />;
  if (toRemove)
    return (
      <RemoveWorkspace name={toRemove} onClose={() => setToRemove(undefined)} />
    );
  if (toEdit)
    return <EditWorkspace name={toEdit} onClose={() => setToEdit(undefined)} />;

  const filtered = Object.entries(list.workspaces)
    .filter(([name]) => name.toLowerCase().includes(query.toLowerCase()))
    .sort(([, a], [, b]) => (b.lastUsed || "").localeCompare(a.lastUsed || ""));

  const allEditors = list.settings.editors;
  const defaultEditor = list.settings.defaultEditor;

  const renderBody = () => {
    if (filtered.length === 0) {
      return (
        <Alert
          type="info"
          showIcon
          message={query ? "No matches" : "No workspaces yet"}
          description={
            query
              ? `No workspace name matches "${query}".`
              : "Use + to create one, or run `qail ws add` from CLI."
          }
        />
      );
    }
    return (
      <div>
        {filtered.map(([name, ws]) => {
          const repos = ws.repos || [];
          const lastUsed = ws.lastUsed ? new Date(ws.lastUsed) : null;
          const effective = ws.editor || defaultEditor;
          const wsHasOverride = !!ws.editor;

          const otherEditors = allEditors.filter((e) => e.name !== effective);
          const openMenu: MenuAction[] = [
            {
              label: effective
                ? `Open in default (${effective})`
                : "Open in default",
              onClick: (id) => open.workspace(id),
            },
            ...(otherEditors.length > 0
              ? [
                  {
                    label: "Open in",
                    children: otherEditors.map<MenuAction>((e) => ({
                      label: e.name,
                      onClick: (id) => open.workspace(id, e.name),
                    })),
                  },
                ]
              : []),
          ];

          const inheritMenu: MenuAction[] = wsHasOverride
            ? [
                {
                  label: "Inherit global default",
                  onClick: (id) => editors.setWorkspace(id, ""),
                },
              ]
            : [];

          return (
            <ToolboxListItem
              key={name}
              id={name}
              primary={
                <div className="flex items-center gap-2">
                  <span>{name}</span>
                  {wsHasOverride && (
                    <Tag color="purple" className="!text-[10px] !leading-4 !m-0">
                      editor: {ws.editor}
                    </Tag>
                  )}
                </div>
              }
              secondary={
                <div className="mt-1">
                  {repos.length > 0 ? (
                    <div className="flex flex-wrap gap-1 mb-1">
                      {repos.map((r) => (
                        <Tag
                          key={r}
                          bordered
                          color="default"
                          className="!text-[10px] !leading-4 !m-0"
                        >
                          {r}
                        </Tag>
                      ))}
                    </div>
                  ) : (
                    <div className="text-xs text-zinc-500 mb-1">
                      No repositories
                    </div>
                  )}
                  {lastUsed && !isNaN(lastUsed.getTime()) && (
                    <div className="flex items-center gap-1 text-xs text-zinc-500">
                      <ClockCircleOutlined />
                      <span>
                        used {formatDistanceToNow(lastUsed, { addSuffix: true })}
                      </span>
                    </div>
                  )}
                </div>
              }
              icon={<FolderOpenOutlined />}
              menuItems={[
                ...openMenu,
                ...inheritMenu,
                { label: "Copy cd", onClick: (id) => cd.workspace(id) },
                {
                  label: "Copy tmux attach",
                  onClick: (id) => {
                    mux.attach(id);
                  },
                },
                { label: "Edit", onClick: (id) => setToEdit(id) },
                { label: "Remove", danger: true, onClick: (id) => setToRemove(id) },
              ]}
              onClick={() => {
                open.workspace(name);
                toast.info(`Resolving ${name}`);
              }}
            />
          );
        })}
      </div>
    );
  };

  return (
    <IndexableLayout
      placeholder="Search workspaces…"
      onAdd={() => setShowAdd(true)}
      onSearch={setQuery}
    >
      <ScrollableLayout>{renderBody()}</ScrollableLayout>
    </IndexableLayout>
  );
};
