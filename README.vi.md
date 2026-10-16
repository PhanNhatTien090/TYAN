<div align="center">
<img src="web/public/tyan-mark.svg" width="80" alt="Biểu tượng TYAN" />

# TYAN

Bảng điều khiển nghiên cứu bảo mật sử dụng AI

[English](README.md) · [Tiếng Việt](README.vi.md) · [Tài liệu gốc tiếng Trung](README.zh-CN.md)
</div>

TYAN là phiên bản tùy chỉnh từ [ARTEX](https://github.com/mhtsec/ARTEX), có giao diện hiện đại mang phong cách terminal, nền tối với điểm nhấn xanh mint. Hệ thống dùng backend Go, frontend Next.js và PostgreSQL.

Thông tin tác giả và những người đóng góp cho ARTEX được giữ lại tại [LICENSE](LICENSE), [NOTICE](NOTICE) và [README gốc](README.zh-CN.md).

> **Chỉ dành cho học tập và nghiên cứu bảo mật trong môi trường cục bộ cô lập. Nghiêm cấm sử dụng TYAN để vi phạm pháp luật, truy cập trái phép, đánh cắp dữ liệu, làm gián đoạn dịch vụ hoặc gây hại cho người khác.** Việc cài đặt hay sở hữu công cụ không tạo ra quyền truy cập bất kỳ hệ thống nào. Đồng thời phải tuân thủ điều khoản gốc bên dưới, bao gồm việc không kiểm thử thực tế trên hệ thống trực tuyến hoặc nối mạng, kể cả khi sở hữu hay được cho phép.

## Các chức năng lab bổ sung

Đã bổ sung lịch tác vụ, finding cases, credentials mã hóa, hỏi người vận hành, lớp access gate và extension capture metadata cho lab. Xem [vị trí chức năng, cấu hình và lưu ý nâng cấp](FEATURES.md) trước khi dùng. Giữ nguyên toàn bộ giới hạn sử dụng hiện tại.


## Hình ảnh giao diện

![Dashboard TYAN với dữ liệu mẫu](screenshots/tyan-dashboard.png)

Dashboard tập hợp hoạt động tác vụ, phát hiện, phân bố tài sản, thống kê lưu lượng và mức sử dụng token trong một màn hình.

| Đăng nhập | Quản lý tác vụ |
| --- | --- |
| ![Màn hình đăng nhập TYAN](screenshots/tyan-login.png) | ![Màn hình quản lý tác vụ TYAN với dữ liệu mẫu](screenshots/tyan-tasks.png) |
| Đăng nhập bằng mật khẩu và đọc điều khoản sử dụng. | Theo dõi trạng thái, mục tiêu, bộ lọc và điều khiển thực thi. |

### Quản lý tài sản

Quản lý phạm vi theo tổ chức và xem domain, IP, ứng dụng, dịch vụ, endpoint.

![Quản lý tài sản TYAN với dữ liệu mẫu](screenshots/tyan-assets.png)

### Phát hiện và bằng chứng

Theo dõi mức độ nghiêm trọng, trạng thái xử lý, tài sản bị ảnh hưởng và bằng chứng lưu lượng liên quan.

![Danh sách phát hiện TYAN với dữ liệu mẫu](screenshots/tyan-findings.png)

### Lưu lượng HTTP

Xem các bản ghi HTTP theo host, phương thức, URL, mã trạng thái phản hồi và kích thước.

![Lưu lượng HTTP TYAN với dữ liệu mẫu](screenshots/tyan-traffic.png)

Các ảnh thể hiện giao diện TYAN; dữ liệu dashboard và tác vụ là dữ liệu mẫu có sẵn. Đây không phải bằng chứng đã triển khai backend thật hoặc đã xác minh kết quả đánh giá. Các ảnh không có tiền tố `tyan-` trong `screenshots/` thuộc giao diện ARTEX gốc.

## Chức năng

| Nhóm chức năng | Nội dung |
| --- | --- |
| **Dashboard** | Theo dõi trạng thái tác vụ, hoạt động gần đây, mức độ nghiêm trọng của phát hiện, phân bố tài sản, mã trạng thái HTTP và lượng token LLM. |
| **Tác vụ và mẫu tác vụ** | Khai báo mục tiêu, phạm vi; tạo mẫu và danh mục; tạm dừng/tiếp tục tác vụ và quản lý dữ liệu tác vụ đã lưu trữ. |
| **Không gian agent** | Xem phiên planner/worker, lời gọi công cụ, đồ thị khám phá, độ phủ tài sản và bảng thông tin tác vụ. |
| **Hội thoại** | Trao đổi với agent, tham chiếu phát hiện hoặc tài sản trong tin nhắn và xem lịch sử hội thoại. |
| **Quản lý tài sản** | Quản lý domain, IP, subdomain, ứng dụng, dịch vụ và endpoint; khai báo phạm vi theo tổ chức và đồng bộ nguồn tài sản bên ngoài. |
| **Phát hiện và báo cáo** | Xem mức độ nghiêm trọng, trạng thái xử lý, tài sản liên quan; tạo phiên kiểm tra lại và xuất báo cáo. Kết luận do AI tạo cần được con người xác minh. |
| **Lưu lượng và bằng chứng** | Xem request/response HTTP, lọc bản ghi và liên kết bằng chứng với phát hiện theo vai trò, ghi chú, thứ tự. Tự động liên kết bằng chứng là tùy chọn và có thể tăng chi phí token. |
| **Cấu hình LLM** | Thiết lập endpoint, mô hình, API key, proxy, giới hạn tốc độ, cấu hình dự phòng và cơ chế thử lại/chờ giữa các lần thử. |
| **Agent, skill và công cụ** | Chỉnh prompt và quyền truy cập của agent, quản lý skill/công cụ tùy chỉnh, kết nối máy chủ MCP. Khả năng thực tế phụ thuộc công cụ đã cài và cấu hình môi trường. |
| **Quy tắc phê duyệt** | Thiết lập quy tắc cho phép/chặn, đánh giá bằng mô hình hoặc phê duyệt thủ công; xem ngữ cảnh và kết quả thực thi. Các cơ chế này không bảo đảm việc sử dụng luôn hợp pháp hoặc không có rủi ro. |
| **Thông báo** | Cấu hình DingTalk, Feishu, WeCom, webhook, email; lọc thông báo và xem lịch sử gửi. Cần cấu hình riêng các dịch vụ bên ngoài. |
| **Vận hành và giao diện** | Xem nhật ký backend, quản lý tệp workspace; sử dụng bố cục thích ứng với màn hình và chế độ sáng/tối mang thương hiệu TYAN. |

Giao diện trong tool dùng **tiếng Anh**; tài liệu repo có **tiếng Anh và tiếng Việt**. Nội dung người dùng, phản hồi mô hình và dữ liệu đã lưu giữ nguyên ngôn ngữ gốc.

## Quy trình sử dụng dự kiến

1. Đọc giới hạn sử dụng, chuẩn bị phòng lab cục bộ cô lập với dữ liệu và mục tiêu thuộc phạm vi được phép.
2. Cấu hình database, kết nối LLM và công cụ cục bộ cần thiết. Kiểm tra thông tin nào được gửi đến nhà cung cấp bên ngoài mà bạn lựa chọn.
3. Khai báo phạm vi, mục tiêu, ràng buộc và quyền truy cập của agent/công cụ trước khi chạy.
4. Theo dõi phiên làm việc và yêu cầu phê duyệt; đối chiếu kết luận với bằng chứng, không mặc định câu trả lời của AI là đúng.
5. Rà soát báo cáo xuất ra, bảo vệ dữ liệu nhạy cảm và lưu giữ/xóa dữ liệu phòng lab theo nghĩa vụ áp dụng.

## Cài đặt từ mã nguồn bằng Docker

Cài Docker có hỗ trợ Compose. Trên Windows, dùng Docker Desktop với Linux containers. Chạy các lệnh tại thư mục gốc repo.

1. Sao chép `.env.example` thành `.env`, rồi đặt `POSTGRES_PASSWORD` đủ mạnh:

   ```powershell
   Copy-Item .env.example .env
   ```

   Linux/macOS: `cp .env.example .env`.

2. Build mã nguồn TYAN và khởi động:

   ```sh
   docker compose up -d --build
   ```

3. Mở [http://localhost:8787](http://localhost:8787). Lần đầu, đặt mật khẩu quản trị tại `/setup`, sau đó đăng nhập.
4. Cấu hình nhà cung cấp LLM, mô hình, base URL và API key trong ứng dụng. Bạn cũng có thể khai báo `ANTHROPIC_API_KEY` hoặc `OPENAI_API_KEY` trong `.env` trước khi chạy.

Compose build image TYAN cục bộ từ repo. Image ARTEX dựng sẵn không chứa các thay đổi này.

```sh
docker compose logs -f artex
docker compose down
```

Giữ volume PostgreSQL và thư mục `data/` để giữ dữ liệu. Trước khi nâng cấp, sao lưu cả PostgreSQL và `data/`, bao gồm bằng chứng.

## Phát triển và xem trước frontend

Yêu cầu Node.js 20.9+ và npm. Dependency được cố định bằng lockfile.

```sh
cd web
npm ci
npm run dev
```

Frontend chạy tại [http://localhost:3000](http://localhost:3000), chuyển tiếp `/api` đến `http://localhost:8787`. Đặt `AUTOPENTEST_API` nếu backend ở địa chỉ khác.

Xem thử giao diện với dữ liệu mẫu, không cần backend và database:

```powershell
cd web
$env:NEXT_PUBLIC_MOCK="1"
npm run dev
```

Linux/macOS: chạy `NEXT_PUBLIC_MOCK=1 npm run dev` trong `web/`. Mock chỉ dùng xem trước giao diện. Xóa biến này và khởi động lại trước khi kết nối backend thật.

Kiểm tra frontend:

```sh
cd web
npx tsc --noEmit
npm run build
```

Build bản tĩnh trên Windows:

```powershell
cd web
$env:NEXT_EXPORT="1"
npm run build
```

Linux/macOS: `NEXT_EXPORT=1 npm run build`. Kết quả nằm ở `web/out/`; Docker tự nhúng frontend vào backend khi build.

## Backend và tính tương thích

### Khởi chạy trực tiếp trên Windows

Sau khi build `tyan.exe` có nhúng frontend production và cấu hình PostgreSQL trong `config.json`, chạy:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\start-tyan.ps1
```

Hoặc nhấp đúp `start-tyan.bat`. Mở [http://127.0.0.1:8787](http://127.0.0.1:8787), rồi tự đặt mật khẩu quản trị khi sử dụng lần đầu. Dừng bản cục bộ bằng `powershell -NoProfile -ExecutionPolicy Bypass -File .\stop-tyan.ps1`.

Script chỉ mở TYAN trên máy hiện tại. Nếu có cụm PostgreSQL portable tại `.tyan-runtime/postgres-data`, script sẽ khởi động cả cụm đó. Nhật ký nằm trong `data/`; thông tin kết nối được lưu tại `config.json` đã được Git bỏ qua. Giữ riêng tư `.tyan-runtime/`, `data/`, `config.json` và sao lưu database trước khi thay thế hoặc xóa runtime. Các tệp cài đặt cục bộ này không được đưa lên repo.

### Phát triển và định danh kỹ thuật

Nếu phát triển backend trực tiếp, cài phiên bản Go khai báo trong [go.mod](go.mod) và PostgreSQL. Sao chép `config.example.json` thành `config.json`, cấu hình database, rồi chạy `go run ./cmd/artex` tại thư mục gốc. Backend mặc định dùng cổng `8787`. Chạy backend trực tiếp không tự nhúng frontend mới; dùng Next.js dev server hoặc build Docker như trên.

TYAN giữ nguyên đường dẫn module Go, `cmd/artex`, dịch vụ Compose `artex`, thông số database mặc định và biến môi trường `ARTEX_*` để tương thích. Định danh tài khoản cố định ở backend vẫn là `ARTEX`; biểu mẫu đăng nhập tự gửi giá trị này. Thương hiệu hiển thị là TYAN.

Để giữ bản tùy chỉnh, cập nhật mã nguồn và build lại. Tự cập nhật binary từ upstream có thể ghi đè chương trình và giao diện bằng bản ARTEX gốc.

## Cấu trúc dự án

| Đường dẫn | Nội dung |
| --- | --- |
| `web/` | Frontend Next.js và component dùng chung |
| `server/`, `cmd/artex/` | HTTP API, xác thực và điểm khởi chạy |
| `agent/`, `llmpool/`, `skills/` | Agent, mô hình và skill có sẵn |
| `db/`, `evidence/`, `traffic/` | Dữ liệu, bằng chứng và lưu lượng |
| `config.example.json`, `.env.example` | Cấu hình mẫu |

## Sử dụng có trách nhiệm và các hành vi bị nghiêm cấm

**Không được sử dụng TYAN cho hành vi vi phạm pháp luật hoặc xâm phạm quyền, quyền riêng tư hay an toàn của cá nhân, tổ chức.** Các hành vi sử dụng sai mục đích bị nghiêm cấm bao gồm:

- Truy cập trái phép, vượt qua cơ chế kiểm soát truy cập, chiếm đoạt tài khoản hoặc thu thập thông tin đăng nhập khi không được cho phép.
- Đánh cắp, tiết lộ, sửa đổi hoặc phá hủy dữ liệu cá nhân, dữ liệu bí mật hay dữ liệu thuộc quyền sở hữu của người khác.
- Lừa đảo, phishing, tống tiền, phát tán mã độc hoặc thiết lập cơ chế duy trì truy cập nhằm mục đích gây hại.
- Tấn công từ chối dịch vụ, gây gián đoạn hoạt động, phá hoại hệ thống hoặc thực hiện hành vi có tính hủy hoại.
- Lợi dụng agent, công cụ, proxy hay dịch vụ bên ngoài để che giấu hoặc hỗ trợ hành vi trái pháp luật.
- Xóa bỏ, giả mạo thông báo bản quyền, giấy phép, ghi nhận tác giả hoặc thông báo an toàn.

**Điều khoản gốc còn chặt chẽ hơn yêu cầu “có sự cho phép”:** chỉ giới hạn ở học tập, nghiên cứu mã nguồn và kiểm chứng kỹ thuật trong môi trường cục bộ cô lập. Điều khoản này không cho phép quét, thăm dò, khai thác, kiểm thử xâm nhập hay tấn công thực tế đối với website, dịch vụ trực tuyến, hệ thống nối mạng hoặc production, kể cả tài sản tự sở hữu hay đã được cho phép kiểm thử. Văn bản cho phép của chủ hệ thống không tự thay thế hoặc gỡ bỏ các giới hạn sử dụng đã nêu.

Người dùng phải tự xem xét điều khoản và pháp luật áp dụng, kiểm soát phạm vi/quyền của từng lần chạy, bảo vệ thông tin đăng nhập và bằng chứng, đồng thời kiểm tra hành động và kết quả của agent. Tính năng công cụ, quyết định phê duyệt, câu trả lời AI hay nội dung README **không phải sự cho phép về mặt pháp lý**. Dừng sử dụng nếu mục đích nằm ngoài phạm vi được phép. Tác giả và người duy trì dự án không cổ súy hoặc cho phép hành vi sử dụng sai mục đích.

## Giấy phép và miễn trừ trách nhiệm

Dự án gốc dùng **GNU AGPL-3.0**. Giữ đầy đủ các thông báo bản quyền và giấy phép, kể cả giấy phép riêng từng thành phần. Xem [LICENSE](LICENSE).

README gốc còn nêu điều khoản bổ sung: chỉ học tập, nghiên cứu mã nguồn và kiểm chứng kỹ thuật trong môi trường cục bộ cô lập; không quét, thăm dò, khai thác, kiểm thử xâm nhập thực tế hoặc tấn công website, dịch vụ trực tuyến, hệ thống nối mạng hay production, kể cả khi sở hữu hoặc được cho phép. Điều khoản và miễn trừ gốc được giữ tại [README.zh-CN.md](README.zh-CN.md) và thông báo đăng nhập trong ứng dụng.

Phần mềm được cung cấp theo hiện trạng (**AS IS**), không kèm bảo đảm. Nội dung AI có thể sai, thiếu hoặc không an toàn để thực thi; cần được con người kiểm tra. Người dùng chịu trách nhiệm về hành động và hậu quả liên quan. Tuyên bố miễn trừ không tạo ra quyền miễn tuân thủ pháp luật hoặc loại bỏ nghĩa vụ mà pháp luật không cho phép loại trừ. Xem thông báo gốc để biết đầy đủ điều khoản; bản giới thiệu tiếng Việt không thay thế các điều khoản đó.
