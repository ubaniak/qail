// Edit existing workspace — same shape as Add but pre-populates name
// + selected repos from list.workspaces[name]. Submit calls the Edit
// mutation which streams progress.

import { FolderOpenOutlined } from "@ant-design/icons";
import { Alert, Checkbox, Input, Tag } from "antd";
import { useEffect, useMemo, useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export type EditWorkspaceProps = {
  name: string;
  onClose: () => void;
};

export const EditWorkspace = ({ name, onClose }: EditWorkspaceProps) => {
  const { list, edit } = useQailService();
  const original = list.workspaces[name];

  const [selected, setSelected] = useState<string[]>([]);
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (original?.repos) setSelected([...original.repos]);
  }, [original]);

  const repos = useMemo(
    () =>
      Object.entries(list.repos).filter(([n]) =>
        n.toLowerCase().includes(query.toLowerCase())
      ),
    [list.repos, query]
  );

  const toggle = (r: string) =>
    setSelected((prev) =>
      prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]
    );

  const originalRepos = original?.repos || [];
  const changed =
    selected.length !== originalRepos.length ||
    !selected.every((r) => originalRepos.includes(r)) ||
    !originalRepos.every((r) => selected.includes(r));
  const valid = selected.length > 0;

  if (!original) {
    return (
      <div className="p-3">
        <Alert type="error" message={`Workspace "${name}" not found`} />
        <div className="mt-3">
          <QButton variant="cancel" onClick={onClose}>
            Back
          </QButton>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2">
        <div className="text-zinc-100 text-base font-semibold">Edit workspace</div>
        <div className="text-zinc-400 text-xs flex items-center gap-2">
          <span>{name}</span>
          {changed && <Tag color="orange">unsaved</Tag>}
        </div>
      </div>

      <div className="px-3 pb-2">
        <Input value={name} disabled />
      </div>

      <div className="flex-1 min-h-0">
        <IndexableLayout placeholder="Search repos…" onSearch={setQuery}>
          <ScrollableLayout>
            {repos.map(([repoName, repo]) => (
              <ToolboxListItem
                key={repoName}
                id={repoName}
                primary={repoName}
                secondary={repo.url}
                icon={<FolderOpenOutlined />}
                selected={selected.includes(repoName)}
                onClick={() => toggle(repoName)}
              >
                <Checkbox checked={selected.includes(repoName)} />
              </ToolboxListItem>
            ))}
          </ScrollableLayout>
        </IndexableLayout>
      </div>

      <div className="flex justify-end gap-2 px-3 py-2 border-t border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!valid || !changed}
          onClick={() => {
            edit.workspace(name, selected);
            onClose();
          }}
        >
          Save
        </QButton>
      </div>
    </div>
  );
};
