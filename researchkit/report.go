package researchkit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

func ReportMarkdown(d FindingDraft) string {
	var out strings.Builder
	title := strings.ReplaceAll(strings.TrimSpace(d.Title), "\n", " ")
	fmt.Fprintf(&out, "# %s\n\n> Draft for human review. Checklist completion does not confirm a vulnerability.\n\n", title)
	for _, section := range []struct{ name, value string }{{"Summary", d.Summary}, {"Expected behavior", d.Expected}, {"Observed behavior", d.Actual}, {"Demonstrated impact", d.Impact}, {"Remediation and retest", d.Remediation}} {
		fmt.Fprintf(&out, "## %s\n\n%s\n\n", section.name, section.value)
	}
	out.WriteString("## Reproduction steps\n\n")
	for i, step := range d.Steps {
		fmt.Fprintf(&out, "%d. %s\n", i+1, step)
	}
	out.WriteString("\n## Evidence references\n\n")
	for _, e := range d.Evidence {
		fmt.Fprintf(&out, "- %s\n", e)
	}
	v := Validate(d)
	if v.CVSS != nil {
		fmt.Fprintf(&out, "\n## CVSS 4.0\n\nVector: `%s`\n\nComputed score: %.1f (%s). Metrics remain the reviewer's responsibility.\n", v.CVSS.Vector, v.CVSS.Score, v.CVSS.Severity)
	}
	fmt.Fprintf(&out, "\n## Review checklist\n\nStatus: %s\n", v.Status)
	for _, g := range v.Checks {
		mark := " "
		if g.Passed {
			mark = "x"
		}
		fmt.Fprintf(&out, "- [%s] %s: %s\n", mark, g.Name, g.Detail)
	}
	redacted, err := Redact(out.String())
	if err == nil {
		return redacted.Text
	}
	return "Report exceeds size limit. Shorten the draft before exporting."
}

// ReportDOCX emits plain text OOXML; no macros, remote assets or executable content.
func ReportDOCX(d FindingDraft) ([]byte, error) {
	markdown := ReportMarkdown(d)
	var document strings.Builder
	document.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, line := range strings.Split(markdown, "\n") {
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(line)); err != nil {
			return nil, err
		}
		document.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + escaped.String() + `</w:t></w:r></w:p>`)
	}
	document.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1134" w:right="1134" w:bottom="1134" w:left="1134"/></w:sectPr></w:body></w:document>`)
	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	entries := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`},
		{"word/document.xml", document.String()},
	}
	for _, entry := range entries {
		w, err := archive.Create(entry.name)
		if err != nil {
			return nil, err
		}
		if _, err = w.Write([]byte(entry.body)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
