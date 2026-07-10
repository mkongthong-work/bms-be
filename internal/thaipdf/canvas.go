package thaipdf

// canvas — ชั้นวาดพื้นฐานที่เอกสารทุกประเภท (documents, payslip) ใช้ร่วมกัน
// รวมโทนสีแบรนด์, ฟอนต์ฝัง, และ helper วาดข้อความ/รูปทรง

import (
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/signintech/gopdf"
)

// ฟอนต์ Sarabun (SIL Open Font License) ฝังลงไบนารี — deploy ไฟล์เดียวจบ
var (
	//go:embed fonts/Sarabun-Regular.ttf
	fontRegular []byte
	//go:embed fonts/Sarabun-Bold.ttf
	fontBold []byte
	//go:embed fonts/Sarabun-SemiBold.ttf
	fontSemi []byte
)

// โทนสีตามแบรนด์ B.M. Service (อ้างอิง wireframe ที่อนุมัติ)
type rgb struct{ r, g, b uint8 }

var (
	cNavy   = rgb{43, 77, 153}   // #2b4d99 หัวข้อ/กรอบหลัก
	cRed    = rgb{200, 46, 38}   // แดงโลโก้ — เส้น accent
	cInk    = rgb{26, 26, 26}    // ตัวหนังสือหลัก
	cMuted  = rgb{119, 114, 109} // ตัวหนังสือรอง
	cLine   = rgb{214, 219, 232} // เส้นตาราง (ฟ้าอ่อน)
	cZebra  = rgb{247, 245, 242} // แถวสลับ (ครีม #f7f5f2)
	cHeadBG = cNavy              // พื้นหัวตาราง
)

type canvas struct{ p *gopdf.GoPdf }

// ---------- helpers วาด/ข้อความ ----------

func (cv *canvas) setFont(name string, size float64, c rgb) {
	cv.p.SetFont(name, "", size)
	cv.p.SetTextColor(c.r, c.g, c.b)
}

func (cv *canvas) textAt(x, y float64, s string) {
	cv.p.SetXY(x, y)
	cv.p.Cell(nil, s)
}

func (cv *canvas) textRight(xRight, y float64, s string) {
	w, _ := cv.p.MeasureTextWidth(s)
	cv.textAt(xRight-w, y, s)
}

// cellText วาดข้อความในคอลัมน์ align L/C/R พร้อม padding
func (cv *canvas) cellText(x, y, w float64, s, align string, pad float64) {
	tw, _ := cv.p.MeasureTextWidth(s)
	switch align {
	case "R":
		cv.textAt(x+w-pad-tw, y, s)
	case "C":
		cv.textAt(x+(w-tw)/2, y, s)
	default:
		cv.textAt(x+pad, y, s)
	}
}

func (cv *canvas) fillRect(x, y, w, h float64, c rgb) {
	cv.p.SetFillColor(c.r, c.g, c.b)
	cv.p.RectFromUpperLeftWithStyle(x, y, w, h, "F")
}

func (cv *canvas) strokeRect(x, y, w, h float64, c rgb) {
	cv.p.SetStrokeColor(c.r, c.g, c.b)
	cv.p.SetLineWidth(0.7)
	cv.p.RectFromUpperLeftWithStyle(x, y, w, h, "D")
}

func (cv *canvas) hline(x, y, w float64, c rgb) {
	cv.p.SetStrokeColor(c.r, c.g, c.b)
	cv.p.SetLineWidth(0.7)
	cv.p.Line(x, y, x+w, y)
}

func (cv *canvas) wrap(s string, w float64) []string {
	// ตัดที่ช่องว่างก่อน (คำอังกฤษ/รหัสไม่ขาดกลางคำ) แล้วค่อย fallback ตัดตามตัวอักษร
	lines, err := cv.p.SplitTextWithWordWrap(s, w)
	if err != nil {
		lines, err = cv.p.SplitText(s, w)
	}
	if err != nil || len(lines) == 0 {
		return []string{s}
	}
	return lines
}

// ---------- format ----------

var thaiMonths = []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.",
	"ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}

// thaiDate  9 ก.ค. 2569 (พ.ศ.)
func thaiDate(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%d %s %d", d, thaiMonths[int(m)-1], y+543)
}

// round2 ปัดทศนิยม 2 ตำแหน่ง (round-half-up) สำหรับยอดเงิน
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// money จัดรูปแบบ 1,234,567.89
func money(v float64) string {
	neg := v < 0
	v = math.Abs(math.Round(v*100) / 100)
	s := strconv.FormatFloat(v, 'f', 2, 64)
	intPart, frac := s[:len(s)-3], s[len(s)-3:]
	var out []byte
	for i, c := range []byte(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out) + frac
	}
	return string(out) + frac
}

// trimQty ตัด .00 เมื่อเป็นจำนวนเต็ม (2 ไม่ใช่ 2.00)
func trimQty(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
