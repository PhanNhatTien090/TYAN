package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Autumn-27/artex/intercept"
	actool "github.com/Autumn-27/norma/tool"
)

type findingReviewDecision struct {
	FindingID         int64  `json:"finding_id"`
	Verdict           string `json:"verdict"`
	Reason            string `json:"reason"`
	RootCauseEvidence string `json:"root_cause_evidence"`
}
type findingReviewConclusion struct {
	Summary   string                  `json:"summary"`
	Decisions []findingReviewDecision `json:"decisions"`
}

func (s *Server) isHistoricalFindingReview(conv int64) (bool, error) {
	var exists bool
	err := s.m.pg.QueryRow(`SELECT EXISTS(SELECT 1 FROM finding_case_review_runs WHERE conversation_id=$1)`, conv).Scan(&exists)
	return exists, err
}

func (s *Server) toolCompleteFindingCaseReview() actool.CoreTool {
	decision := objSchema(map[string]any{
		"finding_id":          idParam("selected standalone finding record ID"),
		"verdict":             map[string]any{"type": "string", "enum": []string{"independent", "merged", "suggested"}},
		"reason":              strParam("verdict reason; different endpoints or read/write impact alone cannot prove different root causes"),
		"root_cause_evidence": strParam("specific material from original reports/evidence supporting the actual shared or distinct root cause; if uncertain, state the missing evidence and the suggestion already saved"),
	}, "finding_id", "verdict", "reason", "root_cause_evidence")
	return wrTool("complete_finding_case_review", "Before finishing a historical review, submit a root-cause verdict for every selected record and pass validation: all records must have been read, suspected duplicates must already have suggestions, merged groups must have a complete up-to-date unified report. Cannot declare completion if validation fails.", objSchema(map[string]any{"summary": strParam("summary of review results"), "decisions": map[string]any{"type": "array", "items": decision}}, "summary", "decisions"), func(ctx context.Context, raw json.RawMessage) (actool.Result, error) {
		var a findingReviewConclusion
		if err := json.Unmarshal(raw, &a); err != nil {
			return actool.Errorf(err.Error()), nil
		}
		conv := intercept.ConvIDFromContext(ctx)
		if err := s.saveFindingReviewConclusion(ctx, conv, a); err != nil {
			return actool.Errorf(err.Error()), nil
		}
		return actool.Text("Historical review accepted; original records and reports preserved, per-record root-cause verdicts saved."), nil
	})
}

func (s *Server) saveFindingReviewConclusion(ctx context.Context, conv int64, a findingReviewConclusion) error {
	if conv <= 0 {
		return errors.New("only valid for historical review sessions with a conversation context")
	}
	return s.m.pg.WithEvidenceTx(ctx, func(tx *sql.Tx) error {
		var idsRaw, readsRaw json.RawMessage
		var state string
		if err := tx.QueryRow(`SELECT finding_ids,reviewed_ids,state FROM finding_case_review_runs WHERE conversation_id=$1 FOR UPDATE`, conv).Scan(&idsRaw, &readsRaw, &state); err != nil {
			return err
		}
		if state != "running" {
			return errors.New("review session is not running, cannot submit acceptance")
		}
		var ids, reads []int64
		if err := json.Unmarshal(idsRaw, &ids); err != nil {
			return err
		}
		if err := json.Unmarshal(readsRaw, &reads); err != nil {
			return err
		}
		read := map[int64]bool{}
		for _, id := range reads {
			read[id] = true
		}
		decisions := map[int64]findingReviewDecision{}
		for _, d := range a.Decisions {
			if _, ok := decisions[d.FindingID]; ok {
				return errors.New("duplicate verdict finding_id")
			}
			decisions[d.FindingID] = d
		}
		if strings.TrimSpace(a.Summary) == "" || len(ids) != len(decisions) {
			return errors.New("must provide a summary and per-record verdict for all selected findings")
		}
		for _, id := range ids {
			d, ok := decisions[id]
			if !ok || !read[id] {
				return fmt.Errorf("please read and judge selected record #%d with get_finding_record first", id)
			}
			if strings.TrimSpace(d.Reason) == "" || strings.TrimSpace(d.RootCauseEvidence) == "" {
				return fmt.Errorf("record #%d is missing verdict reason or root cause evidence", id)
			}
			var cid sql.NullInt64
			if err := tx.QueryRow(`SELECT m.case_id FROM findings f LEFT JOIN finding_case_members m ON m.finding_id=f.id WHERE f.id=$1`, id).Scan(&cid); err != nil {
				return err
			}
			switch d.Verdict {
			case "merged":
				if !cid.Valid {
					return fmt.Errorf("record #%d has not been merged", id)
				}
			case "suggested":
				var pending bool
				if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM finding_case_suggestions WHERE state='pending' AND (left_id=$1 OR right_id=$1))`, id).Scan(&pending); err != nil {
					return err
				}
				if !pending {
					return fmt.Errorf("record #%d must first have a pending suggestion saved", id)
				}
			case "independent":
				if cid.Valid {
					return fmt.Errorf("record #%d is already in a case, cannot be declared independent", id)
				}
			default:
				return errors.New("invalid verdict type")
			}
			if cid.Valid {
				var report string
				var version, rv int64
				var active bool
				if err := tx.QueryRow(`SELECT report,version,report_version,active FROM finding_cases WHERE id=$1 FOR SHARE`, cid.Int64).Scan(&report, &version, &rv, &active); err != nil {
					return err
				}
				if !active || rv != version {
					return fmt.Errorf("unified report for case #%d is outdated, please re-read and update", cid.Int64)
				}
				if err := validateUnifiedFindingReport(report); err != nil {
					return err
				}
			}
		}
		raw, err := json.Marshal(a)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE finding_case_review_runs SET conclusion=$2 WHERE conversation_id=$1`, conv, string(raw))
		return err
	})
}

func (s *Server) validateFindingReviewCompletion(conv int64) error {
	var raw json.RawMessage
	if err := s.m.pg.QueryRow(`SELECT conclusion FROM finding_case_review_runs WHERE conversation_id=$1`, conv).Scan(&raw); err != nil {
		return err
	}
	var a findingReviewConclusion
	if err := json.Unmarshal(raw, &a); err != nil {
		return err
	}
	if len(a.Decisions) == 0 {
		return errors.New("review not accepted: complete_finding_case_review with per-record root-cause verdicts was not submitted; check the execution log")
	}
	// Recheck after the last model turn; concurrent evidence changes invalidate completion.
	return s.saveFindingReviewConclusion(context.Background(), conv, a)
}

// Require substantive sections and balanced fenced blocks. This validates report
// completeness, not the truth of a model's root-cause claim or a live exploit.
func validateUnifiedFindingReport(report string) error {
	aliases := [][]string{
		{"overview", "summary"},
		{"impact"},
		{"prerequisite", "precondition"},
		{"reproduction", "reproduce"},
		{"evidence"},
		{"poc", "proof of concept"},
		{"root cause"},
		{"remediation", "recommendation", "fix"},
	}
	names := []string{"Overview", "Impact", "Prerequisites", "Reproduction", "Evidence", "PoC", "Root Cause", "Remediation"}
	found := make([]bool, len(aliases))
	current := -1
	hasContent := false
	fence := ""
	finish := func() {
		if current >= 0 && hasContent {
			found[current] = true
		}
	}
	for _, line := range strings.Split(report, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			marker := t[:3]
			if fence == "" {
				fence = marker
			} else if fence == marker {
				fence = ""
			}
			continue
		}
		if fence != "" {
			if t != "" {
				hasContent = true
			}
			continue
		}
		if strings.HasPrefix(t, "#") {
			heading := strings.TrimLeft(t, "#")
			if len(heading) == len(t) || !strings.HasPrefix(heading, " ") {
				continue
			}
			// Subheadings belong to the current section unless they start another required section.
			next := -1
			lower := strings.ToLower(heading)
			for i, words := range aliases {
				for _, word := range words {
					if strings.Contains(lower, word) {
						next = i
						break
					}
				}
				if next >= 0 {
					break
				}
			}
			if next >= 0 {
				finish()
				current = next
				hasContent = false
			}
			continue
		}
		if t != "" && t != "---" && t != "***" && t != ">" {
			hasContent = true
		}
	}
	finish()
	if fence != "" {
		return errors.New("unified report has unclosed code fence, possible truncation; please complete before saving")
	}
	missing := []string{}
	for i, ok := range found {
		if !ok {
			missing = append(missing, names[i])
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("unified report is missing sections with content: %s; please complete before saving", strings.Join(missing, ", "))
	}
	return nil
}

func (s *Server) historicalFindingReviewSelection(ctx context.Context) ([]int64, error) {
	conv := intercept.ConvIDFromContext(ctx)
	if conv == 0 {
		return nil, nil
	}
	var raw json.RawMessage
	err := s.m.pg.QueryRow(`SELECT finding_ids FROM finding_case_review_runs WHERE conversation_id=$1`, conv).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	err = json.Unmarshal(raw, &ids)
	return ids, err
}
