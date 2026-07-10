package thaipdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/mkongthong-work/bms-be/internal/thaipdf/bahttext"
	"github.com/signintech/gopdf"
)

// ขนาดหน้า A4 (pt) และระยะขอบ
const (
	pageW, pageH = 595.28, 841.89
	margin       = 40.0
	contentW     = pageW - 2*margin
	footerH      = 46.0 // พื้นที่กันไว้ให้ footer ทุกหน้า
)

// สัดส่วนคอลัมน์ตาราง: ลำดับ/รหัส/รายการ/จำนวน/หน่วย/ราคาต่อหน่วย/ส่วนลด/จำนวนเงิน
var colW = []float64{28, 62, 175, 40, 34, 62, 52, 62.28}

// Render สร้าง PDF จากข้อมูลเอกสาร คืนค่าเป็น bytes พร้อมส่งผ่าน HTTP
// ตัวอย่างใช้งานใน handler:  w.Header().Set("Content-Type","application/pdf"); w.Write(buf)
func Render(doc *Document) ([]byte, error) {
	if err := doc.Validate(); err != nil {
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

	r := &renderer{canvas: canvas{p: p}, doc: doc, meta: metaFor(doc.Type, doc.Company.VATRegistered), totals: doc.ComputeTotals()}
	if err := r.build(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if _, err := p.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type renderer struct {
	canvas
	doc    *Document
	meta   docMeta
	totals Totals
	page   int
}

// ---------- โครงหลัก ----------

func (r *renderer) build() error {
	r.newPage()

	// ตารางรายการ — ขึ้นหน้าใหม่อัตโนมัติเมื่อพื้นที่ไม่พอ
	r.tableHeader()
	for i, li := range r.doc.Items {
		rowH := r.rowHeight(li)
		if r.p.GetY()+rowH > pageH-footerH-170 { // เผื่อที่ให้บล็อกยอดรวมหน้าสุดท้าย
			r.newPage()
			r.tableHeader()
		}
		r.tableRow(i+1, li, i%2 == 1)
	}
	r.tableBottomRule()

	if err := r.totalsBlock(); err != nil {
		return err
	}
	r.notesAndSignatures()
	r.footers()
	return nil
}

func (r *renderer) newPage() {
	r.p.AddPage()
	r.page++
	r.header()
}

// ---------- ส่วนหัวเอกสาร ----------

func (r *renderer) header() {
	p, d := r.p, r.doc

	// โลโก้ตัวอักษร: ชื่อบริษัท EN หนา + แถบแดงใต้ชื่อ (แทนโลโก้จริงจนกว่าจะฝัง PNG)
	r.setFont("sarabun-b", 17, cNavy)
	p.SetXY(margin, margin-6)
	p.Cell(nil, d.Company.NameEN)
	r.fillRect(margin, margin+16, 120, 3, cRed)

	r.setFont("sarabun", 8.5, cMuted)
	y := margin + 26.0
	r.textAt(margin, y, d.Company.NameTH)
	for _, ln := range d.Company.AddressLines {
		y += 12
		r.textAt(margin, y, ln)
	}
	y += 12
	r.textAt(margin, y, "เลขประจำตัวผู้เสียภาษี "+d.Company.TaxID+"   โทร "+d.Company.Phone)

	// ชื่อเอกสาร + กล่องเลขที่ (ขวา)
	r.setFont("sarabun-b", 16, cNavy)
	r.textRight(pageW-margin, margin+4, r.meta.TitleTH)
	r.setFont("sarabun-sb", 9, cRed)
	r.textRight(pageW-margin, margin+20, r.meta.TitleEN)

	boxW, boxX, boxY := 200.0, pageW-margin-200, margin+32
	rows := [][2]string{{"เลขที่ / No.", d.Number}, {"วันที่ / Date", thaiDate(d.IssueDate)}}
	if d.DueDate != nil {
		label := "ยืนราคาถึง / Valid until"
		if d.Type != Quotation {
			label = "ครบกำหนด / Due date"
		}
		rows = append(rows, [2]string{label, thaiDate(*d.DueDate)})
	}
	if d.RefNumber != "" {
		rows = append(rows, [2]string{"อ้างอิง / Ref.", d.RefNumber})
	}
	boxH := float64(len(rows))*15 + 8
	r.strokeRect(boxX, boxY, boxW, boxH, cLine)
	yy := boxY + 6
	for _, row := range rows {
		r.setFont("sarabun", 8.5, cMuted)
		r.textAt(boxX+8, yy, row[0])
		r.setFont("sarabun-sb", 8.5, cInk)
		r.textRight(boxX+boxW-8, yy, row[1])
		yy += 15
	}

	// กล่องลูกค้า
	custY := math.Max(y+22, boxY+boxH+12)
	r.fillRect(margin, custY, contentW, 16, rgb{233, 238, 248}) // #e9eef8
	r.setFont("sarabun-sb", 9, cNavy)
	r.textAt(margin+8, custY+3, "ลูกค้า / Customer")
	if d.Salesperson != "" {
		r.setFont("sarabun", 8.5, cMuted)
		r.textRight(pageW-margin-8, custY+3, "ผู้ขาย: "+d.Salesperson)
	}
	cy := custY + 22
	r.setFont("sarabun-sb", 10, cInk)
	r.textAt(margin+8, cy, d.Customer.Name)
	r.setFont("sarabun", 8.5, cMuted)
	for _, ln := range d.Customer.AddressLines {
		cy += 12.5
		r.textAt(margin+8, cy, ln)
	}
	extra := []string{}
	if d.Customer.TaxID != "" {
		extra = append(extra, "เลขประจำตัวผู้เสียภาษี "+d.Customer.TaxID)
	}
	if d.Customer.Contact != "" {
		extra = append(extra, "ติดต่อ "+d.Customer.Contact)
	}
	if len(extra) > 0 {
		cy += 12.5
		r.textAt(margin+8, cy, strings.Join(extra, "   ·   "))
	}
	p.SetY(cy + 24)
}

// ---------- ตารางรายการ ----------

func colX(i int) float64 {
	x := margin
	for j := 0; j < i; j++ {
		x += colW[j]
	}
	return x
}

func (r *renderer) tableHeader() {
	y := r.p.GetY()
	r.fillRect(margin, y, contentW, 20, cHeadBG)
	heads := []string{"ลำดับ", "รหัส", "รายการ", "จำนวน", "หน่วย", "ราคา/หน่วย", "ส่วนลด", "จำนวนเงิน"}
	aligns := []string{"C", "L", "L", "R", "C", "R", "R", "R"}
	r.setFont("sarabun-sb", 8.5, rgb{255, 255, 255})
	for i, h := range heads {
		r.cellText(colX(i), y+4.5, colW[i], h, aligns[i], 6)
	}
	r.p.SetY(y + 20)
}

// rowHeight ความสูงแถวตามจำนวนบรรทัดชื่อ+รายละเอียด (สำหรับตัดหน้า)
func (r *renderer) rowHeight(li LineItem) float64 {
	r.setFont("sarabun", 9, cInk)
	lines := r.wrap(li.Name, colW[2]-12)
	n := len(lines)
	if li.Detail != "" {
		n += len(r.wrap(li.Detail, colW[2]-12))
	}
	return math.Max(20, float64(n)*12.5+8)
}

func (r *renderer) tableRow(no int, li LineItem, zebra bool) {
	y, h := r.p.GetY(), r.rowHeight(li)
	if zebra {
		r.fillRect(margin, y, contentW, h, cZebra)
	}

	r.setFont("sarabun", 9, cInk)
	r.cellText(colX(0), y+4, colW[0], strconv.Itoa(no), "C", 6)
	r.setFont("sarabun", 8, cMuted)
	r.cellText(colX(1), y+5, colW[1], li.SKU, "L", 6)

	ty := y + 4
	r.setFont("sarabun", 9, cInk)
	for _, ln := range r.wrap(li.Name, colW[2]-12) {
		r.textAt(colX(2)+6, ty, ln)
		ty += 12.5
	}
	if li.Detail != "" {
		r.setFont("sarabun", 7.5, cMuted)
		for _, ln := range r.wrap(li.Detail, colW[2]-12) {
			r.textAt(colX(2)+6, ty, ln)
			ty += 12.5
		}
	}

	r.setFont("sarabun", 9, cInk)
	r.cellText(colX(3), y+4, colW[3], trimQty(li.Qty), "R", 6)
	r.cellText(colX(4), y+4, colW[4], li.Unit, "C", 6)
	r.cellText(colX(5), y+4, colW[5], money(li.UnitPrice), "R", 6)
	dis := "-"
	if li.Discount > 0 {
		dis = money(li.Discount)
	}
	r.cellText(colX(6), y+4, colW[6], dis, "R", 6)
	r.setFont("sarabun-sb", 9, cInk)
	r.cellText(colX(7), y+4, colW[7], money(li.Amount()), "R", 6)

	r.hline(margin, y+h, contentW, cLine)
	r.p.SetY(y + h)
}

func (r *renderer) tableBottomRule() { r.hline(margin, r.p.GetY(), contentW, cLine) }

// ---------- ยอดรวม + ตัวอักษร ----------

func (r *renderer) totalsBlock() error {
	t, d := r.totals, r.doc
	y := r.p.GetY() + 10
	rightW, rx := 240.0, pageW-margin-240

	type line struct {
		label, val string
		strong     bool
	}
	lines := []line{{"รวมเป็นเงิน", money(t.Subtotal), false}}
	if t.BillDiscount > 0 {
		lines = append(lines,
			line{"ส่วนลดท้ายบิล", "-" + money(t.BillDiscount), false},
			line{"ยอดหลังหักส่วนลด", money(t.AfterDiscount), false})
	}
	if d.Company.VATRegistered {
		lines = append(lines, line{"ภาษีมูลค่าเพิ่ม 7%", money(t.VAT), false})
	}
	lines = append(lines, line{"จำนวนเงินรวมทั้งสิ้น", money(t.GrandTotal), true})
	if t.Withholding > 0 {
		lines = append(lines,
			line{fmt.Sprintf("หัก ณ ที่จ่าย %s%%", trimQty(d.WithholdingPct)), "-" + money(t.Withholding), false},
			line{"ยอดชำระสุทธิ", money(t.NetPay), true})
	}

	yy := y
	for _, ln := range lines {
		if ln.strong {
			r.fillRect(rx, yy-3, rightW, 19, rgb{233, 238, 248})
			r.setFont("sarabun-b", 10, cNavy)
		} else {
			r.setFont("sarabun", 9, cInk)
		}
		r.textAt(rx+8, yy, ln.label)
		r.textRight(pageW-margin-8, yy, ln.val)
		yy += 19
	}

	// ยอดตัวอักษร (อ่านจากยอดที่ต้องชำระจริง)
	words := bahttext.FromSatang(int64(math.Round(t.GrandTotal * 100)))
	bw := rx - margin - 12
	r.strokeRect(margin, y-3, bw, 40, cLine)
	r.setFont("sarabun", 7.5, cMuted)
	r.textAt(margin+8, y+1, "จำนวนเงิน (ตัวอักษร)")
	r.setFont("sarabun-sb", 9.5, cNavy)
	r.textAt(margin+8, y+16, "( "+words+" )")

	r.p.SetY(math.Max(yy, y+44) + 6)
	return nil
}

// ---------- หมายเหตุ + ลายเซ็น ----------

func (r *renderer) notesAndSignatures() {
	d := r.doc
	y := r.p.GetY() + 4

	if d.PaymentTerms != "" || len(d.Notes) > 0 {
		r.setFont("sarabun-sb", 8.5, cNavy)
		r.textAt(margin, y, "เงื่อนไขและหมายเหตุ")
		y += 14
		r.setFont("sarabun", 8.5, cMuted)
		if d.PaymentTerms != "" {
			r.textAt(margin, y, "• การชำระเงิน: "+d.PaymentTerms)
			y += 12.5
		}
		for _, n := range d.Notes {
			for _, ln := range r.wrap("• "+n, contentW-160) {
				r.textAt(margin, y, ln)
				y += 12.5
			}
		}
	}

	// บล็อกลายเซ็น 2 ช่อง ชิดล่างเสมอ
	sy := pageH - footerH - 78
	if y+20 > sy {
		sy = y + 20
	}
	slotW := (contentW - 24) / 2
	for i, label := range []string{r.meta.SignLeft, r.meta.SignRight} {
		x := margin + float64(i)*(slotW+24)
		r.hline(x+20, sy+42, slotW-40, cMuted)
		r.setFont("sarabun", 8.5, cInk)
		r.cellText(x, sy+48, slotW, label, "C", 0)
		r.setFont("sarabun", 8, cMuted)
		r.cellText(x, sy+62, slotW, "วันที่ ............ / ............ / ............", "C", 0)
	}
}

// footers เลขหน้า + เลขเอกสาร ทุกหน้า (วาดท้ายสุดเพื่อรู้จำนวนหน้ารวม)
func (r *renderer) footers() {
	total := r.page
	for i := 1; i <= total; i++ {
		r.p.SetPage(i)
		fy := pageH - 30
		r.hline(margin, fy-6, contentW, cLine)
		r.setFont("sarabun", 7.5, cMuted)
		r.textAt(margin, fy, r.doc.Company.NameTH+" · "+r.doc.Number)
		r.textRight(pageW-margin, fy, fmt.Sprintf("หน้า %d / %d", i, total))
	}
}
