// Settings overlay — shown when TopBar's gear is toggled. Two read-only
// rows (root + editor) and an Edit button that opens the edit form.

import { CodeOutlined, EditOutlined, FolderOpenOutlined } from "@ant-design/icons";
import { Tag } from "antd";
import { useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import { EditSettings } from "./edit";

export const SettingsIndex = () => {
  const { list } = useQailService();
  const [editing, setEditing] = useState(false);

  if (editing) {
    return (
      <EditSettings
        root={list.settings.root}
        editor={list.settings.editor}
        onClose={() => setEditing(false)}
      />
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-base font-semibold">Settings</div>
          <div className="text-zinc-500 text-xs">
            Workspace root + editor for qail
          </div>
        </div>
        <QButton
          variant="accent"
          icon={<EditOutlined />}
          onClick={() => setEditing(true)}
        >
          Edit
        </QButton>
      </div>

      <div className="flex-1 min-h-0">
        <ScrollableLayout>
          <ToolboxListItem
            id="root"
            primary={
              <div className="flex items-center gap-2">
                <span>Workspace root</span>
                <Tag color="blue" className="!text-[10px] !leading-4 !m-0">
                  Required
                </Tag>
              </div>
            }
            secondary={
              <span className="font-mono text-xs">
                {list.settings.root || "not configured"}
              </span>
            }
            icon={<FolderOpenOutlined />}
            onClick={() => setEditing(true)}
          />
          <ToolboxListItem
            id="editor"
            primary="Editor"
            secondary={
              <span className="font-mono text-xs">
                {list.settings.editor || "not configured"}
              </span>
            }
            icon={<CodeOutlined />}
            onClick={() => setEditing(true)}
          />
        </ScrollableLayout>
      </div>
    </div>
  );
};
