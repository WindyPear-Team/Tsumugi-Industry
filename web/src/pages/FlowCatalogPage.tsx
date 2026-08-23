import { useEffect, useState } from "react"
import { Edit3, Loader2, Plus } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { api, type FlowDefinition } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function FlowCatalogPage() {
  const navigate = useNavigate(); const [flows, setFlows] = useState<FlowDefinition[]>([]); const [loading, setLoading] = useState(true); const [error, setError] = useState("")
  useEffect(() => { api<{ items: FlowDefinition[] }>("/api/flows?page_size=100").then((result) => setFlows(result.items)).catch((err) => setError(err instanceof Error ? err.message : "加载流程失败")).finally(() => setLoading(false)) }, [])
  return <Card className="border-border/70 shadow-none"><CardHeader><div className="flex items-center justify-between gap-4"><div><CardTitle>流程定义</CardTitle><CardDescription>工程师在此选择流程；编辑器是独立页面，发布后的版本不可直接修改。</CardDescription></div><Button onClick={() => navigate("/flows/new/edit")}><Plus />新建流程</Button></div></CardHeader><CardContent>{loading ? <div className="py-12 text-center"><Loader2 className="mx-auto size-5 animate-spin" /></div> : flows.length === 0 ? <div className="py-12 text-center text-muted-foreground">暂无流程定义</div> : <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">{flows.map((flow) => <div className="rounded-xl border border-border/70 p-4" key={flow.id}><div className="flex items-start justify-between gap-3"><div><p className="font-medium">{flow.name}</p><p className="mt-1 font-mono text-xs text-muted-foreground">{flow.code} · v{flow.version}</p></div><Badge variant={flow.status === "published" ? "secondary" : "outline"}>{flow.status === "published" ? "已发布" : "草稿"}</Badge></div><p className="mt-4 line-clamp-2 text-sm text-muted-foreground">{flow.description || "暂无描述"}</p><Button className="mt-4 w-full" variant="outline" onClick={() => navigate(`/flows/${flow.id}/edit`)}><Edit3 />进入编辑器</Button></div>)}</div>}{error && <p className="mt-3 text-sm text-destructive">{error}</p>}</CardContent></Card>
}
