package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"

	"github.com/mkongthong-work/bms-be/internal/excel"
	"github.com/mkongthong-work/bms-be/internal/thaipdf"
)

// CreateDocument ออกเอกสารใหม่ — จองเลขรัน + insert ใน transaction เดียว (เลขไม่ชน/ไม่ข้าม)
func (s *Store) CreateDocument(ctx context.Context, companyID, docType string, yearBE int,
	insert func(ctx context.Context, tx pgx.Tx, docNumber string) error) (string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var number string
	if err := tx.QueryRow(ctx, `SELECT next_document_number($1,$2,$3)`,
		companyID, docType, yearBE).Scan(&number); err != nil {
		return "", fmt.Errorf("allocate doc number: %w", err)
	}
	// ผู้เรียกทำ INSERT ภายใน tx เดียวกันด้วยเลขที่ได้
	if err := insert(ctx, tx, number); err != nil {
		return "", err
	}
	return number, tx.Commit(ctx)
}

// LoadDocumentPDFModel ประกอบข้อมูลเอกสารจาก DB → โมเดลของ engine thaipdf
func (s *Store) LoadDocumentPDFModel(ctx context.Context, id string) (*thaipdf.Document, error) {
	var (
		doc      thaipdf.Document
		docType  string
		snapshot []byte
		due      *time.Time
		notes    []string
	)
	err := s.Pool.QueryRow(ctx, `
		SELECT d.doc_type, d.doc_number, d.issue_date, d.due_date, d.bill_discount,
		       d.withholding_pct, d.vat_applied, d.customer_snapshot, d.salesperson,
		       d.payment_terms, d.notes,
		       c.name_th, c.name_en, c.address_lines, c.tax_id, c.phone
		FROM documents d JOIN companies c ON c.id = d.company_id
		WHERE d.id = $1`, id).Scan(
		&docType, &doc.Number, &doc.IssueDate, &due, &doc.BillDiscount,
		&doc.WithholdingPct, &doc.Company.VATRegistered, &snapshot, &doc.Salesperson,
		&doc.PaymentTerms, &notes,
		&doc.Company.NameTH, &doc.Company.NameEN, &doc.Company.AddressLines,
		&doc.Company.TaxID, &doc.Company.Phone)
	if err != nil {
		return nil, err
	}
	doc.Type, doc.DueDate, doc.Notes = thaipdf.DocType(docType), due, notes
	if err := json.Unmarshal(snapshot, &doc.Customer); err != nil {
		return nil, fmt.Errorf("customer snapshot: %w", err)
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT sku, name, detail, qty, unit, unit_price, discount
		FROM document_items WHERE document_id = $1 ORDER BY sort_order`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var li thaipdf.LineItem
		if err := rows.Scan(&li.SKU, &li.Name, &li.Detail, &li.Qty, &li.Unit,
			&li.UnitPrice, &li.Discount); err != nil {
			return nil, err
		}
		doc.Items = append(doc.Items, li)
	}
	return &doc, rows.Err()
}

// SalesReportRows ดึงข้อมูลรายงานยอดขายตามช่วงวันที่ (สำหรับ export Excel)
func (s *Store) SalesReportRows(ctx context.Context, from, to time.Time) ([]excel.SalesRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT d.issue_date, d.doc_number, d.doc_type,
		       d.customer_snapshot->>'Name', d.status, d.subtotal, d.vat_amount, d.grand_total
		FROM documents d
		WHERE d.doc_type IN ('INV','RC') AND d.status <> 'void'
		  AND d.issue_date BETWEEN $1 AND $2
		ORDER BY d.issue_date, d.doc_number`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []excel.SalesRow
	for rows.Next() {
		var r excel.SalesRow
		if err := rows.Scan(&r.IssueDate, &r.DocNumber, &r.DocType, &r.Customer,
			&r.Status, &r.Subtotal, &r.VAT, &r.Total); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
