package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

type ctxKey int

const claimsKey ctxKey = 0

// FromContext ดึง claims ของผู้ใช้ที่ middleware ตรวจแล้ว
func FromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}

// Middleware ตรวจ Bearer token · roles ว่าง = ขอแค่ล็อกอิน
func Middleware(secret []byte, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			claims, err := Verify(secret, raw)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if len(roles) > 0 && claims.Role != "ADMIN" && !slices.Contains(roles, claims.Role) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
		})
	}
}
