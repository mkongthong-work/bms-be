// Package excel — สร้างรายงาน Excel (.xlsx) ฝั่ง server ด้วย excelize
// ยอดรวมใช้สูตร SUM ของ Excel (ไม่ hardcode) เพื่อให้ผู้ใช้แก้ตัวเลขแล้วคำนวณต่อได้
package excel

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// SalesRow หนึ่งแถวรายงาน (map จากตาราง documents)
type SalesRow struct {
	IssueDate time.Time
	DocNumber string
	DocType   string // QT / INV / RC
	Customer  string
	Status    string
	Subtotal  float64
	VAT       float64
	Total     float64
}

// SalesReport สร้างรายงานสรุปยอดขายตามช่วงวันที่ คืนค่าเป็น bytes พร้อมส่งดาวน์โหลด
// handler: c.Header("Content-Disposition", `attachment; filename="sales.xlsx"`)
func SalesReport(companyName string, from, to time.Time, rows []SalesRow) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "รายงานยอดขาย"
	f.SetSheetName("Sheet1", sheet)

	// ---- สไตล์ ----
	base := &excelize.Font{Family: "Tahoma", Size: 10} // Tahoma รองรับไทยบน Windows/Mac
	styTitle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "Tahoma", Size: 14, Bold: true, Color: "2B4D99"}})
	styMuted, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Family: "Tahoma", Size: 9, Color: "77726D"}})
	styHead, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Tahoma", Size: 10, Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2B4D99"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    thinBorder("1F3A73"),
	})
	styCell, _ := f.NewStyle(&excelize.Style{Font: base, Border: thinBorder("D6DBE8")})
	styMoney, _ := f.NewStyle(&excelize.Style{Font: base, Border: thinBorder("D6DBE8"),
		NumFmt: 4}) // #,##0.00
	styDate, _ := f.NewStyle(&excelize.Style{Font: base, Border: thinBorder("D6DBE8"),
		CustomNumFmt: strPtr("dd/mm/yyyy")})
	styTotal, _ := f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Family: "Tahoma", Size: 10, Bold: true, Color: "2B4D99"},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E9EEF8"}},
		NumFmt: 4, Border: thinBorder("2B4D99")})

	// ---- หัวรายงาน ----
	f.SetCellValue(sheet, "A1", companyName+" — รายงานสรุปยอดขาย")
	f.SetCellStyle(sheet, "A1", "A1", styTitle)
	f.SetCellValue(sheet, "A2", fmt.Sprintf("ช่วงวันที่ %s ถึง %s · ออกรายงาน %s",
		from.Format("02/01/2006"), to.Format("02/01/2006"), time.Now().Format("02/01/2006 15:04")))
	f.SetCellStyle(sheet, "A2", "A2", styMuted)

	// ---- หัวตาราง ----
	heads := []string{"วันที่", "เลขที่เอกสาร", "ประเภท", "ลูกค้า", "สถานะ",
		"มูลค่าก่อน VAT (บาท)", "VAT (บาท)", "รวม (บาท)"}
	widths := []float64{12, 16, 9, 38, 12, 18, 13, 15}
	for i, h := range heads {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(sheet, col+"4", h)
		f.SetColWidth(sheet, col, col, widths[i])
	}
	f.SetCellStyle(sheet, "A4", "H4", styHead)
	f.SetRowHeight(sheet, 4, 22)

	// ---- ข้อมูล ----
	first, last := 5, 4+len(rows)
	for i, r := range rows {
		n := first + i
		f.SetCellValue(sheet, fmt.Sprintf("A%d", n), r.IssueDate)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", n), r.DocNumber)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", n), r.DocType)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", n), r.Customer)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", n), r.Status)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", n), r.Subtotal)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", n), r.VAT)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", n), r.Total)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", n), fmt.Sprintf("A%d", n), styDate)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", n), fmt.Sprintf("E%d", n), styCell)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", n), fmt.Sprintf("H%d", n), styMoney)
	}

	// ---- แถวรวม: ใช้สูตร SUM (ไม่ hardcode) ----
	if len(rows) > 0 {
		tr := last + 1
		f.SetCellValue(sheet, fmt.Sprintf("E%d", tr), "รวมทั้งสิ้น")
		for _, col := range []string{"F", "G", "H"} {
			f.SetCellFormula(sheet, fmt.Sprintf("%s%d", col, tr),
				fmt.Sprintf("SUM(%s%d:%s%d)", col, first, col, last))
		}
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", tr), fmt.Sprintf("H%d", tr), styTotal)
	}

	// freeze หัวตาราง + auto filter
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 4, TopLeftCell: "A5", ActivePane: "bottomLeft"})
	f.AutoFilter(sheet, fmt.Sprintf("A4:H%d", last), nil)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func thinBorder(color string) []excelize.Border {
	sides := []string{"left", "right", "top", "bottom"}
	bs := make([]excelize.Border, 0, 4)
	for _, s := range sides {
		bs = append(bs, excelize.Border{Type: s, Style: 1, Color: color})
	}
	return bs
}

func strPtr(s string) *string { return &s }
