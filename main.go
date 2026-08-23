package main

import (
	"embed"
	"io/fs"
	"log"
	"time"

	"gorm.io/gorm"
	"tsumugi-industry/internal/api"
	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/config"
	"tsumugi-industry/internal/database"
	"tsumugi-industry/internal/maintenance"
	"tsumugi-industry/internal/models"
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
	maintenanceService := maintenance.New(db, "./data/backups")
	seedTasks(db)
	router := api.NewRouter(db, manager, maintenanceService, dist)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			maintenanceService.RunDueTasks()
		}
	}()

	log.Printf("Tsumugi Industry listening on http://%s", cfg.Address())
	if err := router.Run(cfg.Address()); err != nil {
		log.Fatal(err)
	}
}

func seedTasks(db *gorm.DB) {
	defaults := []models.ScheduledTask{{Name: "设备状态监测", TaskType: "device_monitor", IntervalSeconds: 60}, {Name: "业务日志清理", TaskType: "log_cleanup", IntervalSeconds: 86400}, {Name: "数据库备份", TaskType: "database_backup", IntervalSeconds: 86400}}
	for _, task := range defaults {
		db.Where("task_type = ?", task.TaskType).FirstOrCreate(&task)
	}
}
