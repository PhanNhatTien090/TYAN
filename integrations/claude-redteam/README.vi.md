# Tích hợp claude-redteam vào TYAN

[English và bảng đối chiếu đầy đủ](README.md) · [Hướng dẫn sử dụng](../../docs/RESEARCH-KIT.md)

Nguồn được phân tích là [claude-redteam tại commit 329fc12](https://github.com/PhanNhatTien090/claude-redteam/tree/329fc1273c202844cd041e08e74298d683258158). Giữ nguyên giấy phép MIT trong [LICENSE](LICENSE). Từng skill có đường dẫn và SHA-256 gốc trong [source.json](source.json).

## Kết quả phân tích

Revision này có **65 chủ đề skill, 31 script Python, 6 đặc tả agent và 17 slash command**, kèm thư mục công cụ Frida/Flutter/SQLCipher. Con số 66 skill trên README nguồn chưa khớp với tập tin thực tế. Các con số trên là số lượng tập tin, không phải tổng số công cụ chạy độc lập.

Kiến trúc nguồn dựa vào Claude Code, hook, các tập tin trạng thái và một số giả định Unix. TYAN lại quản lý agent, task, tool và dữ liệu bằng Go/PostgreSQL. Vì vậy, chép nguyên thư mục sẽ không khiến các công cụ hoạt động trong TYAN.

TYAN đã chuyển **đủ 65 chủ đề** thành skill review riêng trong `skills/rt-*/SKILL.md`, có checklist theo từng chủ đề, nguồn truy vết và gợi ý tool phù hợp. Đây là các **bản chuyển thể phục vụ review**, không phải bản sao đầy đủ khả năng thực thi của nguồn. Những tập payload, runtime Frida, tự động khai thác và lớp offload/né bộ lọc mô hình không được đưa vào.

## Có thể dùng ngay trong giao diện

Mở **Functions → Research Kit**:

1. **Skills**: tìm chủ đề, lọc công nghệ, sao chép tên skill để agent tải khi cần.
2. **Analyze artifacts**: phân tích OpenAPI/Swagger JSON hoặc YAML; chuẩn hóa kết quả đã lưu từ Nuclei, ffuf, Nmap; che thông tin nhạy cảm; tính CVSS 4.0.
3. **Finding & report**: nhập bằng chứng, kiểm tra 7 điều kiện sẵn sàng, xem bản Markdown và tải DOCX.
4. **Tool readiness**: xem 19 tên chương trình có trên PATH hay chưa. Đây chỉ là kiểm tra sự hiện diện, không cài đặt hoặc chạy chương trình.

Tám helper native cũng được đăng ký vào tool catalog của agent. Chúng dùng quyền, trạng thái bật/tắt và binding agent hiện có. Thay đổi cấu hình do người dùng thực hiện được giữ lại khi seed lại.

## Phạm vi và giới hạn

- OpenAPI chỉ phân tích nội dung được cung cấp, không truy cập URL hay tải `$ref` bên ngoài. Khai báo authentication trong schema không chứng minh triển khai đúng.
- Kết quả scanner là quan sát chưa xác minh. Không tự biến thành lỗ hổng hay nâng severity của ffuf/Nmap.
- CVSS tính từ vector do người review cung cấp, không tự suy diễn tác động.
- Checklist gồm phạm vi, nhận diện vấn đề, tái hiện, đối chiếu kỳ vọng/thực tế, bằng chứng, tác động/điều kiện tiên quyết và khắc phục. Ô xác nhận phạm vi là tự khai báo; đủ checklist không chứng minh lỗ hổng có thật.
- Redaction là best-effort, cần tự kiểm tra trước khi chia sẻ. API không tạo kho lưu artifact mới; dữ liệu gửi qua agent vẫn có thể nằm trong hội thoại/tool history theo cơ chế TYAN hiện có.
- DOCX là bản nháp OOXML cơ bản, chưa phải bộ template Office đầy đủ của nguồn.
- Sáu agent và 17 lệnh được đối chiếu với workflow TYAN, không được quảng cáo là sáu agent mới hay 17 lệnh thực thi mới.

[Bảng đối chiếu tiếng Anh](README.md#script-by-script-mapping) ghi rõ quyết định cho toàn bộ 31 script, sáu agent và 17 command. Các engine KB, system-model graph, replay, chain và mobile instrumentation chưa được port. Task, finding, session và history tái sử dụng dữ liệu TYAN, không có hai kho trạng thái song song.

Cần build lại frontend và binary Go rồi khởi động lại để chạy phiên bản mới. Pull source không cập nhật binary đang chạy. Các giới hạn sử dụng trong LICENSE/README chính vẫn áp dụng.
