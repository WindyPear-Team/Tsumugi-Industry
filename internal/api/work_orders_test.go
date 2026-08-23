package api

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"tsumugi-industry/internal/auth"
	"tsumugi-industry/internal/maintenance"
	"tsumugi-industry/internal/models"
	"tsumugi-industry/internal/plc"
	"tsumugi-industry/internal/system"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkOrderWorkflowAPI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:work-order-api?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Role{}, &models.Permission{}, &models.SystemSetting{}, &models.Device{}, &models.PLC{}, &models.Alarm{}, &models.AuditLog{}, &models.ScheduledTask{}, &models.Backup{}, &models.WorkOrder{}, &models.WorkOrderStep{}, &models.ProductionEvent{}, &models.PLCVariable{}, &models.FlowDefinition{}, &models.FlowRun{}, &models.FlowNodeRun{}))
	require.NoError(t, system.Initialize(db, system.SetupRequest{Username: "admin", Password: "password123", DisplayName: "管理员"}))
	manager := auth.NewManager(db, "test-secret")
	token, _, err := manager.Authenticate("admin", "password123")
	require.NoError(t, err)
	frontend := fstestMapFS()
	router := NewRouter(db, manager, maintenance.New(db, t.TempDir()), plc.DefaultFactory(), frontend)

	create := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/work-orders", map[string]any{
		"code": "WO-TEST-001", "product_code": "P-001", "product_name": "测试产品", "planned_qty": 10,
		"steps": []map[string]any{{"sequence": 1, "code": "CUT", "name": "切割"}, {"sequence": 2, "code": "PACK", "name": "包装"}},
	})
	require.Equal(t, http.StatusCreated, create.Code)
	var created struct {
		WorkOrder models.WorkOrder `json:"work_order"`
	}
	require.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	require.Len(t, created.WorkOrder.Steps, 2)

	release := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/work-orders/1/release", map[string]any{})
	require.Equal(t, http.StatusOK, release.Code)
	start := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/work-orders/1/start", map[string]any{})
	require.Equal(t, http.StatusOK, start.Code)

	var running struct {
		WorkOrder models.WorkOrder `json:"work_order"`
	}
	require.NoError(t, json.Unmarshal(start.Body.Bytes(), &running))
	require.Len(t, running.WorkOrder.Steps, 2)
	firstStepID := running.WorkOrder.Steps[0].ID
	secondStepID := running.WorkOrder.Steps[1].ID
	firstComplete := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/work-orders/1/steps/"+strconv.Itoa(int(firstStepID))+"/complete", map[string]any{"passed_qty": 10, "failed_qty": 0})
	require.Equal(t, http.StatusOK, firstComplete.Code)
	secondStart := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/work-orders/1/steps/"+strconv.Itoa(int(secondStepID))+"/start", map[string]any{})
	require.Equal(t, http.StatusOK, secondStart.Code)
	secondComplete := doWorkOrderRequestWithHeaders(t, router, token, http.MethodPost, "/api/work-orders/1/steps/"+strconv.Itoa(int(secondStepID))+"/report", map[string]any{"passed_qty": 9, "failed_qty": 1, "reason": "外观缺陷"}, map[string]string{"Idempotency-Key": "report-test-001", "X-Production-Source": "plc"})
	require.Equal(t, http.StatusOK, secondComplete.Code)
	retriedReport := doWorkOrderRequestWithHeaders(t, router, token, http.MethodPost, "/api/work-orders/1/steps/"+strconv.Itoa(int(secondStepID))+"/report", map[string]any{"passed_qty": 9, "failed_qty": 1, "reason": "外观缺陷"}, map[string]string{"Idempotency-Key": "report-test-001", "X-Production-Source": "plc"})
	require.Equal(t, http.StatusOK, retriedReport.Code)

	var completed struct {
		WorkOrder models.WorkOrder `json:"work_order"`
	}
	require.NoError(t, json.Unmarshal(secondComplete.Body.Bytes(), &completed))
	require.Equal(t, workOrderCompleted, completed.WorkOrder.Status)
	require.Equal(t, 9, completed.WorkOrder.CompletedQty)
	require.Equal(t, 1, completed.WorkOrder.FailedQty)
	var eventCount int64
	require.NoError(t, db.Model(&models.ProductionEvent{}).Where("work_order_id = ?", created.WorkOrder.ID).Count(&eventCount).Error)
	require.Equal(t, int64(6), eventCount)

	flow := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/flows", map[string]any{
		"code": "FLOW-TEST", "name": "延时测试流程", "definition": map[string]any{
			"nodes": []map[string]any{{"id": "start", "type": "START", "label": "开始", "config": map[string]any{}}, {"id": "delay", "type": "DELAY", "label": "延时", "config": map[string]any{"seconds": 1}}, {"id": "end", "type": "END", "label": "结束", "config": map[string]any{}}},
			"edges": []map[string]any{{"id": "e1", "source": "start", "target": "delay"}, {"id": "e2", "source": "delay", "target": "end"}},
		},
	})
	require.Equal(t, http.StatusCreated, flow.Code)
	var createdFlow struct {
		Flow models.FlowDefinition `json:"flow"`
	}
	require.NoError(t, json.Unmarshal(flow.Body.Bytes(), &createdFlow))
	published := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/flows/"+strconv.Itoa(int(createdFlow.Flow.ID))+"/publish", map[string]any{})
	require.Equal(t, http.StatusOK, published.Code)
	newVersion := doWorkOrderRequest(t, router, token, http.MethodPost, "/api/flows/"+strconv.Itoa(int(createdFlow.Flow.ID))+"/new-version", map[string]any{})
	require.Equal(t, http.StatusCreated, newVersion.Code)
}

func doWorkOrderRequest(t *testing.T, router *gin.Engine, token, method, path string, payload any) *httptest.ResponseRecorder {
	return doWorkOrderRequestWithHeaders(t, router, token, method, path, payload, nil)
}

func doWorkOrderRequestWithHeaders(t *testing.T, router *gin.Engine, token, method, path string, payload any, headers map[string]string) *httptest.ResponseRecorder {
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func fstestMapFS() fs.FS {
	return mapFS{"index.html": []byte("<html></html>")}
}

type mapFS map[string][]byte

func (m mapFS) Open(name string) (fs.File, error) {
	data, ok := m[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &mapFile{name: name, data: data}, nil
}

type mapFile struct {
	name string
	data []byte
	pos  int
}

func (f *mapFile) Stat() (fs.FileInfo, error) {
	return mapFileInfo{name: f.name, size: int64(len(f.data))}, nil
}
func (f *mapFile) Read(p []byte) (int, error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}
func (f *mapFile) Close() error { return nil }

type mapFileInfo struct {
	name string
	size int64
}

func (i mapFileInfo) Name() string       { return i.name }
func (i mapFileInfo) Size() int64        { return i.size }
func (i mapFileInfo) Mode() fs.FileMode  { return 0 }
func (i mapFileInfo) ModTime() time.Time { return time.Time{} }
func (i mapFileInfo) IsDir() bool        { return false }
func (i mapFileInfo) Sys() any           { return nil }

type testRequire struct{}

var require testRequire

func (testRequire) NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (testRequire) Equal(t *testing.T, expected, actual any) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func (testRequire) Len(t *testing.T, value any, expected int) {
	t.Helper()
	steps, ok := value.([]models.WorkOrderStep)
	if !ok {
		t.Fatalf("unsupported length assertion for %T", value)
	}
	if len(steps) != expected {
		t.Fatalf("expected length %d, got %d", expected, len(steps))
	}
}

func (testRequire) GreaterOrEqual(t *testing.T, actual, expected int64) {
	t.Helper()
	if actual < expected {
		t.Fatalf("expected %d >= %d", actual, expected)
	}
}
