package system

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"tsumugi-industry/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrAlreadyInitialized = errors.New("system is already initialized")

type PermissionSeed struct {
	Code string
	Name string
}

var DefaultPermissions = []PermissionSeed{
	{Code: "dashboard.read", Name: "查看仪表盘"},
	{Code: "users.read", Name: "查看用户"},
	{Code: "users.write", Name: "管理用户"},
	{Code: "roles.read", Name: "查看角色"},
	{Code: "roles.write", Name: "管理角色"},
	{Code: "system.settings", Name: "管理系统设置"},
	{Code: "devices.read", Name: "查看设备"},
	{Code: "devices.write", Name: "管理设备"},
}

type SetupRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"max=128"`
	SystemName  string `json:"system_name" binding:"max=128"`
	Timezone    string `json:"timezone" binding:"max=64"`
}

func IsInitialized(db *gorm.DB) (bool, error) {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	var setting models.SystemSetting
	err := db.First(&setting, "key = ?", "system.initialized").Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil && setting.Value == "true", err
}

func EnsureJWTSecret(db *gorm.DB) (string, error) {
	var setting models.SystemSetting
	err := db.First(&setting, "key = ?", "auth.jwt_secret").Error
	if err == nil && setting.Value != "" {
		return setting.Value, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate auth secret: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(bytes)
	if err := db.Create(&models.SystemSetting{Key: "auth.jwt_secret", Value: secret}).Error; err != nil {
		return "", err
	}
	return secret, nil
}

func Initialize(db *gorm.DB, req SetupRequest) error {
	username := strings.TrimSpace(req.Username)
	displayName := strings.TrimSpace(req.DisplayName)
	systemName := strings.TrimSpace(req.SystemName)
	if displayName == "" {
		displayName = username
	}
	if systemName == "" {
		systemName = "Tsumugi Industry"
	}
	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAlreadyInitialized
		}

		permissions := make([]models.Permission, 0, len(DefaultPermissions))
		for _, seed := range DefaultPermissions {
			permission := models.Permission{Code: seed.Code, Name: seed.Name}
			if err := tx.Where("code = ?", seed.Code).FirstOrCreate(&permission).Error; err != nil {
				return err
			}
			permissions = append(permissions, permission)
		}

		adminRole := models.Role{Name: "admin", DisplayName: "系统管理员", Description: "拥有全部系统权限"}
		if err := tx.Create(&adminRole).Error; err != nil {
			return err
		}
		if err := tx.Model(&adminRole).Association("Permissions").Append(&permissions); err != nil {
			return err
		}

		admin := models.User{Username: username, PasswordHash: string(passwordHash), DisplayName: displayName, IsActive: true}
		if err := tx.Create(&admin).Error; err != nil {
			return err
		}
		if err := tx.Model(&admin).Association("Roles").Append(&adminRole); err != nil {
			return err
		}

		settings := map[string]string{
			"system.initialized": "true",
			"system.name":        systemName,
			"system.timezone":    timezone,
		}
		for key, value := range settings {
			setting := models.SystemSetting{Key: key}
			if err := tx.Where("key = ?", key).Assign(models.SystemSetting{Value: value}).FirstOrCreate(&setting).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
