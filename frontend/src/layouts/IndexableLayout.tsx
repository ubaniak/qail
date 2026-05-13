// Header band that sits above every list — search input on the left
// and an optional "add" button on the right. Frosted-glass styling
// matches TopBar so the visual rhythm stays consistent.

import { PlusOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Input } from "antd";
import type { ReactNode } from "react";

export type IndexableLayoutProps = {
  children: ReactNode;
  onAdd?: () => void;
  onSearch?: (query: string) => void;
  placeholder?: string;
};

export const IndexableLayout = ({
  children,
  onAdd,
  onSearch,
  placeholder = "Search…",
}: IndexableLayoutProps) => {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-3 py-2 bg-zinc-900/60 backdrop-blur border border-zinc-800/60 rounded-lg mx-2 mt-2">
        <Input
          allowClear
          variant="borderless"
          placeholder={placeholder}
          prefix={<SearchOutlined className="text-zinc-500" />}
          onChange={(e) => onSearch?.(e.target.value)}
          className="!bg-transparent"
        />
        {onAdd && (
          <Button
            type="primary"
            shape="circle"
            size="small"
            icon={<PlusOutlined />}
            onClick={onAdd}
            aria-label="Add"
          />
        )}
      </div>
      <div className="flex-1 overflow-hidden">{children}</div>
    </div>
  );
};
