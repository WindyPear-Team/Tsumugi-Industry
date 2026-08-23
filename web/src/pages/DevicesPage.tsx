import { useEffect, useState, type FormEvent } from "react"
import { Activity, Check, Loader2, Plus, Search, Wrench } from "lucide-react"
import { api, type Device } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const statusLabel = { online: "在线", offline: "离线", maintenance: "维护中" }

export function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>([])
  const [search, setSearch] = useState("")
  const [showForm, setShowForm] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [form, setForm] = useState({ code: "", name: "", type: "PLC", location: "", status: "offline" })

  async function loadDevices(nextSearch = search) {
    setLoading(true)
    try {
      const query = nextSearch ? `?search=${encodeURIComponent(nextSearch)}` : ""
      const result = await api<{ items: Device[] }>(`/api/devices${query}`)
      setDevices(result.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载设备失败")
    } finally { setLoading(false) }
  }

  useEffect(() => {
    api<{ items: Device[] }>("/api/devices").then((result) => setDevices(result.items)).catch((err) => setError(err instanceof Error ? err.message : "加载设备失败")).finally(() => setLoading(false))
  }, [])

  async function createDevice(event: FormEvent) {
    event.preventDefault(); setSaving(true); setError("")
    try {
      await api("/api/devices", { method: "POST", body: JSON.stringify(form) })
      setForm({ code: "", name: "", type: "PLC", location: "", status: "offline" }); setShowForm(false); await loadDevices()
    } catch (err) { setError(err instanceof Error ? err.message : "创建设备失败") } finally { setSaving(false) }
  }

  async function changeStatus(device: Device, status: Device["status"]) {
    try { await api(`/api/devices/${device.id}/status`, { method: "PATCH", body: JSON.stringify({ status }) }); await loadDevices() }
    catch (err) { setError(err instanceof Error ? err.message : "更新设备状态失败") }
  }

  return <div className="space-y-6"><Card className="border-border/70 shadow-none"><CardHeader><div className="flex items-center justify-between gap-4"><div><CardTitle>设备接入</CardTitle><CardDescription>维护设备台账、连接状态和现场位置。</CardDescription></div><Button onClick={() => setShowForm((value) => !value)}>{showForm ? "取消" : <><Plus />接入设备</>}</Button></div></CardHeader>{showForm && <form onSubmit={createDevice} className="border-t border-border/70 bg-muted/20 p-6"><div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4"><div className="space-y-2"><Label>设备编码</Label><Input required placeholder="PLC-A-001" value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} /></div><div className="space-y-2"><Label>设备名称</Label><Input required placeholder="装配线主控 PLC" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></div><div className="space-y-2"><Label>设备类型</Label><Input placeholder="PLC / Robot / Sensor" value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })} /></div><div className="space-y-2"><Label>现场位置</Label><Input placeholder="一号车间 / A 线" value={form.location} onChange={(e) => setForm({ ...form, location: e.target.value })} /></div></div><Button className="mt-5" disabled={saving}>{saving && <Loader2 className="animate-spin" />}保存设备</Button></form>}</Card><Card className="border-border/70 shadow-none"><CardContent className="p-5"><div className="relative max-w-sm"><Search className="absolute top-2.5 left-3 size-4 text-muted-foreground" /><Input className="pl-9" placeholder="搜索编码、名称或位置" value={search} onChange={(e) => { setSearch(e.target.value); void loadDevices(e.target.value) }} /></div>{error && <p className="mt-4 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">{error}</p>}<div className="mt-5 overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left text-xs text-muted-foreground"><th className="pb-3 font-medium">设备</th><th className="pb-3 font-medium">类型 / 位置</th><th className="pb-3 font-medium">状态</th><th className="pb-3 text-right font-medium">操作</th></tr></thead><tbody>{loading ? <tr><td colSpan={4} className="py-12 text-center text-muted-foreground"><Loader2 className="mx-auto size-5 animate-spin" /></td></tr> : devices.length === 0 ? <tr><td colSpan={4} className="py-12 text-center text-muted-foreground">暂无设备，先接入一台设备。</td></tr> : devices.map((device) => <tr key={device.id} className="border-b last:border-0"><td className="py-4"><div className="flex items-center gap-3"><div className="flex size-8 items-center justify-center rounded-lg bg-muted"><Wrench className="size-4" /></div><div><p className="font-medium">{device.name}</p><p className="font-mono text-xs text-muted-foreground">{device.code}</p></div></div></td><td className="py-4"><p>{device.type || "未分类"}</p><p className="text-xs text-muted-foreground">{device.location || "未设置位置"}</p></td><td className="py-4"><span className="inline-flex items-center gap-1.5 text-xs"><span className={`size-1.5 rounded-full ${device.status === "online" ? "bg-foreground" : "bg-muted-foreground/50"}`} />{statusLabel[device.status]}</span></td><td className="py-4 text-right"><Select value={device.status} onValueChange={(value) => void changeStatus(device, value as Device["status"])}><SelectTrigger className="ml-auto w-28"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="online"><Check />在线</SelectItem><SelectItem value="offline">离线</SelectItem><SelectItem value="maintenance">维护中</SelectItem></SelectContent></Select></td></tr>)}</tbody></table></div></CardContent></Card><div className="flex items-center gap-2 text-xs text-muted-foreground"><Activity className="size-3.5" />设备状态由接入适配器或运维人员更新。</div></div>
}
