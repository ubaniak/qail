// Edit existing workspace — repos + post-install. Two segmented panes:
// one for repo membership (rebuilds workspace on disk), one for the
// workspace-scope post-install attachment list (metadata-only update).
// Saving each pane fires its own RPC so a failed repo rebuild does not
// roll back a successful post-install change.

import { CodeOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Alert, Checkbox, Input, Segmented, Tag } from "antd";
import { useEffect, useMemo, useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export type EditWorkspaceProps = {
  name: string;
  onClose: () => void;
};

type Pane = "repos" | "scripts";

export const EditWorkspace = ({ name, onClose }: EditWorkspaceProps) => {
  const { list, edit, postInstall } = useQailService();
  const original = list.workspaces[name];
  const scriptCatalogue = list.scripts.workspace;

  const [pane, setPane] = useState<Pane>("repos");
  const [selectedRepos, setSelectedRepos] = useState<string[]>([]);
  const [selectedScripts, setSelectedScripts] = useState<string[]>([]);
  const [query, setQuery] = useState("");

  useEffect(() => {
    if (original?.repos) setSelectedRepos([...original.repos]);
    setSelectedScripts([...(original?.postInstall ?? [])]);
  }, [original]);

  const repos = useMemo(
    () =>
      Object.entries(list.repos).filter(([n]) =>
        n.toLowerCase().includes(query.toLowerCase())
      ),
    [list.repos, query]
  );

  const scripts = useMemo(
    () =>
      scriptCatalogue.filter((n) =>
        n.toLowerCase().includes(query.toLowerCase())
      ),
    [scriptCatalogue, query]
  );

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

  const toggleRepo = (r: string) =>
    setSelectedRepos((prev) =>
      prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r]
    );

  const toggleScript = (s: string) =>
    setSelectedScripts((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]
    );

  const originalRepos = original.repos || [];
  const reposChanged =
    selectedRepos.length !== originalRepos.length ||
    !selectedRepos.every((r) => originalRepos.includes(r)) ||
    !originalRepos.every((r) => selectedRepos.includes(r));
  const reposValid = selectedRepos.length > 0;

  const originalScripts = original.postInstall ?? [];
  const scriptsChanged =
    selectedScripts.length !== originalScripts.length ||
    !selectedScripts.every((s) => originalScripts.includes(s)) ||
    !originalScripts.every((s) => selectedScripts.includes(s));

  return (
    <div className="flex flex-col h-full">
      <div className="flex justify-end gap-2 px-3 py-2 border-b border-zinc-800/60 bg-zinc-900/40">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        {pane === "repos" ? (
          <QButton
            variant="accent"
            disabled={!reposValid || !reposChanged}
            onClick={() => {
              edit.workspace(name, selectedRepos);
              onClose();
            }}
          >
            Save repos
          </QButton>
        ) : (
          <QButton
            variant="accent"
            disabled={!scriptsChanged}
            onClick={() => {
              postInstall.setWorkspace(name, selectedScripts);
              onClose();
            }}
          >
            Save scripts
          </QButton>
        )}
      </div>

      <div className="px-3 py-2 flex items-center justify-between gap-2">
        <div>
          <div className="text-zinc-100 text-base font-semibold">Edit workspace</div>
          <div className="text-zinc-400 text-xs flex items-center gap-2">
            <span>{name}</span>
            {(reposChanged || scriptsChanged) && (
              <Tag color="orange">unsaved</Tag>
            )}
          </div>
        </div>
        <Segmented
          value={pane}
          onChange={(v) => {
            setPane(v as Pane);
            setQuery("");
          }}
          options={[
            { label: "Repos", value: "repos" },
            { label: "Post-install", value: "scripts" },
          ]}
          size="small"
        />
      </div>

      {pane === "repos" && (
        <div className="px-3 pb-2">
          <Input value={name} disabled />
        </div>
      )}

      <div className="flex-1 min-h-0">
        {pane === "repos" ? (
          <IndexableLayout placeholder="Search repos…" onSearch={setQuery}>
            <ScrollableLayout>
              {repos.map(([repoName, repo]) => (
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
              ))}
            </ScrollableLayout>
          </IndexableLayout>
        ) : (
          <IndexableLayout placeholder="Search scripts…" onSearch={setQuery}>
            <ScrollableLayout>
              {scriptCatalogue.length === 0 ? (
                <Alert
                  type="info"
                  showIcon
                  message="No workspace scripts"
                  description="Create one in the Post-install tab first."
                />
              ) : scripts.length === 0 ? (
                <Alert
                  type="info"
                  showIcon
                  message="No matches"
                  description={`No script matches "${query}".`}
                />
              ) : (
                scripts.map((s) => (
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
        )}
      </div>

    </div>
  );
};
