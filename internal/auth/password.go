package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword สร้าง hash สำหรับเก็บใน users.password_hash
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	return string(b), err
}

// CheckPassword เทียบรหัสผ่าน — เวลาเทียบคงที่ ป้องกัน timing attack
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
