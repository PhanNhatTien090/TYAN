# TYAN local lab capture

Optional Chrome/Edge Manifest V3 extension. Load this directory unpacked in a separate lab browser profile through the browser's extension developer mode. It requires the debugger permission; do not install it in your personal browsing profile.

1. Start your TYAN instance on a loopback address, e.g. `http://127.0.0.1:8787`.
2. Sign in locally and obtain your operator bearer token from that instance (the UI stores it under `artex_token`). Treat it as an administrator credential; never post it in issues or chat.
3. In the extension popup, enter the loopback TYAN URL, your token, and the exact origin of a dedicated local lab target, e.g. `http://127.0.0.1:3000`.
4. Save settings, open the lab tab, and select **Start capture**. Use **Flush now** for a batch smaller than ten entries; **Stop** flushes pending entries and detaches the debugger.
5. Review records in TYAN → Traffic. Stop capture before leaving the lab. The token lasts only for the browser session.

Only the configured origin is captured. Changing the tab to another origin stops capture. Targets are limited to loopback, private IPv4, `.test` or `.localhost` names; these names alone do not prove isolation. You must maintain an isolated lab and obey the repository's full usage restrictions.

Request/response bodies are omitted. Query values and URL credentials/fragments are removed; only content-type, content-length, accept and cache-control headers are retained. This intentionally differs from the original archive's full-content capture. URL paths and remaining metadata can still contain sensitive information. Use fake lab data. Captured traffic can be read by the configured agent/LLM through normal traffic tools.

The destination must be loopback and redirects are refused. Metadata is sent solely to the configured local TYAN ingest endpoint. This is not a guarantee against data exposure from other tools, a compromised browser, local malware, or later agent use. Optional Basic Auth may prevent extension ingest; leave the gate off for this local capture workflow rather than weakening it.

## Tiếng Việt

Nạp thư mục này dưới dạng extension unpacked trong Chrome/Edge profile dành riêng cho lab. Nhập URL TYAN loopback, token quản trị của phiên đăng nhập cục bộ và origin lab chính xác. Lưu cấu hình, mở tab lab rồi bấm Start capture; bấm Flush now hoặc Stop để gửi các bản ghi đang chờ.

Extension chỉ lưu metadata: không lấy body, che giá trị query và lọc header. Token chỉ giữ trong phiên trình duyệt. Đường dẫn URL vẫn có thể nhạy cảm; chỉ dùng dữ liệu giả trong lab biệt lập. Không gửi token lên GitHub hay vào chat. Xem giới hạn đầy đủ trong README chính.
