import { useCallback, useEffect, useState, type FormEvent } from "react"
import {
  Database,
  Palette,
  Pencil,
  RefreshCw,
  Save,
  Settings2,
  Timer,
} from "lucide-react"
import { api, type Setting } from "@/lib/api"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { TableColumnMenu } from "@/components/table-column-menu"
import { TablePagination } from "@/components/table-pagination"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

type Task = {
  id: number
  name: string
  task_type: string
  interval_seconds: number
  enabled: boolean
  last_status: string
}
type Backup = {
  id: number
  name: string
  size: number
  status: string
  created_by: string
  created_at: string
}
type Page = {
  page: number
  page_count: number
  total: number
  page_size: number
}

export function SettingsPage() {
  const [settings, setSettings] = useState<Setting[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [backups, setBackups] = useState<Backup[]>([])
  const [page, setPage] = useState<Page>({
    page: 1,
    page_count: 1,
    total: 0,
    page_size: 20,
  })
  const [editing, setEditing] = useState<Setting | null>(null)
  const [value, setValue] = useState("")
  const [sortField, setSortField] = useState("key")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc")
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [error, setError] = useState("")
  const params = new URLSearchParams({
    sort_by: sortField,
    sort_order: sortOrder,
    page: String(page.page),
    page_size: String(page.page_size),
  })
  Object.entries(filters).forEach(([key, filter]) => {
    if (filter) params.set(`filter_${key}`, filter)
  })
  const queryString = params.toString()
  const reloadSettings = useCallback(() => {
    api<{ items: Setting[]; page: Page }>(`/api/settings?${queryString}`)
      .then((result) => {
        setSettings(result.items)
        setPage(result.page)
      })
      .catch((err) =>
        setError(err instanceof Error ? err.message : "加载设置失败")
      )
  }, [queryString])
  useEffect(() => {
    reloadSettings()
  }, [reloadSettings])
  useEffect(() => {
    api<{ items: Task[] }>("/api/tasks")
      .then((result) => setTasks(result.items))
      .catch(() => undefined)
    api<{ items: Backup[] }>("/api/backups")
      .then((result) => setBackups(result.items))
      .catch(() => undefined)
  }, [])
  function setFilter(field: string, filter: string) {
    setFilters((current) => ({ ...current, [field]: filter }))
    setPage((current) => ({ ...current, page: 1 }))
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!editing) return
    try {
      await api(`/api/settings/${encodeURIComponent(editing.key)}`, {
        method: "PUT",
        body: JSON.stringify({ value }),
      })
      setEditing(null)
      reloadSettings()
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存设置失败")
    }
  }
  async function createBackup() {
    try {
      const result = await api<{ backup: Backup }>("/api/backups", {
        method: "POST",
      })
      setBackups((current) => [result.backup, ...current])
    } catch (err) {
      setError(err instanceof Error ? err.message : "备份失败")
    }
  }
  async function toggleTask(task: Task) {
    await api(`/api/tasks/${task.id}`, {
      method: "PUT",
      body: JSON.stringify({ enabled: !task.enabled }),
    })
    setTasks((current) =>
      current.map((item) =>
        item.id === task.id ? { ...item, enabled: !item.enabled } : item
      )
    )
  }
  const settingMenu = (label: string, field: string) => (
    <TableColumnMenu
      label={label}
      field={field}
      filter={filters[field]}
      sortField={sortField}
      sortOrder={sortOrder}
      onSort={(next, order) => {
        setSortField(next)
        setSortOrder(order)
      }}
      onFilter={(filter) => setFilter(field, filter)}
    />
  )
  return (
    <Tabs defaultValue="general" className="space-y-6">
      <TabsList>
        <TabsTrigger value="general">
          <Settings2 />
          通用设置
        </TabsTrigger>
        <TabsTrigger value="brand">
          <Palette />
          品牌与导航
        </TabsTrigger>
        <TabsTrigger value="tasks">
          <Timer />
          定时任务
        </TabsTrigger>
        <TabsTrigger value="maintenance">
          <RefreshCw />
          维护与备份
        </TabsTrigger>
      </TabsList>
      <TabsContent value="general">
        <Card className="border-border/70 shadow-none">
          <CardHeader>
            <CardTitle>通用设置</CardTitle>
            <CardDescription>系统标题、时区和运行偏好。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-muted-foreground">
                    <th className="pb-3">{settingMenu("设置项", "key")}</th>
                    <th className="pb-3">{settingMenu("值", "value")}</th>
                    <th className="pb-3 text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {settings
                    .filter((setting) => setting.key.startsWith("system."))
                    .map((setting) => (
                      <tr key={setting.key} className="border-b last:border-0">
                        <td className="py-4 font-mono text-xs text-muted-foreground">
                          {setting.key}
                        </td>
                        <td className="py-4">{setting.value}</td>
                        <td className="py-4 text-right">
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => {
                              setEditing(setting)
                              setValue(setting.value)
                            }}
                          >
                            <Pencil />
                          </Button>
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
            <TablePagination
              page={page}
              onPageChange={(next) =>
                setPage((current) => ({ ...current, page: next }))
              }
            />
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="brand">
        <BrandSettings settings={settings} onSaved={reloadSettings} />
      </TabsContent>
      <TabsContent value="tasks">
        <TaskSettings tasks={tasks} onToggle={toggleTask} />
      </TabsContent>
      <TabsContent value="maintenance">
        <MaintenanceSettings backups={backups} onBackup={createBackup} />
      </TabsContent>
      <Dialog
        open={Boolean(editing)}
        onOpenChange={(open) => !open && setEditing(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>编辑系统设置</DialogTitle>
            <DialogDescription>{editing?.key}</DialogDescription>
          </DialogHeader>
          <form onSubmit={save} className="space-y-4">
            <div className="space-y-2">
              <Label>值</Label>
              <Input
                value={value}
                onChange={(event) => setValue(event.target.value)}
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setEditing(null)}
              >
                取消
              </Button>
              <Button type="submit">
                <Save />
                保存
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      {error && <p className="text-sm text-destructive">{error}</p>}
    </Tabs>
  )
}

function BrandSettings({
  settings,
  onSaved,
}: {
  settings: Setting[]
  onSaved: () => void
}) {
  const [name, setName] = useState(
    settings.find((item) => item.key === "system.name")?.value ??
      "Tsumugi Industry"
  )
  const [subtitle, setSubtitle] = useState(
    settings.find((item) => item.key === "brand.subtitle")?.value ??
      "工业控制中心"
  )
  const [logo, setLogo] = useState(
    settings.find((item) => item.key === "brand.logo")?.value ?? ""
  )
  const navigation = [
    { id: "overview", label: "运行总览" },
    { id: "work-orders", label: "生产工单" },
    { id: "flows", label: "流程定义" },
    { id: "variables", label: "PLC 变量" },
    { id: "flow-runs", label: "运行记录" },
    { id: "plcs", label: "PLC 管理" },
    { id: "devices", label: "设备接入" },
    { id: "audit", label: "审计日志" },
    { id: "users", label: "用户管理" },
    { id: "roles", label: "角色权限" },
    { id: "settings", label: "系统设置" },
  ]
  const current = (() => {
    try {
      return JSON.parse(
        settings.find((item) => item.key === "navigation.items")?.value ?? "[]"
      ) as string[]
    } catch {
      return navigation.map((item) => item.id)
    }
  })()
  const [visible, setVisible] = useState<string[]>(current)
  async function save() {
    await api("/api/settings/system.name", {
      method: "PUT",
      body: JSON.stringify({ value: name }),
    })
    await api("/api/settings/brand.subtitle", {
      method: "PUT",
      body: JSON.stringify({ value: subtitle }),
    })
    await api("/api/settings/brand.logo", {
      method: "PUT",
      body: JSON.stringify({ value: logo }),
    })
    await api("/api/settings/navigation.items", {
      method: "PUT",
      body: JSON.stringify({ value: JSON.stringify(visible) }),
    })
    onSaved()
  }
  return (
    <Card className="border-border/70 shadow-none">
      <CardHeader>
        <CardTitle>品牌与导航</CardTitle>
        <CardDescription>配置系统标题、品牌标识和侧边栏项目。</CardDescription>
      </CardHeader>
      <CardContent className="max-w-xl space-y-5">
        <div className="space-y-2">
          <Label>系统标题</Label>
          <Input
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>品牌副标题</Label>
          <Input
            value={subtitle}
            onChange={(event) => setSubtitle(event.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>品牌 Logo 地址</Label>
          <Input
            placeholder="https://... 或 /logo.svg"
            value={logo}
            onChange={(event) => setLogo(event.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label>侧边栏项目</Label>
          <div className="grid gap-2 rounded-lg border border-border p-3 sm:grid-cols-2">
            {navigation.map((item) => (
              <label className="flex items-center gap-2 text-sm" key={item.id}>
                <input
                  type="checkbox"
                  checked={visible.includes(item.id)}
                  onChange={() =>
                    setVisible((currentItems) =>
                      currentItems.includes(item.id)
                        ? currentItems.filter((id) => id !== item.id)
                        : [...currentItems, item.id]
                    )
                  }
                />
                {item.label}
              </label>
            ))}
          </div>
        </div>
        <Button onClick={() => void save()}>
          <Save />
          保存品牌设置
        </Button>
      </CardContent>
    </Card>
  )
}

function TaskSettings({
  tasks,
  onToggle,
}: {
  tasks: Task[]
  onToggle: (task: Task) => Promise<void>
}) {
  return (
    <Card className="border-border/70 shadow-none">
      <CardHeader>
        <CardTitle>定时任务</CardTitle>
        <CardDescription>设备状态监测、日志清理和数据库备份。</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {tasks.map((task) => (
            <div
              key={task.id}
              className="flex items-center justify-between rounded-xl border border-border/70 p-4"
            >
              <div>
                <p className="font-medium">{task.name}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  每 {task.interval_seconds} 秒 ·{" "}
                  {task.last_status || "尚未运行"}
                </p>
              </div>
              <Button
                variant={task.enabled ? "default" : "outline"}
                size="sm"
                onClick={() => void onToggle(task)}
              >
                {task.enabled ? "运行中" : "已停用"}
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function MaintenanceSettings({
  backups,
  onBackup,
}: {
  backups: Backup[]
  onBackup: () => Promise<void>
}) {
  const [restoring, setRestoring] = useState<Backup | null>(null)
  async function restore() {
    if (!restoring) return
    await api(`/api/backups/${restoring.id}/restore`, { method: "POST" })
    setRestoring(null)
  }
  return (
    <Card className="border-border/70 shadow-none">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>维护与备份</CardTitle>
            <CardDescription>创建数据库快照或恢复已有备份。</CardDescription>
          </div>
          <Button onClick={() => void onBackup()}>
            <Database />
            立即备份
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {backups.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              暂无备份
            </p>
          ) : (
            backups.map((backup) => (
              <div
                key={backup.id}
                className="flex items-center justify-between border-b border-border/70 py-3 last:border-0"
              >
                <div>
                  <p className="font-mono text-xs">{backup.name}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {new Date(backup.created_at).toLocaleString()} ·{" "}
                    {backup.size} bytes
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs">{backup.status}</span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setRestoring(backup)}
                  >
                    恢复
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>
        <ConfirmDialog
          open={Boolean(restoring)}
          onOpenChange={(open) => !open && setRestoring(null)}
          title="确认恢复备份？"
          description={`恢复“${restoring?.name ?? ""}”会覆盖当前系统数据。`}
          onConfirm={() => void restore()}
        />
      </CardContent>
    </Card>
  )
}
