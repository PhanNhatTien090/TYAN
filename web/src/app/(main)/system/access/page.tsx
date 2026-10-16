"use client";
import { type FormEvent, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
export default function AccessPage() {
  const [enabled, setEnabled] = useState(false),
    [username, setUsername] = useState(""),
    [password, setPassword] = useState(""),
    [error, setError] = useState(""),
    [saved, setSaved] = useState(false),
    [busy, setBusy] = useState(false),
    [loaded, setLoaded] = useState(false);
  useEffect(() => {
    api
      .basicAuthSettings()
      .then((s) => {
        setEnabled(s.enabled);
        setUsername(s.username);
        setLoaded(true);
      })
      .catch((e) => setError(String(e)));
  }, []);
  async function save(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setSaved(false);
    setError("");
    try {
      await api.saveBasicAuthSettings({ enabled, username, ...(password ? { password } : {}) });
      setPassword("");
      setSaved(true);
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-6 p-4 md:p-6">
      <div>
        <h1 className="font-bold font-mono text-2xl">Access gate</h1>
        <p className="text-muted-foreground">
          Add an optional browser authentication gate before the TYAN operator login.
        </p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>HTTP Basic Auth</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={save} className="grid max-w-lg gap-4">
            <p className="text-muted-foreground">
              Keep TYAN local to your lab. If you use a remote connection, protect it with HTTPS. Keep these credentials
              available before enabling the gate.
            </p>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
              Enable access gate
            </label>
            <label htmlFor="access-field-1" className="grid gap-2">
              Username
              <Input
                id="access-field-1"
                autoComplete="off"
                required={enabled}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </label>
            <label htmlFor="access-field-2" className="grid gap-2">
              New password
              <Input
                id="access-field-2"
                type="password"
                autoComplete="new-password"
                placeholder="Leave empty to keep the current password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            {error && (
              <p role="alert" className="text-destructive">
                {error}
              </p>
            )}
            {saved && <p role="status">Access settings saved.</p>}
            <Button type="submit" disabled={busy || !loaded}>
              Save access settings
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
