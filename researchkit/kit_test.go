package researchkit

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestCatalogPinnedAndRouted(t *testing.T) {
	all := Catalog("", "")
	if len(all) != 65 {
		t.Fatalf("catalog has %d entries", len(all))
	}
	names := map[string]bool{}
	for _, skill := range all {
		if names[skill.Name] || skill.SourceSHA256 == "" || skill.Mode != "tyan-review-adaptation" {
			t.Fatalf("invalid entry: %+v", skill)
		}
		names[skill.Name] = true
	}
	routed := Catalog("", "graphql")
	if len(routed) == 0 {
		t.Fatal("no technology matches")
	}
	for _, s := range routed {
		if !strings.HasPrefix(s.Name, "rt-") {
			t.Fatal("not a native skill name")
		}
	}
	if len(Catalog("", "totally-unknown-stack")) != 0 {
		t.Fatal("unknown technology should not recommend every skill")
	}
}

func TestOpenAPIShapeAndAuthOverrides(t *testing.T) {
	input := `{"openapi":"3.0.3","security":[{"bearer":[]}],"paths":{"/users/{id}":{"parameters":[{"$ref":"#/components/parameters/ID"}],"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}}}},"post":{"security":[],"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}}}},"/optional":{"get":{"security":[{}, {"bearer":[]}]}}},"components":{"parameters":{"ID":{"in":"path","name":"id"}},"schemas":{"User":{"type":"object","properties":{"name":{"type":"string","example":"SECRET_SAMPLE"},"password":{"type":"string","default":"SECRET_PASSWORD"}}}}}}`
	result, err := ParseOpenAPI(input, 50)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 {
		t.Fatalf("operation count=%d", result.Total)
	}
	byKey := map[string]APIOperation{}
	for _, op := range result.Operations {
		byKey[op.Method+op.Path] = op
	}
	if byKey["GET/users/{id}"].Authentication != "required_declared" || byKey["POST/users/{id}"].Authentication != "none_declared" || byKey["GET/optional"].Authentication != "optional_declared" {
		t.Fatal("security inheritance/override lost")
	}
	if len(byKey["GET/users/{id}"].PathParameters) != 1 || len(byKey["POST/users/{id}"].BodyFields) != 2 {
		t.Fatal("local reference resolution failed")
	}
	b, _ := json.Marshal(result)
	if bytes.Contains(b, []byte("SECRET_")) {
		t.Fatal("schema values leaked")
	}
	short, err := ParseOpenAPI(input, 1)
	if err != nil || !short.Truncated || short.Total != 3 || len(short.Operations) != 1 {
		t.Fatalf("limit changed total: %+v %v", short, err)
	}
}

func TestSwaggerYAMLAndReferenceCycles(t *testing.T) {
	input := "swagger: '2.0'\npaths:\n  /items:\n    post:\n      parameters:\n        - in: body\n          name: item\n          schema:\n            $ref: '#/definitions/Item'\ndefinitions:\n  Item:\n    type: object\n    properties:\n      label:\n        type: string\n"
	result, err := ParseOpenAPI(input, 10)
	if err != nil || len(result.Operations) != 1 || len(result.Operations[0].BodyFields) != 1 {
		t.Fatalf("Swagger: %+v %v", result, err)
	}
	cyclic := `{"openapi":"3.1.0","paths":{"/a":{"post":{"requestBody":{"$ref":"#/components/requestBodies/A"}}}},"components":{"requestBodies":{"A":{"$ref":"#/components/requestBodies/A"}}}}`
	if _, err := ParseOpenAPI(cyclic, 10); err != nil {
		t.Fatal(err)
	}
	external := strings.ReplaceAll(cyclic, "#/components/requestBodies/A", "https://not-requested.invalid/private")
	if _, err := ParseOpenAPI(external, 10); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeSavedOutputs(t *testing.T) {
	nuclei := `{"template-id":"lab-control","matched-at":"http://user:URL_SECRET@localhost/api?token=QUERY_SECRET","info":{"name":"Lab control observation","severity":"high"},"request":"REQUEST_SECRET","response":"RESPONSE_SECRET"}`
	result, err := Normalize("nuclei", nuclei+"\n"+nuclei, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Duplicates != 1 || result.Items[0].Status != "unverified" {
		t.Fatalf("dedup: %+v", result)
	}
	b, _ := json.Marshal(result)
	if bytes.Contains(b, []byte("SECRET")) {
		t.Fatal("raw evidence leaked into normalized rows")
	}
	ffuf, err := Normalize("ffuf", `{"results":[{"url":"http://localhost/admin","status":200}]}`, 10)
	if err != nil || ffuf.Items[0].Severity != "info" {
		t.Fatal("discovery must not auto-promote to a vulnerability")
	}
	nmap, err := Normalize("nmap", `<nmaprun><host><address addr="127.0.0.1" addrtype="ipv4"/><ports><port protocol="tcp" portid="5432"><state state="open"/><service name="postgresql"/></port><port portid="80"><state state="closed"/></port></ports></host></nmaprun>`, 10)
	if err != nil || nmap.Total != 1 || nmap.Items[0].Severity != "info" {
		t.Fatalf("Nmap: %+v %v", nmap, err)
	}
	if _, err = Normalize("nuclei", nuclei+"\nNOT JSON", 10); err == nil {
		t.Fatal("malformed records silently ignored")
	}
}

func TestPrivacyAndLimits(t *testing.T) {
	for _, input := range []string{`{"user":{"email":"person@example.test","password":"PWD_SENTINEL"},"items":[{"api_key":"KEY_SENTINEL"}]}`, "Authorization: Bearer HEADER_SENTINEL\nCookie: session=COOKIE_SENTINEL\nemail=person@example.test\npassword=PWD_SENTINEL"} {
		result, err := Redact(input)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"person@example.test", "PWD_SENTINEL", "KEY_SENTINEL", "HEADER_SENTINEL", "COOKIE_SENTINEL"} {
			if strings.Contains(result.Text, secret) {
				t.Fatalf("redaction missed %s", secret)
			}
		}
	}
	if _, err := Redact(strings.Repeat("x", MaxInput+1)); err == nil {
		t.Fatal("oversized input accepted")
	}
	if _, err := Run("redact", []byte(`{"input":"ok","command":"anything"}`)); err == nil {
		t.Fatal("unexpected fields accepted")
	}
	if _, err := Run("redact", []byte(`{"input":"ok"} {}`)); err == nil {
		t.Fatal("multiple JSON documents accepted")
	}
}

const highestVector = "CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:H/SI:H/SA:H"

func TestCVSSAndFindingValidation(t *testing.T) {
	zero := 0.0
	score, err := ScoreCVSS(highestVector, &zero)
	if err != nil || score.Score != 10 || score.Severity != "critical" || score.MatchesClaim {
		t.Fatalf("score: %+v %v", score, err)
	}
	if _, err := ScoreCVSS("CVSS:4.0/", nil); err == nil {
		t.Fatal("invalid vector accepted")
	}
	if _, err := ScoreCVSS("CVSS:3.1/AV:N", nil); err == nil {
		t.Fatal("wrong CVSS version accepted")
	}
	draft := FindingDraft{Title: "Lab control", Summary: "Observed fixture failure", Steps: []string{"Use the local fixture"}, Expected: "Denied", Actual: "Allowed", Impact: "Synthetic record read", Remediation: "Enforce policy and retest", ScopeConfirmed: true, PreconditionsProven: true, Evidence: []string{"traffic:1"}}
	if v := Validate(draft); v.Status != "ready_for_human_review" || len(v.Checks) != 7 {
		t.Fatalf("validation: %+v", v)
	}
	draft.Impact = "Could affect other tenants if stolen"
	if v := Validate(draft); v.Status != "needs_review" {
		t.Fatal("hypothetical impact promoted")
	}
	draft.Vector = highestVector
	draft.ClaimedScore = &zero
	if v := Validate(draft); v.Status != "needs_review" || v.CVSS == nil || v.CVSS.MatchesClaim {
		t.Fatal("CVSS mismatch missed")
	}
}

func TestReportDOCXEscapesAndMinimizes(t *testing.T) {
	draft := FindingDraft{Title: "Lab <control> & notes", Summary: "Cookie: COOKIE_SECRET", Evidence: []string{"traffic:1"}}
	body, err := ReportDOCX(draft)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range archive.File {
		f, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(f)
		f.Close()
		decoder := xml.NewDecoder(bytes.NewReader(content))
		for {
			_, err = decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("invalid XML in %s: %v", file.Name, err)
			}
		}
		if file.Name == "word/document.xml" {
			found = true
			if bytes.Contains(content, []byte("COOKIE_SECRET")) || !bytes.Contains(content, []byte("&lt;control&gt;")) {
				t.Fatal("report not minimized/escaped")
			}
		}
	}
	if !found {
		t.Fatal("missing Word document part")
	}
}
