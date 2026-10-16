// Package researchkit provides offline AppSec analysis and report preparation.
// Its workflows are adapted from claude-redteam; no shell or model delegation is used.
package researchkit

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	cvss40 "github.com/pandatix/go-cvss/40"
)

const MaxInput = 1 << 20
const SourceCommit = "329fc1273c202844cd041e08e74298d683258158"

//go:embed catalog.json
var resources embed.FS

type Skill struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	SourcePath   string   `json:"source_path"`
	SourceSHA256 string   `json:"source_sha256"`
	Mode         string   `json:"mode"`
}

func Catalog(query, tech string) []Skill {
	raw, _ := resources.ReadFile("catalog.json")
	var all []Skill
	_ = json.Unmarshal(raw, &all)
	out := []Skill{}
	terms := strings.Fields(strings.ToLower(tech))
	for _, s := range all {
		text := strings.ToLower(s.Slug + " " + s.Name + " " + s.Description + " " + s.Category + " " + strings.Join(s.Tags, " "))
		if query != "" && !strings.Contains(text, strings.ToLower(query)) {
			continue
		}
		if len(terms) > 0 {
			matched := false
			for _, term := range terms {
				for _, tag := range s.Tags {
					if strings.EqualFold(term, tag) {
						matched = true
					}
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

func CheckInput(input string) error {
	if len(input) > MaxInput {
		return errors.New("input exceeds 1 MiB")
	}
	if strings.TrimSpace(input) == "" {
		return errors.New("input is empty")
	}
	return nil
}

type CVSSResult struct {
	Vector       string   `json:"vector"`
	Score        float64  `json:"score"`
	Severity     string   `json:"severity"`
	ClaimedScore *float64 `json:"claimed_score,omitempty"`
	MatchesClaim bool     `json:"matches_claim"`
}

func ScoreCVSS(vector string, claimed *float64) (CVSSResult, error) {
	vector = strings.TrimSpace(vector)
	if len(vector) > 2048 {
		return CVSSResult{}, errors.New("CVSS vector is too long")
	}
	if !strings.HasPrefix(vector, "CVSS:4.0/") {
		return CVSSResult{}, errors.New("provide a CVSS:4.0 vector")
	}
	if claimed != nil && (*claimed < 0 || *claimed > 10 || math.IsNaN(*claimed) || math.IsInf(*claimed, 0)) {
		return CVSSResult{}, errors.New("claimed score must be between 0 and 10")
	}
	parsed, err := cvss40.ParseVector(vector)
	if err != nil {
		return CVSSResult{}, fmt.Errorf("invalid CVSS vector: %w", err)
	}
	score := parsed.Score()
	severity := "none"
	switch {
	case score >= 9:
		severity = "critical"
	case score >= 7:
		severity = "high"
	case score >= 4:
		severity = "medium"
	case score > 0:
		severity = "low"
	}
	return CVSSResult{Vector: vector, Score: score, Severity: severity, ClaimedScore: claimed, MatchesClaim: claimed == nil || *claimed == score}, nil
}

type FindingDraft struct {
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	Steps               []string `json:"steps"`
	Expected            string   `json:"expected"`
	Actual              string   `json:"actual"`
	Impact              string   `json:"impact"`
	Remediation         string   `json:"remediation"`
	ScopeConfirmed      bool     `json:"scope_confirmed"`
	PreconditionsProven bool     `json:"preconditions_proven"`
	Evidence            []string `json:"evidence"`
	Vector              string   `json:"vector,omitempty"`
	ClaimedScore        *float64 `json:"claimed_score,omitempty"`
}

type Gate struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}
type Validation struct {
	Status   string      `json:"status"`
	Checks   []Gate      `json:"checks"`
	Warnings []string    `json:"warnings"`
	CVSS     *CVSSResult `json:"cvss,omitempty"`
}

func nonempty(values []string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return len(values) > 0
}

func Validate(d FindingDraft) Validation {
	result := Validation{Status: "ready_for_human_review", Warnings: []string{}, Checks: []Gate{
		{"scope", d.ScopeConfirmed, "Operator confirms this work belongs to an isolated local lab; not independently verified."},
		{"identity", strings.TrimSpace(d.Title) != "" && strings.TrimSpace(d.Summary) != "", "Name the affected control and summarize the observation."},
		{"reproduction", nonempty(d.Steps), "Provide repeatable steps using dedicated lab data."},
		{"control_comparison", strings.TrimSpace(d.Expected) != "" && strings.TrimSpace(d.Actual) != "", "Separate expected control behavior from the observed result."},
		{"evidence", nonempty(d.Evidence), "Reference evidence records; this check does not validate their contents."},
		{"proven_impact", strings.TrimSpace(d.Impact) != "" && d.PreconditionsProven, "Describe demonstrated impact without assuming an unproven prerequisite."},
		{"remediation", strings.TrimSpace(d.Remediation) != "", "State a concrete fix and retest condition."},
	}}
	for _, g := range result.Checks {
		if !g.Passed {
			result.Status = "needs_review"
		}
	}
	if strings.TrimSpace(d.Vector) != "" {
		c, err := ScoreCVSS(d.Vector, d.ClaimedScore)
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			result.Status = "needs_review"
		} else {
			result.CVSS = &c
			if !c.MatchesClaim {
				result.Warnings = append(result.Warnings, "Claimed severity score differs from the supplied CVSS vector.")
				result.Status = "needs_review"
			}
		}
	}
	if d.ClaimedScore != nil && strings.TrimSpace(d.Vector) == "" {
		result.Warnings = append(result.Warnings, "A claimed CVSS score requires its vector.")
		result.Status = "needs_review"
	}
	for _, word := range []string{"could ", "might ", "potentially ", "if stolen", "assuming "} {
		if strings.Contains(strings.ToLower(d.Impact), word) {
			result.Warnings = append(result.Warnings, "Impact contains conditional language; distinguish observation from hypothesis.")
			result.Status = "needs_review"
			break
		}
	}
	result.Warnings = append(result.Warnings, "Checklist completion is not confirmation of a vulnerability, authorization, or business impact.")
	return result
}

func DecodeStrict(raw []byte, out any) error {
	if len(raw) > MaxInput {
		return errors.New("input exceeds 1 MiB")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}
	if dec.More() {
		return errors.New("multiple JSON values are not allowed")
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		return errors.New("multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return errors.New("invalid trailing JSON")
	}
	return nil
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
