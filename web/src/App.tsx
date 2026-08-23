import { useCallback, useEffect, useState } from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { ConsoleLayout } from "@/components/console-layout"
import { ThemeProvider } from "@/components/theme-provider"
import { api, type User } from "@/lib/api"
import { AuditPage } from "@/pages/AuditPage"
import { DashboardPage } from "@/pages/DashboardPage"
import { DevicesPage } from "@/pages/DevicesPage"
import { LoginPage } from "@/pages/LoginPage"
import { PLCPage } from "@/pages/PLCPage"
import { RolesPage } from "@/pages/RolesPage"
import { SettingsPage } from "@/pages/SettingsPage"
import { SetupPage } from "@/pages/SetupPage"
import { UsersPage } from "@/pages/UsersPage"
import { WorkOrdersPage } from "@/pages/WorkOrdersPage"

type SettingsResponse = { items: { key: string; value: string }[] }

function AppRoutes() {
  const [initialized, setInitialized] = useState<boolean | null>(null)
  const [user, setUser] = useState<User | null>(null)
  const [systemName, setSystemName] = useState("Tsumugi Industry")
  const [visibleItems, setVisibleItems] = useState<string[]>([])

  const loadSystemName = useCallback(() => {
    return api<SettingsResponse>("/api/settings").then((result) => {
      const name = result.items.find((setting) => setting.key === "system.name")?.value
      if (name) setSystemName(name)
      const navigation = result.items.find((setting) => setting.key === "navigation.items")?.value
      if (navigation) { try { const items = JSON.parse(navigation) as string[]; setVisibleItems([...new Set([...items, "work-orders", "plcs"])]) } catch { setVisibleItems([]) } }
    })
  }, [])

  useEffect(() => {
    api<{ initialized: boolean }>("/api/setup/status").then((result) => setInitialized(result.initialized)).catch(() => setInitialized(false))
    const token = localStorage.getItem("tsumugi-token")
    if (token) api<{ user: User }>("/api/auth/me").then((result) => { setUser(result.user); return loadSystemName() }).catch(() => localStorage.removeItem("tsumugi-token"))
  }, [loadSystemName])

  useEffect(() => { if (!user) document.title = initialized ? `登录 - ${systemName}` : `系统初始化 - ${systemName}` }, [initialized, systemName, user])

  if (initialized === null) return <main className="grid min-h-svh place-items-center"><Loader2 className="size-5 animate-spin" /></main>
  if (!initialized) return <SetupPage onComplete={() => setInitialized(true)} />
  if (!user) return <LoginPage onLogin={(token, nextUser) => { localStorage.setItem("tsumugi-token", token); setUser(nextUser); void loadSystemName() }} />
  return <Routes><Route element={<ConsoleLayout user={user} systemName={systemName} visibleItems={visibleItems} onLogout={() => { localStorage.removeItem("tsumugi-token"); setUser(null) }} />}><Route index element={<DashboardPage />} /><Route path="work-orders" element={<WorkOrdersPage />} /><Route path="devices" element={<DevicesPage />} /><Route path="plcs" element={<PLCPage />} /><Route path="audit" element={<AuditPage />} /><Route path="users" element={<UsersPage />} /><Route path="roles" element={<RolesPage />} /><Route path="settings" element={<SettingsPage />} /><Route path="*" element={<Navigate to="/" replace />} /></Route></Routes>
}

export default function App() { return <ThemeProvider defaultTheme="system"><BrowserRouter><AppRoutes /></BrowserRouter></ThemeProvider> }
