package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Autumn-27/artex/researchkit"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
)

type researchToolSpec struct {
	Name, Operation, Description string
	Schema                       map[string]any
}

func researchSpecs() []researchToolSpec {
	input := objSchema(map[string]any{"input": strParam("Supplied local artifact text, up to 1 MiB; never a URL to fetch."), "limit": map[string]any{"type": "integer", "description": "Maximum returned rows (does not change the total)."}}, "input")
	finding := map[string]any{"type": "object", "description": "Structured draft with title, summary, steps[], expected, actual, impact, remediation, evidence[], scope_confirmed, preconditions_proven, vector and optional claimed_score.", "properties": map[string]any{
		"title": strParam("Affected control and observation"), "summary": strParam("Short factual summary"), "steps": map[string]any{"type": "array", "items": strParam("Local reproduction step")}, "expected": strParam("Expected control behavior"), "actual": strParam("Observed result"), "impact": strParam("Demonstrated consequence"), "remediation": strParam("Fix and retest condition"), "evidence": map[string]any{"type": "array", "items": strParam("Evidence record reference")}, "scope_confirmed": map[string]any{"type": "boolean"}, "preconditions_proven": map[string]any{"type": "boolean"}, "vector": strParam("CVSS:4.0 vector"), "claimed_score": map[string]any{"type": "number"},
	}}
	return []researchToolSpec{
		{"research_har_summary", "har", "Summarize saved HAR traffic metadata without replaying requests. Omits headers, cookies and bodies.", input},
		{"research_model_coverage", "coverage", "Validate supplied system model nodes/edges and calculate self-reported review coverage and gaps.", objSchema(map[string]any{"model": map[string]any{"type": "object", "description": "System model with nodes[{id,kind,label,status,references[]}], edges[{from,to,relation}], questions[]. Kinds: tech,role,auth,flow,object,feature,hypothesis,boundary. Status: todo,reviewing,reviewed,supported,rejected."}}, "model")},
		{"research_openapi_summary", "openapi", "Summarize supplied OpenAPI 3.x/Swagger 2.0 JSON or YAML: endpoint/parameter/field names and declared authentication. Offline; no requests or external-reference fetching.", input},
		{"research_normalize_results", "normalize", "Normalize and deduplicate saved Nuclei JSON/JSONL, ffuf JSON or Nmap XML into unverified observations. Does not run scanners or create findings.", objSchema(map[string]any{"input": strParam("Saved tool output"), "source": map[string]any{"type": "string", "enum": []string{"nuclei", "ffuf", "nmap"}}, "limit": map[string]any{"type": "integer"}}, "input", "source")},
		{"research_redact_evidence", "redact", "Best-effort local redaction of credential fields, auth/cookie headers and email addresses in supplied text. Manually inspect remaining sensitive data before sharing.", objSchema(map[string]any{"input": strParam("Text to minimize; use a copy of evidence")}, "input")},
		{"research_cvss_score", "cvss", "Compute a supplied CVSS 4.0 vector and compare an optional claimed score. Does not infer metrics or verify business impact.", objSchema(map[string]any{"vector": strParam("CVSS:4.0 vector"), "claimed_score": map[string]any{"type": "number", "minimum": 0, "maximum": 10}}, "vector")},
		{"research_validate_finding", "validate", "Evaluate seven evidence/readiness gates for a structured finding draft and check its CVSS vector. Self-attested fields are not independently verified.", objSchema(map[string]any{"finding": finding}, "finding")},
		{"research_report_draft", "report", "Prepare a minimized Markdown report draft and readiness checklist from supplied finding data. No publishing, database mutation or external model call.", objSchema(map[string]any{"finding": finding}, "finding")},
	}
}

func researchReadTool(name, description string, schema map[string]any, run func(json.RawMessage) (any, error)) actool.CoreTool {
	return actool.Build(actool.Spec{Name: name, Description: description, Schema: schema,
		ReadOnly: func(json.RawMessage) bool { return true }, Concurrent: func(json.RawMessage) bool { return true },
		Permissions: func(context.Context, json.RawMessage, permission.Context) permission.Decision {
			return permission.Allowed()
		},
		Run: func(_ context.Context, raw json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
			result, err := run(raw)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			return jsonResult(result)
		},
	})
}

func (s *Server) researchTools() []actool.CoreTool {
	tools := []actool.CoreTool{
		researchReadTool("research_skill_catalog", "Search the pinned claude-redteam-derived TYAN review skill catalog by query or exact technology tags. Returns skill names for on-demand Skill loading, not instruction bodies.", objSchema(map[string]any{"query": strParam("Topic/category/name substring"), "tech": strParam("Space-separated technology tags, e.g. node graphql")}), func(raw json.RawMessage) (any, error) {
			var input struct {
				Query string `json:"query"`
				Tech  string `json:"tech"`
			}
			if err := researchkit.DecodeStrict(raw, &input); err != nil {
				return nil, err
			}
			return map[string]any{"items": researchkit.Catalog(input.Query, input.Tech), "source_commit": researchkit.SourceCommit, "mode": "tyan-review-adaptation"}, nil
		}),
		researchReadTool("research_tool_inventory", "Check whether known optional external executables are on PATH. Presence does not verify version or functionality. Nothing is executed or installed.", objSchema(map[string]any{}), func(raw json.RawMessage) (any, error) {
			var input struct{}
			if err := researchkit.DecodeStrict(raw, &input); err != nil {
				return nil, err
			}
			return researchkit.Inventory(), nil
		}),
	}
	for _, spec := range researchSpecs() {
		tools = append(tools, researchReadTool(spec.Name, spec.Description, spec.Schema, func(raw json.RawMessage) (any, error) { return researchkit.Run(spec.Operation, raw) }))
	}
	tools = append(tools, s.researchWorkspaceTools()...)
	return tools
}

func (s *Server) seedResearchTools() {
	if s.m == nil || s.m.pg == nil {
		return
	}
	bindings := json.RawMessage(`["auto","worker","planner","mainagent","pentest","reporter"]`)
	for _, tool := range s.researchTools() {
		schema, _ := json.Marshal(tool.InputSchema())
		if err := s.m.pg.SeedTool(tool.Name(), tool.Description(), schema, bindings); err != nil {
			log.Printf("[research-kit] seed %s: %v", tool.Name(), err)
		}
	}
	toolNames := []string{}
	for _, t := range s.researchTools() {
		toolNames = append(toolNames, t.Name())
	}
	skillNames := []string{}
	for _, skill := range researchkit.Catalog("", "") {
		skillNames = append(skillNames, skill.Name)
	}
	for _, role := range researchkit.ReviewRoles() {
		if err := s.m.pg.SeedResearchRole(role, toolNames, skillNames); err != nil {
			log.Printf("[research-kit] seed role %s: %v", role.Key, err)
		}
	}
}

func (s *Server) registerResearchRoutes(mux *http.ServeMux) {
	s.registerResearchWorkspaceRoutes(mux)
	s.registerResearchWorkflows(mux)
	mux.HandleFunc("GET /api/research-kit/catalog", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"items": researchkit.Catalog(r.URL.Query().Get("q"), r.URL.Query().Get("tech")), "source_commit": researchkit.SourceCommit, "mode": "tyan-review-adaptation"})
	})
	mux.HandleFunc("GET /api/research-kit/tools", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"items": researchkit.Inventory(), "native_tools": len(s.researchTools()), "notice": "PATH presence only; no executables were launched or installed."})
	})
	mux.HandleFunc("POST /api/research-kit/run/{operation}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, researchkit.MaxInput))
		if err != nil {
			writeErr(w, 413, "research input exceeds 1 MiB")
			return
		}
		result, err := researchkit.Run(r.PathValue("operation"), raw)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, result)
	})
	mux.HandleFunc("POST /api/research-kit/report.docx", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, researchkit.MaxInput))
		if err != nil {
			writeErr(w, 413, "report input exceeds 1 MiB")
			return
		}
		finding, err := researchkit.ValidFindingJSON(raw)
		if err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		body, err := researchkit.ReportDOCX(*finding)
		if err != nil {
			writeErr(w, 500, "could not generate report")
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.Header().Set("Content-Disposition", `attachment; filename="tyan-research-draft.docx"`)
		w.WriteHeader(200)
		_, _ = w.Write(body)
	})
}
