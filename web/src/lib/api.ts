export type User = { id: number; username: string; display_name: string; email: string; is_active: boolean; roles?: Role[] }
export type Role = { id: number; name: string; display_name: string; description?: string; permissions?: Permission[] }
export type Permission = { id: number; code: string; name: string; description?: string }
export type Setting = { key: string; value: string }
export type Summary = { users: number; roles: number; devices_online: number; alerts: number; work_orders_running: number; work_orders_waiting: number }
export type Device = { id: number; plc_id?: number; plc?: PLC; code: string; name: string; type: string; location: string; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type PLC = { id: number; code: string; name: string; protocol: string; host: string; port: number; rack?: number; slot?: number; unit_id?: number; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type WorkOrderStep = { id: number; work_order_id: number; sequence: number; code: string; name: string; device_id?: number; device?: Device; planned_qty: number; passed_qty: number; failed_qty: number; status: "pending" | "ready" | "running" | "paused" | "completed"; started_at?: string; completed_at?: string; notes?: string }
export type WorkOrder = { id: number; code: string; product_code: string; product_name: string; planned_qty: number; completed_qty: number; failed_qty: number; status: "draft" | "released" | "running" | "paused" | "completed" | "cancelled"; priority: string; current_sequence: number; scheduled_start?: string; scheduled_end?: string; notes?: string; version: number; steps?: WorkOrderStep[]; events?: ProductionEvent[]; created_at: string; updated_at: string }
export type ProductionEvent = { id: number; work_order_id: number; work_order_step_id?: number; event_type: string; from_status: string; to_status: string; passed_qty: number; failed_qty: number; reason?: string; payload?: string; source: "operator" | "gateway" | "plc"; operator_name: string; created_at: string }
export type AuditLog = { id: number; username: string; action: string; resource: string; detail: string; method?: string; path?: string; status_code?: number; ip?: string; created_at: string }

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
