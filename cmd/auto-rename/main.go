// Entry point for auto-rename app
package main

import (
	"fmt"
	"log"

	"auto-rename/internal/config"
	"auto-rename/internal/delivery"
	"auto-rename/internal/infrastructure"
	"auto-rename/internal/usecase"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Environment variables:")
	cfg := config.ParseFlags()
	// log.Printf("config.Cron=%v", cfg.Cron)
	// log.Printf("config.Dir=%v", cfg.Dir)

	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	db, err := infrastructure.NewDatabase(cfg.DbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if !cfg.WebOnly && cfg.Dir != "" {
		if err := usecase.RenameFiles(cfg, db); err != nil {
			log.Fatalf("Error renaming files: %v", err)
		}
	}

	if cfg.Cron {
		if cfg.Dir == "" {
			log.Fatalf("-cron requires -dir to be specified")
		}
		log.Printf("Cron mode enabled: scanning %s every 60s", cfg.Dir)
		go usecase.StartCronScanner(cfg, db)
	}

	if cfg.WebPort != "" {
		fmt.Printf("\nStarting web server on port %s...\n", cfg.WebPort)
		fmt.Printf("View results at: http://localhost:%s\n", cfg.WebPort)
		webServer := delivery.NewWebServer(db, cfg.WebPort)
		log.Fatal(webServer.Start())
	}

	if cfg.Cron && cfg.WebPort == "" {
		// Block main goroutine so cron scanner keeps running if no web server
		select {}
	}
}
