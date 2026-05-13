// Fixed top bar — Q logo left, Settings icon right. The whole bar is
// the Wails drag region via data-drag (-webkit-app-region: drag in
// index.css). Interactive children inherit no-drag via the override.
//
// Frosted-glass background matches qail_ui; on macOS frameless the
// blur shows the desktop behind in a pleasing way.

import { SettingOutlined } from "@ant-design/icons";
import { Button } from "antd";

export type TopBarProps = {
  onSettingsClick?: () => void;
  settingsActive?: boolean;
};

export default function TopBar({ onSettingsClick, settingsActive }: TopBarProps) {
  return (
    <div
      data-drag
      className="fixed top-0 left-0 right-0 z-50 h-14 px-4 flex items-center justify-between
                 border-b border-zinc-800/60 backdrop-blur-2xl
                 bg-zinc-900/65"
    >
      {/* LEFT — Q logo */}
      <div className="flex items-center gap-2">
        <div
          className="w-8 h-8 rounded-lg flex items-center justify-center
                     bg-emerald-500 text-zinc-950 font-bold text-base select-none"
        >
          Q
        </div>
        <span className="text-zinc-200 text-sm font-medium tracking-tight">
          qail
        </span>
      </div>

      {/* RIGHT — actions */}
      <div className="flex items-center gap-1">
        <Button
          type={settingsActive ? "primary" : "text"}
          shape="circle"
          size="small"
          icon={<SettingOutlined />}
          onClick={onSettingsClick}
          aria-label="Settings"
        />
      </div>
    </div>
  );
}
