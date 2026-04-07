package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
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

	warnIfTransactionPooler(cfg.DSN())

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)

	if err := pingDB(db, 5*time.Second); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	server := api.NewServer(cfg, db)

	log.Printf("admin-api server starting on :%s", cfg.Port)
	if err := server.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func pingDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return db.PingContext(ctx)
}

func warnIfTransactionPooler(dsn string) {
	if strings.Contains(dsn, ".pooler.supabase.com:6543") {
		log.Printf("database configuration warning: transaction pooler detected in DATABASE_URL; use Supabase direct or session connection for this long-lived API server")
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
