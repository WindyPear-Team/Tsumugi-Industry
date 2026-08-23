package models

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	DisplayName  string    `json:"display_name" gorm:"size:128"`
	Email        string    `json:"email" gorm:"size:160"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	Roles        []Role    `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;uniqueIndex;not null"`
	DisplayName string       `json:"display_name" gorm:"size:128"`
	Description string       `json:"description" gorm:"size:255"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"size:128;uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"size:128"`
	Description string    `json:"description" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
}

type SystemSetting struct {
	Key       string    `json:"key" gorm:"primaryKey;size:128"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Device struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	Code       string     `json:"code" gorm:"size:64;uniqueIndex;not null"`
	Name       string     `json:"name" gorm:"size:128;not null"`
	Type       string     `json:"type" gorm:"size:64"`
	Location   string     `json:"location" gorm:"size:160"`
	Status     string     `json:"status" gorm:"size:32;index;not null;default:offline"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Metadata   string     `json:"metadata" gorm:"type:text"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Alarm struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	DeviceID       *uint      `json:"device_id" gorm:"index"`
	Device         *Device    `json:"device,omitempty"`
	Code           string     `json:"code" gorm:"size:64;index"`
	Level          string     `json:"level" gorm:"size:32;index;not null"`
	Message        string     `json:"message" gorm:"size:255;not null"`
	Status         string     `json:"status" gorm:"size:32;index;not null;default:active"`
	OccurredAt     time.Time  `json:"occurred_at" gorm:"index"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     *uint     `json:"user_id" gorm:"index"`
	Username   string    `json:"username" gorm:"size:64"`
	Action     string    `json:"action" gorm:"size:32;index"`
	Resource   string    `json:"resource" gorm:"size:64;index"`
	ResourceID string    `json:"resource_id" gorm:"size:64"`
	Method     string    `json:"-" gorm:"size:16"`
	Path       string    `json:"-" gorm:"size:255"`
	StatusCode int       `json:"-"`
	IP         string    `json:"-" gorm:"size:64"`
	Detail     string    `json:"detail" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"index"`
}

type ScheduledTask struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"size:128;not null"`
	TaskType        string     `json:"task_type" gorm:"size:64;index;not null"`
	IntervalSeconds int        `json:"interval_seconds" gorm:"not null;default:3600"`
	Enabled         bool       `json:"enabled" gorm:"default:true"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at"`
	LastStatus      string     `json:"last_status" gorm:"size:32"`
	LastMessage     string     `json:"last_message" gorm:"size:255"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Backup struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:128;not null"`
	Driver    string    `json:"driver" gorm:"size:32"`
	Path      string    `json:"path" gorm:"size:255"`
	Size      int64     `json:"size"`
	Status    string    `json:"status" gorm:"size:32"`
	CreatedBy string    `json:"created_by" gorm:"size:64"`
	CreatedAt time.Time `json:"created_at"`
}
