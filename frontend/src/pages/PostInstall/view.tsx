// Script viewer / editor. Fetches the file via Bindings.ReadScript and
// renders it in a fixed-width pane. When initialMode is "edit" (or the
// user toggles Edit), the body becomes a textarea that saves back via
// Bindings.WriteScript.

import { useEffect, useState } from "react";
import { Alert, Input, Spin, Tag } from "antd";

import { QButton } from "../../component/Buttons/QButton";
import { fetchScriptContents } from "../../api/qail";
import { useQailService } from "../../providers/qailservice";
import type { Scope } from "../../types";

export type ViewScriptProps = {
  name: string;
  scope: Scope;
  onClose: () => void;
  initialMode?: "view" | "edit";
};

export const ViewScript = ({
  name,
  scope,
  onClose,
  initialMode = "view",
}: ViewScriptProps) => {
  const { scripts: svc } = useQailService();
  const [body, setBody] = useState<string>("");
  const [draft, setDraft] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState(initialMode === "edit");
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);
    fetchScriptContents(name, scope)
      .then((b) => {
        if (cancelled) return;
        setBody(b);
        setDraft(b);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [name, scope]);

  const dirty = editing && draft !== body;

  const save = () => {
    setSaving(true);
    svc.write(name, scope, draft, () => {
      setBody(draft);
      setSaving(false);
      setEditing(false);
    });
  };

  const cancelEdit = () => {
    setDraft(body);
    setEditing(false);
  };

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 flex items-center justify-between">
        <div>
          <div className="text-zinc-100 text-base font-semibold flex items-center gap-2">
            <span>{name}</span>
            <Tag color={scope === "workspace" ? "blue" : "purple"}>{scope}</Tag>
            {editing && (
              <Tag color="gold" className="!m-0">
                Editing
              </Tag>
            )}
          </div>
          <div className="text-zinc-500 text-xs">
            {editing ? "Edits write back to ~/.qail/scripts/" : "Read-only preview"}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {editing ? (
            <>
              <QButton variant="cancel" onClick={cancelEdit} disabled={saving}>
                Cancel
              </QButton>
              <QButton
                variant="accent"
                onClick={save}
                disabled={!dirty || saving}
              >
                {saving ? "Saving…" : "Save"}
              </QButton>
            </>
          ) : (
            <>
              <QButton
                variant="accent"
                onClick={() => setEditing(true)}
                disabled={loading || !!error}
              >
                Edit
              </QButton>
              <QButton variant="cancel" onClick={onClose}>
                Back
              </QButton>
            </>
          )}
        </div>
      </div>

      <div className="flex-1 min-h-0 px-3 pb-3">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <Spin />
          </div>
        ) : error ? (
          <Alert type="error" showIcon message="Read failed" description={error} />
        ) : editing ? (
          <Input.TextArea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            className="!h-full !font-mono !text-xs !bg-zinc-950/80 !text-zinc-200"
            style={{ height: "100%", resize: "none" }}
            spellCheck={false}
            autoSize={false}
          />
        ) : (
          <pre className="bg-zinc-950/80 border border-zinc-800/60 rounded-lg p-3 text-xs text-zinc-200 overflow-auto h-full font-mono whitespace-pre">
            {body}
          </pre>
        )}
      </div>
    </div>
  );
};
