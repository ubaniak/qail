// New-workspace form. Two inputs: name + multi-select repo list.
// Repos come from useQailService().list.repos (already loaded); the
// list is search-filterable. Each repo row toggles selection on click.

import { CheckSquareOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Alert, Checkbox, Input } from "antd";
import { useMemo, useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export type AddWorkspaceProps = {
  onClose: () => void;
};

export const AddWorkspace = ({ onClose }: AddWorkspaceProps) => {
  const { list, create } = useQailService();
  const [name, setName] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [query, setQuery] = useState("");

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

  const valid = name.trim() && selected.length > 0;

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2">
        <div className="text-zinc-100 text-base font-semibold">New workspace</div>
        <div className="text-zinc-400 text-xs">
          Pick a name and the repos to include.
        </div>
      </div>

      <div className="px-3 pb-2">
        <Input
          placeholder="Workspace name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />
      </div>

      <div className="px-3 pb-1 text-xs text-zinc-400 flex items-center gap-2">
        <CheckSquareOutlined />
        <span>{selected.length} selected</span>
      </div>

      <div className="flex-1 min-h-0">
        <IndexableLayout placeholder="Search repos…" onSearch={setQuery}>
          <ScrollableLayout>
            {repos.length === 0 ? (
              <Alert
                type="info"
                showIcon
                message="No repos"
                description="Add a repo first from the Repos tab."
              />
            ) : (
              repos.map(([repoName, repo]) => (
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
              ))
            )}
          </ScrollableLayout>
        </IndexableLayout>
      </div>

      <div className="flex justify-end gap-2 px-3 py-2 border-t border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!valid}
          onClick={() => {
            create.workspace(name.trim(), selected);
            onClose();
          }}
        >
          Create
        </QButton>
      </div>
    </div>
  );
};
