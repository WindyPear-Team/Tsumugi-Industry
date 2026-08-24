import { useEffect, useState } from "react"
import { Eye, Loader2, Pencil, Plus, Trash2 } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { api, type Dashboard } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function DashboardsPage() {
  const navigate = useNavigate(); const [items, setItems] = useState<Dashboard[]>([]); const [loading, setLoading] = useState(true); const [error, setError] = useState("")
  async function load() { try { const result = await api<{ items: Dashboard[] }>("/api/dashboards"); setItems(result.items) } catch (err) { setError(err instanceof Error ? err.message : "加载看板失败") } finally { setLoading(false) } }
  useEffect(() => { void load() }, [])
  async function create() { try { const result = await api<{ dashboard: Dashboard }>("/api/dashboards", { method: "POST", body: JSON.stringify({ name: "新看板", time_range_hours: 24, background_color: "#f8fafc", widgets: [] }) }); navigate("/dashboards/" + result.dashboard.id + "/edit") } catch (err) { setError(err instanceof Error ? err.message : "创建看板失败") } }
  async function remove(id: number) { try { await api("/api/dashboards/" + id, { method: "DELETE" }); await load() } catch (err) { setError(err instanceof Error ? err.message : "删除看板失败") } }
  return <Card className="border-border/70 shadow-none"><CardHeader><div className="flex items-center justify-between"><div><CardTitle>看板列表</CardTitle><CardDescription>查看和编辑分开；查看页用于生产现场，编辑页用于自由布局和看板积木。</CardDescription></div><Button onClick={() => void create()}><Plus />新建看板</Button></div></CardHeader><CardContent>{loading ? <div className="py-12 text-center"><Loader2 className="mx-auto animate-spin" /></div> : <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">{items.map((item) => <div className="rounded-xl border p-4" key={item.id}><p className="font-medium">{item.name}</p><p className="mt-1 text-xs text-muted-foreground">{item.widgets?.length ?? 0} 个组件 · 最近 {item.time_range_hours} 小时</p><div className="mt-4 grid grid-cols-3 gap-2"><Button variant="outline" size="sm" onClick={() => navigate("/dashboards/" + item.id + "/view")}><Eye />查看</Button><Button variant="outline" size="sm" onClick={() => navigate("/dashboards/" + item.id + "/edit")}><Pencil />编辑</Button><Button variant="outline" size="sm" onClick={() => void remove(item.id)}><Trash2 /></Button></div></div>)}</div>}{!loading && items.length === 0 && <div className="py-12 text-center text-muted-foreground">暂无看板</div>}{error && <p className="mt-3 text-sm text-destructive">{error}</p>}</CardContent></Card>
}
