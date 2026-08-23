import { useEffect, useState, type FormEvent } from "react"
import { Cpu, Loader2, Pencil, Plus, RefreshCw, Search, Trash2 } from "lucide-react"
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

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
  rack: string
  slot: string
}
const emptyForm: FormState = {
  code: "",
  name: "",
  protocol: "modbus-tcp",
  host: "",
  port: "502",
  rack: "0",
  slot: "2",
}

type QueryResult = { address: string; value: unknown; quality: string; timestamp: string }

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
  const [statusLoading, setStatusLoading] = useState<number | null>(null)
  const [statusText, setStatusText] = useState<Record<number, string>>({})
  const [queryPLC, setQueryPLC] = useState<PLC | null>(null)
  const [queryAddresses, setQueryAddresses] = useState("DB1.DBW2\nMB10")
  const [queryResults, setQueryResults] = useState<QueryResult[]>([])
  const [queryLoading, setQueryLoading] = useState(false)
  const [queryError, setQueryError] = useState("")
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
            rack: String(plc.rack ?? 0),
            slot: String(plc.slot ?? 2),
          }
        : emptyForm
    )
    setOpen(true)
    setError("")
  }
  async function checkStatus(plc: PLC) {
    setStatusLoading(plc.id)
    try {
      const result = await api<{ status: { code: number; label: string } }>(`/api/plcs/${plc.id}/status`)
      const label = result.status.label === "run" ? "运行" : result.status.label === "stop" ? "停止" : result.status.label
      setStatusText((current) => ({ ...current, [plc.id]: `CPU：${label}（${result.status.code}）` }))
      setPLCs((current) => current.map((item) => item.id === plc.id ? { ...item, status: "online" } : item))
    } catch (err) {
      setStatusText((current) => ({ ...current, [plc.id]: err instanceof Error ? `失败：${err.message}` : "连接失败" }))
      setPLCs((current) => current.map((item) => item.id === plc.id && item.status !== "maintenance" ? { ...item, status: "offline" } : item))
    } finally { setStatusLoading(null) }
  }
  async function queryValues(event: FormEvent) {
    event.preventDefault(); if (!queryPLC) return
    setQueryLoading(true); setQueryError("")
    try {
      const addresses = queryAddresses.split(/\r?\n/).map((address) => address.trim()).filter(Boolean).map((address) => ({ address, length: 8 }))
      if (addresses.length === 0) {
        setQueryError("至少输入一个查询地址")
        return
      }
      const result = await api<{ items: QueryResult[] }>(`/api/plcs/${queryPLC.id}/query`, { method: "POST", body: JSON.stringify({ addresses }) })
      setQueryResults(result.items)
      setPLCs((current) => current.map((item) => item.id === queryPLC.id ? { ...item, status: "online" } : item))
    } catch (err) { setQueryError(err instanceof Error ? err.message : "PLC 查询失败") } finally { setQueryLoading(false) }
  }
  async function save(event: FormEvent) {
    event.preventDefault()
    try {
      await api(editing ? `/api/plcs/${editing.id}` : "/api/plcs", {
        method: editing ? "PUT" : "POST",
        body: JSON.stringify({
          ...form,
          port: Number(form.port),
          rack: Number(form.rack),
          slot: Number(form.slot),
        }),
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
                      <div className="flex flex-wrap items-start justify-end gap-1">
                        <div className="flex flex-col items-end">
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={statusLoading === plc.id}
                            onClick={() => void checkStatus(plc)}
                          >
                            {statusLoading === plc.id ? (
                              <Loader2 className="animate-spin" />
                            ) : (
                              <RefreshCw />
                            )}
                            测试连接
                          </Button>
                          {statusText[plc.id] && (
                            <p className="mt-1 max-w-40 text-xs text-muted-foreground">
                              {statusText[plc.id]}
                            </p>
                          )}
                        </div>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          title="查询数据"
                          onClick={() => {
                            setQueryPLC(plc)
                            setQueryResults([])
                            setQueryError("")
                          }}
                        >
                          <Search />
                        </Button>
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
                      </div>
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
                <Select
                  required
                  value={form.protocol}
                  onValueChange={(protocol) => setForm({ ...form, protocol })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择 PLC 协议" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="s7comm">Siemens S7comm</SelectItem>
                    <SelectItem value="opcua">OPC UA</SelectItem>
                    <SelectItem value="modbus-tcp">Modbus TCP</SelectItem>
                  </SelectContent>
                </Select>
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
              <div className="space-y-2">
                <Label>Rack</Label>
                <Input
                  type="number"
                  min="0"
                  value={form.rack}
                  onChange={(event) =>
                    setForm({ ...form, rack: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>Slot</Label>
                <Input
                  type="number"
                  min="0"
                  value={form.slot}
                  onChange={(event) =>
                    setForm({ ...form, slot: event.target.value })
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
      <Dialog
        open={Boolean(queryPLC)}
        onOpenChange={(value) => !value && setQueryPLC(null)}
      >
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>查询 PLC 数据</DialogTitle>
            <DialogDescription>
              {queryPLC?.name}（{queryPLC?.protocol}）— 每行输入一个地址，查询结果仅用于当前诊断。
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={queryValues} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="plc-query-addresses">查询地址</Label>
              <Textarea
                id="plc-query-addresses"
                rows={5}
                value={queryAddresses}
                onChange={(event) => setQueryAddresses(event.target.value)}
                placeholder={"DB1.DBW2\nMB10\nIB0\nQB0"}
              />
              <p className="text-xs text-muted-foreground">
                S7comm 示例：DB1.DBW2、MB10、IB0、QB0、DB1.DBX0.0。
              </p>
            </div>
            {queryError && <p className="text-sm text-destructive">{queryError}</p>}
            {queryResults.length > 0 && (
              <div className="overflow-x-auto rounded-md border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/40 text-left text-xs text-muted-foreground">
                      <th className="px-3 py-2">地址</th>
                      <th className="px-3 py-2">值</th>
                      <th className="px-3 py-2">质量</th>
                      <th className="px-3 py-2">时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {queryResults.map((item) => (
                      <tr key={`${item.address}-${item.timestamp}`} className="border-b last:border-0">
                        <td className="px-3 py-2 font-mono text-xs">{item.address}</td>
                        <td className="max-w-64 break-all px-3 py-2 font-mono text-xs">
                          {typeof item.value === "string"
                            ? item.value
                            : JSON.stringify(item.value)}
                        </td>
                        <td className="px-3 py-2">{item.quality}</td>
                        <td className="whitespace-nowrap px-3 py-2 text-xs text-muted-foreground">
                          {new Date(item.timestamp).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setQueryPLC(null)}
              >
                关闭
              </Button>
              <Button type="submit" disabled={queryLoading || !queryPLC}>
                {queryLoading && <Loader2 className="animate-spin" />}
                执行查询
              </Button>
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
