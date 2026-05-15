// Edit forms for Settings: root path and editor registration.

import { CodeOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Input } from "antd";
import { useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";

export type EditRootProps = {
  root: string;
  onClose: () => void;
};

export const EditRoot = ({ root: initialRoot, onClose }: EditRootProps) => {
  const { save } = useQailService();
  const [root, setRoot] = useState(initialRoot);

  const changed = root.trim() !== initialRoot.trim();
  const valid = root.trim() !== "";

  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">Edit root</div>
        <div className="text-zinc-400 text-xs">
          Where qail stores workspaces.
        </div>
      </div>

      <div>
        <label className="text-xs text-zinc-400">Workspace root</label>
        <Input
          prefix={<FolderOpenOutlined className="text-zinc-500" />}
          placeholder="/Users/you/Projects"
          value={root}
          onChange={(e) => setRoot(e.target.value)}
          autoFocus
        />
        <div className="text-[11px] text-zinc-500 mt-1">
          All workspace clones live under this directory.
        </div>
      </div>

      <div className="flex-1" />

      <div className="flex justify-end gap-2">
        <QButton variant="cancel" onClick={onClose}>
          Cancel
        </QButton>
        <QButton
          variant="accent"
          disabled={!valid || !changed}
          onClick={() => {
            save.root(root.trim());
            onClose();
          }}
        >
          Save
        </QButton>
      </div>
    </div>
  );
};

export type AddEditorProps = {
  onClose: () => void;
};

export const AddEditor = ({ onClose }: AddEditorProps) => {
  const { editors } = useQailService();
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");

  const valid = name.trim() !== "" && command.trim() !== "";

  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">Add editor</div>
        <div className="text-zinc-400 text-xs">
          Register an editor binding (label + executable).
        </div>
      </div>

      <div>
        <label className="text-xs text-zinc-400">Name</label>
        <Input
          placeholder="vscode"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />
        <div className="text-[11px] text-zinc-500 mt-1">
          Label you'll pick from the workspace menu.
        </div>
      </div>

      <div>
        <label className="text-xs text-zinc-400">Command</label>
        <Input
          prefix={<CodeOutlined className="text-zinc-500" />}
          placeholder="code"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
        />
        <div className="text-[11px] text-zinc-500 mt-1">
          Examples: <code>code</code>, <code>cursor</code>, <code>idea</code>,
          full paths OK.
        </div>
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
            editors.add(name.trim(), command.trim());
            onClose();
          }}
        >
          Add
        </QButton>
      </div>
    </div>
  );
};
