// BMS Admin API — จุดเริ่มระบบ
// ตัวแปรที่ต้องตั้ง: BMS_DATABASE_URL, BMS_JWT_SECRET (ดู .env.example)
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/mkongthong-work/bms-be/internal/config"
	"github.com/mkongthong-work/bms-be/internal/httpapi"
	"github.com/mkongthong-work/bms-be/internal/repo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := repo.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewRouter(cfg, store),
		ReadHeaderTimeout: 5 * time.Second, // กัน slowloris
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second, // เผื่อ PDF/Excel ใหญ่
	}
	go func() {
		log.Printf("BMS API listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done() // graceful shutdown
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	log.Println("shutdown complete")
}
