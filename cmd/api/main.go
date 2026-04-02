package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"

	_ "github.com/lib/pq"
	"github.com/ljj/gugu-admin-api/internal/core/api"
	"github.com/ljj/gugu-admin-api/internal/support/config"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	cfg := config.Load()
	log.Printf(
		"config loaded: affiliate_app_key=%s affiliate_app_secret=%s ds_app_key=%s ds_app_secret=%s",
		fingerprint(cfg.AliExpressAppKey),
		fingerprint(cfg.AliExpressAppSecret),
		fingerprint(cfg.AliExpressDSAppKey),
		fingerprint(cfg.AliExpressDSAppSecret),
	)

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	server := api.NewServer(cfg, db)

	log.Printf("admin-api server starting on :%s", cfg.Port)
	if err := server.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "len=" + itoa(len(value)) + " sha256=" + hex.EncodeToString(sum[:4])
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
