"use client";

// LLM 重试配置的共用件：五层重试各自的「次数 + 间隔」。
//
// 五层从内到外：建连(SDK) → 空响应(SDK) → 同 provider 安全窗口 → 轮询熔断 → 意图重跑。
// 前三层跟着端点走，所以每个模型配置都能覆盖全局默认；后两层是进程级的，只有全局一份。
//
// 所有输入都遵循同一套「留空 = 不配置」语义，与后端 db.RetryRule 一致：
//   次数   空/0 = 用内置默认 | -1 = 关闭这层重试 | >0 = 用这个次数
//   间隔   空/0 = 用这层原本的指数退避 | >0 = 改用这个固定毫秒间隔

import * as React from "react";

import { Loader2Icon, SaveIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/lib/api";
import type { LLMRetryOverride, LLMRetryPolicy, LLMRetryRule } from "@/lib/types";

export const ZERO_RULE: LLMRetryRule = { attempts: 0, interval_ms: 0 };
export const ZERO_OVERRIDE: LLMRetryOverride = {
  connect: ZERO_RULE,
  empty: ZERO_RULE,
  stream: ZERO_RULE,
};
const ZERO_POLICY: LLMRetryPolicy = {
  ...ZERO_OVERRIDE,
  breaker: ZERO_RULE,
  intent: ZERO_RULE,
};

type LayerMeta = {
  title: string;
  /** 这层重试发生在哪、由谁执行 */
  where: string;
  /** 什么样的错误会走到这层——具体到状态码，别让人猜 */
  trigger: string;
  /** 长得像但【不】走这层的错误，省得填了没反应还以为是 bug */
  skips?: string;
  desc: string;
  attemptsLabel: string;
  /** 次数留空时的默认值，用于占位符 */
  defAttempts: number;
  /** 间隔留空时的默认策略，用于占位符 */
  defInterval: string;
  /** 次数填 -1 的含义 */
  offHint: string;
};

export const RETRY_LAYERS = {
  connect: {
    title: "Connection retries",
    where: "SDK... 200 before you get it.",
    trigger:
      "200: Network-level errors such as connect reset / read-write timeout / DNS failure, and HTTP 408, 429, 500, 502, 503, 504.",
    skips: "The rest of the state code (400 / 401 / 403 / 404 / 413 / 422, etc.) is definitive refusal, and re-issuance also fails and is thrown up.",
    desc: "The same request is reissued. Once the flow begins (200 already obtained), the cut-off does not belong to this layer.",
    attemptsLabel: "Retry count",
    defAttempts: 3,
    defInterval: "0.5s  ⁇ 1s  ⁇ 2s Index (top 8s)",
    offHint: "- = Not once.",
  },
  empty: {
    title: "Empty Response Retry",
    where: "SDK. Only openai format",
    trigger:
      "HTTP 200, finish_reason is normal stop, but there's not a single piece of content in the whole response — a gateway frame, a think field frame, a sample hiccup.",
    skips: "It's not counting because max_tokens is cut (it's going to be solved by raising the output ceiling and the re-run will only hit again).",
    desc: "Reissued the whole prompt, so it is more expensive in the context and the number should not be large.",
    attemptsLabel: "Retry count",
    defAttempts: 2,
    defInterval: "0.5s  ⁇ 1s  ⁇ 2s Index (top 8s)",
    offHint: "-1 = empty response is delivered as it is.",
  },
  stream: {
    title: "Retry with provider security window",
    where: ":: Before the undelivered output",
    trigger:
      "The flow has been created (and 200 obtained) before problems arise: the connection has been cut off, the supplier overloaded, the 429 / 5x error in the stream - and none of the tokens have been handed over to the caller.",
    skips:
      "The amount has been depleted (402 / insufficient_quota for a rotation configuration), the context is too long (413 / context length for compression), 400 / 401 / 403 / 404 / 422 for certainty, and is not repeated.",
    desc: "Replays the same request on the same configuration. Relays do not duplicate model outputs or tool execution because no output has been delivered.",
    attemptsLabel: "Retry count",
    defAttempts: 2,
    defInterval: "0.5s  ⁇ 1s index (4s cap)",
    offHint: "-1 = intended to run back directly to the outer layer.",
  },
  breaker: {
    title: "Rotation circuit breaker",
    where: ":: Process level, one global",
    trigger:
      "An instantaneous failure (429, 5x, network error) accrues to the threshold by melting; a failure of sufficient balance (402), failure of the key (401 / 403), absence of a model (404) and failure of such certainty does not look at the threshold for the first time.",
    skips: "Success once is zero, so the configuration of occasional windfalls will not be slowly reduced to melting.",
    desc: "When you melt, you cool, and during the cooling, you jump directly through this configuration. The status library is down and you do not lose.",
    attemptsLabel: "Consecutive failures before opening circuit",
    defAttempts: 3,
    defInterval: "1-min-5-min-30-min gradient",
    offHint: "-1 = instantaneous failure never melts (certainty failure still melts)",
  },
  intent: {
    title: "Intent retries",
    where: ":: Process level, one global",
    trigger:
      "The front floors were left unturned: workingrker closed with model_error — the inner layer was exhausted or the stream had begun to deliver the output before it was broken (it was unsafe to replay the whole article).",
    skips: "The amount is already processed by the rotation configuration and does not run again here; the task is suspended / terminated / immediately relinquished at the end, without taking the time of retreat.",
    desc: "The whole thing's going to run again from the top. It's the top floor, and a rerun means that the number of layers inside is going to go over again.",
    attemptsLabel: "Reruns",
    defAttempts: 2,
    defInterval: "Fixed 3s",
    offHint: "-1 = No rerun, the intention is to be directly judged as blocked",
  },
} satisfies Record<string, LayerMeta>;

type LayerKey = keyof typeof RETRY_LAYERS;

/** 毫秒的人话，只用于在输入框旁边回显，免得数零。 */
function humanMs(ms: number) {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${Number((ms / 1000).toFixed(2))}s`;
  return `${Number((ms / 60_000).toFixed(2))}min`;
}

/** 受控数字输入：空串 ↔ 0，中间态（"-"、"1e"）原样留在本地，不打扰父级。 */
function NumField({
  id,
  value,
  onChange,
  placeholder,
  min,
}: {
  id: string;
  value: number;
  onChange: (n: number) => void;
  placeholder: string;
  min: number;
}) {
  const [text, setText] = React.useState(value === 0 ? "" : String(value));
  // 父级换了一整套值（读取到策略、切换配置）时跟上；自己敲字时不会走到这里，
  // 因为那时 value 已经等于本地文本 parse 后的结果。
  React.useEffect(() => {
    const incoming = value === 0 ? "" : String(value);
    setText((cur) => (Number(cur || 0) === value ? cur : incoming));
  }, [value]);
  return (
    <Input
      id={id}
      type="number"
      min={min}
      className="w-28 shrink-0"
      value={text}
      placeholder={placeholder}
      onChange={(e) => {
        setText(e.target.value);
        const n = Number(e.target.value);
        onChange(e.target.value.trim() === "" || !Number.isFinite(n) ? 0 : Math.trunc(n));
      }}
    />
  );
}

/** 一层重试的两个旋钮。idPrefix 用来在同一页出现多次时保住 label 的 htmlFor。 */
export function RetryRuleFields({
  layer,
  idPrefix,
  value,
  onChange,
  compact,
}: {
  layer: LayerKey;
  idPrefix: string;
  value: LLMRetryRule;
  onChange: (r: LLMRetryRule) => void;
  /** true = 配置抽屉里的紧凑版：省掉展开说明，只留「什么错误会走到这层」这一句 */
  compact?: boolean;
}) {
  const meta = RETRY_LAYERS[layer];
  const human = humanMs(value.interval_ms);
  return (
    <div className={compact ? "grid gap-2" : "grid gap-3 rounded-lg border p-3"}>
      <div className="grid gap-0.5">
        <div className="flex flex-wrap items-baseline gap-2">
          <Label className="text-sm">{meta.title}</Label>
          <span className="text-muted-foreground text-xs">{meta.where}</span>
        </div>
        {/* 哪些错误会走到这层，具体到状态码——填了旋钮却看不到效果，多半是错误压根不落在这层。 */}
        <p className="text-muted-foreground text-xs">
          <span className="font-medium text-foreground">Trigger</span>：{meta.trigger}
        </p>
        {!compact && meta.skips && (
          <p className="text-muted-foreground text-xs">
            <span className="font-medium text-foreground">Not this one.</span>：{meta.skips}
          </p>
        )}
        {!compact && <p className="text-muted-foreground text-xs">{meta.desc}</p>}
      </div>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="flex items-center gap-2">
          <Label htmlFor={`${idPrefix}-${layer}-n`} className="text-muted-foreground text-xs">
            {meta.attemptsLabel}
          </Label>
          <NumField
            id={`${idPrefix}-${layer}-n`}
            min={-1}
            value={value.attempts}
            placeholder={`Default ${meta.defAttempts}`}
            onChange={(n) => onChange({ ...value, attempts: n })}
          />
        </div>
        <div className="flex items-center gap-2">
          <Label htmlFor={`${idPrefix}-${layer}-ms`} className="text-muted-foreground text-xs">
            Interval ms
          </Label>
          <NumField
            id={`${idPrefix}-${layer}-ms`}
            min={0}
            value={value.interval_ms}
            placeholder={"Default backoff"}
            onChange={(n) => onChange({ ...value, interval_ms: n })}
          />
          <span className="text-muted-foreground text-xs">{human ? `Fixed ${human}` : meta.defInterval}</span>
        </div>
      </div>
      {!compact && <p className="text-muted-foreground text-xs">Leave = Default;{meta.offHint}。</p>}
    </div>
  );
}

/** 模型配置抽屉里的三层覆盖（跟着端点走的那三层）。 */
export function ProfileRetryFields({
  value,
  onChange,
}: {
  value: LLMRetryOverride;
  onChange: (o: LLMRetryOverride) => void;
}) {
  return (
    <div className="grid gap-3 rounded-lg border p-3">
      <div className="grid gap-0.5">
        <Label className="text-sm">Retry Overwrite</Label>
        <p className="text-muted-foreground text-xs">
          Only effective with this configuration, overwhelms the global default in the Retry and Evasion. Leave each cell = follow the global; number of times fill -1 = turn off this layer;
          Interval is filled with a fixed interval instead of index evasive. Melting and attempted rerun are process-levels and can only be modulated on the global page.
        </p>
      </div>
      {(["connect", "empty", "stream"] as const).map((k) => (
        <div key={k} className="border-t pt-3 first:border-t-0 first:pt-0">
          <RetryRuleFields
            compact
            layer={k}
            idPrefix="pf"
            value={value[k]}
            onChange={(r) => onChange({ ...value, [k]: r })}
          />
        </div>
      ))}
    </div>
  );
}

/** 「重试与退避」tab：五层的全局默认值。 */
export function RetryPolicyPanel() {
  const [policy, setPolicy] = React.useState<LLMRetryPolicy>(ZERO_POLICY);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const p = await api.llmRetryPolicy();
      setPolicy({ ...ZERO_POLICY, ...p });
    } catch (e) {
      toast.error(`Reading the retry strategy failed:${(e as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    if (saving) return;
    setSaving(true);
    try {
      // 后端会把越界值夹回区间并回传，直接用回传值刷新，所见即所存。
      const saved = await api.saveLLMRetryPolicy(policy);
      setPolicy({ ...ZERO_POLICY, ...saved });
      toast.success("Saved, effective immediately (this rotation, which is running, is still using old parameters)");
    } catch (e) {
      toast.error(`Failed to save:${(e as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  const set = (k: LayerKey) => (r: LLMRetryRule) => setPolicy((p) => ({ ...p, [k]: r }));

  if (loading) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-dashed p-10 text-muted-foreground text-sm">
        <Loader2Icon className="size-4 animate-spin" /> Loading retry policy…
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      <div className="rounded-lg border bg-muted/30 p-3 text-muted-foreground text-xs leading-relaxed">
        The failure of a model call will be retested in five layers, from the inside to the outside:
        <span className="text-foreground"> Networking, empty response, co-provider security window,  ⁇  Query melting,  ⁇  Intending to run again</span>
        . The inner layer is exhausted before it reaches the outer layer, so the number is...
        <span className="text-foreground">Multiplication</span>
        It's -- it's full of every floor, and it burns dozens of requests at a time.
        All empty, the current default, is fully consistent with the behaviour of the absence of this page. The first three layers can be covered in each model configuration separately.
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {(Object.keys(RETRY_LAYERS) as LayerKey[]).map((k) => (
          <RetryRuleFields key={k} layer={k} idPrefix="gl" value={policy[k]} onChange={set(k)} />
        ))}
      </div>

      <div className="flex gap-2">
        <Button onClick={save} disabled={saving}>
          {saving ? <Loader2Icon className="animate-spin" /> : <SaveIcon />}
          Save
        </Button>
        <Button variant="outline" onClick={() => setPolicy(ZERO_POLICY)} disabled={saving}>
          Restore Default All
        </Button>
      </div>
    </div>
  );
}
