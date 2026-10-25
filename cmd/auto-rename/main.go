package main

import (
	"fmt"
	"log"
	"os"

	"auto-rename/internal/config"
	"auto-rename/internal/delivery"
	"auto-rename/internal/infrastructure"
	"auto-rename/internal/usecase"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfg := config.ParseFlags()

	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	db, err := infrastructure.NewDatabase(os.Getenv("AUTO_RENAME_API_URL"), os.Getenv("AUTO_RENAME_API_TOKEN"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("D1 API unavailable: %v", err)
	}

	scanner, err := usecase.NewScanner(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize scanner: %v", err)
	}
	if err := scanner.Reconcile(); err != nil {
		log.Fatalf("Failed to reconcile file operations: %v", err)
	}
	if err := scanner.StartCron(); err != nil {
		log.Fatalf("Failed to initialize cron: %v", err)
	}

	if cfg.WebPort != "" {
		fmt.Printf("\nStarting web server on port %s...\n", cfg.WebPort)
		fmt.Printf("View results at: http://localhost:%s\n", cfg.WebPort)
		webServer := delivery.NewWebServerWithScanner(db, scanner, cfg.WebPort, cfg.Dir)
		log.Fatal(webServer.Start())
	}

	if cfg.Cron && !cfg.WebOnly && cfg.WebPort == "" {
		select {}
	}
}
