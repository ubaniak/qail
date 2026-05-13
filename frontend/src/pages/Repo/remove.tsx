// Confirm-and-remove repo. Like workspace removal, the on-disk clone
// (if any) is untouched.

import { DeleteOutlined, WarningOutlined } from "@ant-design/icons";
import { Alert } from "antd";
import { QButton } from "../../component/Buttons/QButton";
import { useQailService } from "../../providers/qailservice";

export type RemoveRepoProps = {
  name: string;
  onClose: () => void;
};

export const RemoveRepo = ({ name, onClose }: RemoveRepoProps) => {
  const { remove } = useQailService();
  return (
    <div className="flex flex-col h-full p-3 gap-3">
      <div>
        <div className="text-zinc-100 text-base font-semibold">Remove repo</div>
        <div className="text-zinc-400 text-xs">
          Remove <strong>{name}</strong> from qail?
        </div>
      </div>

      <Alert
        type="warning"
        showIcon
        icon={<WarningOutlined />}
        message="This cannot be undone."
        description="On-disk clones and git history stay. Workspaces referencing this repo keep their reference but won't be able to clone it."
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
            remove.repo(name);
            onClose();
          }}
        >
          Remove
        </QButton>
      </div>
    </div>
  );
};
