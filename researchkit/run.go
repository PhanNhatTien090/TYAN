package researchkit

import (
	"encoding/json"
	"errors"
	"os/exec"
)

type Request struct {
	Input        string        `json:"input,omitempty"`
	Source       string        `json:"source,omitempty"`
	Vector       string        `json:"vector,omitempty"`
	ClaimedScore *float64      `json:"claimed_score,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Finding      *FindingDraft `json:"finding,omitempty"`
}

func Run(operation string, raw []byte) (any, error) {
	var request Request
	if err := DecodeStrict(raw, &request); err != nil {
		return nil, err
	}
	switch operation {
	case "openapi":
		return ParseOpenAPI(request.Input, request.Limit)
	case "normalize":
		return Normalize(request.Source, request.Input, request.Limit)
	case "redact":
		return Redact(request.Input)
	case "cvss":
		return ScoreCVSS(request.Vector, request.ClaimedScore)
	case "validate", "report":
		if request.Finding == nil {
			return nil, errors.New("finding is required")
		}
		if operation == "validate" {
			return Validate(*request.Finding), nil
		}
		return map[string]any{"markdown": ReportMarkdown(*request.Finding), "validation": Validate(*request.Finding), "notice": "Draft only; no findings are saved or submitted."}, nil
	default:
		return nil, errors.New("unknown research operation")
	}
}

type ToolAvailability struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	OnPath   bool   `json:"on_path"`
}

// Inventory checks executable names only. It neither launches nor installs tools.
func Inventory() []ToolAvailability {
	groups := []struct {
		category string
		names    []string
	}{
		{"runtime", []string{"python3", "python", "node", "go"}},
		{"discovery", []string{"nmap", "nuclei", "ffuf", "httpx", "dnsx", "subfinder", "katana"}},
		{"code-review", []string{"semgrep", "trivy", "gitleaks"}},
		{"mobile", []string{"frida", "jadx", "apktool"}},
		{"proxy", []string{"burp", "caido"}},
	}
	out := []ToolAvailability{}
	for _, group := range groups {
		for _, name := range group.names {
			_, err := exec.LookPath(name)
			out = append(out, ToolAvailability{Name: name, Category: group.category, OnPath: err == nil})
		}
	}
	return out
}

// ValidFindingJSON validates the complete input before generating a binary report.
func ValidFindingJSON(raw json.RawMessage) (*FindingDraft, error) {
	var req Request
	if err := DecodeStrict(raw, &req); err != nil {
		return nil, err
	}
	if req.Finding == nil {
		return nil, errors.New("finding is required")
	}
	return req.Finding, nil
}
