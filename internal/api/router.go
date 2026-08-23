package api

import (
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/models"
	"tsumugi-industry/internal/system"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	db   *gorm.DB
	auth *auth.Manager
}

func NewRouter(db *gorm.DB, manager *auth.Manager, frontend fs.FS) *gin.Engine {
	service := &Router{db: db, auth: manager}
	router := gin.New()
	router.RedirectTrailingSlash = false
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/health", service.health)
	router.GET("/api/setup/status", service.setupStatus)
	router.POST("/api/setup", service.setup)
	router.POST("/api/auth/login", service.login)

	protected := router.Group("/api")
	protected.Use(manager.Middleware(), service.audit)
	protected.GET("/auth/me", service.me)
	protected.GET("/dashboard/summary", auth.RequirePermission("dashboard.read"), service.dashboard)
	protected.GET("/users", auth.RequirePermission("users.read"), service.users)
	protected.POST("/users", auth.RequirePermission("users.write"), service.createUser)
	protected.GET("/roles", auth.RequirePermission("roles.read"), service.roles)
	protected.POST("/roles", auth.RequirePermission("roles.write"), service.createRole)
	protected.GET("/permissions", auth.RequirePermission("roles.read"), service.permissions)
	protected.GET("/settings", auth.RequirePermission("system.settings"), service.settings)
	protected.GET("/devices", auth.RequirePermission("devices.read"), service.devices)
	protected.POST("/devices", auth.RequirePermission("devices.write"), service.createDevice)
	protected.PATCH("/devices/:id/status", auth.RequirePermission("devices.write"), service.updateDeviceStatus)
	protected.GET("/alarms", auth.RequirePermission("alarms.read"), service.alarms)
	protected.POST("/alarms/:id/ack", auth.RequirePermission("alarms.operate"), service.ackAlarm)
	protected.GET("/audit", auth.RequirePermission("audit.read"), service.auditLogs)

	serveFrontend := func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			serveIndex(c, frontend)
			return
		}
		if _, err := fs.Stat(frontend, path); err != nil {
			serveIndex(c, frontend)
			return
		}
		c.FileFromFS(path, http.FS(frontend))
	}
	router.GET("/", serveFrontend)
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}
		serveFrontend(c)
	})
	return router
}

func serveIndex(c *gin.Context, frontend fs.FS) {
	data, err := fs.ReadFile(frontend, "index.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "frontend entry is unavailable"})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (r *Router) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "tsumugi-industry"})
}

func (r *Router) setupStatus(c *gin.Context) {
	initialized, err := system.IsInitialized(r.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"initialized": initialized})
}

func (r *Router) setup(c *gin.Context) {
	var request system.SetupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := system.Initialize(r.db, request); err != nil {
		if errors.Is(err, system.ErrAlreadyInitialized) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "system initialized"})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (r *Router) login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, user, err := r.auth.Authenticate(request.Username, request.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (r *Router) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": auth.CurrentUser(c)})
}

func (r *Router) dashboard(c *gin.Context) {
	var users, roles, devicesOnline, activeAlarms int64
	r.db.Model(&models.User{}).Count(&users)
	r.db.Model(&models.Role{}).Count(&roles)
	r.db.Model(&models.Device{}).Where("status = ?", "online").Count(&devicesOnline)
	r.db.Model(&models.Alarm{}).Where("status IN ?", []string{"active", "acknowledged"}).Count(&activeAlarms)
	c.JSON(http.StatusOK, gin.H{
		"users":          users,
		"roles":          roles,
		"devices_online": devicesOnline,
		"alerts":         activeAlarms,
	})
}

func (r *Router) users(c *gin.Context) {
	var users []models.User
	if err := r.db.Preload("Roles").Order("id asc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users})
}

type createUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"max=128"`
	Email       string `json:"email" binding:"omitempty,email"`
	RoleIDs     []uint `json:"role_ids"`
}

func (r *Router) createUser(c *gin.Context) {
	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := auth.HashPassword(request.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}
	user := models.User{Username: strings.TrimSpace(request.Username), PasswordHash: hash, DisplayName: strings.TrimSpace(request.DisplayName), Email: strings.TrimSpace(request.Email), IsActive: true}
	if err := r.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	if len(request.RoleIDs) > 0 {
		var roles []models.Role
		if err := r.db.Find(&roles, request.RoleIDs).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := r.db.Model(&user).Association("Roles").Append(&roles); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	r.db.Preload("Roles").First(&user, user.ID)
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (r *Router) roles(c *gin.Context) {
	var roles []models.Role
	if err := r.db.Preload("Permissions").Order("id asc").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": roles})
}

type createRoleRequest struct {
	Name          string `json:"name" binding:"required,max=64"`
	DisplayName   string `json:"display_name" binding:"required,max=128"`
	Description   string `json:"description" binding:"max=255"`
	PermissionIDs []uint `json:"permission_ids"`
}

func (r *Router) createRole(c *gin.Context) {
	var request createRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := models.Role{Name: strings.TrimSpace(request.Name), DisplayName: strings.TrimSpace(request.DisplayName), Description: strings.TrimSpace(request.Description)}
	if err := r.db.Create(&role).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "role name already exists"})
		return
	}
	if len(request.PermissionIDs) > 0 {
		var permissions []models.Permission
		if err := r.db.Find(&permissions, request.PermissionIDs).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := r.db.Model(&role).Association("Permissions").Append(&permissions); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	r.db.Preload("Permissions").First(&role, role.ID)
	c.JSON(http.StatusCreated, gin.H{"role": role})
}

func (r *Router) permissions(c *gin.Context) {
	var permissions []models.Permission
	if err := r.db.Order("code asc").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": permissions})
}

func (r *Router) settings(c *gin.Context) {
	var settings []models.SystemSetting
	if err := r.db.Where("key NOT LIKE ?", "auth.%").Order("key asc").Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": settings})
}

func (r *Router) devices(c *gin.Context) {
	var devices []models.Device
	query := r.db.Order("id asc")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR location LIKE ?", like, like, like)
	}
	if err := query.Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": devices})
}

type createDeviceRequest struct {
	Code     string `json:"code" binding:"required,max=64"`
	Name     string `json:"name" binding:"required,max=128"`
	Type     string `json:"type" binding:"max=64"`
	Location string `json:"location" binding:"max=160"`
	Status   string `json:"status" binding:"max=32"`
	Metadata string `json:"metadata"`
}

func (r *Router) createDevice(c *gin.Context) {
	var request createDeviceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := strings.TrimSpace(request.Status)
	if status == "" {
		status = "offline"
	}
	if !validDeviceStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be online, offline, or maintenance"})
		return
	}
	device := models.Device{Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name), Type: strings.TrimSpace(request.Type), Location: strings.TrimSpace(request.Location), Status: status, Metadata: request.Metadata}
	if status == "online" {
		now := time.Now()
		device.LastSeenAt = &now
	}
	if err := r.db.Create(&device).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "device code already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"device": device})
}

func (r *Router) updateDeviceStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	var payload struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || !validDeviceStatus(payload.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be online, offline, or maintenance"})
		return
	}
	var device models.Device
	if err := r.db.First(&device, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	device.Status = payload.Status
	if payload.Status == "online" {
		now := time.Now()
		device.LastSeenAt = &now
	}
	if err := r.db.Save(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device": device})
}

func validDeviceStatus(status string) bool {
	return status == "online" || status == "offline" || status == "maintenance"
}

func (r *Router) alarms(c *gin.Context) {
	var alarms []models.Alarm
	query := r.db.Preload("Device").Order("occurred_at desc")
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Limit(100).Find(&alarms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": alarms})
}

func (r *Router) ackAlarm(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alarm id"})
		return
	}
	var alarm models.Alarm
	if err := r.db.First(&alarm, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alarm not found"})
		return
	}
	now := time.Now()
	alarm.Status = "acknowledged"
	alarm.AcknowledgedAt = &now
	if err := r.db.Save(&alarm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alarm": alarm})
}

func (r *Router) auditLogs(c *gin.Context) {
	var logs []models.AuditLog
	if err := r.db.Order("created_at desc").Limit(200).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": logs})
}

func (r *Router) audit(c *gin.Context) {
	c.Next()
	user := auth.CurrentUser(c)
	if user == nil || !strings.HasPrefix(c.Request.URL.Path, "/api/") {
		return
	}
	resource := strings.TrimPrefix(c.Request.URL.Path, "/api/")
	if index := strings.IndexByte(resource, '/'); index >= 0 {
		resource = resource[:index]
	}
	action := strings.ToLower(c.Request.Method)
	if action == http.MethodGet {
		action = "read"
	}
	_ = r.db.Create(&models.AuditLog{UserID: &user.ID, Username: user.Username, Action: action, Resource: resource, Method: c.Request.Method, Path: c.Request.URL.Path, StatusCode: c.Writer.Status(), IP: c.ClientIP()}).Error
}
