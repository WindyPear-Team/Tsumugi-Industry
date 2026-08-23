import { useEffect, useState } from "react"
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { ThemeProvider } from "@/components/theme-provider"
import { ConsoleLayout } from "@/components/console-layout"
import { api, type User } from "@/lib/api"
import { AuditPage } from "@/pages/AuditPage"
import { DashboardPage } from "@/pages/DashboardPage"
import { DevicesPage } from "@/pages/DevicesPage"
import { LoginPage } from "@/pages/LoginPage"
import { RolesPage } from "@/pages/RolesPage"
import { SettingsPage } from "@/pages/SettingsPage"
import { SetupPage } from "@/pages/SetupPage"
import { UsersPage } from "@/pages/UsersPage"

function AppRoutes() { const [initialized, setInitialized] = useState<boolean | null>(null); const [user, setUser] = useState<User | null>(null); useEffect(() => { api<{ initialized: boolean }>("/api/setup/status").then((result) => setInitialized(result.initialized)).catch(() => setInitialized(false)); if (localStorage.getItem("tsumugi-token")) api<{ user: User }>("/api/auth/me").then((result) => setUser(result.user)).catch(() => localStorage.removeItem("tsumugi-token")) }, []); if (initialized === null) return <main className="grid min-h-svh place-items-center"><Loader2 className="size-5 animate-spin" /></main>; if (!initialized) return <SetupPage onComplete={() => setInitialized(true)} />; if (!user) return <LoginPage onLogin={(token, nextUser) => { localStorage.setItem("tsumugi-token", token); setUser(nextUser) }} />; return <Routes><Route element={<ConsoleLayout user={user} onLogout={() => { localStorage.removeItem("tsumugi-token"); setUser(null) }} />}><Route index element={<DashboardPage />} /><Route path="devices" element={<DevicesPage />} /><Route path="audit" element={<AuditPage />} /><Route path="users" element={<UsersPage />} /><Route path="roles" element={<RolesPage />} /><Route path="settings" element={<SettingsPage />} /><Route path="*" element={<Navigate to="/" replace />} /></Route></Routes> }

export default function App() { return <ThemeProvider defaultTheme="system"><BrowserRouter><AppRoutes /></BrowserRouter></ThemeProvider> }
