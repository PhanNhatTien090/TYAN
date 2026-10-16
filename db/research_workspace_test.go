package db

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autumn-27/artex/researchkit"
)

func TestResearchWorkspaceVersionsIsolationAndCascade(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	task, err := d.CreateTask(fmt.Sprintf("research-%d", time.Now().UnixNano()), "Local fixture review", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(task.ID)
	row := ResearchRecord{Kind: "memory", TaskID: task.ID, Key: "observation", Title: "Fixture", Content: researchkit.WorkspaceContent{Body: "Observed result", Tags: []string{"lab"}}}
	saved, err := d.ResearchSave(row)
	if err != nil || saved.Version != 1 {
		t.Fatalf("create: %+v %v", saved, err)
	}
	if _, err = d.ResearchSave(row); !errors.Is(err, ErrResearchConflict) {
		t.Fatalf("duplicate create: %v", err)
	}
	other, err := d.ResearchList("memory", 0, "")
	if err != nil || len(other) != 0 {
		t.Fatalf("global scope leaked task notes: %v", err)
	}
	saved.Content.Body = "Concurrent review"
	var successes atomic.Int32
	var conflicts atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := d.ResearchSave(*saved)
			if e == nil {
				successes.Add(1)
			} else if errors.Is(e, ErrResearchConflict) {
				conflicts.Add(1)
			} else {
				t.Errorf("save: %v", e)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || conflicts.Load() != 1 {
		t.Fatalf("concurrent writes: %d successes %d conflicts", successes.Load(), conflicts.Load())
	}
	history, err := d.ResearchHistory("memory", task.ID, row.Key)
	if err != nil || len(history) != 2 || history[0].Version != 2 || history[1].Content.Body != "Observed result" {
		t.Fatalf("history: %+v %v", history, err)
	}
	if err = d.ResearchDelete("memory", task.ID, row.Key, 1); !errors.Is(err, ErrResearchConflict) {
		t.Fatalf("stale delete accepted: %v", err)
	}
	found, err := d.ResearchList("memory", task.ID, "concurrent")
	if err != nil || len(found) != 1 {
		t.Fatalf("search: %v", err)
	}
	if err = d.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	found, err = d.ResearchList("memory", task.ID, "")
	if err != nil || len(found) != 0 {
		t.Fatal("task deletion retained notes")
	}
	var remaining int
	if err = d.QueryRow(`SELECT count(*) FROM research_revisions WHERE record_id NOT IN (SELECT id FROM research_records)`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("orphan revisions")
	}
}

func TestResearchRoleSeedPreservesOperatorEdits(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	role := researchkit.ReviewRoles()[0]
	role.Key = fmt.Sprintf("rt_test_%d", time.Now().UnixNano())
	defer d.DeleteAgent(role.Key)
	if err = d.SeedResearchRole(role, nil, []string{"rt-triage-validation"}); err != nil {
		t.Fatal(err)
	}
	a, err := d.GetAgentByKey(role.Key)
	if err != nil || a == nil {
		t.Fatalf("agent: %v", err)
	}
	prompt, err := d.CurrentPrompt(a.ID)
	if err != nil || prompt != role.Prompt {
		t.Fatalf("prompt: %v", err)
	}
	if _, err = d.SavePrompt(a.ID, "operator customized", "test", "operator"); err != nil {
		t.Fatal(err)
	}
	if err = d.UpdateAgentMeta(role.Key, "Custom reviewer", "User edit"); err != nil {
		t.Fatal(err)
	}
	if err = d.SeedResearchRole(role, nil, nil); err != nil {
		t.Fatal(err)
	}
	a, err = d.GetAgentByKey(role.Key)
	if err != nil || a.Name != "Custom reviewer" {
		t.Fatal("agent edits overwritten")
	}
	prompt, err = d.CurrentPrompt(a.ID)
	if err != nil || prompt != "operator customized" {
		t.Fatal("prompt overwritten")
	}
}
