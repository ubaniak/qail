// Sticky tab bar that sits below TopBar. Antd's Tabs handles indicator
// + keyboard nav; we keep the controlled `value` prop so App can wire
// Cmd/Ctrl+1/2/3 hotkeys + localStorage persistence.

import { Tabs as AntTabs } from "antd";
import type { ReactNode } from "react";

export type ToolboxTab = {
  key: string;
  label: string;
  component: ReactNode;
  shortcut?: string;
};

export type ToolboxTabsProps = {
  tabs: ToolboxTab[];
  activeKey: string;
  onChange: (key: string) => void;
};

export default function ToolboxTabs({
  tabs,
  activeKey,
  onChange,
}: ToolboxTabsProps) {
  const items = tabs.map((t) => ({
    key: t.key,
    label: (
      <span className="flex items-center gap-2">
        <span>{t.label}</span>
        {t.shortcut && (
          <span className="text-[10px] uppercase tracking-wider text-zinc-500">
            {t.shortcut}
          </span>
        )}
      </span>
    ),
    children: <div className="pt-3">{t.component}</div>,
  }));

  return (
    <AntTabs
      activeKey={activeKey}
      onChange={onChange}
      items={items}
      size="small"
      tabBarStyle={{
        marginBottom: 0,
        paddingLeft: 8,
        paddingRight: 8,
      }}
      destroyInactiveTabPane={false}
    />
  );
}
