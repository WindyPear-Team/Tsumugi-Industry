package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/models"
	"tsumugi-industry/internal/plc"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	flowDraft     = "draft"
	flowPublished = "published"
	flowCreated   = "created"
	flowRunning   = "running"
	flowPaused    = "paused"
	flowCompleted = "completed"
	flowFailed    = "failed"
	flowCancelled = "cancelled"
	flowTimeout   = "timeout"
	flowConfirm   = "manual_confirm"
)

var errFlowConditionFalse = errors.New("flow condition false")
var errFlowManualConfirm = errors.New("manual confirmation required")

type flowTimeoutError struct{ message string }

func (e *flowTimeoutError) Error() string { return e.message }

type flowNode struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Label  string         `json:"label"`
	X      float64        `json:"x"`
	Y      float64        `json:"y"`
	Config map[string]any `json:"config"`
}

type flowEdge struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Condition string `json:"condition"`
}

type flowDocument struct {
	Nodes []flowNode `json:"nodes"`
	Edges []flowEdge `json:"edges"`
}

type flowRequest struct {
	Code           string          `json:"code" binding:"required,max=64"`
	Name           string          `json:"name" binding:"required,max=128"`
	Description    string          `json:"description"`
	Definition     json.RawMessage `json:"definition" binding:"required"`
	TimeoutSeconds int             `json:"timeout_seconds"`
}

func (r *Router) flows(c *gin.Context) {
	var items []models.FlowDefinition
	query := applyTableQuery(r.db.Model(&models.FlowDefinition{}), c,
		map[string]string{"id": "id", "code": "code", "name": "name", "version": "version", "status": "status"},
		map[string]string{"filter_code": "code", "filter_name": "name", "filter_status": "status"}, "id")
	query, page := paginate(query, c)
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page})
}

func (r *Router) flow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var item models.FlowDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"flow": item})
}

func (r *Router) createFlow(c *gin.Context) {
	var request flowRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	document, err := decodeFlowDocument(request.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	issues := r.validateFlowDocument(document)
	if len(issues) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow definition failed validation", "issues": issues})
		return
	}
	user := auth.CurrentUser(c)
	item := models.FlowDefinition{Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name), Description: request.Description, Version: 1, Status: flowDraft, Definition: string(request.Definition), TimeoutSeconds: request.TimeoutSeconds}
	if user != nil {
		item.CreatedByID = &user.ID
		item.UpdatedByID = &user.ID
	}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "flow code already exists"})
		return
	}
	r.recordEvent(user, "create", "flow", "创建流程 "+item.Name, c)
	c.JSON(http.StatusCreated, gin.H{"flow": item})
}

func (r *Router) updateFlow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var request flowRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	document, err := decodeFlowDocument(request.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	issues := r.validateFlowDocument(document)
	if len(issues) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow definition failed validation", "issues": issues})
		return
	}
	var item models.FlowDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	if item.Status == flowPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "published flow is immutable; create a new version"})
		return
	}
	item.Code = strings.TrimSpace(request.Code)
	item.Name = strings.TrimSpace(request.Name)
	item.Description = request.Description
	item.Definition = string(request.Definition)
	item.TimeoutSeconds = request.TimeoutSeconds
	item.Version++
	item.UpdatedByID = userID(auth.CurrentUser(c))
	if err := r.db.Save(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "update", "flow", "修改流程 "+item.Name, c)
	c.JSON(http.StatusOK, gin.H{"flow": item})
}

func (r *Router) deleteFlow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var item models.FlowDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	if item.Status == flowPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "published flow cannot be deleted"})
		return
	}
	if err := r.db.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "delete", "flow", "删除流程 "+item.Name, c)
	c.Status(http.StatusNoContent)
}

func (r *Router) validateFlow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var item models.FlowDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	document, err := decodeFlowDocument([]byte(item.Definition))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "issues": []string{err.Error()}})
		return
	}
	issues := r.validateFlowDocument(document)
	c.JSON(http.StatusOK, gin.H{"valid": len(issues) == 0, "issues": issues})
}

func (r *Router) publishFlow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var item models.FlowDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	document, err := decodeFlowDocument([]byte(item.Definition))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	issues := r.validateFlowDocument(document)
	if len(issues) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow definition failed validation", "issues": issues})
		return
	}
	if err := r.db.Model(&item).Updates(map[string]any{"status": flowPublished, "updated_by_id": userID(auth.CurrentUser(c))}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), "publish", "flow", "发布流程 "+item.Name+fmt.Sprintf(" v%d", item.Version), c)
	item.Status = flowPublished
	c.JSON(http.StatusOK, gin.H{"flow": item})
}

func (r *Router) newFlowVersion(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var source models.FlowDefinition
	if err := r.db.First(&source, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	var latest models.FlowDefinition
	if err := r.db.Where("code = ?", source.Code).Order("version DESC").First(&latest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user := auth.CurrentUser(c)
	item := models.FlowDefinition{Code: source.Code, Name: source.Name, Description: source.Description, Version: latest.Version + 1, Status: flowDraft, Definition: source.Definition, TimeoutSeconds: source.TimeoutSeconds, CreatedByID: userID(user), UpdatedByID: userID(user)}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "unable to create flow version: " + err.Error()})
		return
	}
	r.recordEvent(user, "create_version", "flow", fmt.Sprintf("创建流程 %s v%d", item.Name, item.Version), c)
	c.JSON(http.StatusCreated, gin.H{"flow": item})
}

func (r *Router) startFlow(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow id"})
		return
	}
	var definition models.FlowDefinition
	if err := r.db.First(&definition, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}
	if definition.Status != flowPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "only published flows can be started"})
		return
	}
	now := time.Now()
	user := auth.CurrentUser(c)
	run := models.FlowRun{FlowDefinitionID: definition.ID, FlowVersion: definition.Version, Status: flowCreated, StartedAt: &now, StartedByID: userID(user)}
	if err := r.db.Create(&run).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(user, "start", "flow_run", fmt.Sprintf("启动流程 %s v%d", definition.Name, definition.Version), c)
	go r.executeFlow(run.ID, definition)
	c.JSON(http.StatusAccepted, gin.H{"run": run})
}

func (r *Router) flowRuns(c *gin.Context) {
	var runs []models.FlowRun
	query := applyTableQuery(r.db.Model(&models.FlowRun{}).Preload("FlowDefinition"), c, map[string]string{"id": "id", "flow_definition_id": "flow_definition_id", "flow_version": "flow_version", "status": "status", "started_at": "started_at"}, map[string]string{"filter_status": "status"}, "id")
	query, page := paginate(query, c)
	if err := query.Find(&runs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": runs, "page": page})
}

func (r *Router) flowRun(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow run id"})
		return
	}
	var run models.FlowRun
	if err := r.db.Preload("FlowDefinition").Preload("NodeRuns", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).First(&run, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow run not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"run": run})
}

func (r *Router) controlFlowRun(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid flow run id"})
		return
	}
	action := c.Param("action")
	var run models.FlowRun
	if err := r.db.First(&run, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow run not found"})
		return
	}
	updates := map[string]any{}
	switch action {
	case "pause":
		if run.Status != flowRunning {
			c.JSON(http.StatusConflict, gin.H{"error": "only running flow can be paused"})
			return
		}
		updates["status"] = flowPaused
	case "resume":
		if run.Status != flowPaused {
			c.JSON(http.StatusConflict, gin.H{"error": "only paused flow can be resumed"})
			return
		}
		updates["status"] = flowRunning
	case "confirm":
		if run.Status != flowConfirm {
			c.JSON(http.StatusConflict, gin.H{"error": "flow run is not waiting for manual confirmation"})
			return
		}
		updates["status"] = flowRunning
	case "cancel":
		if run.Status == flowCompleted || run.Status == flowFailed || run.Status == flowCancelled {
			c.JSON(http.StatusConflict, gin.H{"error": "flow run is already finished"})
			return
		}
		updates["status"] = flowCancelled
		now := time.Now()
		updates["ended_at"] = &now
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported flow run action"})
		return
	}
	if err := r.db.Model(&run).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r.recordEvent(auth.CurrentUser(c), action, "flow_run", fmt.Sprintf("流程运行 %d：%s", run.ID, action), c)
	c.JSON(http.StatusOK, gin.H{"run": run})
}

func (r *Router) validateFlowDocument(document flowDocument) []string {
	issues := []string{}
	if len(document.Nodes) == 0 {
		return []string{"流程至少需要一个节点"}
	}
	nodes := map[string]flowNode{}
	startCount, endCount := 0, 0
	for _, node := range document.Nodes {
		if strings.TrimSpace(node.ID) == "" {
			issues = append(issues, "存在没有 ID 的节点")
			continue
		}
		if _, exists := nodes[node.ID]; exists {
			issues = append(issues, "节点 ID 重复："+node.ID)
		}
		nodes[node.ID] = node
		switch strings.ToUpper(node.Type) {
		case "START":
			startCount++
		case "END":
			endCount++
		case "SET", "GET", "WAIT", "IF", "DELAY", "MANUAL_CONFIRM", "ALARM", "LOOP", "PARALLEL", "SUBFLOW":
		default:
			issues = append(issues, "不支持的节点类型："+node.Type)
		}
		if variableName, ok := node.Config["variable"].(string); ok && variableName != "" {
			var variable models.PLCVariable
			if err := r.db.Where("name = ?", variableName).First(&variable).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				issues = append(issues, "找不到变量："+variableName)
			} else if err == nil && (strings.EqualFold(node.Type, "WAIT") || strings.EqualFold(node.Type, "IF")) && !variable.ConditionAllowed {
				issues = append(issues, "变量禁止用于条件："+variableName)
			} else if err == nil && strings.EqualFold(node.Type, "SET") && (!variable.FlowWriteAllowed || variable.AccessMode == "read") {
				issues = append(issues, "变量禁止被流程写入："+variableName)
			}
		}
	}
	if startCount != 1 {
		issues = append(issues, "流程必须有且只有一个 START 节点")
	}
	if endCount < 1 {
		issues = append(issues, "流程至少需要一个 END 节点")
	}
	for _, edge := range document.Edges {
		if _, ok := nodes[edge.Source]; !ok {
			issues = append(issues, "连线源节点不存在："+edge.Source)
		}
		if _, ok := nodes[edge.Target]; !ok {
			issues = append(issues, "连线目标节点不存在："+edge.Target)
		}
	}
	if startCount == 1 {
		start := ""
		for _, node := range document.Nodes {
			if strings.EqualFold(node.Type, "START") {
				start = node.ID
			}
		}
		reachable := map[string]bool{start: true}
		changed := true
		for changed {
			changed = false
			for _, edge := range document.Edges {
				if reachable[edge.Source] && !reachable[edge.Target] {
					reachable[edge.Target] = true
					changed = true
				}
			}
		}
		for _, node := range document.Nodes {
			if !reachable[node.ID] {
				issues = append(issues, "节点不可从 START 到达："+node.ID)
			}
		}
	}
	return uniqueStrings(issues)
}

func decodeFlowDocument(data []byte) (flowDocument, error) {
	var document flowDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return document, fmt.Errorf("流程定义不是有效 JSON：%w", err)
	}
	return document, nil
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func (r *Router) executeFlow(runID uint, definition models.FlowDefinition) {
	var document flowDocument
	if err := json.Unmarshal([]byte(definition.Definition), &document); err != nil {
		r.finishFlow(runID, flowFailed, err.Error())
		return
	}
	start := ""
	nodes := map[string]flowNode{}
	next := map[string][]flowEdge{}
	for _, node := range document.Nodes {
		nodes[node.ID] = node
		if strings.EqualFold(node.Type, "START") {
			start = node.ID
		}
	}
	for _, edge := range document.Edges {
		next[edge.Source] = append(next[edge.Source], edge)
	}
	current := start
	deadline := time.Time{}
	if definition.TimeoutSeconds > 0 {
		deadline = time.Now().Add(time.Duration(definition.TimeoutSeconds) * time.Second)
	}
	r.db.Model(&models.FlowRun{}).Where("id = ?", runID).Update("status", flowRunning)
	for current != "" {
		if !deadline.IsZero() && time.Now().After(deadline) {
			r.finishFlow(runID, flowTimeout, "流程超过总超时时间")
			return
		}
		var run models.FlowRun
		if r.db.First(&run, runID).Error != nil {
			return
		}
		if run.Status == flowCancelled {
			return
		}
		for run.Status == flowPaused {
			time.Sleep(250 * time.Millisecond)
			if r.db.First(&run, runID).Error != nil {
				return
			}
			if run.Status == flowCancelled {
				return
			}
		}
		node, ok := nodes[current]
		if !ok {
			r.finishFlow(runID, flowFailed, "当前节点不存在："+current)
			return
		}
		now := time.Now()
		nodeRun := models.FlowNodeRun{FlowRunID: runID, NodeID: node.ID, NodeType: strings.ToUpper(node.Type), Status: "running", StartedAt: &now, Attempt: 1}
		r.db.Create(&nodeRun)
		r.db.Model(&run).Updates(map[string]any{"current_node_id": node.ID})
		attempt := 1
		var nodeErr error
		for {
			nodeErr = r.executeFlowNode(node)
			if nodeErr == nil || errors.Is(nodeErr, errFlowConditionFalse) {
				break
			}
			var timeoutErr *flowTimeoutError
			maxRetries := int(configNumberDefault(node.Config, "max_retries", 0))
			if errors.As(nodeErr, &timeoutErr) && attempt <= maxRetries {
				retrySeconds := configNumberDefault(node.Config, "retry_interval_seconds", 1)
				time.Sleep(time.Duration(retrySeconds * float64(time.Second)))
				attempt++
				r.db.Model(&nodeRun).Update("attempt", attempt)
				continue
			}
			if errors.As(nodeErr, &timeoutErr) && strings.EqualFold(stringValue(node.Config, "timeout_action", "FAIL"), "MANUAL_CONFIRM") {
				r.db.Model(&nodeRun).Updates(map[string]any{"status": flowConfirm, "error_message": nodeErr.Error()})
				r.db.Model(&models.FlowRun{}).Where("id = ?", runID).Update("status", flowConfirm)
				for {
					time.Sleep(250 * time.Millisecond)
					if r.db.First(&run, runID).Error != nil || run.Status == flowCancelled {
						return
					}
					if run.Status == flowRunning {
						break
					}
				}
				nodeErr = nil
			}
			break
		}
		ended := time.Now()
		status := "success"
		conditionFalse := errors.Is(nodeErr, errFlowConditionFalse)
		if errors.Is(nodeErr, errFlowManualConfirm) {
			r.db.Model(&nodeRun).Updates(map[string]any{"status": flowConfirm})
			r.db.Model(&models.FlowRun{}).Where("id = ?", runID).Update("status", flowConfirm)
			for {
				time.Sleep(250 * time.Millisecond)
				if r.db.First(&run, runID).Error != nil || run.Status == flowCancelled {
					return
				}
				if run.Status == flowRunning {
					break
				}
			}
			ended = time.Now()
			nodeErr = nil
		}
		if nodeErr != nil && !conditionFalse {
			status = "failed"
			r.db.Model(&nodeRun).Updates(map[string]any{"status": status, "ended_at": &ended, "error_message": nodeErr.Error()})
			r.finishFlow(runID, flowFailed, nodeErr.Error())
			return
		}
		r.db.Model(&nodeRun).Updates(map[string]any{"status": status, "ended_at": &ended})
		if strings.EqualFold(node.Type, "END") {
			r.finishFlow(runID, flowCompleted, "")
			return
		}
		edges := next[node.ID]
		if len(edges) == 0 {
			r.finishFlow(runID, flowFailed, "节点没有后继连线："+node.ID)
			return
		}
		current = edges[0].Target
		if strings.EqualFold(node.Type, "IF") {
			if conditionFalse && len(edges) > 1 {
				current = edges[1].Target
			}
		}
	}
}

func (r *Router) executeFlowNode(node flowNode) error {
	config := node.Config
	switch strings.ToUpper(node.Type) {
	case "START", "END", "ALARM":
		return nil
	case "GET", "IF":
		variableName, _ := config["variable"].(string)
		if strings.TrimSpace(variableName) == "" {
			return errors.New("节点未配置语义变量")
		}
		value, quality, err := r.readSemanticVariable(variableName)
		if err != nil {
			return err
		}
		if quality != "good" {
			return fmt.Errorf("变量 %s 数据质量为 %s，不能用于流程", variableName, quality)
		}
		if strings.EqualFold(node.Type, "IF") {
			operator, _ := config["operator"].(string)
			if !compareFlowValue(value, operator, config["expected"]) {
				return errFlowConditionFalse
			}
		}
		return nil
	case "DELAY":
		seconds, _ := configNumber(config, "seconds")
		if seconds <= 0 {
			return errors.New("DELAY seconds must be greater than zero")
		}
		time.Sleep(time.Duration(seconds * float64(time.Second)))
		return nil
	case "WAIT":
		variableName, _ := config["variable"].(string)
		if strings.TrimSpace(variableName) == "" {
			return errors.New("WAIT 节点未配置语义变量")
		}
		seconds, ok := configNumber(config, "timeout_seconds")
		if !ok || seconds <= 0 {
			seconds = 10
		}
		deadline := time.Now().Add(time.Duration(seconds * float64(time.Second)))
		for time.Now().Before(deadline) {
			value, quality, err := r.readSemanticVariable(variableName)
			if err == nil && quality == "good" && compareFlowValue(value, stringValue(config, "operator", "=="), config["expected"]) {
				return nil
			}
			time.Sleep(250 * time.Millisecond)
		}
		return &flowTimeoutError{message: fmt.Sprintf("等待变量 %s 条件超时（%.0f 秒）", variableName, seconds)}
	case "SET":
		variableName, _ := config["variable"].(string)
		if strings.TrimSpace(variableName) == "" {
			return errors.New("SET 节点未配置语义变量")
		}
		return r.writeSemanticVariable(variableName, config["value"])
	case "MANUAL_CONFIRM":
		return errFlowManualConfirm
	default:
		return fmt.Errorf("node type %s is not executable in this runtime", node.Type)
	}
}

func (r *Router) readSemanticVariable(name string) (any, string, error) {
	var item models.PLCVariable
	if err := r.db.Where("name = ?", strings.TrimSpace(name)).First(&item).Error; err != nil {
		return nil, "bad", fmt.Errorf("变量 %s 不存在", name)
	}
	adapter, err := r.variableAdapter(item)
	if err != nil {
		markVariableOffline(r.db, &item)
		return nil, "bad", err
	}
	defer adapter.Close(context.Background())
	values, err := adapter.Read(context.Background(), []plc.ReadRequest{{Address: item.Address, Length: variableReadLength(item.DataType)}})
	if err != nil || len(values) == 0 {
		markVariableOffline(r.db, &item)
		if err == nil {
			err = errors.New("PLC returned no value")
		}
		return nil, "bad", err
	}
	decoded, err := decodePLCValue(values[0].Value, item.DataType, item.Address)
	if err != nil {
		return nil, "bad", err
	}
	now := time.Now()
	encoded, _ := json.Marshal(decoded)
	r.db.Model(&item).Updates(map[string]any{"current_value": string(encoded), "last_updated_at": &now, "quality": "good", "communication_state": "online"})
	return decoded, "good", nil
}

func (r *Router) writeSemanticVariable(name string, value any) error {
	var item models.PLCVariable
	if err := r.db.Where("name = ?", strings.TrimSpace(name)).First(&item).Error; err != nil {
		return fmt.Errorf("变量 %s 不存在", name)
	}
	if item.AccessMode == "read" || !item.FlowWriteAllowed {
		return fmt.Errorf("变量 %s 禁止流程写入", name)
	}
	if item.Dangerous {
		return fmt.Errorf("危险变量 %s 不允许被后台流程直接写入，需要人工确认", name)
	}
	if err := validateVariableValue(item, value); err != nil {
		return err
	}
	adapter, err := r.variableAdapter(item)
	if err != nil {
		return err
	}
	defer adapter.Close(context.Background())
	if err := adapter.Write(context.Background(), []plc.WriteRequest{{Address: item.Address, Value: value}}); err != nil {
		return fmt.Errorf("写入变量 %s 失败：%w", name, err)
	}
	encoded, _ := json.Marshal(value)
	now := time.Now()
	r.db.Model(&item).Updates(map[string]any{"current_value": string(encoded), "last_updated_at": &now, "quality": "good", "communication_state": "online"})
	return nil
}

func compareFlowValue(actual any, operator string, expected any) bool {
	operator = strings.TrimSpace(operator)
	if operator == "==" || operator == "!=" {
		left, _ := json.Marshal(actual)
		right, _ := json.Marshal(expected)
		equal := strings.EqualFold(string(left), string(right)) || strings.EqualFold(fmt.Sprint(actual), fmt.Sprint(expected))
		if operator == "!=" {
			return !equal
		}
		return equal
	}
	left, lok := flowNumber(actual)
	right, rok := flowNumber(expected)
	if !lok || !rok {
		return false
	}
	switch operator {
	case ">":
		return left > right
	case "<":
		return left < right
	case ">=":
		return left >= right
	case "<=":
		return left <= right
	default:
		return false
	}
}

func flowNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func stringValue(config map[string]any, key, fallback string) string {
	value, ok := config[key].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func configNumber(config map[string]any, key string) (float64, bool) {
	value, ok := config[key].(float64)
	return value, ok
}

func configNumberDefault(config map[string]any, key string, fallback float64) float64 {
	value, ok := configNumber(config, key)
	if !ok {
		return fallback
	}
	return value
}
func (r *Router) finishFlow(runID uint, status, message string) {
	now := time.Now()
	r.db.Model(&models.FlowRun{}).Where("id = ? AND status NOT IN ?", runID, []string{flowCancelled, flowCompleted, flowFailed, flowTimeout}).Updates(map[string]any{"status": status, "ended_at": &now, "error_message": message})
}
