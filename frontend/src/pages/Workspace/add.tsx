// New-workspace form. Two-step flow: first pick name + repos, then
// optionally pick workspace-scope post-install scripts that will fire
// once the clone finishes. Each step is fail-soft — the user can skip
// the scripts step by clicking Create with nothing selected.

import {
  CheckSquareOutlined,
  CodeOutlined,
  FolderOpenOutlined,
} from "@ant-design/icons";
import { Alert, Checkbox, Input, Tag } from "antd";
import { useMemo, useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export type AddWorkspaceProps = {
  onClose: () => void;
};

type Step = "repos" | "scripts";

export const AddWorkspace = ({ onClose }: AddWorkspaceProps) => {
  const { list, create } = useQailService();
  const [step, setStep] = useState<Step>("repos");
  const [name, setName] = useState("");
  const [selectedRepos, setSelectedRepos] = useState<string[]>([]);
  const [selectedScripts, setSelectedScripts] = useState<string[]>([]);
  const [query, setQuery] = useState("");

  const repos = useMemo(
    () =>
      Object.entries(list.repos).filter(([n]) =>
        n.toLowerCase().includes(query.toLowerCase())
      ),
    [list.repos, query]
  );

  const scriptCatalogue = useMemo(
    () =>
      list.scripts.workspace.filter((n) =>
        n.toLowerCase().includes(query.toLowerCase())
      ),
    [list.scripts.workspace, query]
  );

  const toggleRepo = (r: string) =>
    setSelectedRepos((prev) =>
      prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]
    );

  const toggleScript = (s: string) =>
    setSelectedScripts((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]
    );

  const reposValid = name.trim() && selectedRepos.length > 0;

  if (step === "scripts") {
    return (
      <div className="flex flex-col h-full">
        <div className="flex justify-end gap-2 px-3 py-2 border-b border-zinc-800/60 bg-zinc-900/40">
          <QButton
            variant="cancel"
            onClick={() => {
              setStep("repos");
              setQuery("");
            }}
          >
            Back
          </QButton>
          <QButton
            variant="accent"
            onClick={() => {
              create.workspace(name.trim(), selectedRepos, selectedScripts);
              onClose();
            }}
          >
            Create
          </QButton>
        </div>

        <div className="px-3 py-2">
          <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
            <span>Post-install scripts</span>
            <Tag color="blue">workspace</Tag>
          </div>
          <div className="text-zinc-400 text-xs">
            Pick workspace-scope scripts to run after clone. Optional.
          </div>
        </div>

        <div className="px-3 pb-1 text-xs text-zinc-400 flex items-center gap-2">
          <CheckSquareOutlined />
          <span>{selectedScripts.length} selected</span>
        </div>

        <div className="flex-1 min-h-0">
          <IndexableLayout
            placeholder="Search scripts…"
            onSearch={setQuery}
          >
            <ScrollableLayout>
              {list.scripts.workspace.length === 0 ? (
                <Alert
                  type="info"
                  showIcon
                  message="No workspace scripts"
                  description="Skip this step, or create one in the Post-install tab first."
                />
              ) : scriptCatalogue.length === 0 ? (
                <Alert
                  type="info"
                  showIcon
                  message="No matches"
                  description={`No script matches "${query}".`}
                />
              ) : (
                scriptCatalogue.map((s) => (
                  <ToolboxListItem
                    key={s}
                    id={s}
                    primary={s}
                    icon={<CodeOutlined />}
                    selected={selectedScripts.includes(s)}
                    onClick={() => toggleScript(s)}
                  >
                    <Checkbox checked={selectedScripts.includes(s)} />
                  </ToolboxListItem>
                ))
              )}
            </ScrollableLayout>
          </IndexableLayout>
        </div>

      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-end gap-2 px-3 py-2 border-b border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!reposValid}
          onClick={() => {
            setStep("scripts");
            setQuery("");
          }}
        >
          Next: scripts
        </QButton>
      </div>

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
        <span>{selectedRepos.length} selected</span>
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
                  selected={selectedRepos.includes(repoName)}
                  onClick={() => toggleRepo(repoName)}
                >
                  <Checkbox checked={selectedRepos.includes(repoName)} />
                </ToolboxListItem>
              ))
            )}
          </ScrollableLayout>
        </IndexableLayout>
      </div>

    </div>
  );
};
