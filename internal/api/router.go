package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/maintenance"
	"tsumugi-industry/internal/models"
	"tsumugi-industry/internal/plc"
	"tsumugi-industry/internal/system"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Router struct {
	db          *gorm.DB
	auth        *auth.Manager
	maintenance *maintenance.Service
	plcFactory  *plc.Factory
}

func NewRouter(db *gorm.DB, manager *auth.Manager, maintenanceService *maintenance.Service, plcFactory *plc.Factory, frontend fs.FS) *gin.Engine {
	service := &Router{db: db, auth: manager, maintenance: maintenanceService, plcFactory: plcFactory}
	router := gin.New()
	router.RedirectTrailingSlash = false
	// Business audit events are persisted explicitly by handlers. Do not persist
	// or emit raw HTTP access logs as application audit records.
	router.Use(gin.Recovery())

	router.GET("/api/health", service.health)
	router.GET("/api/setup/status", service.setupStatus)
	router.POST("/api/setup", service.setup)
	router.POST("/api/auth/login", service.login)

	protected := router.Group("/api")
	protected.Use(manager.Middleware())
	protected.GET("/auth/me", service.me)
	protected.GET("/dashboard/summary", auth.RequirePermission("dashboard.read"), service.dashboard)
	protected.GET("/users", auth.RequirePermission("users.read"), service.users)
	protected.POST("/users", auth.RequirePermission("users.write"), service.createUser)
	protected.PUT("/users/:id", auth.RequirePermission("users.write"), service.updateUser)
	protected.DELETE("/users/:id", auth.RequirePermission("users.write"), service.deleteUser)
	protected.GET("/roles", auth.RequirePermission("roles.read"), service.roles)
	protected.POST("/roles", auth.RequirePermission("roles.write"), service.createRole)
	protected.PUT("/roles/:id", auth.RequirePermission("roles.write"), service.updateRole)
	protected.DELETE("/roles/:id", auth.RequirePermission("roles.write"), service.deleteRole)
	protected.GET("/permissions", auth.RequirePermission("roles.read"), service.permissions)
	protected.GET("/settings", auth.RequirePermission("system.settings"), service.settings)
	protected.PUT("/settings/:key", auth.RequirePermission("system.settings"), service.updateSetting)
	protected.GET("/devices", auth.RequirePermission("devices.read"), service.devices)
	protected.POST("/devices", auth.RequirePermission("devices.write"), service.createDevice)
	protected.PUT("/devices/:id", auth.RequirePermission("devices.write"), service.updateDevice)
	protected.DELETE("/devices/:id", auth.RequirePermission("devices.write"), service.deleteDevice)
	protected.PATCH("/devices/:id/status", auth.RequirePermission("devices.write"), service.updateDeviceStatus)
	protected.GET("/plcs", auth.RequirePermission("plcs.read"), service.plcs)
	protected.POST("/plcs", auth.RequirePermission("plcs.write"), service.createPLC)
	protected.PUT("/plcs/:id", auth.RequirePermission("plcs.write"), service.updatePLC)
	protected.DELETE("/plcs/:id", auth.RequirePermission("plcs.write"), service.deletePLC)
	protected.GET("/plcs/:id/status", auth.RequirePermission("plcs.read"), service.plcStatus)
	protected.POST("/plcs/:id/query", auth.RequirePermission("plcs.read"), service.queryPLC)
	protected.GET("/variables", auth.RequirePermission("variables.read"), service.variables)
	protected.POST("/variables", auth.RequirePermission("variables.write"), service.createVariable)
	protected.PUT("/variables/:id", auth.RequirePermission("variables.write"), service.updateVariable)
	protected.DELETE("/variables/:id", auth.RequirePermission("variables.write"), service.deleteVariable)
	protected.POST("/variables/:id/read", auth.RequirePermission("variables.read"), service.readVariable)
	protected.POST("/variables/:id/write", auth.RequirePermission("variables.write"), service.writeVariable)
	protected.GET("/monitor-items", auth.RequirePermission("variables.read"), service.monitorItems)
	protected.POST("/monitor-items", auth.RequirePermission("variables.write"), service.createMonitorItem)
	protected.PUT("/monitor-items/:id", auth.RequirePermission("variables.write"), service.updateMonitorItem)
	protected.DELETE("/monitor-items/:id", auth.RequirePermission("variables.write"), service.deleteMonitorItem)
	protected.GET("/monitor-items/:id/records", auth.RequirePermission("variables.read"), service.monitorRecords)
	protected.POST("/monitor-items/:id/sample", auth.RequirePermission("variables.read"), service.sampleMonitorItem)
	protected.GET("/dashboards", auth.RequirePermission("dashboard.read"), service.dashboards)
	protected.GET("/dashboards/:id", auth.RequirePermission("dashboard.read"), service.dashboardByID)
	protected.POST("/dashboards", auth.RequirePermission("dashboard.read"), service.createDashboard)
	protected.PUT("/dashboards/:id", auth.RequirePermission("dashboard.read"), service.updateDashboard)
	protected.DELETE("/dashboards/:id", auth.RequirePermission("dashboard.read"), service.deleteDashboard)
	protected.GET("/flows", auth.RequirePermission("flows.read"), service.flows)
	protected.GET("/flows/:id", auth.RequirePermission("flows.read"), service.flow)
	protected.POST("/flows", auth.RequirePermission("flows.write"), service.createFlow)
	protected.PUT("/flows/:id", auth.RequirePermission("flows.write"), service.updateFlow)
	protected.DELETE("/flows/:id", auth.RequirePermission("flows.write"), service.deleteFlow)
	protected.POST("/flows/:id/validate", auth.RequirePermission("flows.write"), service.validateFlow)
	protected.POST("/flows/:id/publish", auth.RequirePermission("flows.write"), service.publishFlow)
	protected.POST("/flows/:id/new-version", auth.RequirePermission("flows.write"), service.newFlowVersion)
	protected.POST("/flows/:id/run", auth.RequirePermission("flows.operate"), service.startFlow)
	protected.GET("/flow-functions", auth.RequirePermission("flows.read"), service.flowFunctions)
	protected.GET("/flow-functions/:id", auth.RequirePermission("flows.read"), service.flowFunction)
	protected.POST("/flow-functions", auth.RequirePermission("flows.write"), service.createFlowFunction)
	protected.PUT("/flow-functions/:id", auth.RequirePermission("flows.write"), service.updateFlowFunction)
	protected.DELETE("/flow-functions/:id", auth.RequirePermission("flows.write"), service.deleteFlowFunction)
	protected.GET("/flow-runs", auth.RequirePermission("flows.read"), service.flowRuns)
	protected.GET("/flow-runs/:id", auth.RequirePermission("flows.read"), service.flowRun)
	protected.POST("/flow-runs/:id/:action", auth.RequirePermission("flows.operate"), service.controlFlowRun)
	protected.GET("/alarms", auth.RequirePermission("alarms.read"), service.alarms)
	protected.POST("/alarms/:id/ack", auth.RequirePermission("alarms.operate"), service.ackAlarm)
	protected.GET("/audit", auth.RequirePermission("audit.read"), service.auditLogs)
	protected.GET("/tasks", auth.RequirePermission("system.settings"), service.tasks)
	protected.PUT("/tasks/:id", auth.RequirePermission("system.settings"), service.updateTask)
	protected.POST("/maintenance/cleanup-logs", auth.RequirePermission("system.settings"), service.cleanupLogs)
	protected.GET("/backups", auth.RequirePermission("system.settings"), service.backups)
	protected.POST("/backups", auth.RequirePermission("system.settings"), service.createBackup)
	protected.POST("/backups/:id/restore", auth.RequirePermission("system.settings"), service.restoreBackup)
	protected.GET("/work-orders", auth.RequirePermission("production.read"), service.workOrders)
	protected.GET("/work-orders/:id", auth.RequirePermission("production.read"), service.workOrder)
	protected.POST("/work-orders", auth.RequirePermission("production.write"), service.createWorkOrder)
	protected.PUT("/work-orders/:id", auth.RequirePermission("production.write"), service.updateWorkOrder)
	protected.DELETE("/work-orders/:id", auth.RequirePermission("production.write"), service.deleteWorkOrder)
	protected.POST("/work-orders/:id/release", auth.RequirePermission("production.operate"), service.releaseWorkOrder)
	protected.POST("/work-orders/:id/start", auth.RequirePermission("production.operate"), service.startWorkOrder)
	protected.POST("/work-orders/:id/pause", auth.RequirePermission("production.operate"), service.pauseWorkOrder)
	protected.POST("/work-orders/:id/resume", auth.RequirePermission("production.operate"), service.resumeWorkOrder)
	protected.POST("/work-orders/:id/cancel", auth.RequirePermission("production.operate"), service.cancelWorkOrder)
	protected.POST("/work-orders/:id/complete", auth.RequirePermission("production.operate"), service.completeWorkOrder)
	protected.POST("/work-orders/:id/steps/:stepID/start", auth.RequirePermission("production.operate"), service.startWorkOrderStep)
	protected.POST("/work-orders/:id/steps/:stepID/complete", auth.RequirePermission("production.operate"), service.completeWorkOrderStep)
	protected.POST("/work-orders/:id/steps/:stepID/report", auth.RequirePermission("production.operate"), service.reportWorkOrderStep)
	go service.monitorLoop()

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
	r.recordEvent(user, "login", "auth", "登录系统", c)
}

func (r *Router) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": auth.CurrentUser(c)})
}

func (r *Router) dashboard(c *gin.Context) {
	var users, roles, devicesOnline, activeAlarms, workOrdersRunning, workOrdersWaiting int64
	r.db.Model(&models.User{}).Count(&users)
	r.db.Model(&models.Role{}).Count(&roles)
	r.db.Model(&models.Device{}).Where("status = ?", "online").Count(&devicesOnline)
	r.db.Model(&models.Alarm{}).Where("status IN ?", []string{"active", "acknowledged"}).Count(&activeAlarms)
	r.db.Model(&models.WorkOrder{}).Where("status = ?", workOrderRunning).Count(&workOrdersRunning)
	r.db.Model(&models.WorkOrder{}).Where("status IN ?", []string{workOrderReleased, workOrderPaused}).Count(&workOrdersWaiting)
	c.JSON(http.StatusOK, gin.H{
		"users":               users,
		"roles":               roles,
		"devices_online":      devicesOnline,
		"alerts":              activeAlarms,
		"work_orders_running": workOrdersRunning,
		"work_orders_waiting": workOrdersWaiting,
	})
}

func (r *Router) users(c *gin.Context) {
	var users []models.User
	query := applyTableQuery(r.db.Model(&models.User{}), c,
		map[string]string{"id": "id", "username": "username", "display_name": "display_name", "email": "email", "is_active": "is_active"},
		map[string]string{"filter_username": "username", "filter_display_name": "display_name", "filter_email": "email", "filter_is_active": "is_active"}, "id")
	query, page := paginate(query.Preload("Roles"), c)
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users, "page": page})
}

type createUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"max=128"`
	Email       string `json:"email" binding:"omitempty,email"`
	RoleIDs     []uint `json:"role_ids"`
}

type updateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Password    string `json:"password" binding:"omitempty,min=8"`
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
	r.recordEvent(auth.CurrentUser(c), "create", "user", "创建用户 "+user.Username, c)
}

func (r *Router) updateUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	var request updateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	updates := map[string]any{"username": strings.TrimSpace(request.Username), "display_name": strings.TrimSpace(request.DisplayName), "email": strings.TrimSpace(request.Email)}
	if request.Password != "" {
		hash, hashErr := auth.HashPassword(request.Password)
		if hashErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
			return
		}
		updates["password_hash"] = hash
	}
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		if request.RoleIDs != nil {
			var roles []models.Role
			if err := tx.Find(&roles, request.RoleIDs).Error; err != nil {
				return err
			}
			return tx.Model(&user).Association("Roles").Replace(&roles)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to update user"})
		return
	}
	r.db.Preload("Roles").First(&user, id)
	c.JSON(http.StatusOK, gin.H{"user": user})
	r.recordEvent(auth.CurrentUser(c), "update", "user", "修改用户 "+user.Username, c)
}

func (r *Router) deleteUser(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	current := auth.CurrentUser(c)
	if current != nil && current.ID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current user"})
		return
	}
	if result := r.db.Delete(&models.User{}, id); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	} else if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.Status(http.StatusNoContent)
	r.recordEvent(current, "delete", "user", "删除用户", c)
}

func (r *Router) roles(c *gin.Context) {
	var roles []models.Role
	query := applyTableQuery(r.db.Model(&models.Role{}), c,
		map[string]string{"id": "id", "name": "name", "display_name": "display_name"},
		map[string]string{"filter_name": "name", "filter_display_name": "display_name", "filter_description": "description"}, "id")
	query, page := paginate(query.Preload("Permissions"), c)
	if err := query.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": roles, "page": page})
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
	r.recordEvent(auth.CurrentUser(c), "create", "role", "创建角色 "+role.Name, c)
}

func (r *Router) updateRole(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var request createRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if err := r.db.Model(&role).Updates(map[string]any{"name": strings.TrimSpace(request.Name), "display_name": strings.TrimSpace(request.DisplayName), "description": strings.TrimSpace(request.Description)}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to update role"})
		return
	}
	if request.PermissionIDs != nil {
		var permissions []models.Permission
		if err := r.db.Find(&permissions, request.PermissionIDs).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := r.db.Model(&role).Association("Permissions").Replace(&permissions); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	r.db.Preload("Permissions").First(&role, id)
	c.JSON(http.StatusOK, gin.H{"role": role})
	r.recordEvent(auth.CurrentUser(c), "update", "role", "修改角色 "+role.Name, c)
}

func (r *Router) deleteRole(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	if role.Name == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete admin role"})
		return
	}
	if err := r.db.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
	r.recordEvent(auth.CurrentUser(c), "delete", "role", "删除角色 "+role.Name, c)
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
	query := applyTableQuery(r.db.Model(&models.SystemSetting{}).Where("key NOT LIKE ?", "auth.%"), c,
		map[string]string{"key": "key", "value": "value"},
		map[string]string{"filter_key": "key", "filter_value": "value"}, "key")
	query, page := paginate(query, c)
	if err := query.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": settings, "page": page})
}

func (r *Router) updateSetting(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if key == "" || strings.HasPrefix(key, "auth.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid setting key"})
		return
	}
	var payload struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	setting := models.SystemSetting{Key: key, Value: payload.Value}
	if err := r.db.Where("key = ?", key).Assign(models.SystemSetting{Value: payload.Value}).FirstOrCreate(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"setting": setting})
	r.recordEvent(auth.CurrentUser(c), "update", "setting", "修改设置 "+key, c)
}

func (r *Router) devices(c *gin.Context) {
	var devices []models.Device
	query := applyTableQuery(r.db.Model(&models.Device{}).Preload("PLC"), c,
		map[string]string{"id": "id", "code": "code", "name": "name", "type": "type", "location": "location", "status": "status"},
		map[string]string{"filter_code": "code", "filter_name": "name", "filter_type": "type", "filter_location": "location", "filter_status": "status", "filter_plc_id": "plc_id"}, "id")
	if plcID := strings.TrimSpace(c.Query("plc_id")); plcID != "" {
		query = query.Where("plc_id = ?", plcID)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("code LIKE ? OR name LIKE ? OR location LIKE ?", like, like, like)
	}
	query, page := paginate(query, c)
	if err := query.Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": devices, "page": page})
}

type createDeviceRequest struct {
	PLCID    *uint  `json:"plc_id"`
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
	device := models.Device{PLCID: request.PLCID, Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name), Type: strings.TrimSpace(request.Type), Location: strings.TrimSpace(request.Location), Status: status, Metadata: request.Metadata}
	if status == "online" {
		now := time.Now()
		device.LastSeenAt = &now
	}
	if err := r.db.Create(&device).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "device code already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"device": device})
	r.recordEvent(auth.CurrentUser(c), "create", "device", "接入设备 "+device.Name, c)
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
	r.recordEvent(auth.CurrentUser(c), "update", "device", "修改设备 "+device.Name, c)
}

func (r *Router) updateDevice(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	var request createDeviceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Status != "" && !validDeviceStatus(request.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device status"})
		return
	}
	var device models.Device
	if err := r.db.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	updates := map[string]any{"plc_id": request.PLCID, "code": strings.TrimSpace(request.Code), "name": strings.TrimSpace(request.Name), "type": strings.TrimSpace(request.Type), "location": strings.TrimSpace(request.Location), "metadata": request.Metadata}
	if request.Status != "" {
		updates["status"] = request.Status
		if request.Status == "online" {
			now := time.Now()
			updates["last_seen_at"] = &now
		}
	}
	if err := r.db.Model(&device).Updates(updates).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to update device"})
		return
	}
	r.db.First(&device, id)
	c.JSON(http.StatusOK, gin.H{"device": device})
}

func (r *Router) deleteDevice(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	if result := r.db.Delete(&models.Device{}, id); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	} else if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.Status(http.StatusNoContent)
	r.recordEvent(auth.CurrentUser(c), "delete", "device", "删除设备", c)
}

type plcRequest struct {
	Code     string `json:"code" binding:"required,max=64"`
	Name     string `json:"name" binding:"required,max=128"`
	Protocol string `json:"protocol" binding:"required,max=32"`
	Host     string `json:"host" binding:"max=128"`
	Port     int    `json:"port"`
	Rack     int    `json:"rack"`
	Slot     int    `json:"slot"`
	UnitID   byte   `json:"unit_id"`
	Status   string `json:"status" binding:"max=32"`
	Metadata string `json:"metadata"`
}

func (r *Router) plcs(c *gin.Context) {
	var plcs []models.PLC
	query := applyTableQuery(r.db.Model(&models.PLC{}), c,
		map[string]string{"id": "id", "code": "code", "name": "name", "protocol": "protocol", "host": "host", "port": "port", "status": "status"},
		map[string]string{"filter_code": "code", "filter_name": "name", "filter_protocol": "protocol", "filter_host": "host", "filter_status": "status"}, "id")
	query, page := paginate(query, c)
	if err := query.Find(&plcs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": plcs, "page": page})
}

func (r *Router) createPLC(c *gin.Context) {
	var request plcRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := request.Status
	if status == "" {
		status = "offline"
	}
	if !validPLCStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be online, offline, or maintenance"})
		return
	}
	plc := models.PLC{Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name), Protocol: strings.TrimSpace(request.Protocol), Host: strings.TrimSpace(request.Host), Port: request.Port, Rack: request.Rack, Slot: request.Slot, UnitID: request.UnitID, Status: status, Metadata: request.Metadata}
	if err := r.db.Create(&plc).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "PLC code already exists"})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "create", "plc", "创建 PLC "+plc.Name, c)
	c.JSON(http.StatusCreated, gin.H{"plc": plc})
}

func (r *Router) updatePLC(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PLC id"})
		return
	}
	var request plcRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Status != "" && !validPLCStatus(request.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PLC status"})
		return
	}
	var plc models.PLC
	if err := r.db.First(&plc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PLC not found"})
		return
	}
	updates := map[string]any{"code": strings.TrimSpace(request.Code), "name": strings.TrimSpace(request.Name), "protocol": strings.TrimSpace(request.Protocol), "host": strings.TrimSpace(request.Host), "port": request.Port, "rack": request.Rack, "slot": request.Slot, "unit_id": request.UnitID, "metadata": request.Metadata}
	if request.Status != "" {
		updates["status"] = request.Status
		if request.Status == "online" {
			now := time.Now()
			updates["last_seen_at"] = &now
		}
	}
	if err := r.db.Model(&plc).Updates(updates).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to update PLC"})
		return
	}
	r.db.First(&plc, id)
	r.recordEvent(auth.CurrentUser(c), "update", "plc", "修改 PLC "+plc.Name, c)
	c.JSON(http.StatusOK, gin.H{"plc": plc})
}

func (r *Router) deletePLC(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PLC id"})
		return
	}
	var count int64
	r.db.Model(&models.Device{}).Where("plc_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "PLC has devices; reassign them before deletion"})
		return
	}
	var plc models.PLC
	if err := r.db.First(&plc, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PLC not found"})
		return
	}
	if err := r.db.Delete(&plc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "delete", "plc", "删除 PLC "+plc.Name, c)
	c.Status(http.StatusNoContent)
}

func validPLCStatus(status string) bool {
	return status == "online" || status == "offline" || status == "maintenance"
}

type plcQueryRequest struct {
	Addresses []plc.ReadRequest `json:"addresses" binding:"required,min=1,max=100"`
}

func (r *Router) plcStatus(c *gin.Context) {
	adapter, err := r.plcAdapter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer adapter.Close(context.Background())
	querier, ok := adapter.(plc.StatusQuerier)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "PLC protocol does not support CPU status query"})
		return
	}
	status, err := querier.CPUStatus(c.Request.Context())
	if err != nil {
		r.markPLCUnreachable(c)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	r.markPLCReachable(c)
	c.JSON(http.StatusOK, gin.H{"protocol": adapter.Protocol(), "status": status})
}

func (r *Router) queryPLC(c *gin.Context) {
	var request plcQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adapter, err := r.plcAdapter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer adapter.Close(context.Background())
	values, err := adapter.Read(c.Request.Context(), request.Addresses)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	r.markPLCReachable(c)
	c.JSON(http.StatusOK, gin.H{"protocol": adapter.Protocol(), "items": values})
}

func (r *Router) markPLCReachable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	now := time.Now()
	// Maintenance is an operator-controlled state and must not be overwritten
	// by an automatic connectivity check.
	r.db.Model(&models.PLC{}).Where("id = ? AND status <> ?", id, "maintenance").Updates(map[string]any{
		"status":       "online",
		"last_seen_at": &now,
	})
}

func (r *Router) markPLCUnreachable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	r.db.Model(&models.PLC{}).Where("id = ? AND status <> ?", id, "maintenance").Updates(map[string]any{"status": "offline"})
}

func (r *Router) plcAdapter(c *gin.Context) (plc.Adapter, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, errors.New("invalid PLC id")
	}
	var controller models.PLC
	if err := r.db.First(&controller, id).Error; err != nil {
		return nil, fmt.Errorf("PLC not found: %w", err)
	}
	if r.plcFactory == nil {
		return nil, errors.New("PLC adapter factory is unavailable")
	}
	return r.plcFactory.Create(plc.Config{ID: controller.ID, Code: controller.Code, Name: controller.Name, Protocol: plc.Protocol(controller.Protocol), Host: controller.Host, Port: controller.Port, Rack: controller.Rack, Slot: controller.Slot, UnitID: controller.UnitID})
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
	r.recordEvent(auth.CurrentUser(c), "acknowledge", "alarm", "确认告警", c)
}

func (r *Router) auditLogs(c *gin.Context) {
	var logs []models.AuditLog
	query := applyTableQuery(r.db.Model(&models.AuditLog{}), c,
		map[string]string{"id": "id", "username": "username", "action": "action", "resource": "resource", "detail": "detail", "created_at": "created_at"},
		map[string]string{"filter_username": "username", "filter_action": "action", "filter_resource": "resource", "filter_detail": "detail"}, "created_at")
	query, page := paginate(query, c)
	if err := query.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": logs, "page": page})
}

func parseID(c *gin.Context) (uint, error) {
	parsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	return uint(parsed), err
}

func (r *Router) recordEvent(user *models.User, action, resource, detail string, c *gin.Context) {
	if user == nil {
		return
	}
	_ = r.db.Create(&models.AuditLog{UserID: &user.ID, Username: user.Username, Action: action, Resource: resource, Detail: detail, CreatedAt: time.Now()}).Error
}

func (r *Router) tasks(c *gin.Context) {
	var tasks []models.ScheduledTask
	query := applyTableQuery(r.db.Model(&models.ScheduledTask{}), c, map[string]string{"id": "id", "name": "name", "task_type": "task_type", "enabled": "enabled", "last_status": "last_status"}, map[string]string{"filter_name": "name", "filter_task_type": "task_type", "filter_enabled": "enabled", "filter_last_status": "last_status"}, "id")
	query, page := paginate(query, c)
	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tasks, "page": page})
}

func (r *Router) updateTask(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}
	var payload struct {
		Enabled         *bool `json:"enabled"`
		IntervalSeconds int   `json:"interval_seconds"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var task models.ScheduledTask
	if err := r.db.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	updates := map[string]any{}
	if payload.Enabled != nil {
		updates["enabled"] = *payload.Enabled
	}
	if payload.IntervalSeconds > 0 {
		updates["interval_seconds"] = payload.IntervalSeconds
	}
	if err := r.db.Model(&task).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "update", "task", "修改定时任务 "+task.Name, c)
	c.JSON(http.StatusOK, gin.H{"task": task})
}

func (r *Router) cleanupLogs(c *gin.Context) {
	var payload struct {
		Days int `json:"days"`
	}
	_ = c.ShouldBindJSON(&payload)
	deleted, err := r.maintenance.CleanupAuditLogs(payload.Days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "cleanup", "audit", fmt.Sprintf("清理 %d 条业务日志", deleted), c)
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func (r *Router) backups(c *gin.Context) {
	var backups []models.Backup
	query := applyTableQuery(r.db.Model(&models.Backup{}), c, map[string]string{"id": "id", "name": "name", "status": "status", "created_at": "created_at"}, map[string]string{"filter_name": "name", "filter_status": "status", "filter_created_by": "created_by"}, "created_at")
	query, page := paginate(query, c)
	if err := query.Find(&backups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": backups, "page": page})
}

func (r *Router) createBackup(c *gin.Context) {
	backup, err := r.maintenance.CreateBackup(auth.CurrentUser(c).Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "create", "backup", "创建数据库备份 "+backup.Name, c)
	c.JSON(http.StatusCreated, gin.H{"backup": backup})
}

func (r *Router) restoreBackup(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup id"})
		return
	}
	var backup models.Backup
	if err := r.db.First(&backup, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if err := r.maintenance.RestoreBackup(backup); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "restore", "backup", "恢复数据库备份 "+backup.Name, c)
	c.JSON(http.StatusOK, gin.H{"message": "backup restored"})
}
