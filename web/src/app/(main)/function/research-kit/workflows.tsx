"use client";

import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { http } from "@/lib/api";

interface Workflow {
  name: string;
  description: string;
  writes_database: boolean;
  example: Record<string, unknown>;
}
interface WorkflowResponse {
  command: string;
  mode: string;
  result: unknown;
}

export default function ResearchWorkflows() {
  const [items, setItems] = useState<Workflow[]>([]),
    [name, setName] = useState("skill-load"),
    [input, setInput] = useState("{}");
  const [result, setResult] = useState<WorkflowResponse | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    void http<{ items: Workflow[] }>("/research-kit/workflows")
      .then((r) => {
        setItems(r.items);
        setInput(JSON.stringify(r.items[0].example, null, 2));
      })
      .catch((e) => setError(String(e)));
  }, []);
  const selected = items.find((item) => item.name === name);
  const object =
    result?.result && typeof result.result === "object" ? (result.result as Record<string, unknown>) : null;
  const encoded = typeof object?.docx_base64 === "string" ? object.docx_base64 : "";
  const actionLabel = selected?.writes_database ? "Run and save" : "Run workflow";
  let resultText = JSON.stringify(result?.result, null, 2);
  if (typeof object?.markdown === "string") resultText = object.markdown;
  if (encoded) resultText = "DOCX draft generated. Use the download button below.";
  async function run() {
    setBusy(true);
    setError("");
    setResult(null);
    try {
      const parsed = JSON.parse(input);
      setResult(
        await http<WorkflowResponse>(`/research-kit/workflows/${name}`, {
          method: "POST",
          body: JSON.stringify(parsed),
        }),
      );
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="grid min-w-0 gap-4 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>17 native review workflows</CardTitle>
          <p className="text-muted-foreground text-sm">
            Adaptations of the source command names. Each description states what runs in TYAN. This console accepts
            structured JSON, not shell commands, and makes no external model call.
          </p>
        </CardHeader>
        <CardContent className="grid gap-4">
          <label htmlFor="workflow-name" className="grid gap-2 text-sm">
            Workflow
            <select
              id="workflow-name"
              aria-label="Workflow"
              className="rounded-md border bg-background p-2"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                const row = items.find((item) => item.name === e.target.value);
                setInput(JSON.stringify(row?.example ?? {}, null, 2));
                setResult(null);
                setError("");
              }}
            >
              {items.map((item) => (
                <option key={item.name} value={item.name}>
                  /{item.name}
                  {item.writes_database ? " · writes database" : ""}
                </option>
              ))}
            </select>
          </label>
          <p className="text-sm">{selected?.description}</p>
          <label htmlFor="workflow-input" className="grid gap-2 text-sm">
            Parameters JSON
            <Textarea
              id="workflow-input"
              aria-label="Parameters JSON"
              className="min-h-80 font-mono text-xs"
              value={input}
              onChange={(e) => setInput(e.target.value)}
            />
          </label>
          <p className="text-muted-foreground text-xs">
            Use the actual existing task ID as a quoted string. Example IDs are placeholders. /new-engagement creates a
            task; /update-state saves a record. Other workflows prepare or read data.
          </p>
          {selected?.writes_database && (
            <p className="rounded-md border p-3 text-sm">
              Running this workflow writes to your local database. Check the JSON fields before running it.
            </p>
          )}
          <Button disabled={busy || !selected} onClick={() => void run()}>
            {busy ? "Running…" : actionLabel}
          </Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Workflow result</CardTitle>
        </CardHeader>
        <CardContent className="grid min-w-0 gap-4">
          {error && (
            <p role="alert" className="break-words text-destructive">
              {error}
            </p>
          )}
          {!result && (
            <p className="text-muted-foreground text-sm">
              Choose a workflow and provide the supplied artifact or record fields.
            </p>
          )}
          {result && (
            <>
              <p className="font-mono text-primary text-xs">
                /{result.command} · {result.mode}
              </p>
              <pre className="max-h-[750px] overflow-auto whitespace-pre-wrap break-words rounded-md border p-3 text-xs">
                {resultText}
              </pre>
            </>
          )}
          {encoded && (
            <Button
              onClick={() => {
                const bytes = Uint8Array.from(atob(encoded), (c) => c.charCodeAt(0));
                const url = URL.createObjectURL(
                  new Blob([bytes], {
                    type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
                  }),
                );
                const link = document.createElement("a");
                link.href = url;
                link.download = "tyan-research-draft.docx";
                document.body.appendChild(link);
                link.click();
                link.remove();
                URL.revokeObjectURL(url);
              }}
            >
              Download workflow DOCX
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
