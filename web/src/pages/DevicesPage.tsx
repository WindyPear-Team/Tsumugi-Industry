import { useEffect, useState, type FormEvent } from "react"
import { Activity, Loader2, Pencil, Plus, Trash2, Wrench } from "lucide-react"
import { api, type Device, type PLC } from "@/lib/api"
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

type Page = {
  page: number
  page_count: number
  total: number
  page_size: number
}
type FormState = {
  code: string
  name: string
  type: string
  location: string
  plc_id: string
}
const emptyForm: FormState = {
  code: "",
  name: "",
  type: "Sensor",
  location: "",
  plc_id: "",
}

export function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>([])
  const [plcs, setPLCs] = useState<PLC[]>([])
  const [page, setPage] = useState<Page>({
    page: 1,
    page_count: 1,
    total: 0,
    page_size: 20,
  })
  const [sortField, setSortField] = useState("id")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc")
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [editing, setEditing] = useState<Device | null>(null)
  const [deleting, setDeleting] = useState<Device | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const params = new URLSearchParams({
    sort_by: sortField,
    sort_order: sortOrder,
    page: String(page.page),
    page_size: String(page.page_size),
  })
  Object.entries(filters).forEach(([key, value]) => {
    if (value) params.set(`filter_${key}`, value)
  })
  const query = params.toString()
  useEffect(() => {
    api<{ items: Device[]; page: Page }>(`/api/devices?${query}`)
      .then((result) => {
        setDevices(result.items)
        setPage(result.page)
      })
      .catch((err) =>
        setError(err instanceof Error ? err.message : "加载设备失败")
      )
      .finally(() => setLoading(false))
  }, [query])
  useEffect(() => {
    api<{ items: PLC[] }>("/api/plcs?page_size=100")
      .then((result) => setPLCs(result.items))
      .catch(() => undefined)
  }, [])
  function filter(field: string, value: string) {
    setFilters((current) => ({ ...current, [field]: value }))
    setPage((current) => ({ ...current, page: 1 }))
  }
  const menu = (label: string, field: string) => (
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
      onFilter={(value) => filter(field, value)}
    />
  )
  function edit(device?: Device) {
    setEditing(device ?? null)
    setForm(
      device
        ? {
            code: device.code,
            name: device.name,
            type: device.type,
            location: device.location,
            plc_id: device.plc_id ? String(device.plc_id) : "",
          }
        : emptyForm
    )
    setOpen(true)
    setError("")
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    try {
      await api(editing ? `/api/devices/${editing.id}` : "/api/devices", {
        method: editing ? "PUT" : "POST",
        body: JSON.stringify({
          ...form,
          plc_id: form.plc_id ? Number(form.plc_id) : null,
        }),
      })
      setOpen(false)
      const result = await api<{ items: Device[]; page: Page }>(
        `/api/devices?${query}`
      )
      setDevices(result.items)
      setPage(result.page)
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存设备失败")
    }
  }
  async function remove() {
    if (!deleting) return
    try {
      await api(`/api/devices/${deleting.id}`, { method: "DELETE" })
      setDeleting(null)
      const result = await api<{ items: Device[]; page: Page }>(
        `/api/devices?${query}`
      )
      setDevices(result.items)
      setPage(result.page)
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除设备失败")
      setDeleting(null)
    }
  }
  return (
    <div className="space-y-6">
      <Card className="border-border/70 shadow-none">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>设备接入</CardTitle>
              <CardDescription>
                每个设备是独立对象，并通过来源 PLC 获取数据。
              </CardDescription>
            </div>
            <Button onClick={() => edit()}>
              <Plus />
              接入设备
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs text-muted-foreground">
                  <th className="pb-3">{menu("编码", "code")}</th>
                  <th className="pb-3">{menu("名称", "name")}</th>
                  <th className="pb-3">来源 PLC</th>
                  <th className="pb-3">{menu("类型", "type")}</th>
                  <th className="pb-3">{menu("位置", "location")}</th>
                  <th className="pb-3">{menu("状态", "status")}</th>
                  <th className="pb-3 text-right">操作</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <tr>
                    <td colSpan={7} className="py-12 text-center">
                      <Loader2 className="mx-auto size-5 animate-spin" />
                    </td>
                  </tr>
                ) : devices.length === 0 ? (
                  <tr>
                    <td
                      colSpan={7}
                      className="py-12 text-center text-muted-foreground"
                    >
                      暂无设备
                    </td>
                  </tr>
                ) : (
                  devices.map((device) => (
                    <tr key={device.id} className="border-b last:border-0">
                      <td className="py-4 font-mono text-xs">{device.code}</td>
                      <td className="py-4">
                        <div className="flex items-center gap-3">
                          <div className="flex size-8 items-center justify-center rounded-lg bg-muted">
                            <Wrench className="size-4" />
                          </div>
                          {device.name}
                        </div>
                      </td>
                      <td className="py-4 text-xs">
                        {device.plc?.name ?? "未绑定"}
                      </td>
                      <td className="py-4">{device.type || "未分类"}</td>
                      <td className="py-4 text-muted-foreground">
                        {device.location || "未设置"}
                      </td>
                      <td className="py-4 text-xs">
                        {device.status === "online"
                          ? "在线"
                          : device.status === "maintenance"
                            ? "维护中"
                            : "离线"}
                      </td>
                      <td className="py-4 text-right">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => edit(device)}
                        >
                          <Pencil />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive"
                          onClick={() => setDeleting(device)}
                        >
                          <Trash2 />
                        </Button>
                      </td>
                    </tr>
                  ))
                )}
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
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <Activity className="size-3.5" />
        设备状态由接入适配器或运维人员更新。
      </div>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "编辑设备" : "接入设备"}</DialogTitle>
            <DialogDescription>设备信息会保存到设备台账。</DialogDescription>
          </DialogHeader>
          <form onSubmit={save} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>设备编码</Label>
                <Input
                  required
                  value={form.code}
                  onChange={(event) =>
                    setForm({ ...form, code: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>设备名称</Label>
                <Input
                  required
                  value={form.name}
                  onChange={(event) =>
                    setForm({ ...form, name: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>设备类型</Label>
                <Input
                  value={form.type}
                  onChange={(event) =>
                    setForm({ ...form, type: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>现场位置</Label>
                <Input
                  value={form.location}
                  onChange={(event) =>
                    setForm({ ...form, location: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label>来源 PLC</Label>
                <Select
                  value={form.plc_id}
                  onValueChange={(plc_id) => setForm({ ...form, plc_id })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择设备数据来源 PLC" />
                  </SelectTrigger>
                  <SelectContent>
                    {plcs.map((plc) => (
                      <SelectItem value={String(plc.id)} key={plc.id}>
                        {plc.name} · {plc.code}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setOpen(false)}
              >
                取消
              </Button>
              <Button type="submit">保存</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={Boolean(deleting)}
        onOpenChange={(value) => !value && setDeleting(null)}
        title="确认删除设备？"
        description={`将删除设备“${deleting?.name ?? ""}”，该操作不可撤销。`}
        onConfirm={() => void remove()}
      />
    </div>
  )
}
