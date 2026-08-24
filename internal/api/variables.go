package api

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/models"
	"tsumugi-industry/internal/plc"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var variableTypes = map[string]bool{"BOOL": true, "INT": true, "UINT": true, "DINT": true, "UDINT": true, "WORD": true, "DWORD": true, "REAL": true, "LREAL": true, "STRING": true, "BYTE": true}
var variableAccessModes = map[string]bool{"read": true, "write": true, "read_write": true}

type variableRequest struct {
	Name             string   `json:"name" binding:"required,max=128"`
	Description      string   `json:"description" binding:"max=255"`
	PLCID            uint     `json:"plc_id" binding:"required"`
	Address          string   `json:"address" binding:"required,max=128"`
	DataType         string   `json:"data_type" binding:"required"`
	AccessMode       string   `json:"access_mode" binding:"required"`
	DefaultValue     string   `json:"default_value"`
	Unit             string   `json:"unit" binding:"max=32"`
	MinValue         *float64 `json:"min_value"`
	MaxValue         *float64 `json:"max_value"`
	EnumValues       string   `json:"enum_values"`
	ConditionAllowed *bool    `json:"condition_allowed"`
	FlowWriteAllowed *bool    `json:"flow_write_allowed"`
	Dangerous        bool     `json:"dangerous"`
	FreshnessSeconds int      `json:"freshness_seconds"`
}

func (r *Router) variables(c *gin.Context) {
	var items []models.PLCVariable
	query := applyTableQuery(r.db.Model(&models.PLCVariable{}).Preload("PLC"), c,
		map[string]string{"id": "id", "name": "name", "plc_id": "plc_id", "address": "address", "data_type": "data_type", "access_mode": "access_mode", "quality": "quality", "communication_state": "communication_state"},
		map[string]string{"filter_name": "name", "filter_address": "address", "filter_data_type": "data_type", "filter_access_mode": "access_mode", "filter_quality": "quality", "filter_communication_state": "communication_state"}, "id")
	query, page := paginate(query, c)
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for index := range items {
		refreshVariableQuality(&items[index])
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "data_types": sortedVariableTypes()})
}

func (r *Router) createVariable(c *gin.Context) {
	var request variableRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateVariableRequest(request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var plcRecord models.PLC
	if err := r.db.First(&plcRecord, request.PLCID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PLC not found"})
		return
	}
	conditionAllowed := true
	if request.ConditionAllowed != nil {
		conditionAllowed = *request.ConditionAllowed
	}
	flowWriteAllowed := false
	if request.FlowWriteAllowed != nil {
		flowWriteAllowed = *request.FlowWriteAllowed
	}
	freshness := request.FreshnessSeconds
	if freshness <= 0 {
		freshness = 10
	}
	user := auth.CurrentUser(c)
	item := models.PLCVariable{Name: strings.TrimSpace(request.Name), Description: request.Description, PLCID: plcRecord.ID, Address: strings.TrimSpace(request.Address), DataType: strings.ToUpper(request.DataType), AccessMode: strings.ToLower(request.AccessMode), DefaultValue: request.DefaultValue, Unit: request.Unit, MinValue: request.MinValue, MaxValue: request.MaxValue, EnumValues: request.EnumValues, ConditionAllowed: conditionAllowed, FlowWriteAllowed: flowWriteAllowed, Dangerous: request.Dangerous, FreshnessSeconds: freshness, Quality: "unknown", CommunicationState: "unknown"}
	if user != nil {
		item.CreatedByID = &user.ID
		item.UpdatedByID = &user.ID
	}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "variable name already exists"})
		return
	}
	r.recordEvent(user, "create", "plc_variable", "创建 PLC 变量 "+item.Name, c)
	r.db.Preload("PLC").First(&item, item.ID)
	c.JSON(http.StatusCreated, gin.H{"variable": item})
}

func (r *Router) updateVariable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid variable id"})
		return
	}
	var request variableRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateVariableRequest(request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.PLCVariable
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variable not found"})
		return
	}
	var controller models.PLC
	if err := r.db.First(&controller, request.PLCID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PLC not found"})
		return
	}
	if err := r.db.Model(&item).Updates(map[string]any{"name": strings.TrimSpace(request.Name), "description": request.Description, "plc_id": request.PLCID, "address": strings.TrimSpace(request.Address), "data_type": strings.ToUpper(request.DataType), "access_mode": strings.ToLower(request.AccessMode), "default_value": request.DefaultValue, "unit": request.Unit, "min_value": request.MinValue, "max_value": request.MaxValue, "enum_values": request.EnumValues, "condition_allowed": valueOrDefault(request.ConditionAllowed, true), "flow_write_allowed": valueOrDefault(request.FlowWriteAllowed, false), "dangerous": request.Dangerous, "freshness_seconds": maxInt(request.FreshnessSeconds, 10), "updated_by_id": userID(auth.CurrentUser(c))}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to update variable"})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "update", "plc_variable", "修改 PLC 变量 "+item.Name, c)
	r.db.Preload("PLC").First(&item, id)
	c.JSON(http.StatusOK, gin.H{"variable": item})
}

func (r *Router) deleteVariable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid variable id"})
		return
	}
	var item models.PLCVariable
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variable not found"})
		return
	}
	if err := r.db.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "delete", "plc_variable", "删除 PLC 变量 "+item.Name, c)
	c.Status(http.StatusNoContent)
}

func (r *Router) readVariable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid variable id"})
		return
	}
	var item models.PLCVariable
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variable not found"})
		return
	}
	adapter, err := r.variableAdapter(item)
	if err != nil {
		markVariableOffline(r.db, &item)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer adapter.Close(context.Background())
	values, err := adapter.Read(c.Request.Context(), []plc.ReadRequest{{Address: item.Address, Length: variableReadLength(item.DataType), DataType: item.DataType}})
	if err != nil || len(values) == 0 {
		markVariableOffline(r.db, &item)
		if err == nil {
			err = errors.New("PLC returned no value")
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	decoded, err := decodePLCValue(values[0].Value, item.DataType, item.Address)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	encoded, _ := json.Marshal(decoded)
	r.db.Model(&item).Updates(map[string]any{"current_value": string(encoded), "last_updated_at": &now, "quality": "good", "communication_state": "online"})
	item.CurrentValue = string(encoded)
	item.LastUpdatedAt = &now
	item.Quality = "good"
	item.CommunicationState = "online"
	c.JSON(http.StatusOK, gin.H{"variable": item, "value": decoded})
}

func (r *Router) writeVariable(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid variable id"})
		return
	}
	var payload struct {
		Value            any  `json:"value"`
		ConfirmDangerous bool `json:"confirm_dangerous"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.PLCVariable
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variable not found"})
		return
	}
	if item.AccessMode == "read" || !item.FlowWriteAllowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "variable is not writable"})
		return
	}
	if item.Dangerous && !payload.ConfirmDangerous {
		c.JSON(http.StatusConflict, gin.H{"error": "dangerous variable requires explicit confirmation"})
		return
	}
	if err := validateVariableValue(item, payload.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adapter, err := r.variableAdapter(item)
	if err != nil {
		markVariableOffline(r.db, &item)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer adapter.Close(context.Background())
	if err := adapter.Write(c.Request.Context(), []plc.WriteRequest{{Address: item.Address, DataType: item.DataType, Value: payload.Value}}); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	oldValue := item.CurrentValue
	encoded, _ := json.Marshal(payload.Value)
	now := time.Now()
	r.db.Model(&item).Updates(map[string]any{"current_value": string(encoded), "last_updated_at": &now, "quality": "good", "communication_state": "online"})
	r.recordEvent(auth.CurrentUser(c), "write", "plc_variable", fmt.Sprintf("写入 PLC 变量 %s：%s → %s", item.Name, oldValue, string(encoded)), c)
	c.JSON(http.StatusOK, gin.H{"variable": item, "value": payload.Value})
}

func (r *Router) variableAdapter(item models.PLCVariable) (plc.Adapter, error) {
	var controller models.PLC
	if err := r.db.First(&controller, item.PLCID).Error; err != nil {
		return nil, fmt.Errorf("PLC not found: %w", err)
	}
	if r.plcFactory == nil {
		return nil, errors.New("PLC adapter factory is unavailable")
	}
	return r.plcFactory.Create(plc.Config{ID: controller.ID, Code: controller.Code, Name: controller.Name, Protocol: plc.Protocol(controller.Protocol), Host: controller.Host, Port: controller.Port, Rack: controller.Rack, Slot: controller.Slot, UnitID: controller.UnitID})
}

func validateVariableRequest(request variableRequest) error {
	dataType := strings.ToUpper(strings.TrimSpace(request.DataType))
	access := strings.ToLower(strings.TrimSpace(request.AccessMode))
	if !variableTypes[dataType] {
		return fmt.Errorf("unsupported data type %s", dataType)
	}
	if !variableAccessModes[access] {
		return fmt.Errorf("unsupported access mode %s", access)
	}
	if request.MinValue != nil && request.MaxValue != nil && *request.MinValue > *request.MaxValue {
		return errors.New("min_value cannot exceed max_value")
	}
	if request.FreshnessSeconds < 0 {
		return errors.New("freshness_seconds cannot be negative")
	}
	return nil
}

func validateVariableValue(item models.PLCVariable, value any) error {
	if item.MinValue == nil && item.MaxValue == nil {
		return nil
	}
	number, ok := value.(float64)
	if !ok {
		return errors.New("numeric range requires a numeric value")
	}
	if item.MinValue != nil && number < *item.MinValue || item.MaxValue != nil && number > *item.MaxValue {
		return errors.New("value is outside the configured range")
	}
	return nil
}

func variableReadLength(dataType string) int {
	switch strings.ToUpper(dataType) {
	case "BOOL", "BYTE":
		return 1
	case "INT", "UINT", "WORD":
		return 2
	case "DINT", "UDINT", "DWORD", "REAL":
		return 4
	case "LREAL":
		return 8
	default:
		return 256
	}
}

func decodePLCValue(value any, dataType, address string) (any, error) {
	data, ok := value.([]byte)
	if !ok {
		return value, nil
	}
	if len(data) == 0 {
		return nil, errors.New("PLC returned empty value")
	}
	dataType = strings.ToUpper(dataType)
	switch dataType {
	case "BOOL":
		bit := 0
		if dot := strings.LastIndex(address, "."); dot >= 0 {
			bit, _ = strconv.Atoi(address[dot+1:])
		}
		return data[0]&(1<<bit) != 0, nil
	case "BYTE":
		return data[0], nil
	case "INT":
		if len(data) < 2 {
			return nil, errors.New("INT requires 2 bytes")
		}
		return int16(binary.BigEndian.Uint16(data[:2])), nil
	case "UINT", "WORD":
		if len(data) < 2 {
			return nil, errors.New("UINT requires 2 bytes")
		}
		return binary.BigEndian.Uint16(data[:2]), nil
	case "DINT":
		if len(data) < 4 {
			return nil, errors.New("DINT requires 4 bytes")
		}
		return int32(binary.BigEndian.Uint32(data[:4])), nil
	case "UDINT", "DWORD":
		if len(data) < 4 {
			return nil, errors.New("UDINT requires 4 bytes")
		}
		return binary.BigEndian.Uint32(data[:4]), nil
	case "REAL":
		if len(data) < 4 {
			return nil, errors.New("REAL requires 4 bytes")
		}
		return math.Float32frombits(binary.BigEndian.Uint32(data[:4])), nil
	case "LREAL":
		if len(data) < 8 {
			return nil, errors.New("LREAL requires 8 bytes")
		}
		return math.Float64frombits(binary.BigEndian.Uint64(data[:8])), nil
	case "STRING":
		return strings.TrimRight(string(data), "\x00"), nil
	default:
		return nil, fmt.Errorf("unsupported data type %s", dataType)
	}
}

func refreshVariableQuality(item *models.PLCVariable) {
	if item.LastUpdatedAt == nil {
		if item.Quality == "" {
			item.Quality = "unknown"
		}
		return
	}
	if item.FreshnessSeconds > 0 && time.Since(*item.LastUpdatedAt) > time.Duration(item.FreshnessSeconds)*time.Second {
		item.Quality = "stale"
		item.CommunicationState = "stale"
	}
}

func markVariableOffline(db *gorm.DB, item *models.PLCVariable) {
	db.Model(item).Updates(map[string]any{"quality": "bad", "communication_state": "offline"})
	item.Quality = "bad"
	item.CommunicationState = "offline"
}

func sortedVariableTypes() []string {
	result := make([]string, 0, len(variableTypes))
	for key := range variableTypes {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
func valueOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
func maxInt(value, fallback int) int {
	if value < 1 {
		return fallback
	}
	return value
}

func userID(user *models.User) *uint {
	if user == nil {
		return nil
	}
	return &user.ID
}
