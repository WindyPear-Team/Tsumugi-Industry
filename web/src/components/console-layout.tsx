import { useEffect } from "react"
import { Moon, Sun } from "lucide-react"
import { Outlet, useLocation } from "react-router-dom"
import { AppSidebar } from "@/components/app-sidebar"
import { Button } from "@/components/ui/button"
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"
import { useTheme } from "@/components/theme-provider"
import type { User } from "@/lib/api"

const titles: Record<string, [string, string]> = { "/": ["运行总览", "实时掌握生产系统运行状态"], "/work-orders": ["生产工单", "按工序推进生产计划并记录现场结果"], "/devices": ["设备接入", "监控现场设备与连接状态"], "/plcs": ["PLC 管理", "管理控制器连接与设备数据来源"], "/audit": ["审计日志", "追踪关键系统操作"], "/users": ["用户管理", "维护账号与访问范围"], "/roles": ["角色权限", "配置细粒度访问策略"], "/settings": ["系统设置", "管理保存在数据库中的运行参数"] }

export function ConsoleLayout({ user, onLogout, systemName, visibleItems }: { user: User; onLogout: () => void; systemName: string; visibleItems?: string[] }) {
  const { theme, setTheme } = useTheme(); const location = useLocation(); const [title, description] = titles[location.pathname] ?? titles["/"]
  useEffect(() => { document.title = `${title} - ${systemName}` }, [title, systemName])
  return <TooltipProvider><SidebarProvider><AppSidebar user={user} systemName={systemName} visibleItems={visibleItems} onLogout={onLogout} /><SidebarInset><header className="flex h-16 shrink-0 items-center justify-between border-b border-border/70 px-4 lg:px-6"><div className="flex items-center gap-3"><SidebarTrigger className="-ml-1" /><div className="h-4 w-px bg-border" /><div><h1 className="text-sm font-semibold">{title}</h1><p className="hidden text-xs text-muted-foreground sm:block">{description}</p></div></div><Button variant="ghost" size="icon" onClick={() => setTheme(theme === "dark" ? "light" : "dark")} aria-label="切换主题">{theme === "dark" ? <Sun /> : <Moon />}</Button></header><main className="flex-1 overflow-auto bg-muted/20 p-4 lg:p-6"><div className="mx-auto max-w-[1600px]"><Outlet /></div></main></SidebarInset></SidebarProvider></TooltipProvider>
}
