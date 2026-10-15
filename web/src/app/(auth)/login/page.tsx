"use client";

import { useEffect, useRef, useState } from "react";

import { useRouter } from "next/navigation";

import { ShieldCheck } from "lucide-react";

import { AuthBrandPanel } from "@/components/auth-brand-panel";
import { usageTerms } from "@/config/usage-terms";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/lib/api";
import { auth } from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(true);
  const [agreed, setAgreed] = useState(false);
  const [termsOpen, setTermsOpen] = useState(false);
  const [readToEnd, setReadToEnd] = useState(false);
  const termsBodyRef = useRef<HTMLDivElement>(null);

  // 滚动到条款底部（含无需滚动即可完整展示的情况）方可点击「同意」。
  function handleTermsScroll() {
    const el = termsBodyRef.current;
    if (!el) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 8) setReadToEnd(true);
  }

  useEffect(() => {
    if (!termsOpen) return;
    // 打开时重置，并处理内容本就不足一屏、无法触发滚动的场景。
    setReadToEnd(false);
    const el = termsBodyRef.current;
    if (el && el.scrollHeight <= el.clientHeight + 8) setReadToEnd(true);
  }, [termsOpen]);

  useEffect(() => {
    // 已登录直接进主界面（静态导出下无 middleware 代劳这层跳转）。
    const token = auth.getToken();
    if (token) {
      // localStorage 可能仍有凭据但 cookie 已丢失。先同步，再发起全新请求，
      // 避免服务端守卫或路由缓存把跳转送回仍处于 checking 状态的登录页。
      auth.setToken(token);
      window.location.replace("/function/tasks");
      return;
    }
    api
      .authStatus()
      .then(({ initialized }) => {
        if (!initialized) router.replace("/setup");
      })
      .catch(() => setError("Unable to connect to the backend"))
      .finally(() => setChecking(false));
  }, [router]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!agreed) {
      setError("Read and accept the usage terms first.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const { token } = await api.login("ARTEX", password);
      auth.setToken(token);
      window.location.replace("/function/tasks");
    } catch {
      setError("Incorrect password.");
    } finally {
      setLoading(false);
    }
  }

  if (checking) {
    return <div role="status" className="flex min-h-dvh items-center justify-center text-muted-foreground">Checking sign-in status…</div>;
  }

  return (
    <div className="flex min-h-dvh">
      <AuthBrandPanel />
      <main className="flex min-w-0 flex-1 items-center justify-center p-6 sm:p-12">
        <div className="w-full max-w-sm space-y-8 py-12">
          <div className="space-y-3">
            <img src="/tyan-mark.svg" alt="TYAN" width={44} height={44} className="mb-8 lg:hidden" />
            <p className="font-mono text-[10px] tracking-[0.2em] text-primary">TYAN / OPERATOR ACCESS</p>
            <h2 className="text-3xl font-semibold tracking-tight">Welcome back.</h2>
            <p className="text-sm leading-6 text-muted-foreground">Sign in to your security workspace.</p>
          </div>
          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            <div className="space-y-2">
              <Label htmlFor="username">Account</Label>
              <Input id="username" value="TYAN" readOnly className="bg-muted text-muted-foreground" autoComplete="username" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Enter your password" autoFocus autoComplete="current-password" />
            </div>
            <div className="flex items-start gap-2">
              <Checkbox id="agree-terms" checked={agreed} onCheckedChange={(v) => setAgreed(v === true)} className="mt-0.5" />
              <div className="text-sm leading-6 text-muted-foreground">
                <Label htmlFor="agree-terms" className="inline font-normal">I have read and agree to the </Label>
                <button type="button" onClick={() => setTermsOpen(true)} className="inline text-left text-primary underline underline-offset-4">usage terms and disclaimer</button>
              </div>
            </div>
            {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
            <Button type="submit" className="h-11 w-full" disabled={loading || !password || !agreed}>{loading ? "Signing in…" : "Sign in to TYAN"}</Button>
          </form>
          <p className="border-t pt-5 font-mono text-[10px] tracking-wide text-muted-foreground">SELF-HOSTED WORKSPACE · BUILT ON ARTEX</p>
        </div>
      </main>
      <Dialog open={termsOpen} onOpenChange={setTermsOpen}>
        <DialogContent className="gap-0 p-0 sm:max-w-2xl">
          <DialogHeader className="flex-row items-center gap-3 border-b px-6 py-4">
            <ShieldCheck className="size-6 shrink-0 text-primary" />
            <div className="space-y-1">
              <DialogTitle>TYAN usage terms and disclaimer</DialogTitle>
              <p className="text-xs text-muted-foreground">Upstream notice v1.0 · September 18, 2026 · Read all terms before signing in.</p>
            </div>
          </DialogHeader>
          <div ref={termsBodyRef} onScroll={handleTermsScroll} className="max-h-[60vh] space-y-6 overflow-y-auto px-6 py-5 text-sm leading-7 text-muted-foreground">
            <p className="rounded-lg border bg-muted/40 p-3">This notice sets out the upstream authors’ usage terms for the software. Read it carefully, including the restrictions and limitations of liability. The original notice is retained in README.zh-CN.md.</p>
            {usageTerms.map((term, index) => (
              <section key={term.title} className="space-y-2">
                <h3 className="font-medium text-foreground">{index + 1}. {term.title}</h3>
                <p>{term.body}</p>
                {"items" in term && <ul className="list-disc space-y-2 rounded-lg border border-destructive/20 bg-destructive/5 p-4 pl-8">{term.items.map((item) => <li key={item}>{item}</li>)}</ul>}
              </section>
            ))}
          </div>
          <DialogFooter className="mx-0 mb-0 flex-col gap-3 border-t px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-xs text-muted-foreground">{readToEnd ? "You have reached the end of the terms." : "Scroll to the bottom to confirm."}</p>
            <DialogClose asChild><Button type="button" disabled={!readToEnd} onClick={() => { setAgreed(true); setError(""); }}>I agree to all terms</Button></DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
