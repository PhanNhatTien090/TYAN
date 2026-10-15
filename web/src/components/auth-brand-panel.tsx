export function AuthBrandPanel() {
  return (
    <aside className="tyan-auth-panel relative hidden w-5/12 flex-col justify-between overflow-hidden p-12 lg:flex">
      <div className="flex items-center gap-3 font-mono text-sm font-semibold tracking-[0.2em]">
        <img src="/tyan-mark.svg" alt="" width={36} height={36} /> TYAN
      </div>
      <div className="relative space-y-7 py-16">
        <span className="inline-flex items-center gap-2 rounded-full border border-current/25 px-3 py-1 font-mono text-[10px] tracking-widest">
          <span className="size-1.5 rounded-full bg-current" /> AI-POWERED SECURITY
        </span>
        <h1 className="text-5xl font-semibold leading-[1.12] tracking-tight text-[#e2f8ee]">Understand your<br />attack surface.</h1>
        <p className="max-w-sm text-sm leading-7 text-[#9cb9ac]">One workspace for your assets, agents, evidence, and security research.</p>
        <div className="rounded-xl border border-current/15 bg-black/20 p-5 font-mono text-xs leading-7 text-[#9cb9ac]">
          <p><span className="text-[#69f5c3]">&gt;</span> initialize workspace</p>
          <p><span className="text-[#69f5c3]">&gt;</span> connect intelligence</p>
          <p><span className="text-[#69f5c3]">&gt;</span> investigate with clarity<span className="ml-1 inline-block h-3 w-1.5 bg-[#69f5c3] align-middle" /></p>
        </div>
      </div>
      <p className="font-mono text-[10px] tracking-widest text-[#78968a]">TYAN / BUILT ON ARTEX</p>
    </aside>
  );
}
