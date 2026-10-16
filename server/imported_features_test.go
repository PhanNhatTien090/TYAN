package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
)

func TestImportedScheduleAPI(t *testing.T) {
	_, _, request := trafficEvidenceServer(t)
	response := request("POST", "/api/schedules", `{"name":"local lab schedule test","enabled":false,"timezone_mode":"system","schedule_type":"once","run_date":"2030-01-01","start_time":"09:00","end_time":"17:00","task_ids":[]}`)
	if response.Code != 201 {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var created db.TaskSchedule
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Enabled {
		t.Fatal("new schedule unexpectedly enabled")
	}
	path := fmt.Sprintf("/api/schedules/%d", created.ID)
	defer request("DELETE", path, "")
	for _, action := range []string{"resume", "pause"} {
		response = request("POST", path+"/"+action, `{}`)
		if response.Code != 200 {
			t.Fatalf("%s: %d %s", action, response.Code, response.Body.String())
		}
	}
	response = request("GET", path, "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "local lab schedule test") {
		t.Fatalf("get: %s", response.Body.String())
	}
	response = request("GET", "/api/schedules", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"schedules"`) {
		t.Fatalf("list: %s", response.Body.String())
	}
}

func TestImportedScheduleWindowBoundaries(t *testing.T) {
	s := &db.TaskSchedule{ScheduleType: "once", TimezoneMode: "beijing", RunDate: "2030-01-01", StartTime: "09:00", EndTime: "17:00"}
	loc := scheduleLocation(s.TimezoneMode)
	for _, c := range []struct {
		hour             int
		active, complete bool
	}{{8, false, false}, {9, true, false}, {16, true, false}, {17, false, true}} {
		got := evaluateScheduleWindow(s, time.Date(2030, 1, 1, c.hour, 0, 0, 0, loc))
		if got.active != c.active || got.complete != c.complete {
			t.Fatalf("hour %d: %+v", c.hour, got)
		}
	}
	s.ScheduleType = "weekly"
	s.Weekdays = []int{1}
	s.StartTime = "23:00"
	s.EndTime = "01:00"
	// Monday's window continues across midnight into Tuesday.
	now := time.Date(2030, 1, 8, 0, 30, 0, 0, loc)
	if got := evaluateScheduleWindow(s, now); !got.active {
		t.Fatalf("overnight window not active: %+v", got)
	}
}

func TestImportedFindingCaseMergeAPI(t *testing.T) {
	s, first, request := trafficEvidenceServer(t)
	var taskID int64
	if err := s.m.pg.QueryRow("SELECT task_id FROM findings WHERE id=$1", first.FindingID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	second, err := s.m.pg.AddFinding(taskID, 0, "TEST", "Same root cause", "low", "second affected endpoint", "fixture evidence", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := request("POST", "/api/exploration/finding-cases", fmt.Sprintf(`{"task_id":%d,"finding_ids":[%d,%d],"title":"Shared local fixture root cause","reason":"Both fixture records share the same verified implementation defect"}`, taskID, first.FindingID, second))
	if response.Code != 200 {
		t.Fatalf("merge: %d %s", response.Code, response.Body.String())
	}
	var result struct {
		ID string `json:"case_id"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.ID == "" {
		t.Fatalf("response: %s", response.Body.String())
	}
	response = request("GET", "/api/exploration/finding-cases/"+result.ID, "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), "Shared local fixture root cause") {
		t.Fatalf("case detail: %d %s", response.Code, response.Body.String())
	}
	response = request("GET", "/api/exploration/finding-cases?limit=20", "")
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"case":`) {
		t.Fatalf("list envelope: %d %s", response.Code, response.Body.String())
	}
}
