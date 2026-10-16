"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { api } from "@/lib/api";
import type { FindingCaseRow } from "@/lib/feature-types";

export default function FindingCasesPage() {
  const [reports, setReports] = useState<Record<string,string>>({});
  const [items, setItems] = useState<FindingCaseRow[]>([]);
  const [task, setTask] = useState("");
  const [ids, setIds] = useState("");
  const [title, setTitle] = useState("");
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const load = useCallback(async () => {
    try {
      const data = await api.findingCases(page);
      setItems(data.items ?? []);
      setTotal(data.total);
      setError("");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [page]);
  useEffect(() => {
    void load();
  }, [load]);
  async function merge(e: FormEvent) {
    e.preventDefault();
    const findingIds = ids
      .split(/[\s,]+/)
      .filter(Boolean)
      .map(Number);
    if (
      findingIds.length < 2 ||
      findingIds.some((id) => !Number.isSafeInteger(id) || id <= 0) ||
      !Number.isSafeInteger(Number(task)) ||
      Number(task) <= 0
    ) {
      setError("Enter a task ID and at least two positive finding IDs.");
      return;
    }
    setBusy(true);
    try {
      await api.mergeFindingCase({ task_id: Number(task), finding_ids: findingIds, title, reason });
      setIds("");
      setTitle("");
      setReason("");
      await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-6 p-4 md:p-6">
      <div>
        <h1 className="font-bold font-mono text-2xl">Finding cases</h1>
        <p className="text-muted-foreground">
          Group findings that share a verified root cause. Similar titles alone are not evidence of duplication.
        </p>
      </div>
      {error && (
        <p className="text-destructive" role="alert">
          {error}
        </p>
      )}
      <Card>
        <CardHeader>
          <CardTitle>Merge findings into a case</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={merge} className="grid gap-4 md:grid-cols-2">
            <label htmlFor="finding-cases-field-1" className="grid gap-2">
              Source task ID
              <Input
                id="finding-cases-field-1"
                required
                inputMode="numeric"
                value={task}
                onChange={(e) => setTask(e.target.value)}
              />
            </label>
            <label htmlFor="finding-cases-field-2" className="grid gap-2">
              Finding IDs (comma separated)
              <Input
                id="finding-cases-field-2"
                required
                placeholder="12, 15"
                value={ids}
                onChange={(e) => setIds(e.target.value)}
              />
            </label>
            <label htmlFor="finding-cases-field-3" className="grid gap-2 md:col-span-2">
              Case title
              <Input id="finding-cases-field-3" required value={title} onChange={(e) => setTitle(e.target.value)} />
            </label>
            <label htmlFor="finding-cases-field-4" className="grid gap-2 md:col-span-2">
              Root-cause evidence and merge reason
              <Textarea
                id="finding-cases-field-4"
                required
                value={reason}
                onChange={(e) => setReason(e.target.value)}
              />
            </label>
            <Button type="submit" disabled={busy}>
              Merge findings
            </Button>
          </form>
        </CardContent>
      </Card>
      <div className="flex items-center justify-between">
        <h2 className="font-semibold text-lg">Cases and standalone findings · {total}</h2>
        <Button variant="outline" onClick={() => void load()}>
          Refresh
        </Button>
      </div>
      {loading ? (
        <p>Loading cases…</p>
      ) : !items.length ? (
        <p className="text-muted-foreground">
          No matching cases. Existing standalone findings remain on the Findings page.
        </p>
      ) : (
        items.map((row) => {
          const c = row.case;
          return c ? (
            <Card key={`case-${c.id}`}>
              <CardHeader>
                <CardTitle>
                  #{c.id} · {c.title}
                </CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <p className="text-muted-foreground">
                  {c.severity} · {c.count} findings · version {c.version}
                </p>
                <p className="whitespace-pre-wrap">{c.reason}</p>
                <details>
                  <summary className="cursor-pointer">Unified report</summary>
                  <Button variant="outline" className="mt-3" onClick={() => { void api.findingCase(c.id).then(data => setReports(prev => ({...prev, [c.id]:data.case.report || "No unified report yet."}))).catch(e => setError(String(e))); }}>Load report</Button>
                  <pre className="mt-3 whitespace-pre-wrap break-words font-sans text-sm">
                    {reports[c.id] ?? "Load the current unified report. The reporter can update it after reviewing member evidence."}
                  </pre>
                </details>
              </CardContent>
            </Card>
          ) : (
            <Card key={`finding-${row.finding?.id}`}>
              <CardHeader>
                <CardTitle>
                  Finding #{row.finding?.id} · {row.finding?.name || "Untitled finding"}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p>
                  {row.finding?.severity} · Standalone finding · Task {row.finding?.task_id}
                </p>
                <a className="text-primary underline" href={`/function/findings/detail?id=${row.finding?.id}`}>
                  Open finding
                </a>
              </CardContent>
            </Card>
          );
        })
      )}
      <div className="flex items-center gap-3">
        <Button variant="outline" disabled={page <= 1} onClick={() => setPage(page - 1)}>
          Previous
        </Button>
        <span>Page {page}</span>
        <Button variant="outline" disabled={page * 20 >= total} onClick={() => setPage(page + 1)}>
          Next
        </Button>
      </div>
    </div>
  );
}
