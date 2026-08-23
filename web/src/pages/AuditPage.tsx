import { FileClock } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function AuditPage() { return <Card className="min-h-[420px] border-dashed border-border/80 bg-transparent shadow-none"><CardHeader><CardTitle>审计日志</CardTitle><CardDescription>追踪关键系统操作</CardDescription></CardHeader><CardContent className="grid place-items-center"><div className="max-w-md text-center"><div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-2xl bg-muted"><FileClock className="size-5" /></div><h2 className="text-lg font-semibold">审计日志模型已预留</h2><p className="mt-2 text-sm leading-6 text-muted-foreground">后端将记录登录、权限变更、设备操作和系统配置等关键事件。</p></div></CardContent></Card> }
