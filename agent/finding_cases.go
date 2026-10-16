package agent

import (
	"context"
	actool "github.com/Autumn-27/norma/tool"
)

type findingReviewKey struct{}

// Historical consolidation has a narrower capability set than new reporting.
func WithFindingCaseReview(ctx context.Context) context.Context {
	return context.WithValue(ctx, findingReviewKey{}, true)
}
func isFindingCaseReview(ctx context.Context) bool {
	v, _ := ctx.Value(findingReviewKey{}).(bool)
	return v
}

func historicalFindingCaseTools(tools []actool.CoreTool) []actool.CoreTool {
	allowed := map[string]bool{}
	for _, name := range []string{"search_finding_duplicates", "get_finding_record", "merge_finding_records", "suggest_finding_merge", "get_finding_case", "update_finding_case_report", "complete_finding_case_review", "get_finding_traffic", "traffic_search", "traffic_get", "traffic_blob", "get_task_node_detail", "list_task_findings", "get_task_graph", "list_task_worker_traces", "get_task_worker_trace", "Read"} {
		allowed[name] = true
	}
	out := make([]actool.CoreTool, 0, len(tools))
	for _, tool := range tools {
		if allowed[tool.Name()] {
			out = append(out, tool)
		}
	}
	return out
}

// This guidance is appended to the runtime prompt, including editable old prompts.
// Never reintroduce unbound tools or recreate a deleted Reporter.
func findingCaseWorkflow(key string, tools []actool.CoreTool) ([]actool.CoreTool, string) {
	if key != "reporter" {
		return tools, ""
	}
	has := map[string]bool{}
	for _, t := range tools {
		has[t.Name()] = true
	}
	if !has["search_finding_duplicates"] || !has["merge_finding_records"] || !has["get_finding_case"] || !has["update_finding_case_report"] {
		return tools, ""
	}
	return tools, `

[Deduplication of multiple findings for the same vulnerability] Preserve this standalone report; do not overwrite other original reports. The previous guidance (stop after saving the report) is extended here: save this report first, then call search_finding_duplicates(finding_id=standalone record ID) to page through same-task candidate summaries, using get_finding_record to inspect evidence as needed. Vulnerability type/title similarity or severity alone is not enough to merge; verify target, endpoint, parameter, authorization boundary, root cause, and actual validation evidence.
[Decision criteria] Different endpoints, paths, backends, read/write impacts, or severities are each insufficient on their own to prove non-duplication; the same login entry point, the same token, or the same vulnerability type alone do not prove the same root cause. Compare the actual missing auth checks, object permission boundaries, session gates, or shared components, and cite specific evidence from each record. Read and write impacts may be different effects of the same missing permission check, or they may be independent defects in separate handlers. Only merge automatically when evidence proves a shared actual root cause; only declare independent when evidence proves distinct root causes. When related but unconfirmed, you must call suggest_finding_merge, explaining what is known in common, the missing evidence, and the questions to be confirmed — do not substitute "different path / different impact" for a root-cause judgment. When the same defect is clear, call merge_finding_records(task_id, finding_ids, title, reason). When uncertain, call suggest_finding_merge and leave it for human review; do not force a merge. Do not re-report, delete records, or re-probe targets. During a manual review, only judge the selected records; do not add unselected records (existing group members may be used as reporting context).
When a merge succeeds or a case_id already exists, call get_finding_case to record the actual version, page through all members, and use get_finding_record and get_finding_traffic as needed to read original reports, PoC, and saved traffic. Use update_finding_case_report to save a complete unified report (including sections with content: overview, impact, prerequisites, reproduction, evidence, PoC, root cause analysis, remediation — the report must not be truncated) along with a severity and severity_reason based on verified impact; do not simply take the highest member severity. Provide prerequisites, complete execution order, how inputs and outputs connect, and complete runnable scripts and commands in one place — do not scatter necessary steps across original reports; mark missing conditions and unverified steps honestly. On version conflict, re-read the materials and regenerate; do not simply retry with a different version. Each member still has its own independent report, severity, and evidence. Retain independent records when there is no clear duplicate. Do not bypass the tool configuration when tools are disabled.`
}

const historicalFindingReviewGuidance = `

[Historical review mode — overrides prior reporting and auto-traffic-binding guidance] This session only reviews existing findings. Even when original reports are absent or pending, only record what is missing — never call update_finding_report, report_finding, or bind_finding_traffic; do not change original titles, severities, statuses, evidence, or reports; do not run Bash, write files, or re-probe targets.
Start by calling get_finding_record for each selected record in turn; reuse existing original reports directly as judgment material — do not rewrite them. Check for identical actual defects only within the selected scope; read existing evidence as needed. Form clear merge / independent / suggested verdicts following the root-cause criteria above; save suggestions for human confirmation when unable to confirm.
Existing or newly created cases must have all members read and a complete, up-to-date unified report written; the unified report is a separate document that does not overwrite original reports. Finally, you must call complete_finding_case_review to submit a verdict (independent/merged/suggested), reason, and root_cause_evidence for every selected finding_id, along with a summary. Root-cause evidence must cite the specific shared or distinct checks and their evidence sources; suggested-merge submissions must specify the missing evidence. Only declare completion after the server-side validation passes. If tools are unavailable or validation fails, report honestly — do not simply claim completion in natural language.`
