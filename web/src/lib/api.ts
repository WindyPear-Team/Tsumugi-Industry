export type User = { id: number; username: string; display_name: string; email: string; is_active: boolean; roles?: Role[] }
export type Role = { id: number; name: string; display_name: string; description?: string; permissions?: Permission[] }
export type Permission = { id: number; code: string; name: string; description?: string }
export type Setting = { key: string; value: string }
export type Summary = { users: number; roles: number; devices_online: number; alerts: number }

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem("tsumugi-token")
  const response = await fetch(path, { ...options, headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers } })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new Error(body.error ?? "请求失败")
  }
  return response.json()
}
