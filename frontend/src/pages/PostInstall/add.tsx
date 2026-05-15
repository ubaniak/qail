// Add-script form. Creates an empty bash script in the chosen scope
// (workspace/repo) — backend ensures the `.sh` extension. The scope is
// fixed by the caller so the user can't accidentally drop a workspace
// script into the repo bucket.

import { Input, Tag } from "antd";
import { useState } from "react";

import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";
import type { Scope } from "../../types";

export type AddScriptProps = {
  scope: Scope;
  onClose: () => void;
};

export const AddScript = ({ scope, onClose }: AddScriptProps) => {
  const { scripts } = useQailService();
  const [name, setName] = useState("");

  const valid = name.trim() !== "";

  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div className="flex justify-end gap-2">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!valid}
          onClick={() => {
            scripts.add(name.trim(), scope);
            onClose();
          }}
        >
          Create
        </QButton>
      </div>

      <div>
        <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
          <span>New script</span>
          <Tag color={scope === "workspace" ? "blue" : "purple"}>{scope}</Tag>
        </div>
        <div className="text-zinc-400 text-xs">
          Creates an empty bash script under ~/.qail/scripts/{scope}/.
        </div>
      </div>

      <div>
        <label className="text-xs text-zinc-400">Name</label>
        <Input
          placeholder="bootstrap"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />
        <div className="text-[11px] text-zinc-500 mt-1">
          The <code>.sh</code> suffix is added if missing.
        </div>
      </div>
    </div>
  );
};
