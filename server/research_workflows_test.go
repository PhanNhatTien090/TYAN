package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/researchkit"
)

func TestResearchWorkflowDispatch(t *testing.T) {
	s, _, request := trafficEvidenceServer(t)
	task, err := s.m.pg.CreateTask("Workflow fixtures", "Review only saved fixtures", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.m.pg.DeleteTask(task.ID)
	for _, workflow := range researchkit.Workflows() {
		example := workflow.Example
		if _, ok := example["task_id"]; ok {
			example["task_id"] = fmt.Sprint(task.ID)
		}
		raw, _ := json.Marshal(example)
		rec := request("POST", "/api/research-kit/workflows/"+workflow.Name, string(raw))
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", workflow.Name, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("workflow artifact can be cached")
		}
		if workflow.Name == "new-engagement" {
			var body struct {
				Result db.Task `json:"result"`
			}
			if err = json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Result.ID == 0 {
				t.Fatalf("task creation: %v %s", err, rec.Body.String())
			}
			defer s.m.pg.DeleteTask(body.Result.ID)
		}
		if workflow.Name == "update-state" {
			defer s.m.pg.ResearchDelete("kb", 0, "lab-review", 1)
		}
		if workflow.Name == "gen-office-report" {
			var body struct {
				Result struct {
					Data string `json:"docx_base64"`
				} `json:"result"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			data, e := base64.StdEncoding.DecodeString(body.Result.Data)
			if e != nil || !strings.HasPrefix(string(data), "PK") {
				t.Fatal("DOCX not generated")
			}
		}
	}
	for _, test := range []struct{ name, body string }{{"missing", `{}`}, {"kb", `{"task_id":"-1"}`}, {"replay", `{"baseline":"invalid","candidate":"invalid"}`}, {"update-state", `{}`}, {"new-engagement", `{}`}, {"validate", `{}`}} {
		rec := request("POST", "/api/research-kit/workflows/"+test.name, test.body)
		if rec.Code != 400 {
			t.Fatalf("%s invalid request accepted: %d", test.name, rec.Code)
		}
	}
}
