import { http } from "@/lib/api";
import { auth } from "@/lib/auth";

export interface ResearchSkill {
  slug: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  source_path: string;
  source_sha256: string;
  mode: string;
}
export interface ToolAvailability {
  name: string;
  category: string;
  on_path: boolean;
}
export interface FindingDraft {
  title: string;
  summary: string;
  steps: string[];
  expected: string;
  actual: string;
  impact: string;
  remediation: string;
  evidence: string[];
  scope_confirmed: boolean;
  preconditions_proven: boolean;
  vector?: string;
  claimed_score?: number;
}
export type ResearchOperation = "openapi" | "normalize" | "redact" | "cvss" | "validate" | "report";
export interface ResearchResult {
  total?: number;
  input_count?: number;
  duplicates?: number;
  truncated?: boolean;
  notice?: string;
  text?: string;
  markdown?: string;
  score?: number;
  severity?: string;
  vector?: string;
  matches_claim?: boolean;
  status?: string;
  warnings?: string[];
  checks?: { name: string; passed: boolean; detail: string }[];
  operations?: {
    method: string;
    path: string;
    authentication: string;
    path_parameters: string[];
    query_parameters: string[];
    body_fields: string[];
    response_fields: string[];
  }[];
  items?: { id: string; title: string; url: string; severity: string; status: string }[];
  validation?: ResearchResult;
}

export const researchAPI = {
  catalog: (query = "", tech = "") =>
    http<{ items: ResearchSkill[]; source_commit: string; mode: string }>(
      `/research-kit/catalog?q=${encodeURIComponent(query)}&tech=${encodeURIComponent(tech)}`,
    ),
  inventory: () => http<{ items: ToolAvailability[]; native_tools: number; notice: string }>("/research-kit/tools"),
  run: (operation: ResearchOperation, body: unknown) =>
    http<ResearchResult>(`/research-kit/run/${operation}`, { method: "POST", body: JSON.stringify(body) }),
  downloadReport: async (finding: FindingDraft) => {
    const token = auth.getToken();
    const response = await fetch("/api/research-kit/report.docx", {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ finding }),
    });
    if (!response.ok)
      throw new Error((await response.json().catch(() => ({}))).error || `Report export failed (${response.status})`);
    const url = URL.createObjectURL(await response.blob());
    const link = document.createElement("a");
    link.href = url;
    link.download = "tyan-research-draft.docx";
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  },
};
