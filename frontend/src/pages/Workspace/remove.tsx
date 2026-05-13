// Confirm-and-remove workspace. The action removes the workspace from
// qail's DB; on-disk repos are untouched (matches CLI behavior).

import { DeleteOutlined, WarningOutlined } from "@ant-design/icons";
import { Alert } from "antd";
import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";

export type RemoveWorkspaceProps = {
  name: string;
  onClose: () => void;
};

export const RemoveWorkspace = ({ name, onClose }: RemoveWorkspaceProps) => {
  const { remove } = useQailService();
  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">
          Remove workspace
        </div>
        <div className="text-zinc-400 text-xs">
          You're about to remove <strong>{name}</strong>.
        </div>
      </div>

      <Alert
        type="error"
        showIcon
        icon={<WarningOutlined />}
        message="This cannot be undone."
        description="The workspace definition is removed from qail. On-disk repo clones and git history stay."
      />

      <div className="flex-1" />

      <div className="flex justify-end gap-2">
        <QButton variant="cancel" onClick={onClose}>
          Keep
        </QButton>
        <QButton
          variant="destructive"
          icon={<DeleteOutlined />}
          onClick={() => {
            remove.workspace(name);
            onClose();
          }}
        >
          Remove
        </QButton>
      </div>
    </div>
  );
};
