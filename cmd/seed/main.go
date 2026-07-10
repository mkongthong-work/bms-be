// seed — สร้างข้อมูลบริษัท + ผู้ใช้ ADMIN คนแรก (รันครั้งเดียวหลัง migrate)
// ใช้: BMS_DATABASE_URL=... SEED_EMAIL=admin@x.com SEED_PASSWORD=... go run ./cmd/seed
// รหัสผ่านรับจาก env เท่านั้น — ไม่รับเป็น argument เพื่อไม่ให้ค้างใน shell history
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/mkongthong-work/bms-be/internal/auth"
)

func main() {
	dbURL, email, pass := os.Getenv("BMS_DATABASE_URL"), os.Getenv("SEED_EMAIL"), os.Getenv("SEED_PASSWORD")
	if dbURL == "" || email == "" || len(pass) < 10 {
		log.Fatal("ต้องตั้ง BMS_DATABASE_URL, SEED_EMAIL และ SEED_PASSWORD (>= 10 ตัวอักษร)")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อฐานข้อมูลไม่สำเร็จ: %v", err)
	}
	defer conn.Close(ctx)

	var companyID string
	err = conn.QueryRow(ctx, `
		INSERT INTO companies (name_th, name_en, vat_registered)
		VALUES ('บริษัท บี.เอ็ม. เซอร์วิส จำกัด', 'B.M. SERVICE CO., LTD.', false)
		RETURNING id`).Scan(&companyID)
	if err != nil {
		log.Fatalf("สร้างบริษัทไม่สำเร็จ (อาจ seed ไปแล้ว): %v", err)
	}

	hash, err := auth.HashPassword(pass)
	if err != nil {
		log.Fatal(err)
	}
	_, err = conn.Exec(ctx, `
		INSERT INTO users (company_id, role_id, email, password_hash, display_name)
		SELECT $1, r.id, $2, $3, 'ผู้ดูแลระบบ' FROM roles r WHERE r.code = 'ADMIN'`,
		companyID, email, hash)
	if err != nil {
		log.Fatalf("สร้างผู้ใช้ไม่สำเร็จ: %v", err)
	}
	fmt.Printf("สร้าง ADMIN %s เรียบร้อย — ล็อกอินแล้วเปลี่ยนข้อมูลบริษัทในหน้าตั้งค่า\n", email)
}
