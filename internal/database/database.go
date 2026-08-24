package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tsumugi-industry/internal/config"
	"tsumugi-industry/internal/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver))
	var dialector gorm.Dialector

	switch driver {
	case "sqlite", "sqlite3":
		if dir := filepath.Dir(cfg.DBDSN); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite directory: %w", err)
			}
		}
		dialector = sqlite.Open(cfg.DBDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DBDSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q: use sqlite, mysql, or postgres", cfg.DBDriver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.SystemSetting{},
		&models.Device{},
		&models.PLC{},
		&models.Alarm{},
		&models.AuditLog{},
		&models.ScheduledTask{},
		&models.Backup{},
		&models.WorkOrder{},
		&models.WorkOrderStep{},
		&models.ProductionEvent{},
		&models.PLCVariable{},
		&models.FlowDefinition{},
		&models.FlowRun{},
		&models.FlowNodeRun{},
		&models.MonitorItem{},
		&models.MonitorRecord{},
		&models.Dashboard{},
		&models.DashboardWidget{},
	); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	// Older builds used a globally unique flow code. Flow versions share a
	// code, so remove that legacy index after AutoMigrate and retain the
	// composite (code, version) uniqueness from the model tags.
	if db.Migrator().HasIndex(&models.FlowDefinition{}, "uni_flow_definitions_code") {
		if err := db.Migrator().DropIndex(&models.FlowDefinition{}, "uni_flow_definitions_code"); err != nil {
			return nil, fmt.Errorf("migrate flow version index: %w", err)
		}
	}

	return db, nil
}
