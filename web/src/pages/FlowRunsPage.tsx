import { useEffect, useState } from "react"
import { Eye, Loader2 } from "lucide-react"
import { api, type FlowRun } from "@/lib/api"
import { TableColumnMenu } from "@/components/table-column-menu"
import { TablePagination } from "@/components/table-pagination"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

type Page = { page: number; page_count: number; total: number; page_size: number }
const statusLabels: Record<string, string> = { created: "创建", running: "运行中", paused: "已暂停", completed: "已完成", failed: "失败", cancelled: "已取消", timeout: "超时", manual_confirm: "等待确认" }
function statusBadge(status: string) { return <Badge variant={status === "completed" ? "secondary" : status === "failed" || status === "timeout" ? "destructive" : status === "running" ? "default" : "outline"}>{statusLabels[status] ?? status}</Badge> }

export function FlowRunsPage() {
  const [runs, setRuns] = useState<FlowRun[]>([])
  const [page, setPage] = useState<Page>({ page: 1, page_count: 1, total: 0, page_size: 20 })
  const [sortField, setSortField] = useState("id")
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc")
  const [filterStatus, setFilterStatus] = useState("")
  const [selected, setSelected] = useState<FlowRun | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const params = new URLSearchParams({ sort_by: sortField, sort_order: sortOrder, page: String(page.page), page_size: String(page.page_size) })
  if (filterStatus) params.set("filter_status", filterStatus)
  const query = params.toString()
  useEffect(() => { api<{ items: FlowRun[]; page: Page }>(`/api/flow-runs?${query}`).then((result) => { setRuns(result.items); setPage(result.page) }).catch((err) => setError(err instanceof Error ? err.message : "加载运行记录失败")).finally(() => setLoading(false)) }, [query])
  function menu(label: string, field: string, filter?: string, onFilter?: (value: string) => void) { return <TableColumnMenu label={label} field={field} filter={filter} sortField={sortField} sortOrder={sortOrder} onSort={(next, order) => { setSortField(next); setSortOrder(order) }} onFilter={onFilter ?? (() => undefined)} /> }
  async function openRun(run: FlowRun) { try { const result = await api<{ run: FlowRun }>(`/api/flow-runs/${run.id}`); setSelected(result.run) } catch (err) { setError(err instanceof Error ? err.message : "加载运行详情失败") } }
  return <div className="space-y-6"><Card className="border-border/70 shadow-none"><CardHeader><CardTitle>流程运行记录</CardTitle><CardDescription>每次运行固定绑定启动时的流程版本，节点执行结果不可被后续编辑覆盖。</CardDescription></CardHeader><CardContent><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left text-xs text-muted-foreground"><th className="pb-3">{menu("运行编号", "id")}</th><th className="pb-3">流程</th><th className="pb-3">版本</th><th className="pb-3">{menu("状态", "status", filterStatus, setFilterStatus)}</th><th className="pb-3">当前节点</th><th className="pb-3">开始时间</th><th className="pb-3">结束时间</th><th className="pb-3 text-right">详情</th></tr></thead><tbody>{loading ? <tr><td colSpan={8} className="py-12 text-center"><Loader2 className="mx-auto size-5 animate-spin" /></td></tr> : runs.length === 0 ? <tr><td colSpan={8} className="py-12 text-center text-muted-foreground">暂无流程运行记录</td></tr> : runs.map((run) => <tr key={run.id} className="border-b last:border-0"><td className="py-4 font-mono text-xs">#{run.id}</td><td className="py-4">{run.flow_definition?.name ?? `流程 #${run.flow_definition_id}`}</td><td className="py-4">v{run.flow_version}</td><td className="py-4">{statusBadge(run.status)}</td><td className="py-4 font-mono text-xs">{run.current_node_id || "—"}</td><td className="py-4 text-xs text-muted-foreground">{run.started_at ? new Date(run.started_at).toLocaleString() : "—"}</td><td className="py-4 text-xs text-muted-foreground">{run.ended_at ? new Date(run.ended_at).toLocaleString() : "—"}</td><td className="py-4 text-right"><Button variant="ghost" size="icon-sm" onClick={() => void openRun(run)}><Eye /></Button></td></tr>)}</tbody></table></div><TablePagination page={page} onPageChange={(next) => setPage((current) => ({ ...current, page: next }))} />{error && <p className="mt-3 text-sm text-destructive">{error}</p>}</CardContent></Card>
    <Dialog open={Boolean(selected)} onOpenChange={(open) => !open && setSelected(null)}><DialogContent className="max-w-3xl"><DialogHeader><DialogTitle>流程运行 #{selected?.id}</DialogTitle><DialogDescription>固定版本 v{selected?.flow_version} · {selected && statusLabels[selected.status]}</DialogDescription></DialogHeader>{selected && <div className="space-y-4"><div className="grid gap-3 sm:grid-cols-3"><div className="rounded-lg bg-muted/50 p-3"><p className="text-xs text-muted-foreground">当前节点</p><p className="mt-1 font-mono text-sm">{selected.current_node_id || "—"}</p></div><div className="rounded-lg bg-muted/50 p-3"><p className="text-xs text-muted-foreground">开始时间</p><p className="mt-1 text-sm">{selected.started_at ? new Date(selected.started_at).toLocaleString() : "—"}</p></div><div className="rounded-lg bg-muted/50 p-3"><p className="text-xs text-muted-foreground">结束时间</p><p className="mt-1 text-sm">{selected.ended_at ? new Date(selected.ended_at).toLocaleString() : "—"}</p></div></div>{selected.error_message && <p className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{selected.error_message}</p>}<div className="space-y-2"><p className="text-sm font-medium">节点执行记录</p>{(selected.node_runs ?? []).map((node) => <div className="flex items-center justify-between rounded-lg border p-3" key={node.id}><span><span className="font-medium">{node.node_id}</span><span className="ml-2 text-xs text-muted-foreground">{node.node_type}</span></span>{statusBadge(node.status)}</div>)}</div></div>}<DialogFooter><Button variant="outline" onClick={() => setSelected(null)}>关闭</Button></DialogFooter></DialogContent></Dialog></div>
}
