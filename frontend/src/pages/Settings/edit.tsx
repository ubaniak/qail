// Edit form for the two scalar config fields. Save is disabled until
// either field changes and root is non-empty.

import { CodeOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Input } from "antd";
import { useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";

export type EditSettingsProps = {
  root: string;
  editor: string;
  onClose: () => void;
};

export const EditSettings = ({
  root: initialRoot,
  editor: initialEditor,
  onClose,
}: EditSettingsProps) => {
  const { save } = useQailService();
  const [root, setRoot] = useState(initialRoot);
  const [editor, setEditor] = useState(initialEditor);

  const changed =
    root.trim() !== initialRoot.trim() ||
    editor.trim() !== initialEditor.trim();
  const valid = root.trim() !== "";

  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">Edit settings</div>
        <div className="text-zinc-400 text-xs">
          Where qail stores workspaces + which editor to launch.
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

      <div>
        <label className="text-xs text-zinc-400">Editor command</label>
        <Input
          prefix={<CodeOutlined className="text-zinc-500" />}
          placeholder="code"
          value={editor}
          onChange={(e) => setEditor(e.target.value)}
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
          disabled={!valid || !changed}
          onClick={() => {
            save.settings(root.trim(), editor.trim());
            onClose();
          }}
        >
          Save
        </QButton>
      </div>
    </div>
  );
};
