// Repo listing — register/remove individual repos. The list is plain;
// no edit form because repos are immutable name+url pairs.

import { GithubOutlined } from "@ant-design/icons";
import { Alert, Tag } from "antd";
import { useState } from "react";
import ToolboxListItem from "../../component/Toolbox/ListItem";
import { IndexableLayout, ScrollableLayout } from "../../layouts";
import { useQailService } from "../../providers/qailservice";
import { AddRepo } from "./add";
import { EditRepo } from "./edit";
import { RemoveRepo } from "./remove";

export const RepoIndex = () => {
  const { list, copy } = useQailService();
  const [query, setQuery] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [toRemove, setToRemove] = useState<string | undefined>();
  const [toEdit, setToEdit] = useState<string | undefined>();

  if (showAdd) return <AddRepo onClose={() => setShowAdd(false)} />;
  if (toEdit)
    return <EditRepo name={toEdit} onClose={() => setToEdit(undefined)} />;
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
    return filtered.map(([name, repo]) => {
      const postCount = (repo.postInstall ?? []).length;
      return (
        <ToolboxListItem
          key={name}
          id={name}
          primary={
            <div className="flex items-center gap-2">
              <span>{name}</span>
              {postCount > 0 && (
                <Tag color="purple" className="!text-[10px] !leading-4 !m-0">
                  {postCount} post-install
                </Tag>
              )}
            </div>
          }
          secondary={repo.url}
          icon={<GithubOutlined />}
          menuItems={[
            {
              label: "Copy URL",
              onClick: () => copy.text(repo.url, "Repo URL"),
            },
            {
              label: "Copy name",
              onClick: (id) => copy.text(id, "Repo name"),
            },
            { label: "Edit post-install", onClick: (id) => setToEdit(id) },
            { label: "Remove", danger: true, onClick: (id) => setToRemove(id) },
          ]}
          onClick={() => setToEdit(name)}
        />
      );
    });
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
