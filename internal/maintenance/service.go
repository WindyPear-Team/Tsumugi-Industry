package maintenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tsumugi-industry/internal/models"

	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	dataDir string
}

func New(db *gorm.DB, dataDir string) *Service { return &Service{db: db, dataDir: dataDir} }

func (s *Service) RunDueTasks() {
	var tasks []models.ScheduledTask
	if err := s.db.Where("enabled = ? AND (next_run_at IS NULL OR next_run_at <= ?)", true, time.Now()).Find(&tasks).Error; err != nil {
		return
	}
	for _, task := range tasks {
		s.runTask(task)
	}
}

func (s *Service) runTask(task models.ScheduledTask) {
	now := time.Now()
	next := now.Add(time.Duration(task.IntervalSeconds) * time.Second)
	status, message := "success", "task completed"
	switch task.TaskType {
	case "log_cleanup":
		cutoff := now.AddDate(0, 0, -30)
		result := s.db.Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
		if result.Error != nil {
			status, message = "failed", result.Error.Error()
		} else {
			message = fmt.Sprintf("deleted %d audit events", result.RowsAffected)
		}
	case "device_monitor":
		cutoff := now.Add(-10 * time.Minute)
		s.db.Model(&models.Device{}).Where("last_seen_at IS NOT NULL AND last_seen_at < ? AND status = ?", cutoff, "online").Updates(map[string]any{"status": "offline"})
		s.db.Model(&models.PLC{}).Where("last_seen_at IS NOT NULL AND last_seen_at < ? AND status = ?", cutoff, "online").Updates(map[string]any{"status": "offline"})
	case "database_backup":
		if _, err := s.CreateBackup("scheduled"); err != nil {
			status, message = "failed", err.Error()
		}
	default:
		status, message = "skipped", "unknown task type"
	}
	s.db.Model(&task).Updates(map[string]any{"last_run_at": now, "next_run_at": next, "last_status": status, "last_message": message})
}

func (s *Service) CreateBackup(createdBy string) (models.Backup, error) {
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return models.Backup{}, err
	}
	var settings []models.SystemSetting
	if err := s.db.Find(&settings).Error; err != nil {
		return models.Backup{}, err
	}
	var users []models.User
	var roles []models.Role
	var permissions []models.Permission
	var devices []models.Device
	var workOrders []models.WorkOrder
	var workOrderSteps []models.WorkOrderStep
	var productionEvents []models.ProductionEvent
	s.db.Find(&users)
	s.db.Find(&roles)
	s.db.Find(&permissions)
	s.db.Find(&devices)
	s.db.Find(&workOrders)
	s.db.Find(&workOrderSteps)
	s.db.Find(&productionEvents)
	payload := map[string]any{"settings": settings, "users": users, "roles": roles, "permissions": permissions, "devices": devices, "work_orders": workOrders, "work_order_steps": workOrderSteps, "production_events": productionEvents}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return models.Backup{}, err
	}
	path := filepath.Join(s.dataDir, fmt.Sprintf("backup-%s.json", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return models.Backup{}, err
	}
	backup := models.Backup{Name: filepath.Base(path), Driver: "json", Path: path, Size: int64(len(content)), Status: "completed", CreatedBy: createdBy}
	return backup, s.db.Create(&backup).Error
}

func (s *Service) RestoreBackup(backup models.Backup) error {
	content, err := os.ReadFile(backup.Path)
	if err != nil {
		return err
	}
	var payload struct {
		Settings         []models.SystemSetting   `json:"settings"`
		Users            []models.User            `json:"users"`
		Roles            []models.Role            `json:"roles"`
		Permissions      []models.Permission      `json:"permissions"`
		Devices          []models.Device          `json:"devices"`
		WorkOrders       []models.WorkOrder       `json:"work_orders"`
		WorkOrderSteps   []models.WorkOrderStep   `json:"work_order_steps"`
		ProductionEvents []models.ProductionEvent `json:"production_events"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, setting := range payload.Settings {
			if err := tx.Save(&setting).Error; err != nil {
				return err
			}
		}
		for _, permission := range payload.Permissions {
			if err := tx.Save(&permission).Error; err != nil {
				return err
			}
		}
		for _, role := range payload.Roles {
			if err := tx.Save(&role).Error; err != nil {
				return err
			}
		}
		for _, user := range payload.Users {
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}
		for _, device := range payload.Devices {
			if err := tx.Save(&device).Error; err != nil {
				return err
			}
		}
		for _, order := range payload.WorkOrders {
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
		}
		for _, step := range payload.WorkOrderSteps {
			if err := tx.Save(&step).Error; err != nil {
				return err
			}
		}
		for _, event := range payload.ProductionEvents {
			if err := tx.Save(&event).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
