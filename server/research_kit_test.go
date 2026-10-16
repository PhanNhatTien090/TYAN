package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/researchkit"
	"github.com/Autumn-27/norma/skill"
)

func TestResearchKitNativeToolContracts(t *testing.T) {
	s := &Server{}
	tools := s.researchTools()
	if len(tools) != 8 {
		t.Fatalf("tools=%d", len(tools))
	}
	for _, tool := range tools {
		if !tool.IsReadOnly(json.RawMessage(`{}`)) || !tool.IsConcurrencySafe(json.RawMessage(`{}`)) {
			t.Fatalf("wrong contract: %s", tool.Name())
		}
		if tool.Name() == "research_cvss_score" {
			result, err := tool.Call(context.Background(), json.RawMessage(`{"vector":"CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H"}`), nil)
			if err != nil || result.IsError || !strings.Contains(result.Flatten(), `"score":10`) {
				t.Fatalf("native scoring: %v %s", err, result.Flatten())
			}
		}
	}
	registry, err := skill.LoadDir(filepath.Join("..", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range researchkit.Catalog("", "") {
		loaded, ok := registry.Get(entry.Name)
		if !ok || loaded.Description == "" || !strings.Contains(loaded.Instructions, "research_validate_finding") {
			t.Fatalf("skill not invokable: %s", entry.Name)
		}
	}
}

func TestResearchKitHTTPValidation(t *testing.T) {
	s := &Server{}
	mux := http.NewServeMux()
	s.registerResearchRoutes(mux)
	run := func(path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("POST", path, strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, request)
		return rec
	}
	for _, tc := range []struct {
		path, body string
		code       int
	}{
		{"/api/research-kit/run/unknown", `{}`, 400},
		{"/api/research-kit/run/redact", `{"input":"Authorization: Bearer FAKE_SECRET"}`, 200},
		{"/api/research-kit/run/normalize", `{"input":"{}","source":"shell"}`, 400},
		{"/api/research-kit/run/redact", `{"input":"a","command":"ignored?"}`, 400},
		{"/api/research-kit/run/validate", `{}`, 400},
		{"/api/research-kit/report.docx", `{"finding":{"title":"Local lab draft"}}`, 200},
	} {
		rec := run(tc.path, tc.body)
		if rec.Code != tc.code {
			t.Fatalf("%s: %d %s", tc.path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "FAKE_SECRET") {
			t.Fatal("redaction route returned bearer value")
		}
		if rec.Code == 200 && rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("artifact response can be cached")
		}
	}
	if rec := run("/api/research-kit/run/redact", strings.Repeat("x", researchkit.MaxInput+1)); rec.Code != 413 {
		t.Fatalf("oversize status=%d", rec.Code)
	}
}

func TestResearchKitAuthenticatedRoutesAndBindings(t *testing.T) {
	s, _, request := trafficEvidenceServer(t)
	unauth := httptest.NewRecorder()
	s.Handler().ServeHTTP(unauth, httptest.NewRequest("GET", "/api/research-kit/catalog", nil))
	if unauth.Code != 401 {
		t.Fatalf("catalog not protected: %d", unauth.Code)
	}
	for _, path := range []string{"/api/research-kit/catalog", "/api/research-kit/tools"} {
		rec := request("GET", path, "")
		if rec.Code != 200 {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	rec := request("POST", "/api/research-kit/run/cvss", `{"vector":"CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H"}`)
	if rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"score":10`)) {
		t.Fatalf("scoring: %d %s", rec.Code, rec.Body.String())
	}
	tool, err := s.m.pg.GetTool("research_cvss_score")
	if err != nil || tool == nil || !tool.Enabled {
		t.Fatalf("tool not seeded: %+v %v", tool, err)
	}
	if err = s.m.pg.UpdateTool(tool.Key, "operator-edited", tool.Schema, json.RawMessage(`["reporter"]`), false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.m.pg.Exec(`DELETE FROM tools WHERE key='research_cvss_score'`); s.seedResearchTools() })
	s.seedResearchTools()
	tool, err = s.m.pg.GetTool(tool.Key)
	if err != nil || tool.Enabled || tool.Description != "operator-edited" || len(tool.Agents) != 1 || tool.Agents[0] != "reporter" {
		t.Fatal("startup overwrote operator settings")
	}
}
