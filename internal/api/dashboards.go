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
	Name            string                   `json:"name" binding:"required,max=128"`
	Description     string                   `json:"description"`
	TimeRangeHours  int                      `json:"time_range_hours"`
	StatusRunning   string                   `json:"status_running"`
	StatusIdle      string                   `json:"status_idle"`
	BackgroundColor string                   `json:"background_color"`
	Definition      any                      `json:"definition"`
	Widgets         []dashboardWidgetRequest `json:"widgets"`
}
type dashboardWidgetRequest struct {
	ID         uint    `json:"id"`
	WidgetType string  `json:"widget_type"`
	Title      string  `json:"title"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Config     any     `json:"config"`
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
	definition, _ := json.Marshal(req.Definition)
	item := models.Dashboard{Name: strings.TrimSpace(req.Name), Description: req.Description, TimeRangeHours: maxPositive(req.TimeRangeHours, 24), StatusRunning: req.StatusRunning, StatusIdle: req.StatusIdle, BackgroundColor: req.BackgroundColor, Definition: string(definition)}
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
	item.BackgroundColor = req.BackgroundColor
	definition, _ := json.Marshal(req.Definition)
	item.Definition = string(definition)
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
		var existing []models.DashboardWidget
		if err := tx.Where("dashboard_id = ?", item.ID).Find(&existing).Error; err != nil {
			return err
		}
		existingByID := make(map[uint]*models.DashboardWidget, len(existing))
		for index := range existing {
			existingByID[existing[index].ID] = &existing[index]
		}
		keptIDs := make([]uint, 0, len(widgets))
		for _, widget := range widgets {
			config, err := marshalDashboardWidgetConfig(widget.Config)
			if err != nil {
				return err
			}
			if current, ok := existingByID[widget.ID]; ok {
				current.WidgetType = widget.WidgetType
				current.Title = widget.Title
				current.X = widget.X
				current.Y = widget.Y
				current.Width = maxPositiveFloat(widget.Width, 3)
				current.Height = maxPositiveFloat(widget.Height, 2)
				current.Config = string(config)
				if err := tx.Save(current).Error; err != nil {
					return err
				}
				keptIDs = append(keptIDs, current.ID)
				continue
			}
			created := &models.DashboardWidget{DashboardID: item.ID, WidgetType: widget.WidgetType, Title: widget.Title, X: widget.X, Y: widget.Y, Width: maxPositiveFloat(widget.Width, 3), Height: maxPositiveFloat(widget.Height, 2), Config: string(config)}
			if err := tx.Create(created).Error; err != nil {
				return err
			}
			keptIDs = append(keptIDs, created.ID)
		}
		if len(keptIDs) == 0 {
			return tx.Where("dashboard_id = ?", item.ID).Delete(&models.DashboardWidget{}).Error
		}
		return tx.Where("dashboard_id = ? AND id NOT IN ?", item.ID, keptIDs).Delete(&models.DashboardWidget{}).Error
	})
}

// marshalDashboardWidgetConfig keeps widget config as a JSON object even when
// it came back through the API as an already-encoded string. This prevents
// repeated saves from nesting the JSON and making text values unreadable.
func marshalDashboardWidgetConfig(value any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	for depth := 0; depth < 3; depth++ {
		raw, ok := value.(string)
		if !ok {
			break
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return []byte("{}"), nil
		}
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			break
		}
		value = decoded
		if value == nil {
			return []byte("{}"), nil
		}
	}
	return json.Marshal(value)
}

func maxPositive(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	return value
}

func maxPositiveFloat(value, fallback float64) float64 {
	if value < 1 {
		return fallback
	}
	return value
}
