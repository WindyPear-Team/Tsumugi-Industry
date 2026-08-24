import { useEffect, useState } from "react"
import { Edit3, Loader2, Plus, Trash2 } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { api, type FlowFunction } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

function parametersOf(item: FlowFunction) { if (Array.isArray(item.parameters)) return item.parameters; try { return JSON.parse(item.parameters) as { name: string; type: string }[] } catch { return [] } }

export function FunctionsPage() {
  const navigate = useNavigate(); const [items, setItems] = useState<FlowFunction[]>([]); const [loading, setLoading] = useState(true); const [error, setError] = useState("")
  async function load() { try { const result = await api<{ items: FlowFunction[] }>("/api/flow-functions"); setItems(result.items) } catch (err) { setError(err instanceof Error ? err.message : "加载函数失败") } finally { setLoading(false) } }
  useEffect(() => { void load() }, [])
  async function remove(item: FlowFunction) { if (!item.id) return; try { await api("/api/flow-functions/" + item.id, { method: "DELETE" }); await load() } catch (err) { setError(err instanceof Error ? err.message : "删除函数失败") } }
  return <Card className="border-border/70 shadow-none"><CardHeader><div className="flex items-center justify-between"><div><CardTitle>流程函数</CardTitle><CardDescription>函数是全局资源，创建后所有流程都可以在函数积木盒中调用。</CardDescription></div><Button onClick={() => navigate("/flow-functions/new/edit")}><Plus />新建函数</Button></div></CardHeader><CardContent>{loading ? <div className="py-12 text-center"><Loader2 className="mx-auto animate-spin" /></div> : <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">{items.map((item) => <div className="rounded-xl border p-4" key={item.id}><div className="flex items-start justify-between"><div><p className="font-medium">{item.name}</p><p className="font-mono text-xs text-muted-foreground">{item.code} · {item.return_type === "none" ? "无返回值" : "返回 " + item.return_type}</p></div><div className="flex gap-1"><Button variant="ghost" size="icon-sm" onClick={() => navigate("/flow-functions/" + item.id + "/edit")}><Edit3 /></Button><Button variant="ghost" size="icon-sm" onClick={() => void remove(item)}><Trash2 /></Button></div></div><p className="mt-3 text-sm text-muted-foreground">参数：{parametersOf(item).map((parameter) => parameter.name + ":" + parameter.type).join("、") || "无"}</p></div>)}</div>}{!loading && items.length === 0 && <div className="py-12 text-center text-muted-foreground">暂无全局函数</div>}{error && <p className="mt-3 text-sm text-destructive">{error}</p>}</CardContent></Card>
}
