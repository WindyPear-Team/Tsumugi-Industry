import { useCallback, useEffect, useState, type FormEvent } from "react"
import { Check, Eye, Loader2, Pause, Pencil, Play, Plus, RotateCcw, Send, Trash2, XCircle } from "lucide-react"
import { api, type Device, type WorkOrder, type WorkOrderStep } from "@/lib/api"
import { ConfirmDialog } from "@/components/confirm-dialog"
import { TableColumnMenu } from "@/components/table-column-menu"
import { TablePagination } from "@/components/table-pagination"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { Textarea } from "@/components/ui/textarea"

type Page = { page: number; page_count: number; total: number; page_size: number }
type StepForm = { sequence: string; code: string; name: string; device_id: string; planned_qty: string; notes: string }
type FormState = { code: string; product_code: string; product_name: string; planned_qty: string; priority: string; scheduled_start: string; scheduled_end: string; notes: string; steps: StepForm[] }
type WorkflowAction = "release" | "start" | "pause" | "resume" | "cancel"
type ActionConfirm = { order: WorkOrder; action: WorkflowAction } | null

const newStep = (sequence = 1): StepForm => ({ sequence: String(sequence), code: `OP-${String(sequence).padStart(2, "0")}`, name: "", device_id: "", planned_qty: "", notes: "" })
const emptyForm: FormState = { code: "", product_code: "", product_name: "", planned_qty: "", priority: "normal", scheduled_start: "", scheduled_end: "", notes: "", steps: [newStep()] }
const statusLabels: Record<WorkOrder["status"], string> = { draft: "草稿", released: "已下达", running: "生产中", paused: "已暂停", completed: "已完工", cancelled: "已取消" }
const stepLabels: Record<WorkOrderStep["status"], string> = { pending: "待下达", ready: "待执行", running: "执行中", paused: "已暂停", completed: "已完成" }
const actionLabels: Record<WorkflowAction, string> = { release: "下达", start: "启动", pause: "暂停", resume: "恢复", cancel: "取消" }

function dateInput(value?: string) { return value ? value.slice(0, 16) : "" }
function dateValue(value: string) { return value ? new Date(value).toISOString() : undefined }
function statusBadge(status: WorkOrder["status"]) { return <Badge variant={status === "cancelled" ? "destructive" : status === "completed" ? "secondary" : status === "running" ? "default" : "outline"}>{statusLabels[status]}</Badge> }

export function WorkOrdersPage() {
  const [orders, setOrders] = useState<WorkOrder[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [page, setPage] = useState<Page>({ page: 1, page_count: 1, total: 0, page_size: 20 })
  const [sortField, setSortField] = useState("id")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc")
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [form, setForm] = useState<FormState>(emptyForm)
  const [editing, setEditing] = useState<WorkOrder | null>(null)
  const [deleting, setDeleting] = useState<WorkOrder | null>(null)
  const [actionConfirm, setActionConfirm] = useState<ActionConfirm>(null)
  const [selected, setSelected] = useState<WorkOrder | null>(null)
  const [completeStep, setCompleteStep] = useState<WorkOrderStep | null>(null)
  const [completeForm, setCompleteForm] = useState({ passed_qty: "", failed_qty: "0", reason: "", notes: "" })
  const [openForm, setOpenForm] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [workflowLoading, setWorkflowLoading] = useState(false)
  const [error, setError] = useState("")
  const params = new URLSearchParams({ sort_by: sortField, sort_order: sortOrder, page: String(page.page), page_size: String(page.page_size) })
  Object.entries(filters).forEach(([key, value]) => { if (value) params.set(`filter_${key}`, value) })
  const query = params.toString()

  const loadOrders = useCallback(async () => {
    try { const result = await api<{ items: WorkOrder[]; page: Page }>(`/api/work-orders?${query}`); setOrders(result.items); setPage(result.page) } catch (err) { setError(err instanceof Error ? err.message : "加载生产工单失败") }
  }, [query])
  useEffect(() => {
    api<{ items: WorkOrder[]; page: Page }>(`/api/work-orders?${query}`).then((result) => { setOrders(result.items); setPage(result.page) }).catch((err) => setError(err instanceof Error ? err.message : "加载生产工单失败")).finally(() => setLoading(false))
  }, [query])
  useEffect(() => { api<{ items: Device[] }>("/api/devices?page_size=100").then((result) => setDevices(result.items)).catch(() => undefined) }, [])

  function filter(field: string, value: string) { setFilters((current) => ({ ...current, [field]: value })); setPage((current) => ({ ...current, page: 1 })) }
  function menu(label: string, field: string) { return <TableColumnMenu label={label} field={field} filter={filters[field]} sortField={sortField} sortOrder={sortOrder} onSort={(next, order) => { setSortField(next); setSortOrder(order) }} onFilter={(value) => filter(field, value)} /> }
  function edit(order?: WorkOrder) {
    setEditing(order ?? null)
    setForm(order ? { code: order.code, product_code: order.product_code, product_name: order.product_name, planned_qty: String(order.planned_qty), priority: order.priority, scheduled_start: dateInput(order.scheduled_start), scheduled_end: dateInput(order.scheduled_end), notes: order.notes ?? "", steps: (order.steps ?? []).map((step) => ({ sequence: String(step.sequence), code: step.code, name: step.name, device_id: step.device_id ? String(step.device_id) : "", planned_qty: String(step.planned_qty), notes: step.notes ?? "" })) } : { ...emptyForm, steps: [newStep()] })
    setError(""); setOpenForm(true)
  }
  async function save(event: FormEvent) {
    event.preventDefault(); setSaving(true); setError("")
    try {
      const body = { ...form, planned_qty: Number(form.planned_qty), scheduled_start: dateValue(form.scheduled_start), scheduled_end: dateValue(form.scheduled_end), steps: form.steps.map((step) => ({ ...step, sequence: Number(step.sequence), planned_qty: step.planned_qty ? Number(step.planned_qty) : Number(form.planned_qty), device_id: step.device_id ? Number(step.device_id) : null })) }
      await api(editing ? `/api/work-orders/${editing.id}` : "/api/work-orders", { method: editing ? "PUT" : "POST", body: JSON.stringify(body) })
      setOpenForm(false); await loadOrders()
    } catch (err) { setError(err instanceof Error ? err.message : "保存生产工单失败") } finally { setSaving(false) }
  }
  async function remove() {
    if (!deleting) return
    try { await api(`/api/work-orders/${deleting.id}`, { method: "DELETE" }); setDeleting(null); await loadOrders() } catch (err) { setError(err instanceof Error ? err.message : "删除生产工单失败"); setDeleting(null) }
  }
  async function loadDetail(id: number) {
    try { const result = await api<{ work_order: WorkOrder }>(`/api/work-orders/${id}`); setSelected(result.work_order) } catch (err) { setError(err instanceof Error ? err.message : "加载工单详情失败") }
  }
  async function runWorkflow() {
    if (!actionConfirm) return
    setWorkflowLoading(true)
    try { const result = await api<{ work_order: WorkOrder }>(`/api/work-orders/${actionConfirm.order.id}/${actionConfirm.action}`, { method: "POST", body: "{}" }); setActionConfirm(null); setSelected(result.work_order); await loadOrders() } catch (err) { setError(err instanceof Error ? err.message : "工单流转失败"); setActionConfirm(null) } finally { setWorkflowLoading(false) }
  }
  async function startStep(step: WorkOrderStep) {
    if (!selected) return
    setWorkflowLoading(true)
    try { const result = await api<{ work_order: WorkOrder }>(`/api/work-orders/${selected.id}/steps/${step.id}/start`, { method: "POST", body: "{}" }); setSelected(result.work_order); await loadOrders() } catch (err) { setError(err instanceof Error ? err.message : "启动工序失败") } finally { setWorkflowLoading(false) }
  }
  async function finishStep(event: FormEvent) {
    event.preventDefault(); if (!selected || !completeStep) return
    setWorkflowLoading(true)
    try { const result = await api<{ work_order: WorkOrder }>(`/api/work-orders/${selected.id}/steps/${completeStep.id}/report`, { method: "POST", headers: { "Idempotency-Key": crypto.randomUUID(), "X-Production-Source": "operator" }, body: JSON.stringify({ passed_qty: Number(completeForm.passed_qty), failed_qty: Number(completeForm.failed_qty), reason: completeForm.reason, notes: completeForm.notes }) }); setSelected(result.work_order); setCompleteStep(null); await loadOrders() } catch (err) { setError(err instanceof Error ? err.message : "完成工序失败") } finally { setWorkflowLoading(false) }
  }
  function addStep() { setForm((current) => ({ ...current, steps: [...current.steps, newStep(current.steps.length + 1)] })) }
  function updateStep(index: number, value: Partial<StepForm>) { setForm((current) => ({ ...current, steps: current.steps.map((step, stepIndex) => stepIndex === index ? { ...step, ...value } : step) })) }

  return <div className="space-y-6">
    <Card className="border-border/70 shadow-none">
      <CardHeader><div className="flex items-center justify-between gap-4"><div><CardTitle>生产工单</CardTitle><CardDescription>工单按工序顺序执行，现场数量与异常必须留痕。</CardDescription></div><Button onClick={() => edit()}><Plus />新建工单</Button></div></CardHeader>
      <CardContent><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left text-xs text-muted-foreground"><th className="pb-3">{menu("工单号", "code")}</th><th className="pb-3">{menu("产品编码", "product_code")}</th><th className="pb-3">{menu("产品名称", "product_name")}</th><th className="pb-3">计划 / 完成</th><th className="pb-3">{menu("状态", "status")}</th><th className="pb-3">当前工序</th><th className="pb-3 text-right">操作</th></tr></thead><tbody>
        {loading ? <tr><td colSpan={7} className="py-12 text-center"><Loader2 className="mx-auto size-5 animate-spin" /></td></tr> : orders.length === 0 ? <tr><td colSpan={7} className="py-12 text-center text-muted-foreground">暂无生产工单</td></tr> : orders.map((order) => <tr key={order.id} className="border-b last:border-0"><td className="py-4"><Button variant="link" className="h-auto p-0 font-mono text-xs" onClick={() => void loadDetail(order.id)}>{order.code}</Button></td><td className="py-4 font-mono text-xs">{order.product_code}</td><td className="py-4">{order.product_name}</td><td className="py-4 tabular-nums">{order.completed_qty} / {order.planned_qty}<span className="ml-2 text-xs text-destructive">-{order.failed_qty}</span></td><td className="py-4">{statusBadge(order.status)}</td><td className="py-4 text-xs text-muted-foreground">{order.steps?.find((step) => step.sequence === order.current_sequence)?.name ?? "—"}</td><td className="py-4 text-right"><div className="flex justify-end gap-1">
          <Button variant="ghost" size="icon-sm" title="查看工单" onClick={() => void loadDetail(order.id)}><Eye /></Button>
          {order.status === "draft" && <><Button variant="ghost" size="icon-sm" title="编辑" onClick={() => edit(order)}><Pencil /></Button><Button variant="ghost" size="icon-sm" title="下达" onClick={() => setActionConfirm({ order, action: "release" })}><Send /></Button><Button variant="ghost" size="icon-sm" className="text-destructive" title="删除" onClick={() => setDeleting(order)}><Trash2 /></Button></>}
          {order.status === "released" && <><Button variant="ghost" size="icon-sm" title="启动" onClick={() => setActionConfirm({ order, action: "start" })}><Play /></Button><Button variant="ghost" size="icon-sm" className="text-destructive" title="取消" onClick={() => setActionConfirm({ order, action: "cancel" })}><XCircle /></Button></>}
          {order.status === "running" && <Button variant="ghost" size="icon-sm" title="暂停" onClick={() => setActionConfirm({ order, action: "pause" })}><Pause /></Button>}
          {order.status === "paused" && <><Button variant="ghost" size="icon-sm" title="恢复" onClick={() => setActionConfirm({ order, action: "resume" })}><RotateCcw /></Button><Button variant="ghost" size="icon-sm" className="text-destructive" title="取消" onClick={() => setActionConfirm({ order, action: "cancel" })}><XCircle /></Button></>}
        </div></td></tr>)}</tbody></table></div><TablePagination page={page} onPageChange={(next) => setPage((current) => ({ ...current, page: next }))} /></CardContent>
    </Card>
    {error && <p className="text-sm text-destructive">{error}</p>}

    <Dialog open={openForm} onOpenChange={setOpenForm}><DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto"><DialogHeader><DialogTitle>{editing ? "编辑草稿工单" : "新建生产工单"}</DialogTitle><DialogDescription>只有草稿可以修改。下达后必须按工序顺序执行。</DialogDescription></DialogHeader><form onSubmit={save} className="space-y-5"><div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label>工单号</Label><Input required value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} /></div><div className="space-y-2"><Label>产品编码</Label><Input required value={form.product_code} onChange={(event) => setForm({ ...form, product_code: event.target.value })} /></div><div className="space-y-2"><Label>产品名称</Label><Input required value={form.product_name} onChange={(event) => setForm({ ...form, product_name: event.target.value })} /></div><div className="space-y-2"><Label>计划数量</Label><Input required min="1" type="number" value={form.planned_qty} onChange={(event) => setForm({ ...form, planned_qty: event.target.value })} /></div><div className="space-y-2"><Label>优先级</Label><Select value={form.priority} onValueChange={(priority) => setForm({ ...form, priority })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="low">低</SelectItem><SelectItem value="normal">普通</SelectItem><SelectItem value="high">高</SelectItem><SelectItem value="urgent">紧急</SelectItem></SelectContent></Select></div><div className="space-y-2"><Label>计划开始</Label><Input type="datetime-local" value={form.scheduled_start} onChange={(event) => setForm({ ...form, scheduled_start: event.target.value })} /></div><div className="space-y-2"><Label>计划结束</Label><Input type="datetime-local" value={form.scheduled_end} onChange={(event) => setForm({ ...form, scheduled_end: event.target.value })} /></div></div><div className="space-y-2"><Label>备注</Label><Textarea rows={2} value={form.notes} onChange={(event) => setForm({ ...form, notes: event.target.value })} /></div><Separator /><div className="flex items-center justify-between"><div><p className="text-sm font-medium">工艺路线</p><p className="text-xs text-muted-foreground">每个工序可绑定一个现场设备，执行时必须逐步放行。</p></div><Button type="button" variant="outline" size="sm" onClick={addStep}><Plus />增加工序</Button></div><div className="space-y-3">{form.steps.map((step, index) => <div className="grid gap-3 rounded-lg border p-3 sm:grid-cols-[70px_1fr_1fr_1fr_110px_auto]" key={index}><Input aria-label="工序序号" type="number" min="1" value={step.sequence} onChange={(event) => updateStep(index, { sequence: event.target.value })} /><Input required aria-label="工序编码" placeholder="工序编码" value={step.code} onChange={(event) => updateStep(index, { code: event.target.value })} /><Input required aria-label="工序名称" placeholder="工序名称" value={step.name} onChange={(event) => updateStep(index, { name: event.target.value })} /><Select value={step.device_id} onValueChange={(device_id) => updateStep(index, { device_id })}><SelectTrigger><SelectValue placeholder="绑定设备" /></SelectTrigger><SelectContent>{devices.map((device) => <SelectItem value={String(device.id)} key={device.id}>{device.code} · {device.name}</SelectItem>)}</SelectContent></Select><Input aria-label="工序计划数量" type="number" min="1" placeholder="数量" value={step.planned_qty} onChange={(event) => updateStep(index, { planned_qty: event.target.value })} /><Button type="button" variant="ghost" size="icon" disabled={form.steps.length === 1} onClick={() => setForm((current) => ({ ...current, steps: current.steps.filter((_, stepIndex) => stepIndex !== index) }))}><Trash2 /></Button></div>)}</div>{error && <p className="text-sm text-destructive">{error}</p>}<DialogFooter><Button type="button" variant="outline" onClick={() => setOpenForm(false)}>取消</Button><Button type="submit" disabled={saving}>{saving && <Loader2 className="animate-spin" />}保存草稿</Button></DialogFooter></form></DialogContent></Dialog>

    <Dialog open={Boolean(selected)} onOpenChange={(open) => !open && setSelected(null)}><DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto"><DialogHeader><DialogTitle>{selected?.code} · {selected?.product_name}</DialogTitle><DialogDescription>{selected && statusLabels[selected.status]} · {selected?.completed_qty} / {selected?.planned_qty} 件完成</DialogDescription></DialogHeader>{selected && <div className="space-y-5"><div className="flex flex-wrap items-center gap-2">{statusBadge(selected.status)}<span className="text-xs text-muted-foreground">产品：{selected.product_code}</span><span className="text-xs text-muted-foreground">失败：{selected.failed_qty}</span></div><div className="space-y-3"><h3 className="text-sm font-medium">工艺路线与执行状态</h3>{(selected.steps ?? []).map((step) => <div className="flex flex-wrap items-center gap-3 rounded-lg border p-3" key={step.id}><div className="flex size-7 items-center justify-center rounded-full bg-muted text-xs font-semibold">{step.sequence}</div><div className="min-w-40 flex-1"><p className="text-sm font-medium">{step.name}</p><p className="font-mono text-xs text-muted-foreground">{step.code}{step.device ? ` · ${step.device.name}` : " · 人工工位"}</p></div><div className="text-xs tabular-nums text-muted-foreground">{step.passed_qty} / {step.planned_qty} <span className="text-destructive">失败 {step.failed_qty}</span></div>{<Badge variant={step.status === "running" ? "default" : step.status === "completed" ? "secondary" : "outline"}>{stepLabels[step.status]}</Badge>}{selected.status === "running" && step.status === "ready" && <Button size="sm" variant="outline" disabled={workflowLoading} onClick={() => void startStep(step)}><Play />启动</Button>}{selected.status === "running" && step.status === "running" && <Button size="sm" disabled={workflowLoading} onClick={() => { setCompleteStep(step); setCompleteForm({ passed_qty: String(Math.max(0, step.planned_qty - step.passed_qty - step.failed_qty)), failed_qty: "0", reason: "", notes: "" }) }}><Check />报工</Button>}</div>)}</div><Separator /><div><h3 className="mb-3 text-sm font-medium">最近流转记录</h3><div className="space-y-2">{(selected.events ?? []).slice(0, 8).map((event) => <div className="flex items-center justify-between text-xs" key={event.id}><span>{event.event_type} · {event.operator_name || "系统"}</span><span className="text-muted-foreground">{new Date(event.created_at).toLocaleString()}</span></div>)}</div></div></div>}<DialogFooter><Button variant="outline" onClick={() => setSelected(null)}>关闭</Button></DialogFooter></DialogContent></Dialog>

    <Dialog open={Boolean(completeStep)} onOpenChange={(open) => !open && setCompleteStep(null)}><DialogContent><DialogHeader><DialogTitle>工序报工：{completeStep?.name}</DialogTitle><DialogDescription>必须填报合格数；存在不合格品时必须填写原因。</DialogDescription></DialogHeader><form onSubmit={finishStep} className="space-y-4"><div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label>合格数量</Label><Input required min="0" type="number" value={completeForm.passed_qty} onChange={(event) => setCompleteForm({ ...completeForm, passed_qty: event.target.value })} /></div><div className="space-y-2"><Label>不合格数量</Label><Input min="0" type="number" value={completeForm.failed_qty} onChange={(event) => setCompleteForm({ ...completeForm, failed_qty: event.target.value })} /></div></div><div className="space-y-2"><Label>异常/不合格原因</Label><Input value={completeForm.reason} onChange={(event) => setCompleteForm({ ...completeForm, reason: event.target.value })} /></div><div className="space-y-2"><Label>备注</Label><Textarea value={completeForm.notes} onChange={(event) => setCompleteForm({ ...completeForm, notes: event.target.value })} /></div><DialogFooter><Button type="button" variant="outline" onClick={() => setCompleteStep(null)}>取消</Button><Button type="submit" disabled={workflowLoading}>{workflowLoading && <Loader2 className="animate-spin" />}提交报工</Button></DialogFooter></form></DialogContent></Dialog>

    <ConfirmDialog open={Boolean(actionConfirm)} onOpenChange={(open) => !open && setActionConfirm(null)} title={`${actionConfirm ? actionLabels[actionConfirm.action] : ""}工单？`} description={actionConfirm ? `工单“${actionConfirm.order.code}”将进入${actionLabels[actionConfirm.action]}状态，系统会记录操作人和流转时间。` : ""} confirmLabel={actionConfirm ? actionLabels[actionConfirm.action] : "确认"} destructive={actionConfirm?.action === "cancel"} onConfirm={() => void runWorkflow()} />
    <ConfirmDialog open={Boolean(deleting)} onOpenChange={(open) => !open && setDeleting(null)} title="确认删除工单？" description={`将删除草稿工单“${deleting?.code ?? ""}”及其工艺路线。`} confirmLabel="确认删除" onConfirm={() => void remove()} />
  </div>
}
