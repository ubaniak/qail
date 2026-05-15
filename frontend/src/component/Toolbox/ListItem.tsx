// JetBrains-Toolbox-style row: avatar/icon, primary/secondary text,
// optional injected child slot (checkbox, badge, …) and an overflow
// menu that appears on hover. Selected state applies a subtle accent
// background. Hooks into Antd's Dropdown for the menu.

import { MoreOutlined } from "@ant-design/icons";
import { Avatar, Button, Dropdown } from "antd";
import type { MenuProps } from "antd";
import type { ReactNode } from "react";

export type MenuAction = {
  label: string;
  onClick?: (id: string) => void;
  danger?: boolean;
  children?: MenuAction[];
};

export type ToolboxListItemProps = {
  id: string;
  primary: string | ReactNode;
  secondary?: string | ReactNode;
  icon?: ReactNode;
  selected?: boolean;
  onClick?: () => void;
  menuItems?: MenuAction[];
  children?: ReactNode;
};

export default function ToolboxListItem({
  id,
  primary,
  secondary,
  icon,
  selected = false,
  onClick,
  menuItems = [],
  children,
}: ToolboxListItemProps) {
  const renderPrimary = () =>
    typeof primary === "string" ? (
      <div className="text-sm font-semibold text-zinc-100">{primary}</div>
    ) : (
      primary
    );

  const renderSecondary = () =>
    !secondary ? null : typeof secondary === "string" ? (
      <div className="text-xs text-zinc-400 mt-0.5 truncate">{secondary}</div>
    ) : (
      <div className="mt-0.5">{secondary}</div>
    );

  const toMenuItems = (
    actions: MenuAction[],
    keyPrefix = ""
  ): MenuProps["items"] =>
    actions.map((item, i) => {
      const key = `${keyPrefix}${i}-${item.label}`;
      if (item.children && item.children.length > 0) {
        return {
          key,
          label: item.label,
          danger: item.danger,
          children: toMenuItems(item.children, `${key}-`),
        };
      }
      return {
        key,
        label: item.label,
        danger: item.danger,
        onClick: ({ domEvent }) => {
          domEvent.stopPropagation();
          item.onClick?.(id);
        },
      };
    });

  const dropdownItems: MenuProps["items"] = toMenuItems(menuItems);

  return (
    <div
      onClick={onClick}
      className={[
        "group rounded-lg mb-1 transition-colors relative",
        onClick ? "cursor-pointer" : "",
        selected
          ? "bg-emerald-900/20 hover:bg-emerald-900/30"
          : "hover:bg-zinc-800/50",
      ].join(" ")}
    >
      <div className="flex items-center px-3 py-2 gap-3">
        {children}

        <Avatar
          shape="square"
          size={32}
          className={selected ? "!bg-emerald-600" : "!bg-zinc-700"}
        >
          {icon ?? null}
        </Avatar>

        <div className="flex-1 min-w-0">
          {renderPrimary()}
          {renderSecondary()}
        </div>

        {menuItems.length > 0 && (
          <Dropdown
            menu={{ items: dropdownItems }}
            trigger={["click"]}
            placement="bottomRight"
          >
            <Button
              type="text"
              size="small"
              icon={<MoreOutlined />}
              onClick={(e) => e.stopPropagation()}
              className={
                selected
                  ? "!opacity-70"
                  : "!opacity-0 group-hover:!opacity-70 transition-opacity"
              }
            />
          </Dropdown>
        )}
      </div>
    </div>
  );
}
