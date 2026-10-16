# TYAN feature integration / Tính năng mới

This edition selectively integrates application features from the supplied ARTEX-en archive while retaining TYAN branding, its bundled Norma source, existing Docker source builds, Windows launchers, and original usage restrictions. See NOTICE for provenance.

## English

### Research Kit

Functions → Research Kit adds 65 review skill adaptations and 14 native tools, including offline artifact analysis, CVSS 4.0, finding readiness, versioned KB/session notes/API catalogs/system models, six review chat agents, 17 operator workflow adapters and Markdown/basic DOCX drafts. See the [bilingual user guide](docs/RESEARCH-KIT.md) and [complete source-to-TYAN mapping](integrations/claude-redteam/README.md). Tool readiness detects 19 optional executable names on PATH; it does not install or run them. This is a native review adaptation rather than full upstream runtime parity.

### Where to find the new features

| Feature | Location | Behavior |
| --- | --- | --- |
| Task schedules | Functions → Schedules | Create one-time or weekly windows; edit, enable, pause, run now, and delete schedules. New schedules created here are paused. Task IDs refer to existing local lab tasks. Times use the server timezone or explicit UTC+08:00. |
| Finding cases | Functions → Finding Cases | Browse grouped cases and standalone findings. Merge at least two findings from the same task with an evidence-based root-cause explanation. Reporter tools support unified reports and historical review. |
| Test-account credentials | Task detail → Credentials | Store dedicated lab accounts, reveal/copy passwords as the operator, and remove entries. Passwords are encrypted at rest and omitted from the list/overview tool results sent to agents. |
| Operator handoff | Worker conversation | The `ask_operator` tool can pause a worker and ask a precise question, optionally with a screenshot; answer using the existing message-to-worker flow. Do not paste passwords or personal OTPs into chat. |
| Additional access gate | System → Access Gate | Optional HTTP Basic Auth before normal operator login. It starts disabled. Use HTTPS for any remote connection. |
| Browser capture | Optional unpacked extension in `extension/` | Collect metadata from one configured local lab origin into TYAN's traffic store. See extension/README.md for limits and setup. |
| Script-aware interception | Existing Interception Rules and approval views | Additional shell/Python script analysis and review context. Existing TYAN guard and default rule seed are retained. This is not a complete sandbox or egress firewall. |

### Credential encryption setup

Set `TYAN_CREDENTIAL_KEY` to a **base64-encoded random 32-byte key** before storing a non-empty task password. Without a valid key, password writes fail; TYAN does not silently save plaintext. Keep the same key on restart and retain a separate secure backup. Losing or replacing it makes existing passwords unreadable.

PowerShell, generate once in your local shell:

```powershell
$env:TYAN_CREDENTIAL_KEY = python -c "import secrets,base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
.\start-tyan.ps1
```

Save that key securely before closing the shell and restore it as an environment variable on subsequent launches. Do not regenerate it at every startup. Do not commit it or include it beside database dumps.

For Docker, put the generated value in the ignored `.env` file as `TYAN_CREDENTIAL_KEY=...`; Compose forwards it to the application. Do not commit `.env`.

Only the password field is encrypted; account names, domains and notes remain ordinary database fields. The authenticated operator API can return decrypted passwords for the Credentials tab. Agent list/overview results omit passwords and notes, but tools, screenshots, captured traffic, and anything manually typed into chat can still disclose secrets. This feature is for disposable lab accounts, not a general password manager. Legacy plaintext task passwords are rejected and must be removed and re-entered.

### Local backups

On Linux/macOS or a suitable Bash environment with Docker Compose, `scripts/backup.sh` writes private-permission database dumps; `--with-data` also archives evidence. Outputs in `backups/` are ignored by Git. They are not encrypted: protect them and keep `TYAN_CREDENTIAL_KEY` in a separate secure backup.

### Before updating an existing installation

Back up PostgreSQL and the application's `data/` directory first, keeping secrets protected. The application applies additional tables/columns at startup. Test against a separate database before choosing to migrate an existing instance. Rebuild both the frontend and embedded Go binary; pulling source alone does not update an already-running executable. No LLM service or provider credit is included.

The imported archive's DeepSeek GitHub bot, external Norma replacement, machine-specific deployment notes, Windows shortcut and EDR-bypass skill are not included in this application integration. No automatic third-party marketing or issue replies are enabled.

## Tiếng Việt

- **Schedules:** tạo lịch một lần hoặc hằng tuần, sửa, bật/tạm dừng, chạy ngay và xóa lịch. Lịch mới trên giao diện mặc định tạm dừng. Nhập ID tác vụ lab đã có; chọn múi giờ máy chủ hoặc UTC+08:00.
- **Finding Cases:** xem case và findings độc lập; gộp ít nhất hai findings cùng tác vụ khi đã có bằng chứng chung nguyên nhân gốc rễ. Các công cụ reporter hỗ trợ báo cáo hợp nhất và review lịch sử.
- **Credentials:** quản lý tài khoản lab trong trang chi tiết tác vụ. Mật khẩu được mã hóa khi lưu; kết quả list/overview gửi cho agent không chứa mật khẩu hoặc ghi chú.
- **Operator handoff:** worker có thể đặt câu hỏi kèm ảnh và tạm dừng để người vận hành phản hồi. Không nhập mật khẩu hoặc OTP cá nhân vào chat.
- **Access Gate:** thêm lớp HTTP Basic Auth trước đăng nhập TYAN, mặc định tắt.
- **Extension:** capture metadata của một origin lab đã chỉ định, chỉ gửi về server loopback. Body bị bỏ, giá trị query bị che và header được giới hạn. Đường dẫn URL vẫn có thể chứa dữ liệu nhạy cảm.
- **Interception:** thêm phân tích script shell/Python và ngữ cảnh review; giữ lớp guard và seed luật bảo vệ TYAN hiện tại.

Để lưu mật khẩu, cấu hình `TYAN_CREDENTIAL_KEY` là khóa ngẫu nhiên 32 byte mã hóa base64 theo hướng dẫn phía trên. **Giữ nguyên khóa qua các lần khởi động**, cất riêng khỏi database/backup và không commit lên Git. Khi thiếu khóa, hệ thống từ chối ghi mật khẩu thay vì lưu dạng rõ. Tên tài khoản, domain và ghi chú không được mã hóa. Chỉ dùng tài khoản lab riêng.

Trước nâng cấp, backup PostgreSQL và `data/`, thử migration trên database riêng và build lại cả frontend lẫn binary Go. Bản đang chạy không tự thay đổi chỉ vì source đã cập nhật. Toàn bộ giới hạn sử dụng lab cục bộ biệt lập của TYAN/ARTEX vẫn áp dụng.
