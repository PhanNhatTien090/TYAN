package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/norma/permission"
)

func TestResearchWorkspaceHTTPAndToolScope(t *testing.T) {
	s, _, request := trafficEvidenceServer(t)
	task, err := s.m.pg.CreateTask("Research workspace lab", "Review synthetic records", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.m.pg.DeleteTask(task.ID)
	key := "fixture-note"
	body := fmt.Sprintf(`{"kind":"memory","key":"%s","task_id":"%d","title":"Lab observation","version":0,"content":{"body":"Observed fixture","tags":["lab"],"references":["traffic:1"],"session":"review-1"}}`, key, task.ID)
	url := "/api/research-kit/records/memory/" + key
	rec := request("PUT", url, body)
	if rec.Code != 200 {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	if rec = request("PUT", url, body); rec.Code != 409 {
		t.Fatal("stale create did not conflict")
	}
	for _, path := range []string{fmt.Sprintf("%s?task_id=%d", url, task.ID), fmt.Sprintf("%s/history?task_id=%d", url, task.ID), fmt.Sprintf("/api/research-kit/records?kind=memory&task_id=%d&q=fixture", task.ID), "/api/research-kit/roles"} {
		if rec = request("GET", path, ""); rec.Code != 200 {
			t.Fatalf("read %s: %d", path, rec.Code)
		}
	}
	if rec = request("GET", url+"?task_id=0", ""); rec.Code != 404 {
		t.Fatal("cross-task lookup returned record")
	}
	if rec = request("DELETE", fmt.Sprintf("%s?task_id=%d&version=2", url, task.ID), ""); rec.Code != 409 {
		t.Fatal("stale delete accepted")
	}
	for _, tool := range s.researchWorkspaceTools() {
		if tool.Name() == "research_workspace_put" {
			decision := tool.CheckPermissions(context.Background(), json.RawMessage(`{}`), permission.Context{})
			if decision.Behavior == permission.Allow {
				t.Fatal("write tool bypasses normal write approval")
			}
		}
		if tool.Name() != "research_workspace_get" {
			continue
		}
		ctx := agent.WithRunInfo(context.Background(), agent.RunInfo{TaskID: task.ID})
		result, err := tool.Call(ctx, json.RawMessage(`{"kind":"memory","key":"fixture-note"}`), nil)
		if err != nil || result.IsError || !strings.Contains(result.Flatten(), "Observed fixture") {
			t.Fatalf("task tool: %v %s", err, result.Flatten())
		}
		result, _ = tool.Call(ctx, json.RawMessage(`{"kind":"memory","key":"fixture-note","task_id":999999}`), nil)
		if !result.IsError {
			t.Fatal("cross-task tool allowed")
		}
		result, _ = tool.Call(context.Background(), json.RawMessage(`{"kind":"memory","key":"fixture-note"}`), nil)
		if !result.IsError {
			t.Fatal("unscoped task notes allowed")
		}
	}
	for _, role := range []string{"rt_coordinator", "rt_recon", "rt_recon_ranker", "rt_appsec_review", "rt_chain_scout", "rt_validate"} {
		a, err := s.m.pg.GetAgentByKey(role)
		if err != nil || a == nil {
			t.Fatalf("role not installed: %s %v", role, err)
		}
	}
	if rec = request("DELETE", fmt.Sprintf("%s?task_id=%d&version=1", url, task.ID), ""); rec.Code != 200 {
		t.Fatalf("delete: %d", rec.Code)
	}
}
