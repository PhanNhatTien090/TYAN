"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import type { TaskSchedule } from "@/lib/feature-types";

const empty = {
  name: "",
  schedule_type: "once",
  timezone_mode: "system",
  run_date: "",
  end_date: "",
  start_time: "09:00",
  end_time: "17:00",
  weekdays: [1, 2, 3, 4, 5],
  task_ids: "",
  enabled: false,
};
export default function SchedulesPage() {
  const [items, setItems] = useState<TaskSchedule[]>([]);
  const [form, setForm] = useState(empty);
  const [editing, setEditing] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    try {
      setItems(((await api.schedules()).schedules ?? []).map(s => ({...s, task_ids:s.task_ids ?? []})));
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
  async function action(fn: () => Promise<unknown>) {
    setBusy(true);
    try {
      await fn();
      await load();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  async function save(e: FormEvent) {
    e.preventDefault();
    const ids = form.task_ids
      .split(/[\s,]+/)
      .filter(Boolean)
      .map(Number);
    if (!ids.length || ids.some((id) => !Number.isSafeInteger(id) || id <= 0)) {
      setError("Enter one or more positive task IDs.");
      return;
    }
    await action(async () => {
      const body = { ...form, task_ids: ids, start_immediately: false };
      if (editing) await api.updateSchedule(editing, body);
      else await api.createSchedule(body);
      setForm(empty);
      setEditing(null);
    });
  }
  return (
    <div className="flex flex-col gap-6 p-4 md:p-6">
      <div>
        <h1 className="font-bold font-mono text-2xl">Task schedules</h1>
        <p className="text-muted-foreground">
          Control when local lab tasks may run. New schedules are paused until you enable them.
        </p>
      </div>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <Card>
        <CardHeader>
          <CardTitle>{editing ? `Edit schedule #${editing}` : "Create schedule"}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={save} className="grid gap-4 md:grid-cols-2">
            <label htmlFor="schedules-field-1" className="grid gap-2">
              Name
              <Input
                id="schedules-field-1"
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </label>
            <label htmlFor="schedules-field-2" className="grid gap-2">
              Task IDs (comma separated)
              <Input
                id="schedules-field-2"
                required
                placeholder="1, 2"
                value={form.task_ids}
                onChange={(e) => setForm({ ...form, task_ids: e.target.value })}
              />
            </label>
            <label htmlFor="schedules-field-3" className="grid gap-2">
              Repeat
              <select
                id="schedules-field-3"
                className="rounded-md border bg-background p-2"
                value={form.schedule_type}
                onChange={(e) => setForm({ ...form, schedule_type: e.target.value })}
              >
                <option value="once">Once</option>
                <option value="weekly">Weekly</option>
              </select>
            </label>
            <label htmlFor="schedules-field-4" className="grid gap-2">
              Timezone
              <select
                id="schedules-field-4"
                className="rounded-md border bg-background p-2"
                value={form.timezone_mode}
                onChange={(e) => setForm({ ...form, timezone_mode: e.target.value })}
              >
                <option value="system">Server timezone</option>
                <option value="beijing">UTC+08:00</option>
              </select>
            </label>
            {form.schedule_type === "once" ? (
              <>
                <label htmlFor="schedules-field-5" className="grid gap-2">
                  Start date
                  <Input
                    id="schedules-field-5"
                    type="date"
                    required
                    value={form.run_date}
                    onChange={(e) => setForm({ ...form, run_date: e.target.value })}
                  />
                </label>
                <label htmlFor="schedules-field-6" className="grid gap-2">
                  End date (optional)
                  <Input
                    id="schedules-field-6"
                    type="date"
                    value={form.end_date}
                    onChange={(e) => setForm({ ...form, end_date: e.target.value })}
                  />
                </label>
              </>
            ) : (
              <fieldset className="flex flex-wrap gap-3 md:col-span-2">
                <legend className="mb-2">Weekdays</legend>
                {["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map((day, i) => (
                  <label key={day} className="flex items-center gap-1">
                    <input
                      type="checkbox"
                      checked={form.weekdays.includes(i + 1)}
                      onChange={(e) =>
                        setForm({
                          ...form,
                          weekdays: e.target.checked
                            ? [...form.weekdays, i + 1]
                            : form.weekdays.filter((d) => d !== i + 1),
                        })
                      }
                    />
                    {day}
                  </label>
                ))}
              </fieldset>
            )}
            <label htmlFor="schedules-field-7" className="grid gap-2">
              Start time
              <Input
                id="schedules-field-7"
                type="time"
                required
                value={form.start_time}
                onChange={(e) => setForm({ ...form, start_time: e.target.value })}
              />
            </label>
            <label htmlFor="schedules-field-8" className="grid gap-2">
              End time
              <Input
                id="schedules-field-8"
                type="time"
                required
                value={form.end_time}
                onChange={(e) => setForm({ ...form, end_time: e.target.value })}
              />
            </label>
            <div className="flex gap-2 md:col-span-2">
              <Button disabled={busy} type="submit">
                {editing ? "Save changes" : "Create paused schedule"}
              </Button>
              {editing && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setEditing(null);
                    setForm(empty);
                  }}
                >
                  Cancel
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>
      <div className="flex justify-between">
        <h2 className="font-semibold text-lg">Schedules</h2>
        <Button variant="outline" disabled={busy} onClick={() => void load()}>
          Refresh
        </Button>
      </div>
      {loading ? (
        <p>Loading schedules…</p>
      ) : !items.length ? (
        <p className="text-muted-foreground">
          No schedules yet. Create a paused schedule above, then enable it when your lab is ready.
        </p>
      ) : (
        items.map((s) => (
          <Card key={s.id}>
            <CardHeader>
              <CardTitle>
                {s.name}{" "}
                <span className="text-muted-foreground text-sm">
                  #{s.id} · {s.enabled ? s.status : "paused"}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <p>
                {s.schedule_type === "weekly"
                  ? `Weekly (${s.weekdays?.map((d) => ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"][d - 1]).join(", ")})`
                  : s.run_date}{" "}
                · {s.start_time}–{s.end_time || "open"} · {s.timezone || s.timezone_mode}
              </p>
              <p className="text-muted-foreground">
                Tasks: {s.task_ids.join(", ")}
                {s.next_start ? ` · Next start: ${s.next_start}` : ""}
              </p>
              {s.last_error && <p className="text-destructive">{s.last_error}</p>}
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    setEditing(s.id);
                    setForm({
                      ...empty,
                      ...s,
                      start_time: s.start_time.slice(0, 5),
                      end_time: (s.end_time || "17:00").slice(0, 5),
                      weekdays: s.weekdays || [],
                      task_ids: s.task_ids.join(", "),
                      run_date: s.run_date || "",
                      end_date: s.end_date || "",
                    });
                  }}
                >
                  Edit
                </Button>
                <Button
                  disabled={busy}
                  onClick={() => void action(() => api.scheduleAction(s.id, s.enabled ? "pause" : "resume"))}
                >
                  {s.enabled ? "Pause" : "Enable"}
                </Button>
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    if (window.confirm(`Run the tasks in ${s.name} now?`))
                      void action(() => api.scheduleAction(s.id, "run-now"));
                  }}
                >
                  Run now
                </Button>
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    if (window.confirm(`Delete schedule ${s.name}? Tasks will be kept.`))
                      void action(() => api.deleteSchedule(s.id));
                  }}
                >
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  );
}
