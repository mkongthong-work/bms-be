// Package bahttext แปลงจำนวนเงิน (ตัวเลข) เป็นคำอ่านภาษาไทย
// เช่น 25750.50 -> "สองหมื่นห้าพันเจ็ดร้อยห้าสิบบาทห้าสิบสตางค์"
//
// การคำนวณภายในใช้หน่วยสตางค์ (int64) ทั้งหมด เพื่อเลี่ยงปัญหา
// ความคลาดเคลื่อนของ floating point ในเอกสารการเงิน
package bahttext

import (
	"errors"
	"math"
	"strings"
)

var (
	digitWords = []string{"", "หนึ่ง", "สอง", "สาม", "สี่", "ห้า", "หก", "เจ็ด", "แปด", "เก้า"}
	placeWords = []string{"", "สิบ", "ร้อย", "พัน", "หมื่น", "แสน"}
)

// ErrOutOfRange จำนวนเงินเกินช่วงที่รองรับ (เกิน ~92 ล้านล้านบาท หรือเป็น NaN/Inf)
var ErrOutOfRange = errors.New("bahttext: amount out of supported range")

// FromSatang แปลงจำนวนเงินหน่วยสตางค์เป็นคำอ่านไทย
// รองรับค่าติดลบ (ขึ้นต้นด้วย "ลบ") — ใช้กับใบลดหนี้ได้
func FromSatang(satang int64) string {
	var sb strings.Builder
	if satang < 0 {
		sb.WriteString("ลบ")
		satang = -satang
	}

	baht := satang / 100
	st := satang % 100

	if baht == 0 && st == 0 {
		return "ศูนย์บาทถ้วน"
	}
	if baht > 0 || st == 0 {
		sb.WriteString(readNumber(baht))
		sb.WriteString("บาท")
	}
	if st == 0 {
		sb.WriteString("ถ้วน")
	} else {
		sb.WriteString(readNumber(st))
		sb.WriteString("สตางค์")
	}
	return sb.String()
}

// FromFloat แปลงจำนวนเงินหน่วยบาท (float64) เป็นคำอ่านไทย
// ปัดเศษสตางค์แบบ round-half-up ตามแนวปฏิบัติเอกสารการเงินไทย
func FromFloat(amount float64) (string, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return "", ErrOutOfRange
	}
	satang := math.Round(amount * 100)
	if satang > math.MaxInt64 || satang < math.MinInt64 {
		return "", ErrOutOfRange
	}
	return FromSatang(int64(satang)), nil
}

// readNumber อ่านจำนวนเต็มบวกเป็นคำไทย โดยแบ่งเป็นกลุ่มละ 6 หลัก
// คั่นด้วย "ล้าน" (เช่น 1,000,000 -> หนึ่งล้าน, 10^12 -> หนึ่งล้านล้าน)
func readNumber(n int64) string {
	if n == 0 {
		return ""
	}
	million := n / 1_000_000
	rest := n % 1_000_000

	var sb strings.Builder
	if million > 0 {
		sb.WriteString(readNumber(million))
		sb.WriteString("ล้าน")
	}
	sb.WriteString(readBelowMillion(rest, million > 0))
	return sb.String()
}

// readBelowMillion อ่านเลข 0–999,999 · afterMillion ใช้ตัดสินกรณี "เอ็ด"
// หลักหน่วย = 1: อ่าน "เอ็ด" เมื่อมีหลักหน้ากว่า (11 -> สิบเอ็ด, 1,000,001 -> หนึ่งล้านเอ็ด)
func readBelowMillion(n int64, afterMillion bool) string {
	var sb strings.Builder
	hasHigher := afterMillion
	for place := 5; place >= 0; place-- {
		div := int64(math.Pow10(place))
		d := (n / div) % 10
		if d == 0 {
			continue
		}
		switch {
		case place == 0 && d == 1 && hasHigher:
			sb.WriteString("เอ็ด")
		case place == 1 && d == 1: // 10 -> "สิบ" ไม่ใช่ "หนึ่งสิบ"
			sb.WriteString("สิบ")
		case place == 1 && d == 2: // 20 -> "ยี่สิบ"
			sb.WriteString("ยี่สิบ")
		default:
			sb.WriteString(digitWords[d])
			sb.WriteString(placeWords[place])
		}
		hasHigher = true
	}
	return sb.String()
}
