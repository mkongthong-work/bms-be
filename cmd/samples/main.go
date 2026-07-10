// สร้างไฟล์ตัวอย่างสำหรับตรวจงาน: sales-report.xlsx + payslip PDF (รายวัน/รายเดือน)
package main

import (
	"log"
	"os"
	"time"

	"github.com/mkongthong-work/bms-be/internal/excel"
	"github.com/mkongthong-work/bms-be/internal/thaipdf"
)

func d(y, m, dd int) time.Time { return time.Date(y, time.Month(m), dd, 0, 0, 0, 0, time.Local) }

func main() {
	// ---------- Excel ----------
	rows := []excel.SalesRow{
		{IssueDate: d(2026, 6, 2), DocNumber: "INV-2569/0090", DocType: "INV", Customer: "บริษัท อุตสาหกรรมตัวอย่าง จำกัด (มหาชน)", Status: "ชำระแล้ว", Subtotal: 84700, VAT: 5929, Total: 90629},
		{IssueDate: d(2026, 6, 5), DocNumber: "RC-2569/0065", DocType: "RC", Customer: "หจก. โรงงานตัวอย่าง", Status: "ชำระแล้ว", Subtotal: 10100, VAT: 707, Total: 10807},
		{IssueDate: d(2026, 6, 12), DocNumber: "INV-2569/0091", DocType: "INV", Customer: "บริษัท คลังสินค้าบางนา จำกัด", Status: "ค้างชำระ", Subtotal: 42500, VAT: 2975, Total: 45475},
		{IssueDate: d(2026, 6, 18), DocNumber: "INV-2569/0092", DocType: "INV", Customer: "บริษัท โลจิสติกส์ไทย จำกัด", Status: "ชำระแล้ว", Subtotal: 18900, VAT: 1323, Total: 20223},
		{IssueDate: d(2026, 6, 25), DocNumber: "RC-2569/0066", DocType: "RC", Customer: "โรงงานผลิตชิ้นส่วนยานยนต์", Status: "ชำระแล้ว", Subtotal: 27300, VAT: 1911, Total: 29211},
	}
	xlsx, err := excel.SalesReport("บริษัท บี.เอ็ม. เซอร์วิส จำกัด", d(2026, 6, 1), d(2026, 6, 30), rows)
	if err != nil {
		log.Fatal(err)
	}
	must(os.WriteFile("sales-report-sample.xlsx", xlsx, 0o644))

	// ---------- Payslip (พนักงานรายวัน) ----------
	ps := &thaipdf.Payslip{
		Company:     thaipdf.Company{NameTH: "บริษัท บี.เอ็ม. เซอร์วิส จำกัด", NameEN: "B.M. SERVICE CO., LTD."},
		PeriodStart: d(2026, 6, 1), PeriodEnd: d(2026, 6, 30),
		EmployeeCode: "EMP-014", FullName: "นายสมศักดิ์ ขยันงาน", Position: "ช่างซ่อมไฮดรอลิค",
		WageType: "daily", WorkDays: 25, OTHours: 12,
		Earnings: []thaipdf.PayslipLine{
			{Label: "ค่าแรง 25 วัน × 450 บาท", Amount: 11250},
			{Label: "ค่าล่วงเวลา 12 ชม. × 84.38 บาท (1.5 เท่า)", Amount: 1012.50},
			{Label: "เบี้ยขยัน", Amount: 500},
		},
		Deductions: []thaipdf.PayslipLine{
			{Label: "ประกันสังคม 5%", Amount: 638.13},
			{Label: "เบิกล่วงหน้า", Amount: 1000},
		},
		BankAccount: "กสิกรไทย xxx-x-x1234-x",
	}
	pdf, err := thaipdf.RenderPayslip(ps)
	if err != nil {
		log.Fatal(err)
	}
	must(os.WriteFile("payslip-sample.pdf", pdf, 0o644))
	log.Println("generated sales-report-sample.xlsx + payslip-sample.pdf")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
