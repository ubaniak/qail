// Settings overlay — shown when TopBar's gear is toggled. Root row +
// editor list (one row per registered editor, default badge, menu).
// Add and edit forms live in edit.tsx.

import {
  CodeOutlined,
  EditOutlined,
  FolderOpenOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { Tag } from "antd";
import { useState } from "react";
import { QButton } from "../../component/Buttons/QButton";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import { AddEditor, EditRoot } from "./edit";

type Mode = "view" | "editRoot" | "addEditor";

export const SettingsIndex = () => {
  const { list, editors } = useQailService();
  const [mode, setMode] = useState<Mode>("view");

  if (mode === "editRoot") {
    return <EditRoot root={list.settings.root} onClose={() => setMode("view")} />;
  }
  if (mode === "addEditor") {
    return <AddEditor onClose={() => setMode("view")} />;
  }

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-base font-semibold">Settings</div>
          <div className="text-zinc-500 text-xs">
            Workspace root + editors for qail
          </div>
        </div>
        <QButton
          variant="accent"
          icon={<EditOutlined />}
          onClick={() => setMode("editRoot")}
        >
          Edit root
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
            onClick={() => setMode("editRoot")}
          />

          <div className="px-3 pt-3 pb-1 flex items-center justify-between">
            <div className="text-xs text-zinc-400 uppercase tracking-wider">
              Editors
            </div>
            <QButton
              variant="accent"
              icon={<PlusOutlined />}
              onClick={() => setMode("addEditor")}
            >
              Add
            </QButton>
          </div>

          {list.settings.editors.length === 0 ? (
            <div className="px-3 py-2 text-xs text-zinc-500">
              No editors configured. Add one to enable "open in editor".
            </div>
          ) : (
            list.settings.editors.map((e) => {
              const isDefault = e.name === list.settings.defaultEditor;
              const menuItems = [
                ...(isDefault
                  ? []
                  : [
                      {
                        label: "Set as default",
                        onClick: () => editors.setDefault(e.name),
                      },
                    ]),
                {
                  label: "Remove",
                  danger: true,
                  onClick: () => editors.remove(e.name),
                },
              ];
              return (
                <ToolboxListItem
                  key={e.name}
                  id={`editor-${e.name}`}
                  primary={
                    <div className="flex items-center gap-2">
                      <span>{e.name}</span>
                      {isDefault && (
                        <Tag
                          color="gold"
                          className="!text-[10px] !leading-4 !m-0"
                        >
                          Default
                        </Tag>
                      )}
                    </div>
                  }
                  secondary={
                    <span className="font-mono text-xs">{e.command}</span>
                  }
                  icon={<CodeOutlined />}
                  menuItems={menuItems}
                />
              );
            })
          )}
        </ScrollableLayout>
      </div>
    </div>
  );
};
