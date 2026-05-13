// Register a new repo. Auto-derives the local name from the trailing
// segment of the git URL (so users only really need to paste the URL).

import { Input } from "antd";
import { useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";

export type AddRepoProps = {
  onClose: () => void;
};

export const AddRepo = ({ onClose }: AddRepoProps) => {
  const { create } = useQailService();
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");

  const handleUrl = (v: string) => {
    setUrl(v);
    if (!v.trim()) return;
    const parts = v.replace(/\/$/, "").split("/");
    const last = parts[parts.length - 1] || "";
    const derived = last.replace(/\.git$/, "").trim();
    if (derived) setName(derived);
  };

  const valid = name.trim() && url.trim();

  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">Add repo</div>
        <div className="text-zinc-400 text-xs">
          Paste a git URL; the name will fill in.
        </div>
      </div>

      <div>
        <label className="text-xs text-zinc-400">Repository URL</label>
        <Input
          autoFocus
          placeholder="git@github.com:org/project.git"
          value={url}
          onChange={(e) => handleUrl(e.target.value)}
        />
      </div>

      <div>
        <label className="text-xs text-zinc-400">Repository name</label>
        <Input
          placeholder="project"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>

      <div className="flex-1" />

      <div className="flex justify-end gap-2">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!valid}
          onClick={() => {
            create.repo(name.trim(), url.trim());
            onClose();
          }}
        >
          Add
        </QButton>
      </div>
    </div>
  );
};
