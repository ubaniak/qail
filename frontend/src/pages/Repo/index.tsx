// Repo listing — register/remove individual repos. The list is plain;
// no edit form because repos are immutable name+url pairs.

import { GithubOutlined } from "@ant-design/icons";
import { Alert } from "antd";
import { useState } from "react";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import { AddRepo } from "./add";
import { RemoveRepo } from "./remove";

export const RepoIndex = () => {
  const { list } = useQailService();
  const [query, setQuery] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [toRemove, setToRemove] = useState<string | undefined>();

  if (showAdd) return <AddRepo onClose={() => setShowAdd(false)} />;
  if (toRemove)
    return <RemoveRepo name={toRemove} onClose={() => setToRemove(undefined)} />;

  const filtered = Object.entries(list.repos).filter(([n]) =>
    n.toLowerCase().includes(query.toLowerCase())
  );

  const renderBody = () => {
    if (filtered.length === 0) {
      return (
        <Alert
          type="info"
          showIcon
          message={query ? "No matches" : "No repos yet"}
          description={
            query
              ? `No repo name matches "${query}".`
              : "Use + to register a git URL, or run `qail repo add`."
          }
        />
      );
    }
    return filtered.map(([name, repo]) => (
      <ToolboxListItem
        key={name}
        id={name}
        primary={name}
        secondary={repo.url}
        icon={<GithubOutlined />}
        menuItems={[
          { label: "Remove", danger: true, onClick: (id) => setToRemove(id) },
        ]}
      />
    ));
  };

  return (
    <IndexableLayout
      placeholder="Search repos…"
      onAdd={() => setShowAdd(true)}
      onSearch={setQuery}
    >
      <ScrollableLayout>{renderBody()}</ScrollableLayout>
    </IndexableLayout>
  );
};
