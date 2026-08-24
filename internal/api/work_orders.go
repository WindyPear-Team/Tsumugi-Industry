package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	workOrderDraft     = "draft"
	workOrderReleased  = "released"
	workOrderRunning   = "running"
	workOrderPaused    = "paused"
	workOrderCompleted = "completed"
	workOrderCancelled = "cancelled"
	stepPending        = "pending"
	stepReady          = "ready"
	stepRunning        = "running"
	stepPaused         = "paused"
	stepCompleted      = "completed"
)

type workOrderStepRequest struct {
	Sequence   int    `json:"sequence"`
	Code       string `json:"code" binding:"required,max=64"`
	Name       string `json:"name" binding:"required,max=128"`
	DeviceID   *uint  `json:"device_id"`
	PlannedQty int    `json:"planned_qty"`
	Notes      string `json:"notes"`
}

type workOrderRequest struct {
	Code             string                 `json:"code" binding:"max=64"`
	Name             string                 `json:"name" binding:"max=128"`
	ProductCode      string                 `json:"product_code" binding:"max=64"`
	ProductName      string                 `json:"product_name" binding:"max=128"`
	PlannedQty       int                    `json:"planned_qty"`
	FlowDefinitionID *uint                  `json:"flow_definition_id"`
	FlowVariables    map[string]any         `json:"flow_variables"`
	Priority         string                 `json:"priority" binding:"max=16"`
	ScheduledStart   *time.Time             `json:"scheduled_start"`
	ScheduledEnd     *time.Time             `json:"scheduled_end"`
	Notes            string                 `json:"notes"`
	Steps            []workOrderStepRequest `json:"steps" binding:"max=50"`
}

type stepCompleteRequest struct {
	PassedQty int             `json:"passed_qty"`
	FailedQty int             `json:"failed_qty"`
	Reason    string          `json:"reason" binding:"max=255"`
	Notes     string          `json:"notes"`
	Payload   json.RawMessage `json:"payload"`
}

func (r *Router) workOrders(c *gin.Context) {
	var orders []models.WorkOrder
	query := applyTableQuery(r.db.Model(&models.WorkOrder{}), c,
		map[string]string{
			"id": "id", "code": "code", "product_code": "product_code",
			"product_name": "product_name", "planned_qty": "planned_qty",
			"completed_qty": "completed_qty", "status": "status", "priority": "priority",
		},
		map[string]string{
			"filter_code": "code", "filter_product_code": "product_code",
			"filter_product_name": "product_name", "filter_status": "status",
			"filter_priority": "priority",
		}, "id")
	query, page := paginate(query, c)
	if err := query.Preload("Steps", func(db *gorm.DB) *gorm.DB { return db.Order("sequence ASC") }).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": orders, "page": page})
}

func (r *Router) workOrder(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order id"})
		return
	}
	order, err := loadWorkOrder(r.db, id, true)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "work order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work_order": order})
}

func (r *Router) createWorkOrder(c *gin.Context) {
	var request workOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.FlowDefinitionID != nil {
		if strings.TrimSpace(request.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if request.Code == "" {
			request.Code = "WO-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		if request.ProductName == "" {
			request.ProductName = request.Name
		}
		if request.ProductCode == "" {
			request.ProductCode = request.Code
		}
		if request.PlannedQty < 1 {
			request.PlannedQty = 1
		}
		if len(request.Steps) == 0 {
			request.Steps = []workOrderStepRequest{{Sequence: 1, Code: "FLOW", Name: request.Name, PlannedQty: request.PlannedQty}}
		}
	} else if strings.TrimSpace(request.Code) == "" || strings.TrimSpace(request.ProductCode) == "" || strings.TrimSpace(request.ProductName) == "" || request.PlannedQty < 1 || len(request.Steps) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "传统工单需要编码、产品、计划数量和工序；新工单请填写 name 与 flow_definition_id"})
		return
	}
	steps, err := normalizeWorkOrderSteps(request.Steps, request.PlannedQty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateWorkOrderSchedule(request.ScheduledStart, request.ScheduledEnd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	priority := strings.TrimSpace(request.Priority)
	if priority == "" {
		priority = "normal"
	}
	user := auth.CurrentUser(c)
	variablesJSON, _ := json.Marshal(request.FlowVariables)
	order := models.WorkOrder{
		Code: strings.TrimSpace(request.Code), ProductCode: strings.TrimSpace(request.ProductCode), ProductName: strings.TrimSpace(request.ProductName),
		PlannedQty: request.PlannedQty, Priority: priority, Status: workOrderDraft,
		ScheduledStart: request.ScheduledStart, ScheduledEnd: request.ScheduledEnd, Notes: request.Notes, Version: 1,
		FlowDefinitionID: request.FlowDefinitionID, FlowVariables: string(variablesJSON),
	}
	if user != nil {
		order.CreatedByID = &user.ID
	}
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := validateWorkOrderDevices(tx, steps); err != nil {
			return err
		}
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("work order code already exists: %w", err)
		}
		for _, requestStep := range steps {
			requestStep.WorkOrderID = order.ID
			if err := tx.Create(&requestStep).Error; err != nil {
				return err
			}
		}
		return appendProductionEvent(tx, &order, nil, "created", "", workOrderDraft, 0, 0, "", user)
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(user, "create", "work_order", "创建生产工单 "+order.Code, c)
	order, err = loadWorkOrder(r.db, order.ID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"work_order": order})
}

func (r *Router) updateWorkOrder(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order id"})
		return
	}
	var request workOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.FlowDefinitionID != nil {
		if strings.TrimSpace(request.Name) == "" {
			request.Name = request.ProductName
		}
		if request.Code == "" {
			request.Code = "WO-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		if request.ProductName == "" {
			request.ProductName = request.Name
		}
		if request.ProductCode == "" {
			request.ProductCode = request.Code
		}
		if request.PlannedQty < 1 {
			request.PlannedQty = 1
		}
		if len(request.Steps) == 0 {
			request.Steps = []workOrderStepRequest{{Sequence: 1, Code: "FLOW", Name: request.Name, PlannedQty: request.PlannedQty}}
		}
	}
	steps, err := normalizeWorkOrderSteps(request.Steps, request.PlannedQty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateWorkOrderSchedule(request.ScheduledStart, request.ScheduledEnd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var order models.WorkOrder
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != workOrderDraft {
			return fmt.Errorf("only draft work orders can be edited")
		}
		if err := validateWorkOrderDevices(tx, steps); err != nil {
			return err
		}
		priority := strings.TrimSpace(request.Priority)
		if priority == "" {
			priority = "normal"
		}
		variablesJSON, _ := json.Marshal(request.FlowVariables)
		updates := map[string]any{"code": strings.TrimSpace(request.Code), "product_code": strings.TrimSpace(request.ProductCode), "product_name": strings.TrimSpace(request.ProductName), "planned_qty": request.PlannedQty, "priority": priority, "scheduled_start": request.ScheduledStart, "scheduled_end": request.ScheduledEnd, "notes": request.Notes, "version": order.Version + 1, "flow_definition_id": request.FlowDefinitionID, "flow_variables": string(variablesJSON)}
		if result := tx.Model(&models.WorkOrder{}).Where("id = ? AND version = ?", order.ID, order.Version).Updates(updates); result.Error != nil {
			return result.Error
		} else if result.RowsAffected != 1 {
			return errors.New("work order was changed by another operator; reload and retry")
		}
		if err := tx.Where("work_order_id = ?", order.ID).Delete(&models.WorkOrderStep{}).Error; err != nil {
			return err
		}
		for _, requestStep := range steps {
			requestStep.WorkOrderID = order.ID
			if err := tx.Create(&requestStep).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "work order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "update", "work_order", "修改生产工单 "+request.Code, c)
	order, err = loadWorkOrder(r.db, id, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work_order": order})
}

func (r *Router) deleteWorkOrder(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order id"})
		return
	}
	var order models.WorkOrder
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		if order.Status != workOrderDraft && order.Status != workOrderCancelled {
			return errors.New("only draft or cancelled work orders can be deleted")
		}
		if err := tx.Where("work_order_id = ?", id).Delete(&models.ProductionEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("work_order_id = ?", id).Delete(&models.WorkOrderStep{}).Error; err != nil {
			return err
		}
		return tx.Delete(&order).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "work order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "delete", "work_order", "删除生产工单 "+order.Code, c)
	c.Status(http.StatusNoContent)
}

func (r *Router) releaseWorkOrder(c *gin.Context)  { r.transitionWorkOrder(c, "release") }
func (r *Router) startWorkOrder(c *gin.Context)    { r.transitionWorkOrder(c, "start") }
func (r *Router) pauseWorkOrder(c *gin.Context)    { r.transitionWorkOrder(c, "pause") }
func (r *Router) resumeWorkOrder(c *gin.Context)   { r.transitionWorkOrder(c, "resume") }
func (r *Router) cancelWorkOrder(c *gin.Context)   { r.transitionWorkOrder(c, "cancel") }
func (r *Router) completeWorkOrder(c *gin.Context) { r.transitionWorkOrder(c, "complete") }

func (r *Router) transitionWorkOrder(c *gin.Context, action string) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order id"})
		return
	}
	user := auth.CurrentUser(c)
	var order models.WorkOrder
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&order, id).Error; err != nil {
			return err
		}
		return applyWorkOrderTransition(tx, &order, action, user)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "work order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(user, "workflow", "work_order", workflowDetail(action, order.Code), c)
	order, err = loadWorkOrder(r.db, id, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work_order": order})
}

func (r *Router) startWorkOrderStep(c *gin.Context) {
	r.transitionWorkOrderStep(c, "start", "operator", "")
}
func (r *Router) completeWorkOrderStep(c *gin.Context) {
	r.transitionWorkOrderStep(c, "complete", "operator", "")
}

// reportWorkOrderStep is the machine/gateway-facing entry point. The
// Idempotency-Key is mandatory because PLC gateways retry messages when a
// connection drops; a retry must never count the same production result twice.
func (r *Router) reportWorkOrderStep(c *gin.Context) {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required and must be at most 128 characters"})
		return
	}
	source := strings.TrimSpace(c.GetHeader("X-Production-Source"))
	if source == "" {
		source = "gateway"
	}
	if source != "gateway" && source != "plc" && source != "operator" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Production-Source must be gateway, plc, or operator"})
		return
	}
	r.transitionWorkOrderStep(c, "complete", source, key)
}

func (r *Router) transitionWorkOrderStep(c *gin.Context, action, source, idempotencyKey string) {
	orderID, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order id"})
		return
	}
	stepID, err := strconv.ParseUint(c.Param("stepID"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid work order step id"})
		return
	}
	var request stepCompleteRequest
	if action == "complete" {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	user := auth.CurrentUser(c)
	if idempotencyKey != "" {
		var existing models.ProductionEvent
		lookupErr := r.db.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
		if lookupErr == nil {
			if existing.WorkOrderID != orderID || existing.WorkOrderStepID == nil || *existing.WorkOrderStepID != uint(stepID) {
				c.JSON(http.StatusConflict, gin.H{"error": "Idempotency-Key was already used for another production report"})
				return
			}
			order, loadErr := loadWorkOrder(r.db, orderID, true)
			if loadErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": loadErr.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"work_order": order, "idempotent": true})
			return
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": lookupErr.Error()})
			return
		}
	}
	var order models.WorkOrder
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		var step models.WorkOrderStep
		if err := tx.Where("id = ? AND work_order_id = ?", uint(stepID), order.ID).First(&step).Error; err != nil {
			return err
		}
		if action == "start" {
			if order.Status != workOrderRunning || step.Status != stepReady {
				return errors.New("only a ready step on a running work order can be started")
			}
			var running int64
			if err := tx.Model(&models.WorkOrderStep{}).Where("work_order_id = ? AND status = ?", order.ID, stepRunning).Count(&running).Error; err != nil {
				return err
			}
			if running > 0 {
				return errors.New("another work order step is already running")
			}
			now := time.Now()
			if err := tx.Model(&step).Updates(map[string]any{"status": stepRunning, "started_at": &now}).Error; err != nil {
				return err
			}
			if err := updateWorkOrderVersion(tx, &order, map[string]any{"current_sequence": step.Sequence}); err != nil {
				return err
			}
			return appendProductionEvent(tx, &order, &step, "step_started", stepReady, stepRunning, 0, 0, "", user)
		}
		if order.Status != workOrderRunning || step.Status != stepRunning {
			return errors.New("only a running step on a running work order can be completed")
		}
		if request.PassedQty < 0 || request.FailedQty < 0 || request.PassedQty+request.FailedQty == 0 {
			return errors.New("passed_qty plus failed_qty must be greater than zero")
		}
		if request.PassedQty+request.FailedQty > step.PlannedQty {
			return fmt.Errorf("step quantity cannot exceed planned quantity %d", step.PlannedQty)
		}
		if request.FailedQty > 0 && strings.TrimSpace(request.Reason) == "" {
			return errors.New("a failure reason is required when failed_qty is greater than zero")
		}
		now := time.Now()
		if err := tx.Model(&step).Updates(map[string]any{"status": stepCompleted, "passed_qty": request.PassedQty, "failed_qty": request.FailedQty, "completed_at": &now, "notes": request.Notes}).Error; err != nil {
			return err
		}
		var next models.WorkOrderStep
		nextErr := tx.Where("work_order_id = ? AND sequence > ?", order.ID, step.Sequence).Order("sequence ASC").First(&next).Error
		if errors.Is(nextErr, gorm.ErrRecordNotFound) {
			if err := updateWorkOrderVersion(tx, &order, map[string]any{"status": workOrderCompleted, "completed_qty": request.PassedQty, "failed_qty": gorm.Expr("failed_qty + ?", request.FailedQty), "current_sequence": step.Sequence}); err != nil {
				return err
			}
			return appendProductionEventWithMeta(tx, &order, &step, "step_completed", stepRunning, stepCompleted, request.PassedQty, request.FailedQty, request.Reason, string(request.Payload), source, idempotencyKey, user)
		}
		if nextErr != nil {
			return nextErr
		}
		if err := tx.Model(&next).Updates(map[string]any{"status": stepReady}).Error; err != nil {
			return err
		}
		if err := updateWorkOrderVersion(tx, &order, map[string]any{"failed_qty": gorm.Expr("failed_qty + ?", request.FailedQty), "current_sequence": next.Sequence}); err != nil {
			return err
		}
		return appendProductionEventWithMeta(tx, &order, &step, "step_completed", stepRunning, stepCompleted, request.PassedQty, request.FailedQty, request.Reason, string(request.Payload), source, idempotencyKey, user)
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "work order or step not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(user, "workflow", "work_order", workflowDetail("step_"+action, order.Code), c)
	order, err = loadWorkOrder(r.db, orderID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work_order": order})
}

func applyWorkOrderTransition(tx *gorm.DB, order *models.WorkOrder, action string, user *models.User) error {
	var step models.WorkOrderStep
	if action == "release" {
		if order.Status != workOrderDraft {
			return errors.New("only draft work orders can be released")
		}
		if err := tx.Where("work_order_id = ?", order.ID).Order("sequence ASC").First(&step).Error; err != nil {
			return errors.New("a work order must contain at least one process step")
		}
		if err := tx.Model(&step).Updates(map[string]any{"status": stepReady}).Error; err != nil {
			return err
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderReleased, "current_sequence": step.Sequence}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, &step, "released", workOrderDraft, workOrderReleased, 0, 0, "", user)
	}
	if action == "start" {
		if order.Status != workOrderReleased {
			return errors.New("only released work orders can be started")
		}
		if err := tx.Where("work_order_id = ? AND status = ?", order.ID, stepReady).Order("sequence ASC").First(&step).Error; err != nil {
			return errors.New("no ready process step is available")
		}
		now := time.Now()
		if err := tx.Model(&step).Updates(map[string]any{"status": stepRunning, "started_at": &now}).Error; err != nil {
			return err
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderRunning, "current_sequence": step.Sequence}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, &step, "started", workOrderReleased, workOrderRunning, 0, 0, "", user)
	}
	if action == "pause" {
		if order.Status != workOrderRunning {
			return errors.New("only running work orders can be paused")
		}
		if err := tx.Where("work_order_id = ? AND status = ?", order.ID, stepRunning).First(&step).Error; err != nil {
			return errors.New("no running process step is available")
		}
		if err := tx.Model(&step).Updates(map[string]any{"status": stepPaused}).Error; err != nil {
			return err
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderPaused}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, &step, "paused", workOrderRunning, workOrderPaused, 0, 0, "", user)
	}
	if action == "resume" {
		if order.Status != workOrderPaused {
			return errors.New("only paused work orders can be resumed")
		}
		if err := tx.Where("work_order_id = ? AND status = ?", order.ID, stepPaused).First(&step).Error; err != nil {
			return errors.New("no paused process step is available")
		}
		if err := tx.Model(&step).Updates(map[string]any{"status": stepRunning}).Error; err != nil {
			return err
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderRunning}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, &step, "resumed", workOrderPaused, workOrderRunning, 0, 0, "", user)
	}
	if action == "cancel" {
		if order.Status != workOrderDraft && order.Status != workOrderReleased && order.Status != workOrderPaused {
			return errors.New("only draft, released, or paused work orders can be cancelled")
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderCancelled}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, nil, "cancelled", order.Status, workOrderCancelled, 0, 0, "", user)
	}
	if action == "complete" {
		if order.Status != workOrderRunning {
			return errors.New("only running work orders can be completed")
		}
		var unfinished int64
		if err := tx.Model(&models.WorkOrderStep{}).Where("work_order_id = ? AND status NOT IN ?", order.ID, []string{stepCompleted, "skipped"}).Count(&unfinished).Error; err != nil {
			return err
		}
		if unfinished > 0 {
			return errors.New("all process steps must be completed before closing the work order")
		}
		if err := updateWorkOrderVersion(tx, order, map[string]any{"status": workOrderCompleted}); err != nil {
			return err
		}
		return appendProductionEvent(tx, order, nil, "completed", workOrderRunning, workOrderCompleted, 0, 0, "", user)
	}
	return fmt.Errorf("unsupported work order action %s", action)
}

func updateWorkOrderVersion(tx *gorm.DB, order *models.WorkOrder, updates map[string]any) error {
	if order.Version <= 0 {
		order.Version = 1
	}
	updates["version"] = order.Version + 1
	result := tx.Model(&models.WorkOrder{}).Where("id = ? AND version = ?", order.ID, order.Version).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("work order was changed by another operator; reload and retry")
	}
	order.Version++
	for key, value := range updates {
		switch key {
		case "status":
			if status, ok := value.(string); ok {
				order.Status = status
			}
		case "current_sequence":
			if sequence, ok := value.(int); ok {
				order.CurrentSequence = sequence
			}
		}
	}
	return nil
}

func appendProductionEvent(tx *gorm.DB, order *models.WorkOrder, step *models.WorkOrderStep, eventType, from, to string, passed, failed int, reason string, user *models.User) error {
	return appendProductionEventWithMeta(tx, order, step, eventType, from, to, passed, failed, reason, "", "operator", "", user)
}

func appendProductionEventWithMeta(tx *gorm.DB, order *models.WorkOrder, step *models.WorkOrderStep, eventType, from, to string, passed, failed int, reason, payload, source, idempotencyKey string, user *models.User) error {
	event := models.ProductionEvent{WorkOrderID: order.ID, EventType: eventType, FromStatus: from, ToStatus: to, PassedQty: passed, FailedQty: failed, Reason: strings.TrimSpace(reason), Payload: payload, Source: source, CreatedAt: time.Now()}
	if idempotencyKey != "" {
		event.IdempotencyKey = &idempotencyKey
	}
	if step != nil {
		event.WorkOrderStepID = &step.ID
		event.DeviceID = step.DeviceID
	}
	if user != nil {
		event.OperatorID = &user.ID
		event.OperatorName = user.Username
	}
	return tx.Create(&event).Error
}

func loadWorkOrder(db *gorm.DB, id uint, includeEvents bool) (models.WorkOrder, error) {
	var order models.WorkOrder
	query := db.Preload("FlowDefinition").Preload("Steps", func(tx *gorm.DB) *gorm.DB { return tx.Preload("Device").Order("sequence ASC") })
	if includeEvents {
		query = query.Preload("Events", func(tx *gorm.DB) *gorm.DB { return tx.Order("created_at DESC").Limit(100) })
	}
	err := query.First(&order, id).Error
	return order, err
}

func normalizeWorkOrderSteps(requests []workOrderStepRequest, defaultQty int) ([]models.WorkOrderStep, error) {
	steps := make([]models.WorkOrderStep, 0, len(requests))
	seen := map[int]bool{}
	for index, request := range requests {
		sequence := request.Sequence
		if sequence == 0 {
			sequence = index + 1
		}
		if sequence < 1 || seen[sequence] {
			return nil, errors.New("process step sequence must be positive and unique")
		}
		qty := request.PlannedQty
		if qty == 0 {
			qty = defaultQty
		}
		if qty < 1 {
			return nil, errors.New("process step planned quantity must be greater than zero")
		}
		seen[sequence] = true
		steps = append(steps, models.WorkOrderStep{Sequence: sequence, Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name), DeviceID: request.DeviceID, PlannedQty: qty, Status: stepPending, Notes: request.Notes})
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].Sequence < steps[j].Sequence })
	return steps, nil
}

func validateWorkOrderDevices(tx *gorm.DB, steps []models.WorkOrderStep) error {
	for _, step := range steps {
		if step.DeviceID == nil {
			continue
		}
		var count int64
		if err := tx.Model(&models.Device{}).Where("id = ?", *step.DeviceID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("device %d does not exist", *step.DeviceID)
		}
	}
	return nil
}

func validateWorkOrderSchedule(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return errors.New("scheduled_end must be after scheduled_start")
	}
	return nil
}

func workflowDetail(action, code string) string {
	labels := map[string]string{"release": "下达", "start": "启动", "pause": "暂停", "resume": "恢复", "cancel": "取消", "complete": "完工", "step_start": "启动工序", "step_complete": "完成工序"}
	label := labels[action]
	if label == "" {
		label = action
	}
	return label + "生产工单 " + code
}
