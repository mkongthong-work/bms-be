package thaipdf

// payslip.go — สลิปเงินเดือน (Payslip) ใช้ canvas ร่วมกับเอกสารงานขาย
// หนึ่งหน้า A4 = สลิป 2 ใบเหมือนกัน (ต้นฉบับให้พนักงาน / สำเนาแนบ HR)

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mkongthong-work/bms-be/internal/thaipdf/bahttext"
	"github.com/signintech/gopdf"
)

// PayslipLine หนึ่งบรรทัดรายได้/รายการหัก บนสลิป
type PayslipLine struct {
	Label  string
	Amount float64
}

// Payslip ข้อมูลสลิปหนึ่งคนหนึ่งงวด (map จาก payroll_items + employees)
type Payslip struct {
	Company      Company
	PeriodStart  time.Time
	PeriodEnd    time.Time
	EmployeeCode string
	FullName     string
	Position     string
	WageType     string // "daily" | "monthly"
	WorkDays     float64
	OTHours      float64
	Earnings     []PayslipLine // เงินเดือน/ค่าแรง, OT, เบี้ยขยัน ฯลฯ
	Deductions   []PayslipLine // ประกันสังคม, เบิกล่วงหน้า ฯลฯ
	BankAccount  string        // แสดงแบบ mask แล้วจากชั้น service
}

// TotalEarnings / TotalDeductions / NetPay ยอดรวมคำนวณจากบรรทัด
func (ps *Payslip) TotalEarnings() float64 {
	var s float64
	for _, l := range ps.Earnings {
		s += l.Amount
	}
	return s
}
func (ps *Payslip) TotalDeductions() float64 {
	var s float64
	for _, l := range ps.Deductions {
		s += l.Amount
	}
	return s
}
func (ps *Payslip) NetPay() float64 { return ps.TotalEarnings() - ps.TotalDeductions() }

func (ps *Payslip) validate() error {
	switch {
	case ps.FullName == "":
		return errors.New("thaipdf: payslip employee name is required")
	case len(ps.Earnings) == 0:
		return errors.New("thaipdf: payslip needs at least one earning line")
	case ps.NetPay() < 0:
		return errors.New("thaipdf: payslip net pay is negative — ตรวจรายการหัก")
	}
	return nil
}

// RenderPayslip สร้าง PDF สลิปเงินเดือน (2 สำเนาต่อหน้า)
func RenderPayslip(ps *Payslip) ([]byte, error) {
	if err := ps.validate(); err != nil {
		return nil, err
	}
	p := &gopdf.GoPdf{}
	p.Start(gopdf.Config{PageSize: gopdf.Rect{W: pageW, H: pageH}})
	for name, data := range map[string][]byte{
		"sarabun": fontRegular, "sarabun-b": fontBold, "sarabun-sb": fontSemi,
	} {
		if err := p.AddTTFFontData(name, data); err != nil {
			return nil, fmt.Errorf("thaipdf: load font %s: %w", name, err)
		}
	}
	p.AddPage()
	cv := &canvas{p: p}

	half := pageH / 2
	drawSlip(cv, ps, margin, "ต้นฉบับ — พนักงาน")
	// เส้นประรอยตัดกลางหน้า
	p.SetStrokeColor(cMuted.r, cMuted.g, cMuted.b)
	p.SetLineWidth(0.5)
	for x := margin; x < pageW-margin; x += 12 {
		p.Line(x, half, x+6, half)
	}
	drawSlip(cv, ps, half+14, "สำเนา — ฝ่ายบุคคล")

	var buf bytes.Buffer
	if _, err := p.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawSlip วาดสลิปหนึ่งใบเริ่มที่ตำแหน่ง y
func drawSlip(cv *canvas, ps *Payslip, top float64, copyLabel string) {
	// หัว: ชื่อบริษัท + ชื่อเอกสาร
	cv.setFont("sarabun-b", 13, cNavy)
	cv.textAt(margin, top, ps.Company.NameEN)
	cv.fillRect(margin, top+17, 90, 2.5, cRed)
	cv.setFont("sarabun", 8, cMuted)
	cv.textAt(margin, top+24, ps.Company.NameTH)

	cv.setFont("sarabun-b", 13, cNavy)
	cv.textRight(pageW-margin, top, "สลิปเงินเดือน")
	cv.setFont("sarabun-sb", 8, cRed)
	cv.textRight(pageW-margin, top+16, "PAYSLIP · "+copyLabel)
	cv.setFont("sarabun", 8.5, cMuted)
	cv.textRight(pageW-margin, top+30,
		"งวด "+thaiDate(ps.PeriodStart)+" – "+thaiDate(ps.PeriodEnd))

	// แถบข้อมูลพนักงาน
	iy := top + 46
	cv.fillRect(margin, iy, contentW, 34, rgb{233, 238, 248})
	wage := "รายเดือน"
	if ps.WageType == "daily" {
		wage = "รายวัน"
	}
	cv.setFont("sarabun-sb", 9.5, cInk)
	cv.textAt(margin+10, iy+5, ps.FullName+"  ·  "+ps.Position)
	cv.setFont("sarabun", 8.5, cMuted)
	cv.textAt(margin+10, iy+19, fmt.Sprintf("รหัส %s   ·   ประเภท%s   ·   ทำงาน %s วัน   ·   OT %s ชม.",
		ps.EmployeeCode, wage, trimQty(ps.WorkDays), trimQty(ps.OTHours)))
	if ps.BankAccount != "" {
		cv.setFont("sarabun", 8.5, cMuted)
		cv.textRight(pageW-margin-10, iy+19, "โอนเข้าบัญชี "+ps.BankAccount)
	}

	// สองคอลัมน์: รายได้ | รายการหัก
	colGap, colWd := 16.0, (contentW-16)/2
	ty := iy + 44
	drawPayCol(cv, margin, ty, colWd, "รายได้ / Earnings", ps.Earnings, ps.TotalEarnings())
	drawPayCol(cv, margin+colWd+colGap, ty, colWd, "รายการหัก / Deductions", ps.Deductions, ps.TotalDeductions())

	// ยอดสุทธิ + ตัวอักษร
	rows := math.Max(float64(len(ps.Earnings)), float64(len(ps.Deductions)))
	ny := ty + 22 + rows*15 + 24
	cv.fillRect(margin, ny, contentW, 24, cNavy)
	cv.setFont("sarabun-b", 10.5, rgb{255, 255, 255})
	cv.textAt(margin+10, ny+5, "เงินได้สุทธิ / NET PAY")
	cv.textRight(pageW-margin-10, ny+5, money(ps.NetPay())+" บาท")
	cv.setFont("sarabun", 8.5, cNavy)
	cv.textAt(margin+10, ny+30, "( "+bahttext.FromSatang(int64(math.Round(ps.NetPay()*100)))+" )")

	// ลายเซ็นผู้รับเงิน
	cv.setFont("sarabun", 8, cMuted)
	cv.textRight(pageW-margin, ny+30, "ลงชื่อผู้รับเงิน ......................................")
}

func drawPayCol(cv *canvas, x, y, w float64, title string, lines []PayslipLine, total float64) {
	cv.fillRect(x, y, w, 18, cHeadBG)
	cv.setFont("sarabun-sb", 8.5, rgb{255, 255, 255})
	cv.textAt(x+8, y+3, title)
	yy := y + 22
	cv.setFont("sarabun", 9, cInk)
	for _, l := range lines {
		cv.textAt(x+8, yy, l.Label)
		cv.textRight(x+w-8, yy, money(l.Amount))
		cv.hline(x, yy+13, w, cLine)
		yy += 15
	}
	cv.setFont("sarabun-sb", 9, cNavy)
	cv.textAt(x+8, yy+3, "รวม")
	cv.textRight(x+w-8, yy+3, money(total))
}
