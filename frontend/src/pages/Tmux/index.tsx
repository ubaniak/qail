// Tmux session list — live sessions on the host, plus copy-attach /
// kill actions. The list refreshes on Refresh click + after every kill.

import { CodeOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Popconfirm } from "antd";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";

export const TmuxIndex = () => {
  const { list, mux, remove } = useQailService();

  const renderSessions = () => {
    if (!list.tmux || list.tmux.length === 0) {
      return (
        <Alert
          type="info"
          showIcon
          message="No active tmux sessions"
          description="Use a workspace's tmux action to start one."
        />
      );
    }
    return list.tmux.map((session) => (
      <ToolboxListItem
        key={session}
        id={session}
        primary={session}
        icon={<CodeOutlined />}
        menuItems={[
          { label: "Copy attach command", onClick: (id) => mux.attach(id) },
          { label: "Kill session", danger: true, onClick: (id) => remove.tmux(id) },
        ]}
      />
    ));
  };

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-sm font-semibold">Sessions</div>
          <div className="text-zinc-500 text-xs">
            {list.tmux?.length ?? 0} active
          </div>
        </div>
        <Popconfirm
          title="Refresh sessions?"
          okText="Refresh"
          cancelText="Cancel"
          onConfirm={() => mux.refresh()}
        >
          <Button icon={<ReloadOutlined />} size="small">
            Refresh
          </Button>
        </Popconfirm>
      </div>
      <div className="flex-1 min-h-0">
        <ScrollableLayout>{renderSessions()}</ScrollableLayout>
      </div>
    </div>
  );
};
