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

	"golang.org/x/crypto/bcrypt"
	"helpdesk/backend/internal/config"
	"helpdesk/backend/internal/httpapi"
	"helpdesk/backend/internal/repository"
	"helpdesk/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := repository.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer store.DB.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := store.Seed(ctx, cfg.SeedUsers, func(password string) (string, error) {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		return string(hash), err
	}); err != nil {
		log.Fatalf("seed accounts: %v", err)
	}

	appService := service.New(store, cfg.JWTSecret, cfg.TokenTTL)
	server := &http.Server{
		Addr: cfg.Addr, Handler: httpapi.New(appService, cfg.JWTSecret).Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		log.Printf("helpdesk API listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
}
