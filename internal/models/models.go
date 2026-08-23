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
