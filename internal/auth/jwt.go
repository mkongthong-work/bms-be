package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims ที่ฝังใน access token — เก็บเท่าที่จำเป็น ไม่ใส่ข้อมูลอ่อนไหว
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"` // ADMIN / SALES / ACCOUNT / HR
	jwt.RegisteredClaims
}

// Issue ออก token อายุตาม ttl
func Issue(secret []byte, ttl time.Duration, userID, role string) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID, Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "bms-admin",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	})
	return t.SignedString(secret)
}

// Verify ตรวจ token — บังคับ HS256 เท่านั้น กัน algorithm confusion
func Verify(secret []byte, tokenStr string) (*Claims, error) {
	var claims Claims
	t, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return &claims, nil
}
