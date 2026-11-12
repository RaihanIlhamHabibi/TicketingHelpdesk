package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   string
	TokenTTL    time.Duration
	SeedUsers   []SeedUser
}

type SeedUser struct {
	Name, Email, Password, Role string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        env("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	ttl, err := strconv.Atoi(env("JWT_TTL_HOURS", "24"))
	if err != nil || ttl < 1 || ttl > 24*30 {
		return Config{}, fmt.Errorf("JWT_TTL_HOURS must be between 1 and 720")
	}
	cfg.TokenTTL = time.Duration(ttl) * time.Hour
	cfg.SeedUsers = []SeedUser{
		{env("SEED_REQUESTER_NAME", "Demo Requester"), env("SEED_REQUESTER_EMAIL", "requester@example.com"), os.Getenv("SEED_REQUESTER_PASSWORD"), "requester"},
		{env("SEED_AGENT_NAME", "Demo Agent"), env("SEED_AGENT_EMAIL", "agent@example.com"), os.Getenv("SEED_AGENT_PASSWORD"), "agent"},
		{env("SEED_ADMIN_NAME", "Demo Admin"), env("SEED_ADMIN_EMAIL", "admin@example.com"), os.Getenv("SEED_ADMIN_PASSWORD"), "admin"},
	}
	for i := range cfg.SeedUsers {
		cfg.SeedUsers[i].Email = strings.ToLower(strings.TrimSpace(cfg.SeedUsers[i].Email))
		if len(cfg.SeedUsers[i].Password) < 8 {
			return Config{}, fmt.Errorf("SEED_%s_PASSWORD must be at least 8 characters", strings.ToUpper(cfg.SeedUsers[i].Role))
		}
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
