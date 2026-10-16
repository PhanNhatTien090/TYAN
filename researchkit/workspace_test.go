package researchkit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestModelCoverageChecksRelationshipsAndEvidence(t *testing.T) {
	model := SystemModel{Nodes: []ModelNode{{ID: "api", Kind: "feature", Label: "Lab API", Status: "reviewed"}, {ID: "policy", Kind: "hypothesis", Label: "Policy", Status: "todo"}}, Edges: []ModelEdge{{From: "api", To: "policy", Relation: "requires"}}}
	result, err := ModelCoverage(model)
	if err != nil || result.Percent != 50 || len(result.Gaps) != 1 {
		t.Fatalf("coverage: %+v %v", result, err)
	}
	model.Nodes[1].Status = "supported"
	if _, err = ModelCoverage(model); err == nil {
		t.Fatal("unsupported evidence accepted")
	}
	model.Nodes[1].References = []string{""}
	if _, err = ModelCoverage(model); err == nil {
		t.Fatal("blank evidence accepted")
	}
	model.Nodes[1].References = []string{"traffic:1"}
	if _, err = ModelCoverage(model); err != nil {
		t.Fatal(err)
	}
	model.Edges[0].To = "missing"
	if _, err = ModelCoverage(model); err == nil {
		t.Fatal("dangling edge accepted")
	}
	model.Edges = nil
	model.Nodes[1].ID = "api"
	if _, err = ModelCoverage(model); err == nil {
		t.Fatal("duplicate node accepted")
	}
}
func TestWorkspaceBoundsAndHARPrivacy(t *testing.T) {
	if err := ValidateWorkspace("memory", "note", "Note", 0, WorkspaceContent{Body: "observation"}); err == nil {
		t.Fatal("unscoped memory accepted")
	}
	if err := ValidateWorkspace("kb", "../../path", "Note", 0, WorkspaceContent{Body: "observation"}); err == nil {
		t.Fatal("invalid key accepted")
	}
	if err := ValidateWorkspace("kb", "note", "Note", 0, WorkspaceContent{Body: strings.Repeat("x", (256<<10)+1)}); err == nil {
		t.Fatal("oversized record accepted")
	}
	if err := ValidateWorkspace("kb", "note", "Note", 0, WorkspaceContent{Body: "observation"}); err != nil {
		t.Fatal(err)
	}
	value, err := HARSummary(`{"log":{"entries":[{"request":{"method":"GET","url":"http://user:secret@127.0.0.1/items?token=SECRET","headers":[{"name":"Authorization","value":"SECRET"}]},"response":{"status":200,"content":{"mimeType":"application/json","text":"SECRET"}},"time":12}]}}`)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(value)
	if err != nil || strings.Contains(string(b), "SECRET") || strings.Contains(string(b), "user:secret") {
		t.Fatalf("metadata not minimized: %s %v", b, err)
	}
	if _, err = HARSummary(`{"log":{}}`); err == nil {
		t.Fatal("missing entries accepted")
	}
}

func TestSavedHARComparisonAndWorkflowCatalog(t *testing.T) {
	before := `{"log":{"entries":[{"request":{"method":"GET","url":"http://127.0.0.1/item?token=BEFORE_SECRET"},"response":{"status":200}}]}}`
	after := `{"log":{"entries":[{"request":{"method":"GET","url":"http://127.0.0.1/item?token=AFTER_SECRET"},"response":{"status":403}}]}}`
	result, err := CompareHAR(before, after)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	if !strings.Contains(string(raw), `"200":1`) || !strings.Contains(string(raw), `"403":1`) || strings.Contains(string(raw), "_SECRET") {
		t.Fatalf("comparison: %s", raw)
	}
	if len(Workflows()) != 17 {
		t.Fatal("workflow mapping incomplete")
	}
	seen := map[string]bool{}
	writes := 0
	for _, w := range Workflows() {
		if seen[w.Name] || w.Description == "" || w.Example == nil {
			t.Fatal("invalid workflow catalog")
		}
		seen[w.Name] = true
		if w.Writes {
			writes++
		}
	}
	if writes != 2 {
		t.Fatal("mutation contract changed")
	}
}
