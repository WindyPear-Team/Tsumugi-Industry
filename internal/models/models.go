package models

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	DisplayName  string    `json:"display_name" gorm:"size:128"`
	Email        string    `json:"email" gorm:"size:160"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	Roles        []Role    `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"size:64;uniqueIndex;not null"`
	DisplayName string       `json:"display_name" gorm:"size:128"`
	Description string       `json:"description" gorm:"size:255"`
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type Permission struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Code        string    `json:"code" gorm:"size:128;uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"size:128"`
	Description string    `json:"description" gorm:"size:255"`
	CreatedAt   time.Time `json:"created_at"`
}

type SystemSetting struct {
	Key       string    `json:"key" gorm:"primaryKey;size:128"`
	Value     string    `json:"value" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Device struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	PLCID      *uint      `json:"plc_id" gorm:"index"`
	PLC        *PLC       `json:"plc,omitempty"`
	Code       string     `json:"code" gorm:"size:64;uniqueIndex;not null"`
	Name       string     `json:"name" gorm:"size:128;not null"`
	Type       string     `json:"type" gorm:"size:64"`
	Location   string     `json:"location" gorm:"size:160"`
	Status     string     `json:"status" gorm:"size:32;index;not null;default:offline"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Metadata   string     `json:"metadata" gorm:"type:text"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type PLC struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	Code       string     `json:"code" gorm:"size:64;uniqueIndex;not null"`
	Name       string     `json:"name" gorm:"size:128;not null"`
	Protocol   string     `json:"protocol" gorm:"size:32;not null"`
	Host       string     `json:"host" gorm:"size:128"`
	Port       int        `json:"port"`
	Rack       int        `json:"rack"`
	Slot       int        `json:"slot"`
	UnitID     byte       `json:"unit_id"`
	Status     string     `json:"status" gorm:"size:32;index;not null;default:offline"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Metadata   string     `json:"metadata" gorm:"type:text"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Alarm struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	DeviceID       *uint      `json:"device_id" gorm:"index"`
	Device         *Device    `json:"device,omitempty"`
	Code           string     `json:"code" gorm:"size:64;index"`
	Level          string     `json:"level" gorm:"size:32;index;not null"`
	Message        string     `json:"message" gorm:"size:255;not null"`
	Status         string     `json:"status" gorm:"size:32;index;not null;default:active"`
	OccurredAt     time.Time  `json:"occurred_at" gorm:"index"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     *uint     `json:"user_id" gorm:"index"`
	Username   string    `json:"username" gorm:"size:64"`
	Action     string    `json:"action" gorm:"size:32;index"`
	Resource   string    `json:"resource" gorm:"size:64;index"`
	ResourceID string    `json:"resource_id" gorm:"size:64"`
	Method     string    `json:"-" gorm:"size:16"`
	Path       string    `json:"-" gorm:"size:255"`
	StatusCode int       `json:"-"`
	IP         string    `json:"-" gorm:"size:64"`
	Detail     string    `json:"detail" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at" gorm:"index"`
}

type ScheduledTask struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name" gorm:"size:128;not null"`
	TaskType        string     `json:"task_type" gorm:"size:64;index;not null"`
	IntervalSeconds int        `json:"interval_seconds" gorm:"not null;default:3600"`
	Enabled         bool       `json:"enabled" gorm:"default:true"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at"`
	LastStatus      string     `json:"last_status" gorm:"size:32"`
	LastMessage     string     `json:"last_message" gorm:"size:255"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Backup struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:128;not null"`
	Driver    string    `json:"driver" gorm:"size:32"`
	Path      string    `json:"path" gorm:"size:255"`
	Size      int64     `json:"size"`
	Status    string    `json:"status" gorm:"size:32"`
	CreatedBy string    `json:"created_by" gorm:"size:64"`
	CreatedAt time.Time `json:"created_at"`
}

// WorkOrder is the executable production plan. Its status is changed only by
// workflow endpoints so operators cannot accidentally skip a process step.
type WorkOrder struct {
	ID               uint              `json:"id" gorm:"primaryKey"`
	Code             string            `json:"code" gorm:"size:64;uniqueIndex;not null"`
	ProductCode      string            `json:"product_code" gorm:"size:64;index;not null"`
	ProductName      string            `json:"product_name" gorm:"size:128;not null"`
	PlannedQty       int               `json:"planned_qty" gorm:"not null"`
	CompletedQty     int               `json:"completed_qty" gorm:"not null;default:0"`
	FailedQty        int               `json:"failed_qty" gorm:"not null;default:0"`
	Status           string            `json:"status" gorm:"size:32;index;not null;default:draft"`
	Priority         string            `json:"priority" gorm:"size:16;index;not null;default:normal"`
	CurrentSequence  int               `json:"current_sequence" gorm:"not null;default:0"`
	ScheduledStart   *time.Time        `json:"scheduled_start"`
	ScheduledEnd     *time.Time        `json:"scheduled_end"`
	Notes            string            `json:"notes" gorm:"type:text"`
	Version          int               `json:"version" gorm:"not null;default:1"`
	FlowDefinitionID *uint             `json:"flow_definition_id" gorm:"index"`
	FlowDefinition   *FlowDefinition   `json:"flow_definition,omitempty" gorm:"foreignKey:FlowDefinitionID"`
	FlowVariables    string            `json:"flow_variables" gorm:"type:text"`
	CreatedByID      *uint             `json:"created_by_id" gorm:"index"`
	CreatedBy        *User             `json:"created_by,omitempty" gorm:"foreignKey:CreatedByID"`
	Steps            []WorkOrderStep   `json:"steps,omitempty" gorm:"foreignKey:WorkOrderID;constraint:OnDelete:CASCADE"`
	Events           []ProductionEvent `json:"events,omitempty" gorm:"foreignKey:WorkOrderID;constraint:OnDelete:CASCADE"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type WorkOrderStep struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	WorkOrderID uint       `json:"work_order_id" gorm:"index;not null;uniqueIndex:idx_work_order_step_sequence"`
	Sequence    int        `json:"sequence" gorm:"not null;uniqueIndex:idx_work_order_step_sequence"`
	Code        string     `json:"code" gorm:"size:64;not null"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	DeviceID    *uint      `json:"device_id" gorm:"index"`
	Device      *Device    `json:"device,omitempty"`
	PlannedQty  int        `json:"planned_qty" gorm:"not null"`
	PassedQty   int        `json:"passed_qty" gorm:"not null;default:0"`
	FailedQty   int        `json:"failed_qty" gorm:"not null;default:0"`
	Status      string     `json:"status" gorm:"size:32;index;not null;default:pending"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Notes       string     `json:"notes" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type ProductionEvent struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	WorkOrderID     uint      `json:"work_order_id" gorm:"index;not null"`
	WorkOrderStepID *uint     `json:"work_order_step_id" gorm:"index"`
	DeviceID        *uint     `json:"device_id" gorm:"index"`
	EventType       string    `json:"event_type" gorm:"size:32;index;not null"`
	FromStatus      string    `json:"from_status" gorm:"size:32"`
	ToStatus        string    `json:"to_status" gorm:"size:32"`
	PassedQty       int       `json:"passed_qty"`
	FailedQty       int       `json:"failed_qty"`
	Reason          string    `json:"reason" gorm:"size:255"`
	Payload         string    `json:"payload" gorm:"type:text"`
	Source          string    `json:"source" gorm:"size:32;index;not null;default:operator"`
	IdempotencyKey  *string   `json:"-" gorm:"size:128;uniqueIndex"`
	OperatorID      *uint     `json:"operator_id" gorm:"index"`
	OperatorName    string    `json:"operator_name" gorm:"size:64"`
	CreatedAt       time.Time `json:"created_at" gorm:"index"`
}

// PLCVariable is the protocol-neutral semantic variable layer. Flow
// definitions reference Name, never a Siemens/Modbus address.
type PLCVariable struct {
	ID                 uint       `json:"id" gorm:"primaryKey"`
	Name               string     `json:"name" gorm:"size:128;uniqueIndex;not null"`
	Description        string     `json:"description" gorm:"size:255"`
	PLCID              uint       `json:"plc_id" gorm:"index;not null"`
	PLC                *PLC       `json:"plc,omitempty"`
	Address            string     `json:"address" gorm:"size:128;not null"`
	DataType           string     `json:"data_type" gorm:"size:16;not null"`
	AccessMode         string     `json:"access_mode" gorm:"size:16;not null;default:read"`
	DefaultValue       string     `json:"default_value" gorm:"type:text"`
	Unit               string     `json:"unit" gorm:"size:32"`
	MinValue           *float64   `json:"min_value"`
	MaxValue           *float64   `json:"max_value"`
	EnumValues         string     `json:"enum_values" gorm:"type:text"`
	ConditionAllowed   bool       `json:"condition_allowed" gorm:"default:true"`
	FlowWriteAllowed   bool       `json:"flow_write_allowed" gorm:"default:false"`
	Dangerous          bool       `json:"dangerous" gorm:"default:false"`
	FreshnessSeconds   int        `json:"freshness_seconds" gorm:"not null;default:10"`
	CurrentValue       string     `json:"current_value" gorm:"type:text"`
	LastUpdatedAt      *time.Time `json:"last_updated_at"`
	Quality            string     `json:"quality" gorm:"size:16;not null;default:unknown"`
	CommunicationState string     `json:"communication_state" gorm:"size:16;not null;default:unknown"`
	CreatedByID        *uint      `json:"created_by_id" gorm:"index"`
	UpdatedByID        *uint      `json:"updated_by_id" gorm:"index"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type FlowDefinition struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Code           string    `json:"code" gorm:"size:64;index:idx_flow_code_version,unique;not null"`
	Name           string    `json:"name" gorm:"size:128;not null"`
	Description    string    `json:"description" gorm:"type:text"`
	Version        int       `json:"version" gorm:"index:idx_flow_code_version,unique;not null;default:1"`
	Status         string    `json:"status" gorm:"size:16;index;not null;default:draft"`
	Definition     string    `json:"definition" gorm:"type:text;not null"`
	TimeoutSeconds int       `json:"timeout_seconds" gorm:"not null;default:0"`
	CreatedByID    *uint     `json:"created_by_id" gorm:"index"`
	UpdatedByID    *uint     `json:"updated_by_id" gorm:"index"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MonitorItem describes a PLC variable sampling plan. Samples are kept
// separately so charts can be queried without changing the semantic variable.
type MonitorItem struct {
	ID              uint         `json:"id" gorm:"primaryKey"`
	Name            string       `json:"name" gorm:"size:128;not null"`
	PLCID           uint         `json:"plc_id" gorm:"index;not null"`
	PLC             *PLC         `json:"plc,omitempty"`
	VariableID      uint         `json:"variable_id" gorm:"index;not null"`
	Variable        *PLCVariable `json:"variable,omitempty"`
	IntervalSeconds int          `json:"interval_seconds" gorm:"not null;default:10"`
	RetentionDays   int          `json:"retention_days" gorm:"not null;default:30"`
	Enabled         bool         `json:"enabled" gorm:"default:true"`
	LastSampledAt   *time.Time   `json:"last_sampled_at"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type MonitorRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	MonitorItemID uint      `json:"monitor_item_id" gorm:"index;not null"`
	Value         string    `json:"value" gorm:"type:text"`
	Quality       string    `json:"quality" gorm:"size:16"`
	RecordedAt    time.Time `json:"recorded_at" gorm:"index"`
}

type Dashboard struct {
	ID             uint              `json:"id" gorm:"primaryKey"`
	Name           string            `json:"name" gorm:"size:128;not null"`
	Description    string            `json:"description" gorm:"type:text"`
	TimeRangeHours int               `json:"time_range_hours" gorm:"not null;default:24"`
	StatusRunning  string            `json:"status_running" gorm:"size:255"`
	StatusIdle     string            `json:"status_idle" gorm:"size:255"`
	Widgets        []DashboardWidget `json:"widgets,omitempty" gorm:"foreignKey:DashboardID;constraint:OnDelete:CASCADE"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type DashboardWidget struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	DashboardID uint      `json:"dashboard_id" gorm:"index;not null"`
	WidgetType  string    `json:"widget_type" gorm:"size:32;not null"`
	Title       string    `json:"title" gorm:"size:128"`
	X           int       `json:"x"`
	Y           int       `json:"y"`
	Width       int       `json:"width" gorm:"default:3"`
	Height      int       `json:"height" gorm:"default:2"`
	Config      string    `json:"config" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FlowRun struct {
	ID               uint            `json:"id" gorm:"primaryKey"`
	FlowDefinitionID uint            `json:"flow_definition_id" gorm:"index;not null"`
	FlowDefinition   *FlowDefinition `json:"flow_definition,omitempty"`
	FlowVersion      int             `json:"flow_version" gorm:"not null"`
	Status           string          `json:"status" gorm:"size:16;index;not null;default:created"`
	CurrentNodeID    string          `json:"current_node_id" gorm:"size:128"`
	StartedAt        *time.Time      `json:"started_at"`
	EndedAt          *time.Time      `json:"ended_at"`
	ErrorMessage     string          `json:"error_message" gorm:"type:text"`
	StartedByID      *uint           `json:"started_by_id" gorm:"index"`
	NodeRuns         []FlowNodeRun   `json:"node_runs,omitempty" gorm:"foreignKey:FlowRunID;constraint:OnDelete:CASCADE"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type FlowNodeRun struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	FlowRunID    uint       `json:"flow_run_id" gorm:"index;not null"`
	NodeID       string     `json:"node_id" gorm:"size:128;index;not null"`
	NodeType     string     `json:"node_type" gorm:"size:32;not null"`
	Status       string     `json:"status" gorm:"size:16;index;not null"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	ErrorMessage string     `json:"error_message" gorm:"type:text"`
	Attempt      int        `json:"attempt" gorm:"not null;default:1"`
	Output       string     `json:"output" gorm:"type:text"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
