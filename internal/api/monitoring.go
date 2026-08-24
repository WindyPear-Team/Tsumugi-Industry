package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/models"
)

type monitorItemRequest struct {
	Name            string `json:"name" binding:"required,max=128"`
	PLCID           uint   `json:"plc_id" binding:"required"`
	VariableID      uint   `json:"variable_id" binding:"required"`
	DeviceID        *uint  `json:"device_id"`
	IntervalSeconds int    `json:"interval_seconds"`
	RetentionDays   int    `json:"retention_days"`
	Enabled         *bool  `json:"enabled"`
}

func (r *Router) monitorItems(c *gin.Context) {
	var items []models.MonitorItem
	query := r.db.Preload("PLC").Preload("Variable").Preload("Device").Order("id DESC")
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (r *Router) createMonitorItem(c *gin.Context) {
	var req monitorItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.IntervalSeconds < 1 {
		req.IntervalSeconds = 10
	}
	if req.RetentionDays < 1 {
		req.RetentionDays = 30
	}
	var variable models.PLCVariable
	if err := r.db.First(&variable, req.VariableID).Error; err != nil || variable.PLCID != req.PLCID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "变量不存在或不属于所选 PLC"})
		return
	}
	if req.DeviceID != nil {
		var device models.Device
		if err := r.db.First(&device, *req.DeviceID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在"})
			return
		}
	}
	item := models.MonitorItem{Name: strings.TrimSpace(req.Name), PLCID: req.PLCID, VariableID: req.VariableID, DeviceID: req.DeviceID, IntervalSeconds: req.IntervalSeconds, RetentionDays: req.RetentionDays, Enabled: boolValue(req.Enabled, true)}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "create", "monitor_item", "创建监控项 "+item.Name, c)
	r.db.Preload("PLC").Preload("Variable").Preload("Device").First(&item, item.ID)
	c.JSON(http.StatusCreated, gin.H{"item": item})
}

func (r *Router) updateMonitorItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor item id"})
		return
	}
	var req monitorItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.MonitorItem
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor item not found"})
		return
	}
	if req.IntervalSeconds < 1 {
		req.IntervalSeconds = item.IntervalSeconds
	}
	if req.RetentionDays < 1 {
		req.RetentionDays = item.RetentionDays
	}
	if req.DeviceID != nil {
		var device models.Device
		if err := r.db.First(&device, *req.DeviceID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "设备不存在"})
			return
		}
	}
	if err := r.db.Model(&item).Updates(map[string]any{"name": strings.TrimSpace(req.Name), "plc_id": req.PLCID, "variable_id": req.VariableID, "device_id": req.DeviceID, "interval_seconds": req.IntervalSeconds, "retention_days": req.RetentionDays, "enabled": boolValue(req.Enabled, item.Enabled)}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r.db.Preload("PLC").Preload("Variable").Preload("Device").First(&item, id)
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (r *Router) deleteMonitorItem(c *gin.Context) {
	if err := r.db.Delete(&models.MonitorItem{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.db.Where("monitor_item_id = ?", c.Param("id")).Delete(&models.MonitorRecord{})
	c.Status(http.StatusNoContent)
}

func (r *Router) monitorRecords(c *gin.Context) {
	id := c.Param("id")
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	if hours < 1 {
		hours = 24
	}
	var records []models.MonitorRecord
	if err := r.db.Where("monitor_item_id = ? AND recorded_at >= ?", id, time.Now().Add(-time.Duration(hours)*time.Hour)).Order("recorded_at ASC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": records})
}

func (r *Router) sampleMonitorItem(c *gin.Context) {
	var item models.MonitorItem
	if err := r.db.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "monitor item not found"})
		return
	}
	record, variable, err := r.sampleMonitor(&item)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": record, "value": variable})
}

func (r *Router) sampleMonitor(item *models.MonitorItem) (models.MonitorRecord, any, error) {
	variable, quality, err := r.readSemanticVariableByID(item.VariableID)
	if err != nil {
		return models.MonitorRecord{}, nil, err
	}
	encoded, _ := json.Marshal(variable)
	now := time.Now()
	record := models.MonitorRecord{MonitorItemID: item.ID, Value: string(encoded), Quality: quality, RecordedAt: now}
	if err := r.db.Create(&record).Error; err != nil {
		return models.MonitorRecord{}, nil, err
	}
	r.db.Model(item).Updates(map[string]any{"last_sampled_at": now})
	cutoff := now.AddDate(0, 0, -item.RetentionDays)
	r.db.Where("monitor_item_id = ? AND recorded_at < ?", item.ID, cutoff).Delete(&models.MonitorRecord{})
	return record, variable, nil
}

func (r *Router) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		var items []models.MonitorItem
		if r.db.Where("enabled = ?", true).Find(&items).Error != nil {
			continue
		}
		now := time.Now()
		for index := range items {
			item := &items[index]
			if item.LastSampledAt != nil && now.Sub(*item.LastSampledAt) < time.Duration(item.IntervalSeconds)*time.Second {
				continue
			}
			_, _, _ = r.sampleMonitor(item)
		}
	}
}

func (r *Router) readSemanticVariableByID(id uint) (any, string, error) {
	var item models.PLCVariable
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, "bad", err
	}
	return r.readSemanticVariable(item.Name)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
