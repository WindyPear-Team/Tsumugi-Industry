export type User = { id: number; username: string; display_name: string; email: string; is_active: boolean; roles?: Role[] }
export type Role = { id: number; name: string; display_name: string; description?: string; permissions?: Permission[] }
export type Permission = { id: number; code: string; name: string; description?: string }
export type Setting = { key: string; value: string }
export type Summary = { users: number; roles: number; devices_online: number; alerts: number; work_orders_running: number; work_orders_waiting: number }
export type Device = { id: number; plc_id?: number; plc?: PLC; code: string; name: string; type: string; location: string; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type PLC = { id: number; code: string; name: string; protocol: string; host: string; port: number; rack?: number; slot?: number; unit_id?: number; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type WorkOrderStep = { id: number; work_order_id: number; sequence: number; code: string; name: string; device_id?: number; device?: Device; planned_qty: number; passed_qty: number; failed_qty: number; status: "pending" | "ready" | "running" | "paused" | "completed"; started_at?: string; completed_at?: string; notes?: string }
export type WorkOrder = { id: number; code: string; name?: string; product_code: string; product_name: string; planned_qty: number; completed_qty: number; failed_qty: number; status: "draft" | "released" | "running" | "paused" | "completed" | "cancelled"; priority: string; current_sequence: number; scheduled_start?: string; scheduled_end?: string; notes?: string; version: number; flow_definition_id?: number; flow_definition?: FlowDefinition; flow_variables?: string; steps?: WorkOrderStep[]; events?: ProductionEvent[]; created_at: string; updated_at: string }
export type ProductionEvent = { id: number; work_order_id: number; work_order_step_id?: number; event_type: string; from_status: string; to_status: string; passed_qty: number; failed_qty: number; reason?: string; payload?: string; source: "operator" | "gateway" | "plc"; operator_name: string; created_at: string }
export type PLCVariable = { id: number; name: string; description: string; plc_id: number; plc?: PLC; address: string; data_type: string; access_mode: "read" | "write" | "read_write"; default_value?: string; unit?: string; min_value?: number; max_value?: number; enum_values?: string; condition_allowed: boolean; flow_write_allowed: boolean; dangerous: boolean; freshness_seconds: number; current_value?: string; last_updated_at?: string; quality: "good" | "bad" | "stale" | "unknown"; communication_state: "online" | "offline" | "stale" | "unknown" }
export type FlowNode = { id: string; type: string; label: string; x: number; y: number; config: Record<string, unknown> }
export type FlowEdge = { id: string; source: string; target: string; condition?: string }
export type FlowParameter = { name: string; type: "number" | "string" | "boolean" | "device" | "option" | "select"; required?: boolean; options?: string[] }
export type FlowFunction = { id?: number; name: string; code: string; description?: string; return_type: "none" | "number" | "string" | "boolean"; parameters: FlowParameter[] | string; definition?: string }
export type FlowDocument = { nodes: FlowNode[]; edges: FlowEdge[]; variables?: { name: string; type: string; default_value?: string }[]; options?: string[]; functions?: FlowFunction[] }
export type FlowDefinition = { id: number; code: string; name: string; description: string; version: number; status: "draft" | "published"; definition: string; timeout_seconds: number; created_at: string; updated_at: string }
export type FlowRun = { id: number; flow_definition_id: number; flow_definition?: FlowDefinition; flow_version: number; status: "created" | "running" | "paused" | "completed" | "failed" | "cancelled" | "timeout" | "manual_confirm"; current_node_id: string; started_at?: string; ended_at?: string; error_message?: string; node_runs?: { id: number; node_id: string; node_type: string; status: string; started_at?: string; ended_at?: string; error_message?: string }[] }
export type AuditLog = { id: number; username: string; action: string; resource: string; detail: string; method?: string; path?: string; status_code?: number; ip?: string; created_at: string }
export type MonitorItem = { id: number; name: string; plc_id: number; plc?: PLC; variable_id: number; variable?: PLCVariable; interval_seconds: number; retention_days: number; enabled: boolean; last_sampled_at?: string }
export type MonitorRecord = { id: number; monitor_item_id: number; value: string; quality: string; recorded_at: string }
export type DashboardWidget = { id?: number; dashboard_id?: number; widget_type: "text" | "image" | "status" | "variable" | "chart"; title: string; x: number; y: number; width: number; height: number; config: string | Record<string, unknown> }
export type Dashboard = { id: number; name: string; description: string; time_range_hours: number; status_running?: string; status_idle?: string; widgets?: DashboardWidget[] }

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem("tsumugi-token")
  const response = await fetch(path, { ...options, headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error ?? "请求失败")
  }
  if (response.status === 204) return undefined as T
  return response.json()
}
