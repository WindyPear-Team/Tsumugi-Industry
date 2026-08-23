import { useEffect, useState, type FormEvent } from "react"
import { Cpu, Loader2, Pencil, Plus, Trash2 } from "lucide-react"
import { api, type PLC } from "@/lib/api"
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

type Page = {
  page: number
  page_count: number
  total: number
  page_size: number
}
type FormState = {
  code: string
  name: string
  protocol: string
  host: string
  port: string
}
const emptyForm: FormState = {
  code: "",
  name: "",
  protocol: "modbus-tcp",
  host: "",
  port: "502",
}

export function PLCPage() {
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
  const [editing, setEditing] = useState<PLC | null>(null)
  const [deleting, setDeleting] = useState<PLC | null>(null)
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
    api<{ items: PLC[]; page: Page }>(`/api/plcs?${query}`)
      .then((result) => {
        setPLCs(result.items)
        setPage(result.page)
      })
      .catch((err) =>
        setError(err instanceof Error ? err.message : "加载 PLC 失败")
      )
      .finally(() => setLoading(false))
  }, [query])
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
  function edit(plc?: PLC) {
    setEditing(plc ?? null)
    setForm(
      plc
        ? {
            code: plc.code,
            name: plc.name,
            protocol: plc.protocol,
            host: plc.host,
            port: String(plc.port || ""),
          }
        : emptyForm
    )
    setOpen(true)
    setError("")
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    try {
      await api(editing ? `/api/plcs/${editing.id}` : "/api/plcs", {
        method: editing ? "PUT" : "POST",
        body: JSON.stringify({ ...form, port: Number(form.port) }),
      })
      setOpen(false)
      const result = await api<{ items: PLC[]; page: Page }>(
        `/api/plcs?${query}`
      )
      setPLCs(result.items)
      setPage(result.page)
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存 PLC 失败")
    }
  }
  async function remove() {
    if (!deleting) return
    try {
      await api(`/api/plcs/${deleting.id}`, { method: "DELETE" })
      setDeleting(null)
      const result = await api<{ items: PLC[]; page: Page }>(
        `/api/plcs?${query}`
      )
      setPLCs(result.items)
      setPage(result.page)
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除 PLC 失败")
      setDeleting(null)
    }
  }
  return (
    <Card className="border-border/70 shadow-none">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>PLC 管理</CardTitle>
            <CardDescription>
              PLC 是独立的控制器对象，设备通过 PLC 取得数据。
            </CardDescription>
          </div>
          <Button onClick={() => edit()}>
            <Plus />
            新增 PLC
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
                <th className="pb-3">{menu("协议", "protocol")}</th>
                <th className="pb-3">{menu("地址", "host")}</th>
                <th className="pb-3">{menu("状态", "status")}</th>
                <th className="pb-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="py-12 text-center">
                    <Loader2 className="mx-auto size-5 animate-spin" />
                  </td>
                </tr>
              ) : plcs.length === 0 ? (
                <tr>
                  <td
                    colSpan={6}
                    className="py-12 text-center text-muted-foreground"
                  >
                    暂无 PLC
                  </td>
                </tr>
              ) : (
                plcs.map((plc) => (
                  <tr key={plc.id} className="border-b last:border-0">
                    <td className="py-4 font-mono text-xs">{plc.code}</td>
                    <td className="py-4">
                      <div className="flex items-center gap-3">
                        <div className="flex size-8 items-center justify-center rounded-lg bg-muted">
                          <Cpu className="size-4" />
                        </div>
                        {plc.name}
                      </div>
                    </td>
                    <td className="py-4">{plc.protocol}</td>
                    <td className="py-4 font-mono text-xs text-muted-foreground">
                      {plc.host}:{plc.port}
                    </td>
                    <td className="py-4 text-xs">
                      {plc.status === "online"
                        ? "在线"
                        : plc.status === "maintenance"
                          ? "维护中"
                          : "离线"}
                    </td>
                    <td className="py-4 text-right">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => edit(plc)}
                      >
                        <Pencil />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive"
                        onClick={() => setDeleting(plc)}
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
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "编辑 PLC" : "新增 PLC"}</DialogTitle>
            <DialogDescription>
              配置 PLC 连接信息，设备将通过此 PLC 获取数据。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={save} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>PLC 编码</Label>
                <Input
                  required
                  value={form.code}
                  onChange={(event) =>
                    setForm({ ...form, code: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>PLC 名称</Label>
                <Input
                  required
                  value={form.name}
                  onChange={(event) =>
                    setForm({ ...form, name: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>协议</Label>
                <Input
                  required
                  placeholder="modbus-tcp / opcua"
                  value={form.protocol}
                  onChange={(event) =>
                    setForm({ ...form, protocol: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>主机</Label>
                <Input
                  required
                  value={form.host}
                  onChange={(event) =>
                    setForm({ ...form, host: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>端口</Label>
                <Input
                  type="number"
                  value={form.port}
                  onChange={(event) =>
                    setForm({ ...form, port: event.target.value })
                  }
                />
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
        title="确认删除 PLC？"
        description={`将删除 PLC“${deleting?.name ?? ""}”。必须先解除其设备关联。`}
        onConfirm={() => void remove()}
      />
    </Card>
  )
}
