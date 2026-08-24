package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"tsumugi-industry/internal/models"
)

type dashboardRequest struct {
	Name           string                   `json:"name" binding:"required,max=128"`
	Description    string                   `json:"description"`
	TimeRangeHours int                      `json:"time_range_hours"`
	StatusRunning  string                   `json:"status_running"`
	StatusIdle     string                   `json:"status_idle"`
	Widgets        []dashboardWidgetRequest `json:"widgets"`
}
type dashboardWidgetRequest struct {
	ID         uint   `json:"id"`
	WidgetType string `json:"widget_type"`
	Title      string `json:"title"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Config     any    `json:"config"`
}

func (r *Router) dashboards(c *gin.Context) {
	var items []models.Dashboard
	if err := r.db.Preload("Widgets").Order("id DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
func (r *Router) dashboardByID(c *gin.Context) {
	var item models.Dashboard
	if err := r.db.Preload("Widgets").First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dashboard": item})
}
func (r *Router) createDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := models.Dashboard{Name: strings.TrimSpace(req.Name), Description: req.Description, TimeRangeHours: maxPositive(req.TimeRangeHours, 24), StatusRunning: req.StatusRunning, StatusIdle: req.StatusIdle}
	if err := r.saveDashboard(&item, req.Widgets); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"dashboard": item})
}
func (r *Router) updateDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.Dashboard
	if err := r.db.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
		return
	}
	item.Name = strings.TrimSpace(req.Name)
	item.Description = req.Description
	item.TimeRangeHours = maxPositive(req.TimeRangeHours, 24)
	item.StatusRunning = req.StatusRunning
	item.StatusIdle = req.StatusIdle
	if err := r.saveDashboard(&item, req.Widgets); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r.db.Preload("Widgets").First(&item, item.ID)
	c.JSON(http.StatusOK, gin.H{"dashboard": item})
}
func (r *Router) deleteDashboard(c *gin.Context) {
	r.db.Delete(&models.Dashboard{}, c.Param("id"))
	c.Status(http.StatusNoContent)
}
func (r *Router) saveDashboard(item *models.Dashboard, widgets []dashboardWidgetRequest) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(item).Error; err != nil {
			return err
		}
		if err := tx.Where("dashboard_id = ?", item.ID).Delete(&models.DashboardWidget{}).Error; err != nil {
			return err
		}
		for _, widget := range widgets {
			config, err := json.Marshal(widget.Config)
			if err != nil {
				return err
			}
			if err := tx.Create(&models.DashboardWidget{DashboardID: item.ID, WidgetType: widget.WidgetType, Title: widget.Title, X: widget.X, Y: widget.Y, Width: maxPositive(widget.Width, 3), Height: maxPositive(widget.Height, 2), Config: string(config)}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func maxPositive(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	return value
}
