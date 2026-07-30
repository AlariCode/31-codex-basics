package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uptime-backend/internal/auth"
	"uptime-backend/internal/config"
	"uptime-backend/internal/database"
	"uptime-backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get SQL database: %v", err)
	}
	defer sqlDB.Close()
	service := auth.NewService(auth.NewGormUserStore(db), auth.NewGormSessionStore(db), cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	routes := http.NewServeMux()
	auth.NewController(service, auth.HTTPConfig{RefreshTokenTTL: cfg.RefreshTokenTTL, CookieSecure: cfg.CookieSecure}).RegisterRoutes(routes)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: server.WithCORS(cfg.CORSOrigin, routes), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	}()
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-signalContext.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("shutdown HTTP server: %v", err)
	}
}
