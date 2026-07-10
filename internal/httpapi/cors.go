package httpapi

import "net/http"

// corsMiddleware อนุญาตเฉพาะ origin ที่กำหนดใน config (frontend production)
// ไม่ใช้ "*" เพื่อไม่เปิดให้เว็บอื่นยิง API ข้ามโดเมนได้
func corsMiddleware(allowedOrigin string, next http.Handler) http.Handler {
	if allowedOrigin == "" {
		return next // dev ใช้ proxy ของ ng serve — ไม่ต้องเปิด CORS
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == allowedOrigin {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", allowedOrigin)
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions { // preflight
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
