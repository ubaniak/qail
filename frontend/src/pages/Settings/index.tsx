// Settings overlay — shown when TopBar's gear is toggled. Two tabs:
// "General" (root + editors) and "Cleanup" (delete orphan workspace
// directories under root that aren't registered).

import {
  CodeOutlined,
  DeleteOutlined,
  EditOutlined,
  FolderOpenOutlined,
  PlusOutlined,
  ReloadOutlined,
  UndoOutlined,
} from "@ant-design/icons";
import { Alert, App, Checkbox, Modal, Segmented, Spin, Tag } from "antd";
import { useEffect, useMemo, useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import type { models } from "../../types";
import { AddEditor, EditRoot } from "./edit";

type Tab = "general" | "cleanup";
type Mode = "view" | "editRoot" | "addEditor";

export const SettingsIndex = () => {
  const { list, editors, cleanup, copy } = useQailService();
  const { modal } = App.useApp();
  const [mode, setMode] = useState<Mode>("view");
  const [tab, setTab] = useState<Tab>("general");
  const [selected, setSelected] = useState<string[]>([]);
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null);

  if (mode === "editRoot") {
    return <EditRoot root={list.settings.root} onClose={() => setMode("view")} />;
  }
  if (mode === "addEditor") {
    return <AddEditor onClose={() => setMode("view")} />;
  }

  const orphans = cleanup.orphans;
  const allSelected = selected.length > 0 && selected.length === orphans.length;
  const toggle = (name: string) =>
    setSelected((prev) =>
      prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]
    );
  const toggleAll = () =>
    setSelected(allSelected ? [] : [...orphans]);

  const confirmDelete = () => {
    if (selected.length === 0) return;
    modal.confirm({
      title: `Delete ${selected.length} ${selected.length === 1 ? "directory" : "directories"}?`,
      content: (
        <div>
          <p className="!mb-2">These directories live under your workspace root but aren't tracked by qail. They'll be permanently deleted.</p>
          <ul className="!mt-0 !mb-0 pl-5">
            {selected.map((n) => (
              <li key={n} className="font-mono text-xs">{n}</li>
            ))}
          </ul>
        </div>
      ),
      okText: "Delete",
      okButtonProps: { danger: true },
      onOk: () => {
        const names = [...selected];
        setSelected([]);
        cleanup.remove(names);
      },
    });
  };

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-base font-semibold">Settings</div>
          <div className="text-zinc-500 text-xs">
            {tab === "general"
              ? "Workspace root + editors for qail"
              : "Delete untracked directories under the workspace root"}
          </div>
        </div>
        {tab === "general" ? (
          <QButton
            variant="accent"
            icon={<EditOutlined />}
            onClick={() => setMode("editRoot")}
          >
            Edit root
          </QButton>
        ) : (
          <div className="flex items-center gap-2">
            <QButton
              variant="cancel"
              icon={<ReloadOutlined />}
              onClick={() => {
                setSelected([]);
                cleanup.refresh();
              }}
            >
              Rescan
            </QButton>
            <QButton
              variant="accent"
              icon={<DeleteOutlined />}
              disabled={selected.length === 0}
              onClick={confirmDelete}
            >
              Delete{selected.length > 0 ? ` (${selected.length})` : ""}
            </QButton>
          </div>
        )}
      </div>

      <div className="px-3 pb-2">
        <Segmented
          value={tab}
          onChange={(v) => {
            setTab(v as Tab);
            setSelected([]);
          }}
          options={[
            { label: "General", value: "general" },
            {
              label: (
                <span className="flex items-center gap-1.5">
                  Cleanup
                  {orphans.length > 0 && (
                    <Tag color="orange" className="!text-[10px] !leading-4 !m-0">
                      {orphans.length}
                    </Tag>
                  )}
                </span>
              ),
              value: "cleanup",
            },
          ]}
          size="small"
          block
        />
      </div>

      <div className="flex-1 min-h-0">
        {tab === "general" ? (
          <GeneralPane onAddEditor={() => setMode("addEditor")} onEditRoot={() => setMode("editRoot")} />
        ) : (
          <CleanupPane
            root={list.settings.root}
            orphans={orphans}
            selected={selected}
            allSelected={allSelected}
            onToggle={toggle}
            onToggleAll={toggleAll}
            onOpenEditor={cleanup.openEditor}
            onCopyPath={(path) => copy.text(path, "Path")}
            onRestore={(name) => setRestoreTarget(name)}
          />
        )}
      </div>
      {restoreTarget && (
        <RestoreModal
          name={restoreTarget}
          inspect={cleanup.inspect}
          onCancel={() => setRestoreTarget(null)}
          onConfirm={(name, repos) => {
            cleanup.restore(name, repos);
            setRestoreTarget(null);
          }}
        />
      )}
    </div>
  );

  function GeneralPane({ onAddEditor, onEditRoot }: { onAddEditor: () => void; onEditRoot: () => void }) {
    return (
      <ScrollableLayout>
        <ToolboxListItem
          id="root"
          primary={
            <div className="flex items-center gap-2">
              <span>Workspace root</span>
              <Tag color="blue" className="!text-[10px] !leading-4 !m-0">
                Required
              </Tag>
            </div>
          }
          secondary={
            <span className="font-mono text-xs">
              {list.settings.root || "not configured"}
            </span>
          }
          icon={<FolderOpenOutlined />}
          onClick={onEditRoot}
        />

        <div className="px-3 pt-3 pb-1 flex items-center justify-between">
          <div className="text-xs text-zinc-400 uppercase tracking-wider">
            Editors
          </div>
          <QButton
            variant="accent"
            icon={<PlusOutlined />}
            onClick={onAddEditor}
          >
            Add
          </QButton>
        </div>

        {list.settings.editors.length === 0 ? (
          <div className="px-3 py-2 text-xs text-zinc-500">
            No editors configured. Add one to enable "open in editor".
          </div>
        ) : (
          list.settings.editors.map((e) => {
            const isDefault = e.name === list.settings.defaultEditor;
            const menuItems = [
              ...(isDefault
                ? []
                : [
                    {
                      label: "Set as default",
                      onClick: () => editors.setDefault(e.name),
                    },
                  ]),
              {
                label: "Remove",
                danger: true,
                onClick: () => editors.remove(e.name),
              },
            ];
            return (
              <ToolboxListItem
                key={e.name}
                id={`editor-${e.name}`}
                primary={
                  <div className="flex items-center gap-2">
                    <span>{e.name}</span>
                    {isDefault && (
                      <Tag color="gold" className="!text-[10px] !leading-4 !m-0">
                        Default
                      </Tag>
                    )}
                  </div>
                }
                secondary={
                  <span className="font-mono text-xs">{e.command}</span>
                }
                icon={<CodeOutlined />}
                menuItems={menuItems}
              />
            );
          })
        )}
      </ScrollableLayout>
    );
  }
};

type CleanupPaneProps = {
  root: string;
  orphans: string[];
  selected: string[];
  allSelected: boolean;
  onToggle: (name: string) => void;
  onToggleAll: () => void;
  onOpenEditor: (name: string) => void;
  onCopyPath: (path: string) => void;
  onRestore: (name: string) => void;
};

const CleanupPane = ({
  root,
  orphans,
  selected,
  allSelected,
  onToggle,
  onToggleAll,
  onOpenEditor,
  onCopyPath,
  onRestore,
}: CleanupPaneProps) => {
  const sorted = useMemo(() => [...orphans].sort(), [orphans]);

  if (!root) {
    return (
      <div className="p-3">
        <Alert
          type="warning"
          showIcon
          message="Workspace root not configured"
          description="Set a root in the General tab before running cleanup."
        />
      </div>
    );
  }

  if (sorted.length === 0) {
    return (
      <div className="p-3">
        <Alert
          type="success"
          showIcon
          message="No orphan directories"
          description={`Every subdirectory of ${root} is tracked as a workspace.`}
        />
      </div>
    );
  }

  return (
    <ScrollableLayout>
      <div className="px-3 py-2 flex items-center justify-between border-b border-zinc-800/60">
        <div className="text-xs text-zinc-400">
          Found <span className="text-zinc-200">{sorted.length}</span> untracked{" "}
          {sorted.length === 1 ? "directory" : "directories"} under{" "}
          <span className="font-mono">{root}</span>
        </div>
        <Checkbox checked={allSelected} onChange={onToggleAll}>
          <span className="text-xs">Select all</span>
        </Checkbox>
      </div>
      {sorted.map((name) => {
        const fullPath = `${root}/${name}`;
        return (
          <ToolboxListItem
            key={name}
            id={name}
            primary={name}
            secondary={<span className="font-mono text-xs">{fullPath}</span>}
            icon={<FolderOpenOutlined />}
            selected={selected.includes(name)}
            onClick={() => onToggle(name)}
            menuItems={[
              { label: "Restore as workspace", onClick: () => onRestore(name) },
              { label: "Open in editor", onClick: () => onOpenEditor(name) },
              { label: "Copy path", onClick: () => onCopyPath(fullPath) },
            ]}
          >
            <Checkbox checked={selected.includes(name)} />
          </ToolboxListItem>
        );
      })}
    </ScrollableLayout>
  );
};

type RestoreModalProps = {
  name: string;
  inspect: (name: string) => Promise<models.OrphanInspectionDTO>;
  onCancel: () => void;
  onConfirm: (name: string, repos: string[]) => void;
};

const RestoreModal = ({ name, inspect, onCancel, onConfirm }: RestoreModalProps) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [inspection, setInspection] = useState<models.OrphanInspectionDTO | null>(null);
  const [picked, setPicked] = useState<string[]>([]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    inspect(name)
      .then((res) => {
        if (cancelled) return;
        setInspection(res);
        const preselected = (res.repos ?? [])
          .map((r) => r.match)
          .filter((m): m is string => !!m);
        setPicked(preselected);
        setLoading(false);
      })
      .catch((e: Error) => {
        if (cancelled) return;
        setError(e.message);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [name, inspect]);

  const matched = (inspection?.repos ?? []).filter((r) => r.match);
  const unknown = (inspection?.repos ?? []).filter((r) => !r.match);
  const preserved = inspection?.preservedScripts ?? [];

  const togglePick = (repo: string) =>
    setPicked((prev) =>
      prev.includes(repo) ? prev.filter((p) => p !== repo) : [...prev, repo]
    );

  return (
    <Modal
      open
      title={`Restore "${name}"`}
      onCancel={onCancel}
      onOk={() => onConfirm(name, picked)}
      okText="Restore"
      okButtonProps={{ disabled: loading || !!error }}
    >
      {loading ? (
        <div className="flex items-center justify-center py-6">
          <Spin />
        </div>
      ) : error ? (
        <Alert type="error" showIcon message="Inspection failed" description={error} />
      ) : (
        <div className="flex flex-col gap-3">
          <div className="text-xs text-zinc-400">
            Re-register the directory as a workspace. Detected repos are listed
            below; matched repos are pre-selected. Unknown directories will be
            skipped — register them in the repo list first if you want them
            attached.
          </div>
          {matched.length === 0 ? (
            <Alert
              type="info"
              showIcon
              message="No matched repos detected"
              description="The workspace will be restored with no attached repos. You can add them later via Edit."
            />
          ) : (
            <div className="flex flex-col gap-1">
              <div className="text-xs uppercase tracking-wider text-zinc-500">
                Matched repos
              </div>
              {matched.map((r) => (
                <label
                  key={r.dir}
                  className="flex items-center gap-2 text-sm cursor-pointer"
                >
                  <Checkbox
                    checked={picked.includes(r.match)}
                    onChange={() => togglePick(r.match)}
                  />
                  <span>{r.match}</span>
                  <span className="text-xs text-zinc-500 font-mono">
                    ({r.dir}
                    {r.remoteUrl ? ` → ${r.remoteUrl}` : ""})
                  </span>
                </label>
              ))}
            </div>
          )}
          {unknown.length > 0 && (
            <div className="flex flex-col gap-1">
              <div className="text-xs uppercase tracking-wider text-zinc-500">
                Unknown directories (skipped)
              </div>
              {unknown.map((r) => (
                <div key={r.dir} className="text-xs font-mono text-zinc-400">
                  {r.dir}
                  {r.remoteUrl ? ` — ${r.remoteUrl}` : " — no git remote"}
                </div>
              ))}
            </div>
          )}
          {preserved.length > 0 && (
            <Alert
              type="info"
              showIcon
              icon={<UndoOutlined />}
              message="Post-install scripts will be re-attached"
              description={
                <ul className="!mb-0 pl-5">
                  {preserved.map((s) => (
                    <li key={s} className="font-mono text-xs">
                      {s}
                    </li>
                  ))}
                </ul>
              }
            />
          )}
        </div>
      )}
    </Modal>
  );
};
