"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  type ModelCoverage,
  type ResearchKind,
  type ResearchRecord,
  type ReviewRole,
  type SystemModel,
  type WorkspaceContent,
  workspaceAPI,
} from "@/lib/research-workspace";

const sampleModel: SystemModel = {
  nodes: [
    { id: "client", kind: "role", label: "Lab viewer", status: "reviewed", references: ["source:roles"] },
    { id: "api", kind: "feature", label: "Local item API", status: "reviewing", references: [] },
    { id: "policy", kind: "hypothesis", label: "Object authorization", status: "todo", references: [] },
  ],
  edges: [
    { from: "client", to: "api", relation: "calls" },
    { from: "api", to: "policy", relation: "requires control review" },
  ],
  questions: ["Which fixture demonstrates policy enforcement?"],
};
const emptyContent = (): WorkspaceContent => ({ body: "", tags: [], references: [], session: "" });
const exampleAPI = {
  version: "3.0.3",
  total: 0,
  by_method: {},
  operations: [],
  truncated: false,
  notice: "Paste the complete research_openapi_summary result.",
};

function ModelView({ model }: { model: SystemModel }) {
  const nodes = model.nodes.slice(0, 24);
  const positions = new Map(
    nodes.map((node, index) => {
      const angle = (index / Math.max(nodes.length, 1)) * Math.PI * 2;
      return [node.id, { x: 300 + Math.cos(angle) * 200, y: 230 + Math.sin(angle) * 160 }];
    }),
  );
  return (
    <div className="min-w-0 overflow-auto rounded-lg border">
      <svg viewBox="0 0 600 460" className="w-full min-w-80" role="img" aria-label="System model relationships">
        <title>System model relationships, first 24 nodes</title>
        {Array.from(new Map(model.edges.map((edge) => [JSON.stringify(edge), edge])).values()).map((edge) => {
          const a = positions.get(edge.from),
            b = positions.get(edge.to);
          if (!a || !b) return null;
          return (
            <g key={JSON.stringify(edge)}>
              <line x1={a.x} y1={a.y} x2={b.x} y2={b.y} stroke="currentColor" opacity="0.3" />
              <text x={(a.x + b.x) / 2} y={(a.y + b.y) / 2} fill="currentColor" fontSize="10" textAnchor="middle">
                {edge.relation.slice(0, 28)}
              </text>
            </g>
          );
        })}
        {nodes.map((node) => {
          const p = positions.get(node.id);
          if (!p) return null;
          return (
            <g key={node.id}>
              <circle cx={p.x} cy={p.y} r="27" fill="var(--background)" stroke="var(--primary)" />
              <text x={p.x} y={p.y + 4} fill="currentColor" fontSize="10" textAnchor="middle">
                {node.id.slice(0, 9)}
              </text>
              <text x={p.x} y={p.y + 43} fill="currentColor" fontSize="11" textAnchor="middle">
                {node.label.slice(0, 28)}
              </text>
            </g>
          );
        })}
      </svg>
      <p className="p-3 text-muted-foreground text-xs">
        Visual preview shows up to 24 nodes. The complete model is retained in JSON. Relations are review hypotheses,
        not verified attack paths.
      </p>
    </div>
  );
}

export default function ResearchWorkspace() {
  const [kind, setKind] = useState<ResearchKind>("kb"),
    [task, setTask] = useState("0"),
    [query, setQuery] = useState("");
  const [items, setItems] = useState<ResearchRecord[]>([]),
    [roles, setRoles] = useState<ReviewRole[]>([]);
  const [key, setKey] = useState(""),
    [title, setTitle] = useState(""),
    [version, setVersion] = useState(0);
  const [content, setContent] = useState<WorkspaceContent>(emptyContent),
    [structured, setStructured] = useState("");
  const [history, setHistory] = useState<ResearchRecord[]>([]),
    [coverage, setCoverage] = useState<ModelCoverage | null>(null),
    [model, setModel] = useState<SystemModel | null>(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [deleting, setDeleting] = useState(false);
  useEffect(() => {
    void workspaceAPI
      .roles()
      .then((r) => setRoles(r.items))
      .catch((e) => setError(String(e)));
  }, []);
  function clear() {
    setKey("");
    setTitle("");
    setVersion(0);
    setContent(emptyContent());
    setStructured("");
    setHistory([]);
    setCoverage(null);
    setModel(null);
    setDeleting(false);
    setNotice("");
  }
  async function action(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  async function load() {
    setItems((await workspaceAPI.list(kind, task, query)).items);
  }
  function select(row: ResearchRecord, keepVersion = false) {
    setKey(row.key);
    setTitle(row.title);
    if (!keepVersion) setVersion(row.version);
    setContent(row.content);
    let data = "";
    if (row.kind === "model") data = JSON.stringify(row.content.model, null, 2);
    if (row.kind === "api") data = JSON.stringify(row.content.api, null, 2);
    setStructured(data);
    setModel(row.content.model ?? null);
    setCoverage(null);
    setDeleting(false);
  }
  function record(): ResearchRecord {
    const data = { ...content };
    if (kind === "model") data.model = JSON.parse(structured);
    if (kind === "api") data.api = JSON.parse(structured);
    return { kind, task_id: task || "0", key, title, version, content: data };
  }
  return (
    <div className="grid min-w-0 gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Persistent research workspace</CardTitle>
          <p className="text-muted-foreground text-sm">
            Knowledge, session notes, system models and endpoint catalogs. Records and revisions are ordinary database
            content; use the Credentials tab for secrets. Task deletion removes its workspace records.
          </p>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-3 md:grid-cols-4">
            <label htmlFor="workspace-kind" className="grid gap-2 text-sm">
              Collection
              <select
                id="workspace-kind"
                aria-label="Collection"
                className="rounded-md border bg-background p-2"
                value={kind}
                onChange={(e) => {
                  setKind(e.target.value as ResearchKind);
                  setItems([]);
                  clear();
                }}
              >
                <option value="kb">Knowledge base</option>
                <option value="memory">Session memory</option>
                <option value="model">System models</option>
                <option value="api">API catalogs</option>
              </select>
            </label>
            <label htmlFor="workspace-task" className="grid gap-2 text-sm">
              Task ID (0 = global KB)
              <Input
                id="workspace-task"
                inputMode="numeric"
                value={task}
                onChange={(e) => {
                  setTask(e.target.value);
                  setItems([]);
                  clear();
                }}
              />
            </label>
            <label htmlFor="workspace-query" className="grid gap-2 text-sm">
              Search text
              <Input id="workspace-query" value={query} onChange={(e) => setQuery(e.target.value)} />
            </label>
            <Button disabled={busy} className="self-end" onClick={() => void action(load)}>
              Load records
            </Button>
          </div>
          <p className="text-muted-foreground text-xs">
            Only KB supports task 0. Other collections require an existing task. Search returns the latest 200 matching
            records. Select a record to edit; stale saves return a conflict.
          </p>
        </CardContent>
      </Card>
      {error && (
        <p role="alert" className="break-words rounded-lg border border-destructive p-3 text-destructive">
          {error}
        </p>
      )}
      {notice && (
        <p role="status" className="rounded-lg border p-3 text-primary">
          {notice}
        </p>
      )}
      <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(240px,1fr)_minmax(0,2fr)]">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between gap-2">
              <CardTitle>{items.length} records</CardTitle>
              <Button variant="outline" onClick={clear}>
                New record
              </Button>
            </div>
          </CardHeader>
          <CardContent className="grid max-h-[680px] gap-2 overflow-auto">
            {items.length === 0 && (
              <p className="text-muted-foreground text-sm">Load this collection or create its first record.</p>
            )}
            {items.map((row) => (
              <button
                type="button"
                key={row.key}
                onClick={() => {
                  select(row);
                  setHistory([]);
                  setNotice("");
                }}
                className="rounded-md border p-3 text-left hover:border-primary"
              >
                <p className="break-words font-medium">{row.title}</p>
                <p className="break-all font-mono text-muted-foreground text-xs">
                  {row.key} · v{row.version}
                </p>
              </button>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{version ? `Edit record · v${version}` : "New record"}</CardTitle>
          </CardHeader>
          <CardContent className="grid min-w-0 gap-4">
            <label htmlFor="workspace-key" className="grid gap-2 text-sm">
              Record key
              <Input
                id="workspace-key"
                value={key}
                disabled={version > 0}
                placeholder="authorization-review"
                onChange={(e) => setKey(e.target.value)}
              />
            </label>
            <label htmlFor="workspace-title" className="grid gap-2 text-sm">
              Record title
              <Input id="workspace-title" value={title} onChange={(e) => setTitle(e.target.value)} />
            </label>
            <label htmlFor="workspace-body" className="grid gap-2 text-sm">
              Notes / factual summary
              <Textarea
                id="workspace-body"
                aria-label="Notes / factual summary"
                value={content.body}
                onChange={(e) => setContent({ ...content, body: e.target.value })}
              />
            </label>
            <label htmlFor="workspace-tags" className="grid gap-2 text-sm">
              Tags (comma-separated)
              <Input
                id="workspace-tags"
                value={content.tags.join(",")}
                onChange={(e) => setContent({ ...content, tags: e.target.value.split(",") })}
              />
            </label>
            <label htmlFor="workspace-refs" className="grid gap-2 text-sm">
              Evidence references (one per line)
              <Textarea
                id="workspace-refs"
                aria-label="Evidence references (one per line)"
                value={content.references.join("\n")}
                onChange={(e) => setContent({ ...content, references: e.target.value.split("\n") })}
              />
            </label>
            {kind === "memory" && (
              <label htmlFor="workspace-session" className="grid gap-2 text-sm">
                Session label
                <Input
                  id="workspace-session"
                  value={content.session}
                  onChange={(e) => setContent({ ...content, session: e.target.value })}
                />
              </label>
            )}
            {(kind === "model" || kind === "api") && (
              <>
                <label htmlFor="workspace-json" className="grid gap-2 text-sm">
                  {kind === "model" ? "System model JSON" : "OpenAPI summary JSON"}
                  <Textarea
                    id="workspace-json"
                    aria-label={kind === "model" ? "System model JSON" : "OpenAPI summary JSON"}
                    className="min-h-72 font-mono text-xs"
                    value={structured}
                    onChange={(e) => setStructured(e.target.value)}
                  />
                </label>
                <Button
                  variant="outline"
                  onClick={() => setStructured(JSON.stringify(kind === "model" ? sampleModel : exampleAPI, null, 2))}
                >
                  Load structure example
                </Button>
              </>
            )}
            <div className="flex flex-wrap gap-2">
              <Button
                disabled={busy || !title.trim() || !key.trim()}
                onClick={() =>
                  void action(async () => {
                    const saved = await workspaceAPI.save(record());
                    select(saved);
                    await load();
                    setHistory([]);
                    setNotice(`Saved revision ${saved.version}.`);
                  })
                }
              >
                Save record
              </Button>
              {version > 0 && (
                <>
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() => void action(async () => setHistory((await workspaceAPI.history(record())).items))}
                  >
                    Revision history
                  </Button>
                  <Button
                    variant="outline"
                    disabled={busy}
                    onClick={() => {
                      if (!deleting) {
                        setDeleting(true);
                        return;
                      }
                      void action(async () => {
                        await workspaceAPI.remove(record());
                        clear();
                        await load();
                        setNotice("Record and its revisions deleted.");
                      });
                    }}
                  >
                    {deleting ? "Confirm delete record and revisions" : "Delete record"}
                  </Button>
                </>
              )}
              {kind === "model" && (
                <Button
                  variant="outline"
                  disabled={busy || !structured}
                  onClick={() =>
                    void action(async () => {
                      const parsed = JSON.parse(structured) as SystemModel;
                      const result = await workspaceAPI.coverage(parsed);
                      setCoverage(result);
                      setModel(parsed);
                    })
                  }
                >
                  Check model coverage
                </Button>
              )}
              <Button
                variant="outline"
                disabled={!key || !title}
                onClick={() =>
                  void action(async () => {
                    const url = URL.createObjectURL(
                      new Blob([JSON.stringify(record(), null, 2)], { type: "application/json" }),
                    );
                    const a = document.createElement("a");
                    a.href = url;
                    a.download = `tyan-${kind}-${key}.json`;
                    a.click();
                    URL.revokeObjectURL(url);
                  })
                }
              >
                Export record JSON
              </Button>
            </div>
            {history.length > 0 && (
              <div className="grid gap-2 rounded-lg border p-3">
                <p className="font-medium">Latest 100 revisions</p>
                {history.map((row) => (
                  <div key={row.version} className="flex flex-wrap items-center justify-between gap-2">
                    <span>
                      v{row.version} · {row.title}
                    </span>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        select(row, true);
                        setNotice(`Loaded v${row.version} into the editor. Save creates a new revision.`);
                      }}
                    >
                      Load revision content
                    </Button>
                  </div>
                ))}
              </div>
            )}
            {coverage && (
              <div className="grid gap-2 rounded-lg border p-3">
                <p className="font-mono">
                  {coverage.reviewed}/{coverage.total} reviewed · {coverage.percent.toFixed(1)}%
                </p>
                <p className="text-muted-foreground text-xs">{coverage.notice}</p>
                {coverage.gaps.map((node) => (
                  <p key={node.id} className="text-sm">
                    {node.id}: {node.label} ({node.status})
                  </p>
                ))}
              </div>
            )}
            {model && <ModelView model={model} />}
          </CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Six native review agents</CardTitle>
          <p className="text-muted-foreground text-sm">
            Conversational agents under System → Agents. They use your configured LLM and normal permissions; no
            automatic task or external model call starts here. Names and prompts remain editable.
          </p>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {roles.map((role) => (
            <div key={role.key} className="grid gap-2 rounded-lg border p-3">
              <p className="font-medium">{role.name}</p>
              <p className="font-mono text-primary text-xs">{role.key}</p>
              <p className="text-muted-foreground text-sm">{role.description}</p>
              <details>
                <summary className="cursor-pointer text-xs">Default role instructions</summary>
                <p className="mt-2 whitespace-pre-wrap text-xs">{role.prompt}</p>
              </details>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
