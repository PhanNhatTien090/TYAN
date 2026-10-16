# Research Kit / Hướng dẫn sử dụng

Research Kit is under **Functions → Research Kit** (`/function/research-kit`). The interface stays in English. [Integration analysis](../integrations/claude-redteam/README.md) · [Phân tích tiếng Việt](../integrations/claude-redteam/README.vi.md).

## English

**Skills:** search the 65 review adaptations by topic or exact, space-separated technology tags. Technology tags use OR matching. Copy an `rt-*` name and ask the agent to load it through the existing Skill tool; the library does not automatically execute it.

**Analyze artifacts:** paste text or upload a local UTF-8 artifact (UI limit 900 KiB; API request limit 1 MiB including JSON encoding). Choose OpenAPI, Normalize scanner output, Redact evidence or CVSS. “Load example” supplies synthetic local-lab data. These helpers do not fetch URLs, run scanners or call an external model. For normalization, select the actual saved format: Nuclei JSON/JSONL, ffuf JSON or Nmap XML. Preview limits preserve total counts; normalized observations are not confirmed findings. URL query values, user information and fragments are removed; paths can still contain sensitive identifiers. OpenAPI summaries omit examples/default values, but field/path names may themselves contain sensitive information.

**Finding & report:** fill in the title, factual summary, reproduction steps, expected and observed behavior, demonstrated impact, evidence references and remediation. Confirm scope and prerequisites only when established. The readiness check has seven gates and optional CVSS 4.0 validation. A ready status still requires human review. Preview a Markdown draft or download a basic DOCX draft. Neither action publishes a report or creates a finding record; use TYAN's normal Findings workflow to save an approved conclusion.

**Tool readiness:** checks the server process's PATH for 19 optional executable names. A detected program may still need dependencies/configuration or be an incompatible version. These programs are not needed by the eight native helpers. Detection does not execute or install them, and a missing program is not evidence the helper is broken.

Native agent tools: `research_skill_catalog`, `research_tool_inventory`, `research_openapi_summary`, `research_normalize_results`, `research_redact_evidence`, `research_cvss_score`, `research_validate_finding`, `research_report_draft`. Existing tool enable/disable, agent bindings and guard controls apply. New rows are seeded at startup without overwriting edits. Report DOCX export is an authenticated operator endpoint, not a ninth agent tool.

Artifact APIs require login and use `Cache-Control: no-store` for analysis/export responses. Helpers do not persist the supplied artifacts in a separate database. Agent tool calls can still be recorded in normal task history; redaction does not guarantee removal of all secrets or personal information. Review the output before sharing. All existing local-lab usage restrictions apply.

## Tiếng Việt

- **Skills:** tra cứu 65 skill chuyển thể, lọc theo công nghệ, sao chép tên `rt-*` rồi yêu cầu agent tải skill qua cơ chế có sẵn. Tra cứu không tự thực thi skill.
- **Analyze artifacts:** dán hoặc tải tập tin cục bộ, chọn chức năng và chạy phân tích. Có dữ liệu mẫu để thử. Không truy cập đích hay khởi chạy scanner. Chọn đúng định dạng Nuclei/ffuf/Nmap khi chuẩn hóa.
- **Finding & report:** điền thông tin thực tế và tham chiếu bằng chứng, kiểm tra 7 điều kiện, xem Markdown hoặc tải DOCX cơ bản. Đủ checklist vẫn cần người xác minh. Bản nháp không tự được lưu thành finding hay đăng ra ngoài.
- **Tool readiness:** chỉ kiểm tra tên chương trình trên PATH của máy chủ. Không tự cài, chạy hoặc xác minh phiên bản. Tám helper Go vẫn dùng được khi các chương trình tùy chọn chưa được cài.

Giới hạn upload giao diện là 900 KiB, giới hạn request API là 1 MiB tính cả JSON. Redaction chỉ hỗ trợ che thông tin thường gặp; hãy kiểm tra lại nội dung trước khi chia sẻ. Dữ liệu qua tool của agent vẫn có thể được lưu trong lịch sử task hiện có. Muốn dùng code mới cần build lại frontend/binary và restart; việc này không tự thay thế bản TYAN đang chạy.
