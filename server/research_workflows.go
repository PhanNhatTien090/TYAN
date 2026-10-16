package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/researchkit"
)

type researchWorkflowRequest struct {
	TaskID    int64                     `json:"task_id,string"`
	Query     string                    `json:"query"`
	Tech      string                    `json:"tech"`
	Title     string                    `json:"title"`
	Source    string                    `json:"source"`
	Input     string                    `json:"input"`
	Baseline  string                    `json:"baseline"`
	Candidate string                    `json:"candidate"`
	Model     *researchkit.SystemModel  `json:"model"`
	Finding   *researchkit.FindingDraft `json:"finding"`
	Record    *db.ResearchRecord        `json:"record"`
}

func (s *Server) registerResearchWorkflows(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/research-kit/workflows", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"items": researchkit.Workflows(), "mode": "native-review-adaptation"})
	})
	mux.HandleFunc("POST /api/research-kit/workflows/{command}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, researchkit.MaxInput))
		if err != nil {
			writeErr(w, 413, "request exceeds 1 MiB")
			return
		}
		var input researchWorkflowRequest
		if err = researchkit.DecodeStrict(raw, &input); err != nil {
			writeErr(w, 400, err.Error())
			return
		}
		if input.TaskID < 0 {
			writeErr(w, 400, "invalid task ID")
			return
		}
		command := r.PathValue("command")
		var result any
		switch command {
		case "skill-load":
			result = map[string]any{"items": researchkit.Catalog(input.Query, input.Tech)}
		case "kb", "kb_search":
			result, err = s.m.pg.ResearchList("kb", input.TaskID, input.Query)
		case "burp-analyze":
			result, err = researchkit.HARSummary(input.Input)
		case "recon":
			switch input.Source {
			case "openapi":
				result, err = researchkit.ParseOpenAPI(input.Input, 100)
			case "har":
				result, err = researchkit.HARSummary(input.Input)
			default:
				result, err = researchkit.Normalize(input.Source, input.Input, 100)
			}
		case "model-system", "chain":
			if input.Model != nil {
				result, err = researchkit.ModelCoverage(*input.Model)
			} else if input.TaskID > 0 {
				result, err = s.m.pg.ResearchList("model", input.TaskID, "")
			} else {
				err = fmt.Errorf("provide a model or positive task ID")
			}
		case "replay":
			result, err = researchkit.CompareHAR(input.Baseline, input.Candidate)
		case "session-start", "session-end":
			if input.TaskID <= 0 {
				writeErr(w, 400, "positive task ID required")
				return
			}
			var out strings.Builder
			fmt.Fprintf(&out, "# TYAN task %d review handoff\n\nStored reference material; observations require human verification.\n", input.TaskID)
			for _, kind := range []string{"memory", "model", "api"} {
				rows, e := s.m.pg.ResearchList(kind, input.TaskID, input.Query)
				if e != nil {
					err = e
					break
				}
				fmt.Fprintf(&out, "\n## %s (latest %d matching records; cap 200)\n", kind, len(rows))
				for _, row := range rows {
					fmt.Fprintf(&out, "\n### %s · %s · v%d\n\n%s\n\nReferences: %s\n", row.Title, row.Key, row.Version, row.Content.Body, strings.Join(row.Content.References, ", "))
					if out.Len() > researchkit.MaxInput {
						writeErr(w, 413, "handoff too large; narrow the search query")
						return
					}
					if row.Content.Model != nil {
						coverage, e := researchkit.ModelCoverage(*row.Content.Model)
						if e != nil {
							err = e
							break
						}
						fmt.Fprintf(&out, "\nModel review coverage: %d/%d.\n", coverage.Reviewed, coverage.Total)
						for _, node := range row.Content.Model.Nodes {
							fmt.Fprintf(&out, "- %s: %s (%s, %s)\n", node.ID, node.Label, node.Kind, node.Status)
						}
						for _, edge := range row.Content.Model.Edges {
							fmt.Fprintf(&out, "- %s -> %s: %s\n", edge.From, edge.To, edge.Relation)
						}
					}
					if row.Content.API != nil {
						for _, op := range row.Content.API.Operations {
							fmt.Fprintf(&out, "- %s %s [%s]\n", op.Method, op.Path, op.Authentication)
						}
					}
					if out.Len() > researchkit.MaxInput {
						writeErr(w, 413, "handoff too large; narrow the search query")
						return
					}
				}
			}
			if out.Len() > researchkit.MaxInput {
				writeErr(w, 413, "handoff too large; narrow the search query")
				return
			}
			redacted, e := researchkit.Redact(out.String())
			if e != nil {
				err = e
			} else {
				result = map[string]any{"markdown": redacted.Text, "notice": "Best-effort minimization. No notes cleared and no session lifecycle changed. Inspect before sharing."}
			}
		case "new-engagement":
			if strings.TrimSpace(input.Title) == "" || len(input.Title) > 300 || strings.TrimSpace(input.Input) == "" {
				writeErr(w, 400, "title and task goal are required")
				return
			}
			result, err = s.m.pg.CreateTask(input.Title, input.Input, nil, 0, 0)
		case "update-state":
			if input.Record == nil {
				writeErr(w, 400, "record is required")
				return
			}
			result, err = s.m.pg.ResearchSave(*input.Record)
		case "validate", "new-finding", "generate", "bb-report", "gen-office-report":
			if input.Finding == nil {
				writeErr(w, 400, "finding is required")
				return
			}
			switch command {
			case "validate", "new-finding":
				result = map[string]any{"draft": input.Finding, "validation": researchkit.Validate(*input.Finding), "notice": "Draft only; use the Findings workflow to save a reviewed conclusion."}
			case "generate", "bb-report":
				result = map[string]any{"markdown": researchkit.ReportMarkdown(*input.Finding), "validation": researchkit.Validate(*input.Finding)}
			case "gen-office-report":
				var data []byte
				data, err = researchkit.ReportDOCX(*input.Finding)
				if err == nil {
					result = map[string]any{"docx_base64": base64.StdEncoding.EncodeToString(data), "filename": "tyan-research-draft.docx"}
				}
			}
		default:
			writeErr(w, 400, "unknown research workflow")
			return
		}
		if err != nil {
			if command == "update-state" {
				researchError(w, err)
			} else {
				writeErr(w, 400, err.Error())
			}
			return
		}
		writeJSON(w, 200, map[string]any{"command": command, "result": result, "mode": "native-review-adaptation"})
	})
}
