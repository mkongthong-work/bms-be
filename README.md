# BMS Admin — ระบบหลังบ้าน B.M. Service

Backend (Go 1.22) — คู่กับ frontend ที่ https://github.com/mkongthong-work/bms-fe

## เริ่มใช้งาน

### 1) ฐานข้อมูล
```bash
createdb bms && psql bms < migrations/001_init.sql
```

### 2) Backend
```bash
cd backend
cp .env.example .env        # เติม BMS_DATABASE_URL และ BMS_JWT_SECRET (openssl rand -base64 48)
go run ./cmd/api            # ฟังที่ :8080
go test ./...               # รันเทสต์ (bahttext ครบเคสอ่านเลขไทย)
go run ./cmd/samples        # สร้างไฟล์ตัวอย่าง xlsx + payslip pdf
```
> หมายเหตุ: บรรทัด `replace golang.org/x/... => github.com/golang/...` ใน go.mod
> ใส่ไว้เพื่อ build ในสภาพแวดล้อมที่จำกัดเครือข่าย — ถ้าเครื่องคุณใช้ Go module proxy
> ปกติ ลบ replace ทั้งหมดแล้ว `go mod tidy` ได้เลย


## Endpoint หลัก (v1)
| Method | Path | สิทธิ์ | หน้าที่ |
|---|---|---|---|
| POST | /api/v1/auth/login | - | รับ access token (JWT HS256, 8 ชม.) |
| GET | /api/v1/products?q=&page= | ล็อกอิน | ค้นหาสินค้า แบ่งหน้า |
| POST | /api/v1/products | SALES | เพิ่มสินค้า |
| GET | /api/v1/documents/{id}/pdf | SALES | ใบเสนอราคา/ใบแจ้งหนี้/ใบเสร็จ เป็น PDF ไทย |
| GET | /api/v1/reports/sales.xlsx?from=&to= | ACCOUNT | รายงานยอดขาย Excel (สูตร SUM จริง) |

ADMIN ผ่านทุก route โดยอัตโนมัติ (ดู internal/auth/middleware.go)

## จุดออกแบบด้านความปลอดภัย
- secret จาก env เท่านั้น, JWT บังคับ HS256, bcrypt cost 12
- login ตอบข้อความเดียวทุกกรณี (ไม่เผยว่ามีอีเมลในระบบ), จำกัดขนาด body ทุก endpoint
- SQL parameterized ทั้งหมด, security headers ทุก response, token ฝั่งเว็บอยู่ใน sessionStorage
- เลขเอกสารจองใน transaction ผ่าน next_document_number() — ไม่ชน/ไม่ข้ามเลข

## โครงถัดไป (เฟส 3-4)
สร้าง/แปลงเอกสารครบวงจร (repo.CreateDocument พร้อมแล้ว), โมดูลพนักงาน + payroll
(engine สลิป internal/thaipdf/payslip.go พร้อมแล้ว), Dashboard
