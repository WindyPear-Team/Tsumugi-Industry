export type User = { id: number; username: string; display_name: string; email: string; is_active: boolean; roles?: Role[] }
export type Role = { id: number; name: string; display_name: string; description?: string; permissions?: Permission[] }
export type Permission = { id: number; code: string; name: string; description?: string }
export type Setting = { key: string; value: string }
export type Summary = { users: number; roles: number; devices_online: number; alerts: number }
export type Device = { id: number; plc_id?: number; plc?: PLC; code: string; name: string; type: string; location: string; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type PLC = { id: number; code: string; name: string; protocol: string; host: string; port: number; status: "online" | "offline" | "maintenance"; last_seen_at?: string }
export type AuditLog = { id: number; username: string; action: string; resource: string; detail: string; method?: string; path?: string; status_code?: number; ip?: string; created_at: string }

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem("tsumugi-token")
  const response = await fetch(path, { ...options, headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error ?? "请求失败")
  }
  return response.json()
}
