package main

import (
	"database/sql"
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
