package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/intercept"
	actool "github.com/Autumn-27/norma/tool"
)

type findingCaseRequest struct {
	TaskID         json.RawMessage   `json:"task_id"`
	FindingID      json.RawMessage   `json:"finding_id"`
	IDs            []json.RawMessage `json:"finding_ids"`
	CaseID         json.RawMessage   `json:"case_id"`
	Version        int64             `json:"version"`
	Title          string            `json:"title"`
	Reason         string            `json:"reason"`
	Report         string            `json:"report"`
	Severity       string            `json:"severity"`
	SeverityReason string            `json:"severity_reason"`
	Page           int               `json:"page"`
}

func (a findingCaseRequest) ids() []int64 {
	out := []int64{}
	for _, raw := range a.IDs {
		out = append(out, parseProfileID(raw))
	}
	return out
}
func caseHTTPError(w http.ResponseWriter, err error) {
	code := 400
	if errors.Is(err, db.ErrFindingCaseConflict) || errors.Is(err, db.ErrFindingCaseReviewBusy) {
		code = 409
	}
	writeErr(w, code, err.Error())
}
func decodeCaseRequest(w http.ResponseWriter, r *http.Request, a any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(a); err != nil {
		writeErr(w, 400, "invalid request body")
		return false
	}
	return true
}
func (s *Server) caseFindingDTO(f *db.DBFinding) FindingDTO {
	return findingFromDB(f, s.resolveAssetIDs(f.AssetIDs))
}
func (s *Server) listFindingCases(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page := findingPaginationParam(q.Get("page"), 1, 0)
	size := findingPaginationParam(q.Get("limit"), 20, 100)
	rows, total, err := s.m.pg.ListFindingCases(findingFilterFromQuery(q), page, size)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	caseIDs := []int64{}
	for _, row := range rows {
		if row.Case != nil {
			caseIDs = append(caseIDs, row.Case.ID)
		}
	}
	contexts, err := s.m.pg.FindingCaseListContexts(caseIDs)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	assetIDs := []int64{}
	for _, c := range contexts {
		assetIDs = append(assetIDs, c.AssetIDs...)
	}
	assets := s.resolveAssetIDs(assetIDs)
	items := []map[string]any{}
	for _, row := range rows {
		v := map[string]any{"case": row.Case, "matched_ids": row.MatchedIDs}
		if row.Case != nil {
			c := contexts[row.Case.ID]
			v["task_name"], v["task_description"], v["last_found_at"], v["asset_count"] = c.TaskName, c.TaskDescription, c.LastFoundAt, c.AssetCount
			v["assets"] = append([]FindingAssetDTO{}, findingFromDB(&db.DBFinding{AssetIDs: c.AssetIDs}, assets).Assets...)
		}
		if row.FindingID > 0 {
			f, err := s.m.pg.FindingCaseSummary(row.FindingID, q.Get("original_rows") == "1")
			if err != nil {
				caseHTTPError(w, err)
				return
			}
			if f == nil {
				continue
			}
			f.Report = ""
			v["finding"] = s.caseFindingDTO(f)
		}
		items = append(items, v)
	}
	stats, err := s.m.pg.DistinctFindingStats(q.Get("task_id"))
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	matchingReports, err := s.m.pg.MatchingFindingReportCount(findingFilterFromQuery(q))
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "matching_reports": matchingReports, "page": page, "page_size": size, "stats": stats})
}
func (s *Server) findingCaseAccess(w http.ResponseWriter, r *http.Request, mutation bool) (*db.FindingCase, bool) {
	id := int64(atoiDefault(r.PathValue("id"), 0))
	c, err := s.m.pg.GetFindingCase(id)
	if err != nil {
		caseHTTPError(w, err)
		return nil, false
	}
	if c == nil {
		writeErr(w, 404, "finding case does not exist")
		return nil, false
	}
	if ctx := r.URL.Query().Get("context_task"); ctx != "" {
		t := s.m.ResolveTask(ctx)
		_, inherited, allowed := findingProvenanceInTask(t, c.TaskID)
		if !allowed {
			writeErr(w, 404, "finding case not found in task")
			return nil, false
		}
		if mutation && inherited {
			writeErr(w, 403, "inherited finding case is read-only")
			return nil, false
		}
	}
	return c, true
}
func (s *Server) getFindingCase(w http.ResponseWriter, r *http.Request) {
	c, ok := s.findingCaseAccess(w, r, false)
	if !ok {
		return
	}
	events, err := s.m.pg.FindingCaseEvents(c.ID)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"case": c, "events": events})
}
func (s *Server) getFindingCaseMembers(w http.ResponseWriter, r *http.Request) {
	c, ok := s.findingCaseAccess(w, r, false)
	if !ok {
		return
	}
	page := findingPaginationParam(r.URL.Query().Get("page"), 1, 0)
	size := findingPaginationParam(r.URL.Query().Get("limit"), 20, 100)
	fs, total, err := s.m.pg.FindingCaseMembers(c.ID, page, size)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	assets := s.resolveFindingAssets(fs)
	items := []FindingDTO{}
	for _, f := range fs {
		dto := findingFromDB(f, assets)
		if ctx := r.URL.Query().Get("context_task"); ctx != "" {
			dto.SourceTaskID, dto.Inherited, _ = findingProvenanceInTask(s.m.ResolveTask(ctx), f.TaskID)
		}
		items = append(items, dto)
	}
	writeJSON(w, 200, map[string]any{"items": items, "total": total, "page": page, "page_size": size})
}
func (s *Server) removeFindingCaseMember(w http.ResponseWriter, r *http.Request) {
	c, ok := s.findingCaseAccess(w, r, true)
	if !ok {
		return
	}
	fid := int64(atoiDefault(r.PathValue("fid"), 0))
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeCaseRequest(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "manually removed from finding case"
	}
	if err := s.m.pg.RemoveFindingCaseMember(r.Context(), c.ID, fid, body.Reason); err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
func (s *Server) caseSuggestions(w http.ResponseWriter, r *http.Request) {
	items, err := s.m.pg.FindingCaseSuggestions(int64(atoiDefault(r.URL.Query().Get("task_id"), 0)))
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, items)
}
func (s *Server) resolveCaseSuggestion(w http.ResponseWriter, r *http.Request) {
	var a struct {
		Accept bool `json:"accept"`
	}
	if !decodeCaseRequest(w, r, &a) {
		return
	}
	id, err := s.m.pg.ResolveFindingCaseSuggestion(r.Context(), int64(atoiDefault(r.PathValue("id"), 0)), a.Accept)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"case_id": i64s(id)})
}
func (s *Server) mergeFindingCase(w http.ResponseWriter, r *http.Request) {
	var a findingCaseRequest
	if !decodeCaseRequest(w, r, &a) {
		return
	}
	id, err := s.m.pg.MergeFindingCase(r.Context(), parseProfileID(a.TaskID), a.ids(), a.Title, a.Reason, "human")
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"case_id": i64s(id)})
}
func (s *Server) updateCaseReport(w http.ResponseWriter, r *http.Request) {
	c, ok := s.findingCaseAccess(w, r, true)
	if !ok {
		return
	}
	var a findingCaseRequest
	if !decodeCaseRequest(w, r, &a) {
		return
	}
	if err := validateUnifiedFindingReport(a.Report); err != nil {
		caseHTTPError(w, err)
		return
	}
	if err := s.m.pg.UpdateFindingCaseReport(r.Context(), c.ID, a.Version, a.Title, a.Report, a.Severity, a.SeverityReason); err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func idParam(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func (s *Server) findingCaseTools() []actool.CoreTool {
	props := map[string]any{
		"finding_id":      idParam("standalone finding record ID, not an exploration node ID"),
		"finding_ids":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "list of standalone finding record IDs"},
		"task_id":         idParam("source task ID"),
		"case_id":         idParam("finding case ID"),
		"version":         idParam("version from the actual get_finding_case read"),
		"page":            idParam("pagination, starting from 1, 20 per page"),
		"title":           strParam("title of the identical actual defect"),
		"reason":          strParam("specific evidence and merge reason — type similarity alone is insufficient"),
		"report":          strParam("complete unified Markdown report combining all member evidence"),
		"severity":        strParam("critical|high|medium|low"),
		"severity_reason": strParam("severity rationale based on verified impact, not simply the highest member severity"),
	}
	specs := []struct {
		name, desc string
		required   []string
	}{
		{"search_finding_duplicates", "Read-only search for same-task candidates, returns summary and assets; different severities may still be duplicates. Supports pagination, fetch details as needed.", []string{"finding_id"}},
		{"get_finding_record", "Read the original evidence and report for a standalone finding, returns exploration node ID for trace lookup.", []string{"finding_id"}},
		{"merge_finding_records", "Merge when evidence clearly proves the same defect; all original records are preserved. No cross-task merges, no severity changes, cannot bypass manual rejections.", []string{"task_id", "finding_ids", "title", "reason"}},
		{"suggest_finding_merge", "Submit a suspected-duplicate suggestion for human confirmation when evidence is insufficient; not an automatic merge. Exactly two same-task records only.", []string{"task_id", "finding_ids", "title", "reason"}},
		{"get_finding_case", "Read the finding case and its member pages. Record the version first, then read member reports, evidence and traffic; re-read on version change.", []string{"case_id"}},
		{"update_finding_case_report", "Save the unified report and severity rationale; version must come from an actual read. Each member's original report is preserved — no new findings created.", []string{"case_id", "version", "title", "report", "severity", "severity_reason"}},
	}
	out := []actool.CoreTool{}
	for _, spec := range specs {
		spec := spec
		handler := func(ctx context.Context, in json.RawMessage) (actool.Result, error) {
			var a findingCaseRequest
			if err := json.Unmarshal(in, &a); err != nil {
				return actool.Errorf(err.Error()), nil
			}
			v, err := s.performFindingCaseTool(ctx, spec.name, a)
			if err != nil {
				return actool.Errorf(err.Error()), nil
			}
			raw, _ := json.Marshal(v)
			return actool.Text(string(raw)), nil
		}
		schemaProps := map[string]any{}
		for _, key := range spec.required {
			schemaProps[key] = props[key]
		}
		if spec.name == "search_finding_duplicates" || spec.name == "get_finding_case" {
			schemaProps["page"] = props["page"]
		}
		schema := objSchema(schemaProps, spec.required...)
		if strings.HasPrefix(spec.name, "get_") || strings.HasPrefix(spec.name, "search_") {
			out = append(out, roTool(spec.name, spec.desc, schema, handler))
		} else {
			out = append(out, wrTool(spec.name, spec.desc, schema, handler))
		}
	}
	out = append(out, s.toolCompleteFindingCaseReview())
	return out
}
func (s *Server) performFindingCaseTool(ctx context.Context, name string, a findingCaseRequest) (any, error) {
	if err := s.checkCaseReviewScope(ctx, name, a); err != nil {
		return nil, err
	}
	fid := parseProfileID(a.FindingID)
	cid := parseProfileID(a.CaseID)
	page := a.Page
	if page < 1 {
		page = 1
	}
	switch name {
	case "search_finding_duplicates":
		scope, err := s.historicalFindingReviewSelection(ctx)
		if err != nil {
			return nil, err
		}
		var scopes [][]int64
		if scope != nil {
			scopes = append(scopes, scope)
		}
		items, total, err := s.m.pg.FindingCaseCandidates(fid, page, 20, scopes...)
		return map[string]any{"items": items, "total": total, "page": page, "page_size": 20}, err
	case "get_finding_record":
		f, err := s.m.pg.GetFinding(fid)
		if err != nil {
			return nil, err
		}
		if f == nil {
			return nil, errors.New("finding does not exist")
		}
		dto := s.caseFindingDTO(f)
		cid, err := s.m.pg.FindingCaseID(f.ID)
		if err != nil {
			return nil, err
		}
		if cid > 0 {
			dto.CaseID = i64s(cid)
		}
		if conv := intercept.ConvIDFromContext(ctx); conv > 0 {
			if _, err := s.m.pg.Exec(`UPDATE finding_case_review_runs SET reviewed_ids=CASE WHEN reviewed_ids @> jsonb_build_array($2::bigint) THEN reviewed_ids ELSE reviewed_ids || jsonb_build_array($2::bigint) END WHERE conversation_id=$1 AND finding_ids @> jsonb_build_array($2::bigint)`, conv, f.ID); err != nil {
				return nil, err
			}
		}
		return map[string]any{"finding": dto, "finding_node_id": f.NodeID}, nil
	case "merge_finding_records":
		id, err := s.m.pg.MergeFindingCase(ctx, parseProfileID(a.TaskID), a.ids(), a.Title, a.Reason, "reporter")
		return map[string]string{"case_id": i64s(id)}, err
	case "suggest_finding_merge":
		ids := a.ids()
		if len(ids) != 2 {
			return nil, errors.New("duplicate suggestion requires exactly two records")
		}
		id, err := s.m.pg.SuggestFindingCase(ctx, parseProfileID(a.TaskID), ids[0], ids[1], a.Title, a.Reason)
		return map[string]string{"suggestion_id": i64s(id)}, err
	case "get_finding_case":
		c, err := s.m.pg.GetFindingCase(cid)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, errors.New("finding case does not exist")
		}
		fs, total, err := s.m.pg.FindingCaseMembers(cid, page, 20)
		members := []FindingDTO{}
		for _, f := range fs {
			members = append(members, s.caseFindingDTO(f))
		}
		return map[string]any{"case": c, "members": members, "total": total, "page": page, "page_size": 20}, err
	case "update_finding_case_report":
		if err := validateUnifiedFindingReport(a.Report); err != nil {
			return nil, err
		}
		err := s.m.pg.UpdateFindingCaseReport(ctx, cid, a.Version, a.Title, a.Report, a.Severity, a.SeverityReason)
		return map[string]bool{"ok": err == nil}, err
	}
	return nil, errors.New("unknown merge tool")
}
func (s *Server) seedFindingCaseTools() {
	_, _ = s.m.pg.Exec(`UPDATE finding_case_review_runs SET state='failed',error='service restart interrupted review, please resubmit' WHERE state IN ('queued','running')`)
	a, err := s.m.pg.GetAgentByKey("reporter")
	if err != nil || a == nil {
		return
	}
	const flag = "finding_case_tools_v2_review"
	if v, _, _ := s.m.pg.GetSetting(flag); v == "true" {
		return
	}
	names := []string{}
	previous, _, _ := s.m.pg.GetSetting("finding_case_tools_v1")
	for _, t := range s.findingCaseTools() {
		schema, _ := json.Marshal(t.InputSchema())
		if err := s.m.pg.SeedTool(t.Name(), t.Description(), schema, json.RawMessage(`["reporter"]`)); err != nil {
			return
		}
		if previous != "true" || t.Name() == "complete_finding_case_review" {
			names = append(names, t.Name())
		}
	}
	if err := s.m.pg.AddAgentToToolBinding("reporter", names); err != nil {
		return
	}
	_ = s.m.pg.SetSetting(flag, "true")
}
func (s *Server) reviewFindingCases(w http.ResponseWriter, r *http.Request) {
	var a findingCaseRequest
	if !decodeCaseRequest(w, r, &a) {
		return
	}
	ids := a.ids()
	if cid := parseProfileID(a.CaseID); cid > 0 {
		c, err := s.m.pg.GetFindingCase(cid)
		if err != nil || c == nil || !c.Active {
			writeErr(w, 400, "finding case is not available")
			return
		}
		if ctx := r.URL.Query().Get("context_task"); ctx != "" && (c.TaskID == nil || ctx != i64s(*c.TaskID)) {
			writeErr(w, 403, "inherited finding case is read-only")
			return
		}
		fs, _, err := s.m.pg.FindingCaseMembers(cid, 1, 100000)
		if err != nil {
			caseHTTPError(w, err)
			return
		}
		ids = nil
		for _, f := range fs {
			ids = append(ids, f.ID)
		}
	}
	if len(ids) == 0 || (len(a.CaseID) == 0 && len(ids) > 200) {
		writeErr(w, 400, "please select 1 to 200 original findings")
		return
	}
	reporter, err := s.m.pg.GetAgentByKey("reporter")
	if err != nil || reporter == nil || !reporter.Enabled || s.chatAgentRef() == nil {
		writeErr(w, 409, "reporter agent is not enabled or model is unavailable")
		return
	}
	batches := map[int64][]int64{}
	for _, fid := range ids {
		f, err := s.m.pg.FindingCaseSummary(fid)
		if err != nil || f == nil || f.TaskID == nil {
			writeErr(w, 400, "selected finding source task is not available")
			return
		}
		batches[*f.TaskID] = append(batches[*f.TaskID], fid)
	}
	// Validate and snapshot every source before starting any batch.
	taskContexts := map[int64]triggeredRun{}
	for taskID := range batches {
		t, ok := s.m.Task(i64s(taskID))
		if !ok {
			writeErr(w, 409, "task is not available")
			return
		}
		taskContexts[taskID] = triggeredRun{taskDesc: t.Description, taskGoal: t.Goal}
	}
	reservations, err := s.m.pg.CreateFindingCaseReviews(r.Context(), batches)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	runs := []map[string]any{}
	for _, reservation := range reservations {
		taskID, fids := reservation.TaskID, reservation.FindingIDs
		taskContext := taskContexts[taskID]
		raw, _ := json.Marshal(fids)
		msg := fmt.Sprintf("Manual review (read-only original findings): selected finding_ids=%s for this task. Read each report, judge only identical actual root causes — merge if clear, suggest if uncertain; update unified report for existing groups. Do not create or overwrite original reports, change severity/status, add evidence, merge unselected findings, or probe targets. Call complete_finding_case_review before finishing to submit per-record verdicts and pass validation.", raw)
		item := triggeredRun{agentKey: "reporter", title: "Finding Review · task#" + i64s(taskID), message: msg, taskID: taskID, taskDesc: taskContext.taskDesc, taskGoal: taskContext.taskGoal, conversationID: reservation.ConversationID}
		cfg := s.readTriggerBehavior("reporter")
		s.queueMu.Lock()
		s.triggerCfg["reporter"] = cfg
		s.triggerQ["reporter"] = append(s.triggerQ["reporter"], item)
		s.pumpLocked("reporter")
		s.queueMu.Unlock()
		runs = append(runs, map[string]any{"conversation_id": reservation.ConversationID, "task_id": i64s(taskID)})
	}
	writeJSON(w, 202, map[string]any{"runs": runs})
}
func (s *Server) caseReviewRuns(w http.ResponseWriter, r *http.Request) {
	rows, err := s.m.pg.Query(`SELECT conversation_id,task_id,state,error FROM finding_case_review_runs ORDER BY conversation_id DESC LIMIT 50`)
	if err != nil {
		caseHTTPError(w, err)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var conv int64
		var task sql.NullInt64
		var state, msg string
		if err := rows.Scan(&conv, &task, &state, &msg); err != nil {
			caseHTTPError(w, err)
			return
		}
		items = append(items, map[string]any{"conversation_id": conv, "task_id": i64s(task.Int64), "state": state, "error": msg})
	}
	if err := rows.Err(); err != nil {
		caseHTTPError(w, err)
		return
	}
	writeJSON(w, 200, items)
}

// Manual reviews cannot mutate a finding outside the explicit selection.
func (s *Server) checkCaseReviewScope(ctx context.Context, name string, a findingCaseRequest) error {
	conv := intercept.ConvIDFromContext(ctx)
	if conv == 0 {
		return nil
	}
	var raw, conclusion json.RawMessage
	err := s.m.pg.QueryRow(`SELECT finding_ids,conclusion FROM finding_case_review_runs WHERE conversation_id=$1`, conv).Scan(&raw, &conclusion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	var ids []int64
	if err := json.Unmarshal(raw, &ids); err != nil {
		return err
	}
	if string(conclusion) != "{}" && (name == "merge_finding_records" || name == "suggest_finding_merge" || name == "update_finding_case_report") {
		return errors.New("review decisions have been submitted, no further changes after acceptance")
	}
	allowed := map[int64]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	switch name {
	case "search_finding_duplicates":
		if !allowed[parseProfileID(a.FindingID)] {
			return errors.New("historical review can only search candidates for selected records")
		}
	case "update_finding_report", "bind_finding_traffic", "report_finding":
		return errors.New("historical review cannot modify original reports, findings or evidence; only merge relations and unified report")
	case "merge_finding_records", "suggest_finding_merge":
		checkIDs := a.ids()
		for _, id := range checkIDs {
			if !allowed[id] {
				return errors.New("manual review can only merge selected records")
			}
		}
	case "update_finding_case_report":
		fs, _, err := s.m.pg.FindingCaseMembers(parseProfileID(a.CaseID), 1, 100000)
		if err != nil {
			return err
		}
		found := false
		for _, f := range fs {
			if allowed[f.ID] {
				found = true
			}
		}
		if !found {
			return errors.New("this finding case is not within the manual selection scope")
		}
	}
	return nil
}
