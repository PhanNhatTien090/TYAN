"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { BookOpen, Check, Copy, Download, FlaskConical, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import {
  type FindingDraft,
  type ResearchOperation,
  type ResearchResult,
  type ResearchSkill,
  researchAPI,
  type ToolAvailability,
} from "@/lib/research-kit";

import ResearchWorkflows from "./workflows";
import ResearchWorkspace from "./workspace";

const selectStyle = "min-w-0 rounded-md border bg-background px-3 py-2 text-sm";
const emptyDraft: FindingDraft = {
  title: "",
  summary: "",
  steps: [],
  expected: "",
  actual: "",
  impact: "",
  remediation: "",
  evidence: [],
  scope_confirmed: false,
  preconditions_proven: false,
  vector: "",
};
const examples = {
  openapi: `{
  "openapi": "3.0.3",
  "info": {"title": "Local lab API", "version": "1"},
  "security": [{"labSession": []}],
  "paths": {
    "/items/{id}": {
      "get": {
        "parameters": [{"in": "path", "name": "id", "required": true, "schema": {"type": "string"}}],
        "responses": {"200": {"description": "Lab item", "content": {"application/json": {"schema": {"type": "object", "properties": {"label": {"type": "string"}}}}}}}
      }
    }
  }
}`,
  normalize:
    '{"template-id":"lab-control","matched-at":"http://127.0.0.1:3000/items/1","info":{"name":"Lab control observation","severity":"info"}}',
  redact:
    'Authorization: Bearer FAKE_LAB_TOKEN\nCookie: session=FAKE_LAB_SESSION\n{"email":"sample@example.test","password":"FAKE_PASSWORD"}',
  cvss: "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:L/VI:N/VA:N/SC:N/SI:N/SA:N",
  har: '{"log":{"entries":[{"request":{"method":"GET","url":"http://127.0.0.1:3000/items/1?session=FAKE"},"response":{"status":200,"content":{"mimeType":"application/json"}},"time":12}]}}',
};

function ResultView({ result }: { result: ResearchResult | null }) {
  if (!result)
    return <p className="text-muted-foreground">Choose an operation and supply an artifact to see the result.</p>;
  const validation = result.validation ?? result;
  return (
    <div className="grid gap-4" aria-live="polite">
      {result.score !== undefined && (
        <div className="rounded-lg border p-5">
          <p className="font-mono text-4xl">
            {result.score.toFixed(1)} <span className="text-lg">{result.severity}</span>
          </p>
          <p className="mt-2 break-all font-mono text-xs">{result.vector}</p>
          {!result.matches_claim && (
            <p className="mt-2 text-destructive">The claimed score differs from this vector.</p>
          )}
        </div>
      )}
      {result.total !== undefined && (
        <p className="font-mono">
          {result.total} unique records
          {result.duplicates !== undefined ? ` · ${result.duplicates} duplicates removed` : ""}
          {result.truncated ? " · preview limited" : ""}
        </p>
      )}
      {result.operations && (
        <div className="max-h-[560px] overflow-auto rounded-lg border">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b">
                <th className="p-3">Operation</th>
                <th className="p-3">Declared auth</th>
                <th className="p-3">Shape</th>
              </tr>
            </thead>
            <tbody>
              {result.operations.map((op) => (
                <tr key={`${op.method} ${op.path}`} className="border-b">
                  <td className="p-3 font-mono">
                    {op.method} {op.path}
                  </td>
                  <td className="p-3">{op.authentication}</td>
                  <td className="p-3">
                    Path: {op.path_parameters.join(", ") || "—"}
                    <br />
                    Query: {op.query_parameters.join(", ") || "—"}
                    <br />
                    Body: {op.body_fields.join(", ") || "—"}
                    <br />
                    Response: {op.response_fields.join(", ") || "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {result.items && (
        <div className="grid max-h-[560px] gap-2 overflow-auto">
          {result.items.map((item) => (
            <div className="rounded-lg border p-3" key={item.id}>
              <p className="font-medium">{item.title}</p>
              <p className="break-all font-mono text-muted-foreground text-xs">{item.url}</p>
              <p className="mt-1 text-xs">
                {item.severity} · {item.status}
              </p>
            </div>
          ))}
        </div>
      )}
      {result.entries && (
        <div className="grid max-h-[560px] gap-2 overflow-auto">
          {result.entries.map((entry, index) => (
            <div key={`${entry.method}-${entry.url}-${index}`} className="grid gap-1 rounded-lg border p-3">
              <p className="break-all font-mono text-xs">
                {entry.method} {entry.url}
              </p>
              <p className="text-sm">
                HTTP {entry.status} · {entry.mime} · {entry.time_ms} ms
              </p>
            </div>
          ))}
        </div>
      )}
      {validation.checks && (
        <div className="grid gap-2">
          <p className="font-mono">{validation.status?.replaceAll("_", " ")}</p>
          {validation.checks.map((check) => (
            <div key={check.name} className="rounded-md border p-3">
              <span className={check.passed ? "text-primary" : "text-muted-foreground"}>
                {check.passed ? "✓" : "○"} {check.name.replaceAll("_", " ")}
              </span>
              <p className="mt-1 text-muted-foreground text-xs">{check.detail}</p>
            </div>
          ))}
        </div>
      )}
      {(result.text || result.markdown) && (
        <pre className="max-h-[650px] overflow-auto whitespace-pre-wrap break-words rounded-lg border bg-muted/30 p-4 text-sm">
          {result.text ?? result.markdown}
        </pre>
      )}
      {validation.warnings?.map((warning) => (
        <p key={warning} className="text-muted-foreground text-xs">
          {warning}
        </p>
      ))}
      {result.notice && <p className="text-muted-foreground text-xs">{result.notice}</p>}
    </div>
  );
}

export default function ResearchKitPage() {
  const [skills, setSkills] = useState<ResearchSkill[]>([]),
    [loading, setLoading] = useState(true);
  const [query, setQuery] = useState(""),
    [tech, setTech] = useState(""),
    [category, setCategory] = useState("all");
  const [sourceCommit, setSourceCommit] = useState("");
  const [error, setError] = useState("");
  const [operation, setOperation] = useState<"openapi" | "normalize" | "redact" | "cvss" | "har">("openapi");
  const [input, setInput] = useState(""),
    [format, setFormat] = useState("nuclei"),
    [claimed, setClaimed] = useState("");
  const [result, setResult] = useState<ResearchResult | null>(null),
    [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState<FindingDraft>(emptyDraft),
    [reportResult, setReportResult] = useState<ResearchResult | null>(null);
  const [tools, setTools] = useState<ToolAvailability[]>([]),
    [inventoryLoaded, setInventoryLoaded] = useState(false);
  const [copied, setCopied] = useState("");
  const load = useCallback(async (q = "", t = "") => {
    setLoading(true);
    try {
      const data = await researchAPI.catalog(q, t);
      setSkills(data.items ?? []);
      setSourceCommit(data.source_commit);
      setError("");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  async function run(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body =
        operation === "cvss"
          ? { vector: input, ...(claimed !== "" ? { claimed_score: Number(claimed) } : {}) }
          : { input, ...(operation === "normalize" ? { source: format } : {}), limit: 100 };
      setResult(await researchAPI.run(operation, body));
    } catch (e) {
      setError(String(e));
      setResult(null);
    } finally {
      setBusy(false);
    }
  }
  async function prepare(action: ResearchOperation) {
    setBusy(true);
    setError("");
    try {
      setReportResult(await researchAPI.run(action, { finding: draft }));
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  async function inventory() {
    setBusy(true);
    try {
      const data = await researchAPI.inventory();
      setTools(data.items ?? []);
      setInventoryLoaded(true);
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  async function copy(name: string) {
    try {
      await navigator.clipboard.writeText(name);
      setCopied(name);
    } catch {
      setError("Clipboard access was unavailable; select the skill name to copy it.");
    }
  }
  async function upload(file: File | undefined) {
    if (!file) return;
    if (file.size > 900_000) {
      setError("Use an artifact smaller than 900 KB to leave room for the request envelope.");
      return;
    }
    setInput(await file.text());
    setResult(null);
  }
  const visible = skills.filter((skill) => category === "all" || skill.category === category);
  return (
    <div className="flex min-w-0 flex-col gap-6 p-4 md:p-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-2 font-mono text-primary text-xs tracking-widest">TYAN / RESEARCH WORKSPACE</p>
          <h1 className="font-bold font-mono text-2xl">Research Kit</h1>
          <p className="mt-2 max-w-3xl text-muted-foreground">
            Topic-focused review skills, offline artifact analysis and evidence-driven report drafts.
          </p>
        </div>
        <div className="rounded-lg border px-4 py-3 font-mono text-xs">
          65 review skills · 14 native tools · 6 review agents
        </div>
      </div>
      {error && (
        <p role="alert" className="rounded-lg border border-destructive/40 p-3 text-destructive">
          {error}
        </p>
      )}
      <Tabs defaultValue="library" className="min-w-0">
        <TabsList className="mb-4 flex h-auto w-full flex-wrap justify-start gap-1 md:w-fit">
          <TabsTrigger value="library">
            <BookOpen className="size-4" />
            Skills
          </TabsTrigger>
          <TabsTrigger value="analyze">
            <FlaskConical className="size-4" />
            Analyze artifacts
          </TabsTrigger>
          <TabsTrigger value="report">Finding & report</TabsTrigger>
          <TabsTrigger value="workspace">Knowledge & models</TabsTrigger>
          <TabsTrigger value="workflows">Workflows</TabsTrigger>
          <TabsTrigger
            value="tools"
            onClick={() => {
              if (!inventoryLoaded) void inventory();
            }}
          >
            Tool readiness
          </TabsTrigger>
        </TabsList>
        <TabsContent value="library" className="grid gap-4">
          <Card>
            <CardContent className="pt-5">
              <form
                className="grid gap-3 md:grid-cols-4"
                onSubmit={(e) => {
                  e.preventDefault();
                  void load(query, tech);
                }}
              >
                <label htmlFor="research-search" className="grid gap-2 text-sm">
                  Topic
                  <Input
                    id="research-search"
                    placeholder="OAuth, storage, evidence…"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </label>
                <label htmlFor="research-tech" className="grid gap-2 text-sm">
                  Tech stack tags
                  <Input
                    id="research-tech"
                    placeholder="node graphql"
                    value={tech}
                    onChange={(e) => setTech(e.target.value)}
                  />
                </label>
                <label htmlFor="research-category" className="grid gap-2 text-sm">
                  Category
                  <select
                    id="research-category"
                    aria-label="Category"
                    className={selectStyle}
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                  >
                    {[
                      "all",
                      "web",
                      "identity",
                      "code",
                      "cloud",
                      "mobile",
                      "ai",
                      "infrastructure",
                      "inventory",
                      "workflow",
                      "evidence",
                      "reporting",
                    ].map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </label>
                <Button type="submit" className="self-end" disabled={loading}>
                  Find skills
                </Button>
              </form>
            </CardContent>
          </Card>
          <p className="text-muted-foreground text-sm">
            {visible.length} matching packages · TYAN review adaptations from claude-redteam{" "}
            <span className="font-mono">{sourceCommit.slice(0, 8)}</span>. Copy a name to invoke through the existing
            Skill tool.
          </p>
          {loading && <p>Loading skill catalog…</p>}
          {!loading && !visible.length && (
            <p>
              No matches. Try a broader topic or exact tags such as node, graphql, java, go, android, aws or report.
            </p>
          )}
          <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
            {visible.map((skill) => (
              <Card key={skill.name}>
                <CardHeader>
                  <div className="flex items-start justify-between gap-2">
                    <CardTitle className="break-words text-base">{skill.name}</CardTitle>
                    <Button
                      aria-label={`Copy ${skill.name}`}
                      variant="ghost"
                      size="icon"
                      onClick={() => void copy(skill.name)}
                    >
                      {copied === skill.name ? <Check className="size-4" /> : <Copy className="size-4" />}
                    </Button>
                  </div>
                  <p className="font-mono text-primary text-xs">{skill.category} / review workflow</p>
                </CardHeader>
                <CardContent className="grid gap-3">
                  <p className="text-muted-foreground text-sm">{skill.description}</p>
                  <p className="break-words font-mono text-xs">{skill.tags.join(" · ")}</p>
                  <details>
                    <summary className="cursor-pointer text-muted-foreground text-xs">Source and compatibility</summary>
                    <p className="mt-2 break-all text-xs">
                      {skill.source_path}
                      <br />
                      SHA-256: {skill.source_sha256}
                      <br />
                      Adapted review procedure; not the upstream executable or offensive automation.
                    </p>
                  </details>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
        <TabsContent value="workspace">
          <ResearchWorkspace />
        </TabsContent>
        <TabsContent value="workflows">
          <ResearchWorkflows />
        </TabsContent>
        <TabsContent value="analyze" className="grid gap-4 xl:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Analyze a supplied artifact</CardTitle>
              <p className="text-muted-foreground text-sm">
                These helpers run locally without contacting targets or calling an LLM.
              </p>
            </CardHeader>
            <CardContent>
              <form onSubmit={run} className="grid gap-4">
                <label htmlFor="research-operation" className="grid gap-2">
                  Operation
                  <select
                    id="research-operation"
                    aria-label="Operation"
                    className={selectStyle}
                    value={operation}
                    onChange={(e) => {
                      setOperation(e.target.value as typeof operation);
                      setInput("");
                      setResult(null);
                      setError("");
                    }}
                  >
                    <option value="openapi">OpenAPI / Swagger summary</option>
                    <option value="normalize">Normalize saved tool output</option>
                    <option value="redact">Minimize evidence text</option>
                    <option value="cvss">Compute CVSS 4.0</option>
                    <option value="har">Saved HAR metadata</option>
                  </select>
                </label>
                {operation === "normalize" && (
                  <label htmlFor="research-source" className="grid gap-2">
                    Output format
                    <select
                      id="research-source"
                      aria-label="Output format"
                      className={selectStyle}
                      value={format}
                      onChange={(e) => setFormat(e.target.value)}
                    >
                      <option value="nuclei">Nuclei JSON / JSONL</option>
                      <option value="ffuf">ffuf JSON</option>
                      <option value="nmap">Nmap XML</option>
                    </select>
                  </label>
                )}
                <label htmlFor="research-artifact" className="grid gap-2">
                  {operation === "cvss" ? "CVSS 4.0 vector" : "Artifact text"}
                  <Textarea
                    id="research-artifact"
                    aria-label={operation === "cvss" ? "CVSS 4.0 vector" : "Artifact text"}
                    required
                    className="min-h-64 font-mono text-xs"
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder={operation === "cvss" ? "CVSS:4.0/…" : "Paste the local artifact here"}
                  />
                </label>
                {operation === "cvss" ? (
                  <label htmlFor="research-claimed" className="grid gap-2">
                    Claimed score (optional)
                    <Input
                      id="research-claimed"
                      type="number"
                      min="0"
                      max="10"
                      step="0.1"
                      value={claimed}
                      onChange={(e) => setClaimed(e.target.value)}
                    />
                  </label>
                ) : (
                  <label htmlFor="research-file" className="grid gap-2 text-sm">
                    Or open a local text file
                    <Input
                      id="research-file"
                      type="file"
                      accept=".json,.jsonl,.yaml,.yml,.xml,.txt,.log"
                      onChange={(e) => void upload(e.target.files?.[0])}
                    />
                  </label>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button type="submit" disabled={busy}>
                    Analyze
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setInput(examples[operation]);
                      if (operation === "normalize") setFormat("nuclei");
                      setResult(null);
                    }}
                  >
                    Load example
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Result</CardTitle>
            </CardHeader>
            <CardContent className="min-w-0">
              <ResultView result={result} />
              {result && (
                <Button
                  variant="outline"
                  className="mt-4"
                  onClick={() => {
                    void navigator.clipboard
                      .writeText(JSON.stringify(result, null, 2))
                      .catch((e) => setError(String(e)));
                  }}
                >
                  Copy result JSON
                </Button>
              )}
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="report" className="grid gap-4 xl:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Finding draft</CardTitle>
              <p className="text-muted-foreground text-sm">
                Structure the evidence before review. This form does not create or publish a finding.
              </p>
            </CardHeader>
            <CardContent className="grid gap-4">
              {(
                [
                  ["title", "Title"],
                  ["summary", "Summary"],
                  ["expected", "Expected control behavior"],
                  ["actual", "Observed behavior"],
                  ["impact", "Demonstrated impact"],
                  ["remediation", "Remediation and retest"],
                ] as const
              ).map(([key, label]) => (
                <label key={key} htmlFor={`draft-${key}`} className="grid gap-2 text-sm">
                  {label}
                  <Textarea
                    id={`draft-${key}`}
                    aria-label={label}
                    value={draft[key]}
                    onChange={(e) => setDraft({ ...draft, [key]: e.target.value })}
                  />
                </label>
              ))}
              <label htmlFor="draft-steps" className="grid gap-2 text-sm">
                Reproduction steps (one per line)
                <Textarea
                  id="draft-steps"
                  aria-label="Reproduction steps (one per line)"
                  value={draft.steps.join("\n")}
                  onChange={(e) => setDraft({ ...draft, steps: e.target.value.split("\n") })}
                />
              </label>
              <label htmlFor="draft-evidence" className="grid gap-2 text-sm">
                Evidence references (one per line)
                <Textarea
                  id="draft-evidence"
                  aria-label="Evidence references (one per line)"
                  placeholder="traffic:123"
                  value={draft.evidence.join("\n")}
                  onChange={(e) => setDraft({ ...draft, evidence: e.target.value.split("\n") })}
                />
              </label>
              <label htmlFor="draft-vector" className="grid gap-2 text-sm">
                CVSS 4.0 vector (optional)
                <Input
                  id="draft-vector"
                  value={draft.vector}
                  onChange={(e) => setDraft({ ...draft, vector: e.target.value })}
                />
              </label>
              <label className="flex items-start gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={draft.scope_confirmed}
                  onChange={(e) => setDraft({ ...draft, scope_confirmed: e.target.checked })}
                />
                This work belongs to my isolated local lab scope.
              </label>
              <label className="flex items-start gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={draft.preconditions_proven}
                  onChange={(e) => setDraft({ ...draft, preconditions_proven: e.target.checked })}
                />
                The prerequisites in the impact statement are demonstrated by the evidence.
              </label>
              <div className="flex flex-wrap gap-2">
                <Button disabled={busy} onClick={() => void prepare("validate")}>
                  Check readiness
                </Button>
                <Button variant="outline" disabled={busy} onClick={() => void prepare("report")}>
                  Preview draft
                </Button>
                <Button
                  variant="outline"
                  disabled={busy || !draft.title.trim()}
                  onClick={() => {
                    setBusy(true);
                    void researchAPI
                      .downloadReport(draft)
                      .catch((e) => setError(String(e)))
                      .finally(() => setBusy(false));
                  }}
                >
                  <Download className="size-4" />
                  DOCX draft
                </Button>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Review & preview</CardTitle>
            </CardHeader>
            <CardContent className="min-w-0">
              <ResultView result={reportResult} />
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="tools" className="grid gap-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between gap-3">
                <CardTitle>Optional external tooling</CardTitle>
                <Button variant="outline" disabled={busy} onClick={() => void inventory()}>
                  <RefreshCw className="size-4" />
                  Refresh
                </Button>
              </div>
              <p className="text-muted-foreground text-sm">
                The native Research Kit tools work without these programs. PATH detection does not verify versions or
                functionality; no program is executed or installed here.
              </p>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {tools.map((tool) => (
                  <div key={tool.name} className="flex justify-between gap-3 rounded-lg border p-3">
                    <div>
                      <p className="font-mono">{tool.name}</p>
                      <p className="text-muted-foreground text-xs">{tool.category}</p>
                    </div>
                    <span className={`self-center text-xs ${tool.on_path ? "text-primary" : "text-muted-foreground"}`}>
                      {tool.on_path ? "On PATH" : "Not on PATH"}
                    </span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
