package main

import (
	"embed"
	"io/fs"
	"log"

	"tsumugi-industry/internal/api"
	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/config"
	"tsumugi-industry/internal/database"
	"tsumugi-industry/internal/system"
)

// The frontend is built into web/dist before the Go binary is built.
//
//go:embed web/dist
var frontend embed.FS

func main() {
	cfg := config.Load()
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database startup failed: %v", err)
	}
	if err := system.EnsureDefaultPermissions(db); err != nil {
		log.Fatalf("permission startup failed: %v", err)
	}
	secret, err := system.EnsureJWTSecret(db)
	if err != nil {
		log.Fatalf("auth startup failed: %v", err)
	}
	dist, err := fs.Sub(frontend, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	manager := auth.NewManager(db, secret)
	router := api.NewRouter(db, manager, dist)

	log.Printf("Tsumugi Industry listening on http://%s", cfg.Address())
	if err := router.Run(cfg.Address()); err != nil {
		log.Fatal(err)
	}
}
