import { useEffect, useState } from "react"
import { FileClock, Loader2 } from "lucide-react"
import { api, type AuditLog } from "@/lib/api"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function AuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(true)
  useEffect(() => { api<{ items: AuditLog[] }>("/api/audit").then((result) => setLogs(result.items)).catch(() => undefined).finally(() => setLoading(false)) }, [])
  return <Card className="border-border/70 shadow-none"><CardHeader><CardTitle>审计日志</CardTitle><CardDescription>记录用户对系统资源的访问和变更。</CardDescription></CardHeader><CardContent><div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr className="border-b text-left text-xs text-muted-foreground"><th className="pb-3 font-medium">时间</th><th className="pb-3 font-medium">用户</th><th className="pb-3 font-medium">动作</th><th className="pb-3 font-medium">资源</th><th className="pb-3 font-medium">结果</th></tr></thead><tbody>{loading ? <tr><td colSpan={5} className="py-12 text-center"><Loader2 className="mx-auto size-5 animate-spin" /></td></tr> : logs.length === 0 ? <tr><td colSpan={5} className="py-12 text-center text-muted-foreground"><FileClock className="mx-auto mb-2 size-5" />暂无审计记录</td></tr> : logs.map((log) => <tr key={log.id} className="border-b last:border-0"><td className="whitespace-nowrap py-4 text-xs text-muted-foreground">{new Date(log.created_at).toLocaleString()}</td><td className="py-4 font-medium">{log.username || "系统"}</td><td className="py-4"><span className="rounded-md bg-muted px-2 py-1 font-mono text-xs">{log.action}</span></td><td className="py-4"><p>{log.resource}</p><p className="font-mono text-xs text-muted-foreground">{log.method} {log.path}</p></td><td className="py-4 text-xs">{log.status_code}</td></tr>)}</tbody></table></div></CardContent></Card>
}
