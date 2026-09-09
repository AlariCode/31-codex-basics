package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"uptime-backend/internal/auth"
	"uptime-backend/internal/config"
	"uptime-backend/internal/database"
	"uptime-backend/internal/monitor"
	"uptime-backend/internal/server"

	_ "uptime-backend/docs"
)

// @title Uptime API
// @version 1.0
// @description API for user authentication, profiles, file uploads, and URL monitors.
// @host localhost:8080
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer <access_token>".
//
//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -d ../.. -o ../../docs --parseInternal
//go:generate npx --yes redoc-cli@0.13.21 bundle ../../docs/swagger.yaml -o ../../docs/swagger.html

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
	controller := auth.NewController(service, auth.HTTPConfig{RefreshTokenTTL: cfg.RefreshTokenTTL, CookieSecure: cfg.CookieSecure, AvatarDir: cfg.AvatarDir})
	controller.RegisterRoutes(routes)
	monitorStore := monitor.NewGormStore(db)
	scheduler := monitor.NewScheduler(monitorStore, monitor.NewChecker())
	monitor.NewController(service, monitorStore, monitor.NewFaviconFetcher(cfg.FaviconDir)).
		WithMonitoring(scheduler, monitorStore).RegisterRoutes(routes)
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	schedulerDone := make(chan struct{})
	go func() { defer close(schedulerDone); scheduler.Run(signalContext) }()
	routes.Handle("GET /uploads/avatars/", http.StripPrefix("/uploads/avatars/", http.FileServer(http.Dir(cfg.AvatarDir))))
	routes.Handle("GET /uploads/files/", http.StripPrefix("/uploads/files/", http.FileServer(http.Dir(filepath.Join(cfg.AvatarDir, "..", "files")))))
	routes.Handle("GET /uploads/favicons/", http.StripPrefix("/uploads/favicons/", http.FileServer(http.Dir(cfg.FaviconDir))))
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: server.WithCORS(cfg.CORSOrigin, routes), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	}()
	<-signalContext.Done()
	<-schedulerDone
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("shutdown HTTP server: %v", err)
	}
}
