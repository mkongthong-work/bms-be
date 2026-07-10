// Package config โหลดค่าตั้งจาก environment variables เท่านั้น
// ห้าม hardcode secret ในโค้ด — JWT_SECRET ต้องถูกตั้งเสมอ ไม่มีค่า default
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr        string        // BMS_ADDR       เช่น :8080
	DatabaseURL string        // BMS_DATABASE_URL postgres://...
	JWTSecret   []byte        // BMS_JWT_SECRET  ต้องยาว >= 32 ตัวอักษร
	JWTTTL      time.Duration // อายุ token
	CORSOrigin  string        // BMS_CORS_ORIGIN เช่น https://admin.yourdomain.com (ว่าง = ปิด CORS)
}

func Load() (*Config, error) {
	sec := os.Getenv("BMS_JWT_SECRET")
	if len(sec) < 32 {
		return nil, fmt.Errorf("config: BMS_JWT_SECRET must be set and >= 32 chars")
	}
	db := os.Getenv("BMS_DATABASE_URL")
	if db == "" {
		return nil, fmt.Errorf("config: BMS_DATABASE_URL is required")
	}
	addr := os.Getenv("BMS_ADDR")
	if addr == "" {
		// แพลตฟอร์มอย่าง Render กำหนดพอร์ตผ่านตัวแปร PORT
		if p := os.Getenv("PORT"); p != "" {
			addr = ":" + p
		} else {
			addr = ":8080"
		}
	}
	return &Config{Addr: addr, DatabaseURL: db, JWTSecret: []byte(sec), JWTTTL: 8 * time.Hour,
		CORSOrigin: os.Getenv("BMS_CORS_ORIGIN")}, nil
}
