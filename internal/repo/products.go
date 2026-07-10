package repo

import (
	"context"
	"time"
)

type Product struct {
	ID        string    `json:"id"`
	SKU       string    `json:"sku"`
	NameTH    string    `json:"name_th"`
	NameEN    string    `json:"name_en"`
	Unit      string    `json:"unit"`
	SellPrice float64   `json:"sell_price"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListProducts ค้นหา + แบ่งหน้า — q ผูกเป็นพารามิเตอร์เสมอ กัน SQL injection
func (s *Store) ListProducts(ctx context.Context, q string, limit, offset int) ([]Product, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT id, sku, name_th, name_en, unit, sell_price, status, updated_at
		FROM products
		WHERE deleted_at IS NULL
		  AND ($1 = '' OR sku ILIKE '%'||$1||'%' OR name_th ILIKE '%'||$1||'%')
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3`, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.NameTH, &p.NameEN, &p.Unit,
			&p.SellPrice, &p.Status, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateProduct เพิ่มสินค้าใหม่ คืน id
func (s *Store) CreateProduct(ctx context.Context, p *Product) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO products (sku, name_th, name_en, unit, sell_price, status)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		p.SKU, p.NameTH, p.NameEN, p.Unit, p.SellPrice, p.Status).Scan(&id)
	return id, err
}
