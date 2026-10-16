package researchkit

import (
	"encoding/json"
	"fmt"
	"sort"
)

type Workflow struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Writes      bool           `json:"writes_database"`
	Example     map[string]any `json:"example"`
}

func Workflows() []Workflow {
	finding := FindingDraft{Title: "Local fixture control", Summary: "A synthetic record was returned", Steps: []string{"Run the supplied local fixture"}, Expected: "Denied", Actual: "Synthetic record returned", Impact: "The fixture exposes its synthetic record", Remediation: "Enforce policy and retest", Evidence: []string{"traffic:1"}}
	return []Workflow{
		{"skill-load", "Find native review skills by topic/technology; loading stays in the agent Skill tool.", false, map[string]any{"query": "authorization", "tech": ""}},
		{"kb", "Read/search stored knowledge in the selected task or global scope.", false, map[string]any{"task_id": "0", "query": ""}},
		{"kb_search", "Search persisted KB title, tags, references and body using literal text.", false, map[string]any{"task_id": "0", "query": "authorization"}},
		{"burp-analyze", "Summarize saved HAR exported by a browser or proxy; not raw Burp project files.", false, map[string]any{"input": `{"log":{"entries":[]}}`}},
		{"recon", "Inventory supplied OpenAPI, HAR or saved Nuclei/ffuf/Nmap output. Does not launch discovery.", false, map[string]any{"source": "openapi", "input": `{"openapi":"3.0.3","paths":{}}`}},
		{"model-system", "Validate a supplied system model and preview coverage; omit model to list stored models.", false, map[string]any{"task_id": "1"}},
		{"chain", "Review saved model relationships and coverage. Does not execute chains.", false, map[string]any{"task_id": "1"}},
		{"replay", "Compare saved baseline/candidate HAR metadata. Does not resend HTTP requests.", false, map[string]any{"baseline": `{"log":{"entries":[]}}`, "candidate": `{"log":{"entries":[]}}`}},
		{"session-start", "Build a task handoff from stored notes, models and API catalogs; no automatic capture hook.", false, map[string]any{"task_id": "1", "query": ""}},
		{"session-end", "Export a task handoff as Markdown; notes are retained and no agent is stopped.", false, map[string]any{"task_id": "1", "query": ""}},
		{"new-engagement", "Create a local TYAN task without starting agents or scans.", true, map[string]any{"title": "Local lab review", "input": "Review the supplied synthetic fixture"}},
		{"update-state", "Create/update a versioned workspace record; expected version is required.", true, map[string]any{"record": map[string]any{"kind": "kb", "task_id": "0", "key": "lab-review", "title": "Lab review note", "version": 0, "content": WorkspaceContent{Body: "Observed fixture behavior"}}}},
		{"new-finding", "Prepare a structured finding draft and readiness check; does not save a confirmed finding.", false, map[string]any{"finding": finding}},
		{"validate", "Check seven finding readiness gates and an optional CVSS vector.", false, map[string]any{"finding": finding}},
		{"generate", "Prepare a minimized Markdown report draft.", false, map[string]any{"finding": finding}},
		{"bb-report", "Prepare an evidence-based Markdown submission draft without publishing it.", false, map[string]any{"finding": finding}},
		{"gen-office-report", "Generate a basic DOCX draft, not the upstream Office template engine.", false, map[string]any{"finding": finding}},
	}
}

func CompareHAR(baseline, candidate string) (any, error) {
	left, err := HARSummary(baseline)
	if err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}
	right, err := HARSummary(candidate)
	if err != nil {
		return nil, fmt.Errorf("candidate: %w", err)
	}
	type row struct {
		Method string `json:"method"`
		URL    string `json:"url"`
		Status int    `json:"status"`
	}
	type summary struct {
		Total     int   `json:"total"`
		Truncated bool  `json:"truncated"`
		Entries   []row `json:"entries"`
	}
	decode := func(v any) summary { raw, _ := json.Marshal(v); var s summary; _ = json.Unmarshal(raw, &s); return s }
	a, b := decode(left), decode(right)
	group := func(s summary) map[string]map[int]int {
		out := map[string]map[int]int{}
		for _, r := range s.Entries {
			k := r.Method + " " + r.URL
			if out[k] == nil {
				out[k] = map[int]int{}
			}
			out[k][r.Status]++
		}
		return out
	}
	ga, gb := group(a), group(b)
	keys := map[string]bool{}
	for k := range ga {
		keys[k] = true
	}
	for k := range gb {
		keys[k] = true
	}
	type change struct {
		Operation string      `json:"operation"`
		Before    map[int]int `json:"before_status_counts"`
		After     map[int]int `json:"after_status_counts"`
	}
	changes := []change{}
	sorted := []string{}
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		x, _ := json.Marshal(ga[k])
		y, _ := json.Marshal(gb[k])
		if string(x) != string(y) {
			changes = append(changes, change{k, ga[k], gb[k]})
		}
	}
	return map[string]any{"baseline_total": a.Total, "candidate_total": b.Total, "changes": changes, "partial": a.Truncated || b.Truncated, "notice": "Offline metadata comparison, first 200 entries per file. Query values are minimized and may collapse distinct requests. Status changes do not prove a fix or vulnerability. No HTTP traffic sent."}, nil
}
