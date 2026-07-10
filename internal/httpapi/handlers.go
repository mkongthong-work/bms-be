package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/mkongthong-work/bms-be/internal/auth"
	"github.com/mkongthong-work/bms-be/internal/config"
	"github.com/mkongthong-work/bms-be/internal/excel"
	"github.com/mkongthong-work/bms-be/internal/repo"
	"github.com/mkongthong-work/bms-be/internal/thaipdf"
)

type API struct {
	cfg   *config.Config
	store *repo.Store
}

// ---------- auth ----------

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil ||
		req.Email == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "email และ password จำเป็น")
		return
	}
	var (
		id, hash, role string
		active         bool
	)
	err := a.store.Pool.QueryRow(r.Context(), `
		SELECT u.id, u.password_hash, r.code, u.is_active
		FROM users u JOIN roles r ON r.id = u.role_id
		WHERE u.email = $1`, req.Email).Scan(&id, &hash, &role, &active)
	// ตอบข้อความเดียวกันทุกกรณี — ไม่บอกใบ้ว่าอีเมลมีในระบบหรือไม่
	if err != nil || !active || !auth.CheckPassword(hash, req.Password) {
		writeErr(w, http.StatusUnauthorized, "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}
	token, err := auth.Issue(a.cfg.JWTSecret, a.cfg.JWTTTL, id, role)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ออก token ไม่สำเร็จ")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"access_token": token, "role": role})
}

// ---------- products ----------

func (a *API) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	const size = 20
	items, err := a.store.ListProducts(r.Context(), q, size, (page-1)*size)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ดึงรายการสินค้าไม่สำเร็จ")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "page": page})
}

func (a *API) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p repo.Product
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "รูปแบบข้อมูลไม่ถูกต้อง")
		return
	}
	if p.SKU == "" || p.NameTH == "" || p.SellPrice < 0 {
		writeErr(w, http.StatusUnprocessableEntity, "ต้องระบุ sku, name_th และราคาต้องไม่ติดลบ")
		return
	}
	if p.Status != "published" {
		p.Status = "draft"
	}
	id, err := a.store.CreateProduct(r.Context(), &p)
	if err != nil {
		writeErr(w, http.StatusConflict, "บันทึกไม่สำเร็จ (SKU อาจซ้ำ)")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ---------- document PDF ----------

func (a *API) DocumentPDF(w http.ResponseWriter, r *http.Request) {
	doc, err := a.store.LoadDocumentPDFModel(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "ไม่พบเอกสาร")
		return
	}
	buf, err := thaipdf.Render(doc)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "สร้าง PDF ไม่สำเร็จ")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+doc.Number+`.pdf"`)
	_, _ = w.Write(buf)
}

// ---------- report Excel ----------

func (a *API) SalesReportXLSX(w http.ResponseWriter, r *http.Request) {
	from, err1 := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	to, err2 := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil || to.Before(from) {
		writeErr(w, http.StatusBadRequest, "ระบุ from/to เป็น YYYY-MM-DD และ to >= from")
		return
	}
	rows, err := a.store.SalesReportRows(r.Context(), from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ดึงข้อมูลรายงานไม่สำเร็จ")
		return
	}
	buf, err := excel.SalesReport("บริษัท บี.เอ็ม. เซอร์วิส จำกัด", from, to, rows)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "สร้างไฟล์ Excel ไม่สำเร็จ")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="sales-report.xlsx"`)
	_, _ = w.Write(buf)
}
