// Package httpapi — REST router ด้วย net/http (Go 1.22 pattern routing)
package httpapi

import (
	"net/http"

	"github.com/mkongthong-work/bms-be/internal/auth"
	"github.com/mkongthong-work/bms-be/internal/config"
	"github.com/mkongthong-work/bms-be/internal/repo"
)

func NewRouter(cfg *config.Config, store *repo.Store) http.Handler {
	mux := http.NewServeMux()
	api := &API{cfg: cfg, store: store}

	authed := auth.Middleware(cfg.JWTSecret) // ทุก role ที่ล็อกอิน
	sales := auth.Middleware(cfg.JWTSecret, "SALES", "ACCOUNT")
	account := auth.Middleware(cfg.JWTSecret, "ACCOUNT")

	mux.HandleFunc("POST /api/v1/auth/login", api.Login)
	mux.HandleFunc("GET /api/v1/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("GET /api/v1/products", authed(http.HandlerFunc(api.ListProducts)))
	mux.Handle("POST /api/v1/products", sales(http.HandlerFunc(api.CreateProduct)))
	mux.Handle("GET /api/v1/documents/{id}/pdf", sales(http.HandlerFunc(api.DocumentPDF)))
	mux.Handle("GET /api/v1/reports/sales.xlsx", account(http.HandlerFunc(api.SalesReportXLSX)))

	return securityHeaders(mux)
}

// securityHeaders ใส่ header ป้องกันพื้นฐานทุก response
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
