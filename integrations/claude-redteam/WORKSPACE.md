# Research workspace upgrade / Bản nâng cấp workspace

[Source integration mapping](README.md) · [Tiếng Việt](README.vi.md)

This upgrade brings the total to **65 review skills, 14 native agent tools, six conversational review agents and 17 operator workflow adapters**. It is a native review implementation, not full execution parity with the source repository.

## English

Open **Functions → Research Kit → Knowledge & models**. Choose a collection and task ID. Only Knowledge base accepts task `0` for shared operator knowledge. Session memory, system models and API catalogs require an existing task. Click **Load records**, then select a record or create a new one.

Each record has a stable key, title, body, tags and evidence references. Session notes also have a session label. Save creates revision 1; subsequent saves require the version loaded in the editor. A conflicting write or stale delete returns HTTP 409. Revision history shows the latest 100 versions; loading old content into the editor and saving creates a new version without rewriting history. Searches return the latest 200 matching records and use literal case-insensitive text matching, not embeddings, fuzzy search or the source Supabase engine.

**System models** contain up to 500 typed nodes, 1,000 relationships and 100 open questions. Node kinds are `tech`, `role`, `auth`, `flow`, `object`, `feature`, `hypothesis`, `boundary`; statuses are `todo`, `reviewing`, `reviewed`, `supported`, `rejected`. Supported observations require non-empty evidence references, but reference contents are not independently checked. Coverage counts reviewed/supported/rejected nodes; it does not measure security or exploitability. Check coverage to validate the JSON and preview relationships. The visual preview shows the first 24 nodes; the full model is preserved in JSON.

**API catalogs** store the complete structured result from Analyze artifacts → OpenAPI → Copy result JSON. This includes any parser preview truncation, which is retained in the stored summary. It is not a complete endpoint crawler or automatic graph importer. The structure example is an empty catalog, not evidence of endpoint review.

**Session memory** is manually saved task-scoped context. Use session labels, tags and references to relate observations. `/session-start` and `/session-end` in Workflows prepare a Markdown handoff from the stored notes/models/API summaries, with best-effort redaction. They do not automatically capture all conversations, clear notes or stop an agent. Oversized handoffs require a narrower query. This is not the source automatic memory hook.

**Workflows** offers all 17 command-name adaptations. Select a command, read its description and edit the example JSON. Task IDs are quoted strings; sample IDs must be replaced. Only `/new-engagement` and `/update-state` write database records, as labeled by the UI. Other commands read/prepare artifacts. `/new-finding` prepares a draft; save reviewed findings through the regular Findings workflow. `/replay` compares the first 200 saved HAR entries per file, grouped by sanitized method/URL and response status counts; it never resends requests. Query-only differences may collapse. A changed status is not proof of a fix.

Six new editable chat agents appear under **System → Agents**, using keys `rt_coordinator`, `rt_recon`, `rt_recon_ranker`, `rt_appsec_review`, `rt_chain_scout`, `rt_validate`. They use the existing chat runtime and configured LLM, with the research tool/skill bindings, 24-turn defaults, and web search/interactive shell disabled initially. No agent call starts simply by opening Research Kit. Creation is atomic; subsequent startup preserves agent names, prompts and bindings edited by the operator.

The four workspace agent tools are `research_workspace_list`, `research_workspace_get`, `research_workspace_history`, `research_workspace_put`. They infer the task from run context and reject access to a different task. Non-task conversations can access global KB only. The write tool is non-read-only and uses the normal permission pipeline. `research_har_summary` and `research_model_coverage` are additional read-only artifact tools. No source hook grants extra permissions.

Records, revisions and exports are **ordinary unencrypted research content**. Do not put credentials in them. There is no automatic redaction on save or JSON export. Artifacts remain untrusted reference data. Operator APIs use existing authentication; agents remain subject to bindings and task scope. Deleting a record also deletes its revisions; deleting a task cascades to its workspace records. Current schema does not add a separate research archive. Existing database backups include these tables.

Rebuild the frontend and Go binary, back up the existing database, then restart to apply the additive tables and seed roles/tools. No Frida/Flutter/SQLCipher runtime, source probe/scan/flow engine, HTTP replay executor, source Office template engine, or model-control bypass/offload hook is installed by this upgrade.

## Tiếng Việt

- Mở **Knowledge & models**, chọn loại dữ liệu và task ID. Chỉ KB dùng được task `0`; các loại khác cần task đã tồn tại.
- Lưu/sửa/xóa bản ghi, tìm theo nội dung/tag, xem lịch sử và xuất JSON. Mỗi lần lưu là một phiên bản mới; dữ liệu bị sửa đồng thời sẽ báo conflict thay vì ghi đè.
- System model có node, quan hệ, câu hỏi, trạng thái review và bằng chứng tham chiếu. Có đồ thị xem trước tối đa 24 node và thống kê độ phủ. Đây là độ phủ review tự khai báo, không phải tỷ lệ an toàn.
- API catalog lưu JSON kết quả OpenAPI. Dùng **Copy result JSON** ở tab Analyze artifacts rồi dán vào bản ghi API catalog; nếu kết quả bị giới hạn preview thì giới hạn đó vẫn được giữ.
- Ghi chú phiên được lưu theo task, có nhãn phiên và tham chiếu. Hai workflow đầu/cuối phiên tạo bản bàn giao từ dữ liệu đã lưu; không tự thu mọi hội thoại, xóa ghi chú hay dừng agent.
- **Workflows** có 17 adapter chạy với tham số JSON. Thay task ID mẫu bằng ID thực tế. Hai lệnh có nhãn ghi database là `new-engagement` và `update-state`. `new-finding` chỉ chuẩn bị bản nháp; `replay` chỉ so sánh HAR offline, không phát lại request.
- Sáu agent review là agent hội thoại thực tế trong TYAN, dùng LLM bạn đã cấu hình. Prompt, binding và tên có thể sửa, và được giữ nguyên khi restart.
- Bản ghi và lịch sử không được mã hóa/che bí mật tự động. Xóa bản ghi sẽ xóa lịch sử của nó; xóa task sẽ xóa dữ liệu workspace thuộc task. Backup database trước khi nâng cấp.

Đây vẫn chưa phải bản sao toàn bộ khả năng thực thi của repo nguồn. Các runtime mobile, engine scan/probe/flow, phát lại HTTP thật và hook né kiểm soát mô hình không được đưa vào bản này.
