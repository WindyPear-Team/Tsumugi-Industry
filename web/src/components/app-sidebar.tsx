import { useLocation, useNavigate } from "react-router-dom"
import { Activity, CircleGauge, ClipboardList, Cpu, Database, FileClock, History, KeyRound, LayoutDashboard, Network, Settings2, ShieldCheck, SlidersHorizontal, Users, Wrench } from "lucide-react"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Sidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupContent, SidebarGroupLabel, SidebarHeader, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar"
import type { User } from "@/lib/api"

const items = [
  { id: "overview", path: "/", label: "运行总览", icon: CircleGauge, group: "工作台" },
  { id: "work-orders", path: "/work-orders", label: "生产工单", icon: ClipboardList, group: "生产执行" },
  { id: "flows", path: "/flows", label: "流程定义", icon: Network, group: "工程配置" },
  { id: "variables", path: "/variables", label: "PLC 变量", icon: SlidersHorizontal, group: "工程配置" },
  { id: "monitoring", path: "/monitoring", label: "监控曲线", icon: Activity, group: "工作台" },
  { id: "dashboards", path: "/dashboards", label: "运行看板", icon: LayoutDashboard, group: "工作台" },
  { id: "flow-runs", path: "/flow-runs", label: "运行记录", icon: History, group: "工程配置" },
  { id: "devices", path: "/devices", label: "设备接入", icon: Database, group: "工作台" },
  { id: "plcs", path: "/plcs", label: "PLC 管理", icon: Cpu, group: "工作台" },
  { id: "audit", path: "/audit", label: "审计日志", icon: FileClock, group: "工作台" },
  { id: "users", path: "/users", label: "用户管理", icon: Users, group: "访问控制" },
  { id: "roles", path: "/roles", label: "角色权限", icon: ShieldCheck, group: "访问控制" },
  { id: "settings", path: "/settings", label: "系统设置", icon: Settings2, group: "系统" },
]

export function AppSidebar({ user, onLogout, systemName, visibleItems }: { user: User; onLogout: () => void; systemName: string; visibleItems?: string[] }) {
  const location = useLocation(); const navigate = useNavigate()
  const visible = visibleItems?.length ? items.filter((item) => visibleItems.includes(item.id)) : items
  const groups = [...new Set(visible.map((item) => item.group))]
  return <Sidebar variant="inset"><SidebarHeader><div className="flex items-center gap-3 px-2 py-2"><div className="flex size-9 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground"><Wrench className="size-4" /></div><div className="grid flex-1 text-left text-sm leading-tight"><span className="truncate font-semibold">{systemName}</span><span className="truncate text-xs text-sidebar-foreground/60">工业控制中心</span></div><Badge variant="outline" className="font-mono text-[10px]">v0.1</Badge></div></SidebarHeader><SidebarContent>{groups.map((group) => <SidebarGroup key={group}><SidebarGroupLabel>{group}</SidebarGroupLabel><SidebarGroupContent><SidebarMenu>{visible.filter((item) => item.group === group).map(({ path, label, icon: Icon }) => <SidebarMenuItem key={path}><SidebarMenuButton isActive={location.pathname === path} tooltip={label} onClick={() => navigate(path)}><Icon /><span>{label}</span></SidebarMenuButton></SidebarMenuItem>)}</SidebarMenu></SidebarGroupContent></SidebarGroup>)}</SidebarContent><SidebarFooter><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" className="h-auto w-full justify-start gap-3 px-2 py-2"><Avatar className="size-8 rounded-lg"><AvatarFallback className="rounded-lg bg-sidebar-primary text-xs text-sidebar-primary-foreground">{(user.display_name || user.username).slice(0, 1).toUpperCase()}</AvatarFallback></Avatar><span className="grid flex-1 text-left text-xs"><span className="truncate font-medium">{user.display_name || user.username}</span><span className="truncate text-sidebar-foreground/60">{user.roles?.[0]?.display_name || "系统用户"}</span></span><KeyRound className="size-3.5 text-sidebar-foreground/50" /></Button></DropdownMenuTrigger><DropdownMenuContent side="top" align="start" className="w-56"><DropdownMenuLabel>账号</DropdownMenuLabel><DropdownMenuSeparator /><DropdownMenuItem onClick={onLogout}>退出登录</DropdownMenuItem></DropdownMenuContent></DropdownMenu></SidebarFooter></Sidebar>
}
