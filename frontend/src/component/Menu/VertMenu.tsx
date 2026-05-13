// Standalone overflow menu (three-dot) — used by callers that want a
// dropdown outside a ToolboxListItem. Internally same Dropdown + Menu
// as ToolboxListItem, kept here for parity with qail_ui's API.

import { MoreOutlined } from "@ant-design/icons";
import { Button, Dropdown } from "antd";
import type { MenuProps } from "antd";

export type VertMenuOptions = {
  label: string;
  onClick: (id: string) => void;
  danger?: boolean;
};

export type VertMenuProps = {
  id: string;
  options: VertMenuOptions[];
};

export const VertMenu = ({ id, options }: VertMenuProps) => {
  const items: MenuProps["items"] = options.map((o, i) => ({
    key: `${i}-${o.label}`,
    label: o.label,
    danger: o.danger,
    onClick: ({ domEvent }) => {
      domEvent.stopPropagation();
      o.onClick(id);
    },
  }));

  return (
    <Dropdown menu={{ items }} trigger={["click"]} placement="bottomRight">
      <Button type="text" size="small" icon={<MoreOutlined />} />
    </Dropdown>
  );
};
