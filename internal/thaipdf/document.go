// Package thaipdf — engine สร้างเอกสารธุรกิจภาษาไทยเป็น PDF (A4)
// รองรับ ใบเสนอราคา / ใบแจ้งหนี้ / ใบเสร็จรับเงิน / ใบส่งสินค้า / ใบลดหนี้
// ฟอนต์ Sarabun ฝังในไบนารี · โทนสีตามแบรนด์ B.M. Service
package thaipdf

import (
	"errors"
	"time"
)

// DocType ประเภทเอกสาร — กำหนดหัวเอกสารและช่องลายเซ็น
type DocType string

const (
	Quotation  DocType = "QT"
	Invoice    DocType = "INV"
	Receipt    DocType = "RC"
	Delivery   DocType = "DO"
	CreditNote DocType = "CN"
)

// docMeta ข้อความหัวเอกสารต่อประเภท (แปรตามสถานะจด VAT สำหรับใบเสร็จ)
type docMeta struct {
	TitleTH, TitleEN string
	SignLeft         string // ช่องลายเซ็นซ้าย
	SignRight        string // ช่องลายเซ็นขวา
}

func metaFor(t DocType, vatRegistered bool) docMeta {
	switch t {
	case Quotation:
		return docMeta{"ใบเสนอราคา", "QUOTATION", "ผู้เสนอราคา / Proposed by", "ผู้อนุมัติสั่งซื้อ / Approved by"}
	case Invoice:
		if vatRegistered {
			return docMeta{"ใบแจ้งหนี้ / ใบวางบิล", "INVOICE / BILLING NOTE", "ผู้วางบิล / Issued by", "ผู้รับวางบิล / Received by"}
		}
		return docMeta{"ใบแจ้งหนี้", "INVOICE", "ผู้วางบิล / Issued by", "ผู้รับวางบิล / Received by"}
	case Receipt:
		if vatRegistered {
			return docMeta{"ใบเสร็จรับเงิน / ใบกำกับภาษี", "RECEIPT / TAX INVOICE", "ผู้รับเงิน / Collector", "ผู้จ่ายเงิน / Payer"}
		}
		return docMeta{"ใบเสร็จรับเงิน", "RECEIPT", "ผู้รับเงิน / Collector", "ผู้จ่ายเงิน / Payer"}
	case Delivery:
		return docMeta{"ใบส่งสินค้า", "DELIVERY ORDER", "ผู้ส่งสินค้า / Delivered by", "ผู้รับสินค้า / Received by"}
	case CreditNote:
		return docMeta{"ใบลดหนี้", "CREDIT NOTE", "ผู้ออกเอกสาร / Issued by", "ผู้อนุมัติ / Approved by"}
	default:
		return docMeta{"เอกสาร", "DOCUMENT", "ผู้ออกเอกสาร", "ผู้อนุมัติ"}
	}
}

// Company ข้อมูลผู้ออกเอกสาร (จากหน้าตั้งค่าบริษัท)
type Company struct {
	NameTH, NameEN string
	AddressLines   []string
	TaxID          string
	Phone, Email   string
	VATRegistered  bool   // สวิตช์ VAT — คุมหัวเอกสารและบรรทัด VAT 7%
	LogoPNG        []byte // โลโก้ (ไม่บังคับ) — ว่าง = พิมพ์ชื่อบริษัทแบบตัวอักษร
}

// Party คู่ค้า/ลูกค้าบนเอกสาร
type Party struct {
	Name         string
	AddressLines []string
	TaxID        string
	Contact      string // ชื่อผู้ติดต่อ / โทร
}

// LineItem รายการสินค้า/บริการหนึ่งแถว
type LineItem struct {
	SKU       string
	Name      string // ชื่อรายการ (ตัดบรรทัดอัตโนมัติ)
	Detail    string // รายละเอียดเสริม (บรรทัดเล็กใต้ชื่อ)
	Qty       float64
	Unit      string  // หน่วย เช่น คัน, ชุด, ตัว
	UnitPrice float64 // ราคา/หน่วย (บาท)
	Discount  float64 // ส่วนลดต่อแถว (บาท)
}

// Amount ยอดสุทธิของแถว = Qty×UnitPrice − Discount
func (li LineItem) Amount() float64 { return li.Qty*li.UnitPrice - li.Discount }

// Document ข้อมูลครบหนึ่งเอกสาร พร้อมส่งเข้า Render
type Document struct {
	Type           DocType
	Number         string // เลขที่เอกสาร เช่น QT-2569/0128
	IssueDate      time.Time
	DueDate        *time.Time // ยืนราคาถึง / ครบกำหนดชำระ (ไม่บังคับ)
	RefNumber      string     // อ้างอิงเอกสารต้นทาง เช่น เลข QT บน INV
	Company        Company
	Customer       Party
	Items          []LineItem
	BillDiscount   float64 // ส่วนลดท้ายบิล (บาท)
	WithholdingPct float64 // หัก ณ ที่จ่าย % (0 = ไม่หัก)
	Salesperson    string
	PaymentTerms   string   // เงื่อนไขชำระเงิน เช่น เครดิต 30 วัน
	Notes          []string // หมายเหตุ/เงื่อนไข แสดงท้ายเอกสาร
}

// Totals ยอดรวมที่คำนวณแล้ว
type Totals struct {
	Subtotal, BillDiscount, AfterDiscount float64
	VAT, GrandTotal, Withholding, NetPay  float64
}

// Validate ตรวจความครบถ้วนก่อน render — กันเอกสารเสียหลุดถึงลูกค้า
func (d *Document) Validate() error {
	switch {
	case d.Number == "":
		return errors.New("thaipdf: document number is required")
	case d.Company.NameTH == "":
		return errors.New("thaipdf: company name is required")
	case d.Customer.Name == "":
		return errors.New("thaipdf: customer name is required")
	case len(d.Items) == 0:
		return errors.New("thaipdf: at least one line item is required")
	}
	for i, li := range d.Items {
		if li.Qty < 0 || li.UnitPrice < 0 || li.Discount < 0 {
			return errors.New("thaipdf: negative qty/price/discount at item " + li.SKU)
		}
		_ = i
	}
	return nil
}

// ComputeTotals คำนวณยอดรวมทั้งเอกสาร · VAT คิดเมื่อบริษัทจดทะเบียนเท่านั้น
func (d *Document) ComputeTotals() Totals {
	var t Totals
	for _, li := range d.Items {
		t.Subtotal += li.Amount()
	}
	t.BillDiscount = d.BillDiscount
	t.AfterDiscount = t.Subtotal - t.BillDiscount
	if d.Company.VATRegistered {
		t.VAT = round2(t.AfterDiscount * 0.07)
	}
	t.GrandTotal = round2(t.AfterDiscount + t.VAT)
	if d.WithholdingPct > 0 {
		t.Withholding = round2(t.AfterDiscount * d.WithholdingPct / 100)
	}
	t.NetPay = round2(t.GrandTotal - t.Withholding)
	return t
}
