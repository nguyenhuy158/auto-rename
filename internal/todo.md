# Kế hoạch tách code theo kiến trúc Clean Architecture

- [x] Phân tích code hiện tại trong main.go
- [ ] Lập kế hoạch tách các layer (domain, usecase, infrastructure, delivery)
- [ ] Tạo các file và thư mục cho từng layer
- [ ] Di chuyển code vào các layer phù hợp
- [ ] Refactor lại các phần phụ thuộc giữa các layer
- [ ] Viết lại hàm main để khởi tạo các layer
- [ ] Kiểm tra lại hoạt động của chương trình

## Đề xuất cấu trúc thư mục:
- /cmd/auto-rename/main.go (entrypoint)
- /internal/domain/file.go (FileRecord, các entity)
- /internal/usecase/rename.go (renameFiles, startCronScanner, business logic)
- /internal/infrastructure/database.go (Database, SQLite logic)
- /internal/infrastructure/filesystem.go (file system helpers)
- /internal/delivery/webserver.go (WebServer)
- /internal/config/config.go (Config, parseFlags, validateConfig)

---

# TODO: Phân trang cho records

- [ ] Phân tích yêu cầu phân trang cho records
- [ ] Sửa backend để trả về dữ liệu records theo từng trang (webserver.go, database.go)
- [ ] Sửa template/records.html để hiển thị nút chuyển trang và dữ liệu từng trang
- [ ] Xử lý các trường hợp đặc biệt (ví dụ: trang cuối, trang đầu, không có dữ liệu)
- [ ] Kiểm tra và test tính năng phân trang
- [ ] Xác nhận kết quả
