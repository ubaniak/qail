// Repo edit — the only mutable field today is the post-install script
// list (repo URL is immutable; rename means remove+add). Multi-select
// over the repo-scope script catalogue; submit calls SetRepoPostInstall.

import { Alert, Checkbox, Tag } from "antd";
import { useEffect, useState } from "react";
import { CodeOutlined } from "@ant-design/icons";

import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export type EditRepoProps = {
  name: string;
  onClose: () => void;
};

export const EditRepo = ({ name, onClose }: EditRepoProps) => {
  const { list, postInstall } = useQailService();
  const repo = list.repos[name];
  const scriptCatalogue = list.scripts.repo;

  const [selected, setSelected] = useState<string[]>([]);
  const [query, setQuery] = useState("");

  useEffect(() => {
    setSelected([...(repo?.postInstall ?? [])]);
  }, [repo?.postInstall]);

  if (!repo) {
    return (
      <div className="p-3">
        <Alert type="error" message={`Repo "${name}" not found`} />
        <div className="mt-3">
          <QButton variant="cancel" onClick={onClose}>
            Back
          </QButton>
        </div>
      </div>
    );
  }

  const toggle = (s: string) =>
    setSelected((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]
    );

  const filtered = scriptCatalogue.filter((n) =>
    n.toLowerCase().includes(query.toLowerCase())
  );

  const original = repo.postInstall ?? [];
  const changed =
    selected.length !== original.length ||
    !selected.every((s) => original.includes(s)) ||
    !original.every((s) => selected.includes(s));

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-end gap-2 px-3 py-2 border-b border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!changed}
          onClick={() => {
            postInstall.setRepo(name, selected);
            onClose();
          }}
        >
          Save
        </QButton>
      </div>

      <div className="px-3 py-2">
        <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
          <span>Edit repo</span>
          {changed && <Tag color="orange">unsaved</Tag>}
        </div>
        <div className="text-zinc-400 text-xs">
          {name} — pick repo-scope post-install scripts.
        </div>
      </div>

      <div className="flex-1 min-h-0">
        <IndexableLayout
          placeholder="Search scripts…"
          onSearch={setQuery}
        >
          <ScrollableLayout>
            {scriptCatalogue.length === 0 ? (
              <Alert
                type="info"
                showIcon
                message="No repo scripts"
                description="Add a repo-scope script in the Post-install tab first."
              />
            ) : filtered.length === 0 ? (
              <Alert
                type="info"
                showIcon
                message="No matches"
                description={`No script matches "${query}".`}
              />
            ) : (
              filtered.map((scriptName) => (
                <ToolboxListItem
                  key={scriptName}
                  id={scriptName}
                  primary={scriptName}
                  icon={<CodeOutlined />}
                  selected={selected.includes(scriptName)}
                  onClick={() => toggle(scriptName)}
                >
                  <Checkbox checked={selected.includes(scriptName)} />
                </ToolboxListItem>
              ))
            )}
          </ScrollableLayout>
        </IndexableLayout>
      </div>

    </div>
  );
};
