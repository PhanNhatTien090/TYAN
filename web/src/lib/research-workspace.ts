import { http } from "@/lib/api";

export type ResearchKind = "kb" | "memory" | "model" | "api";
export interface ModelNode {
  id: string;
  kind: string;
  label: string;
  status: string;
  references: string[];
}
export interface SystemModel {
  nodes: ModelNode[];
  edges: { from: string; to: string; relation: string }[];
  questions: string[];
}
export interface WorkspaceContent {
  body: string;
  tags: string[];
  references: string[];
  session: string;
  model?: SystemModel;
  api?: unknown;
}
export interface ResearchRecord {
  kind: ResearchKind;
  task_id: string;
  key: string;
  title: string;
  version: number;
  content: WorkspaceContent;
  updated_at?: string;
}
export interface ReviewRole {
  key: string;
  name: string;
  description: string;
  prompt: string;
}
export interface ModelCoverage {
  total: number;
  reviewed: number;
  percent: number;
  by_kind: Record<string, number>;
  gaps: ModelNode[];
  notice: string;
}
const route = (r: Pick<ResearchRecord, "kind" | "key">) =>
  `/research-kit/records/${r.kind}/${encodeURIComponent(r.key)}`;
export const workspaceAPI = {
  list: (kind: ResearchKind, task: string, query: string) =>
    http<{ items: ResearchRecord[]; limit: number }>(
      `/research-kit/records?kind=${kind}&task_id=${encodeURIComponent(task || "0")}&q=${encodeURIComponent(query)}`,
    ),
  save: (record: ResearchRecord) =>
    http<ResearchRecord>(route(record), { method: "PUT", body: JSON.stringify(record) }),
  remove: (record: ResearchRecord) =>
    http(`${route(record)}?task_id=${record.task_id}&version=${record.version}`, { method: "DELETE" }),
  history: (record: ResearchRecord) =>
    http<{ items: ResearchRecord[] }>(`${route(record)}/history?task_id=${record.task_id}`),
  roles: () => http<{ items: ReviewRole[] }>("/research-kit/roles"),
  coverage: (model: SystemModel) =>
    http<ModelCoverage>("/research-kit/run/coverage", { method: "POST", body: JSON.stringify({ model }) }),
};
