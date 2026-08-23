import { useEffect, useState } from "react"
import { Database, Save } from "lucide-react"
import { api, type Setting } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

export function SettingsPage() { const [settings, setSettings] = useState<Setting[]>([]); useEffect(() => { api<{ items: Setting[] }>("/api/settings").then((r) => setSettings(r.items)).catch(() => undefined) }, []); return <div className="grid gap-6 xl:grid-cols-[1.4fr_1fr]"><Card className="border-border/70 shadow-none"><CardHeader><CardTitle>运行设置</CardTitle><CardDescription>应用设置存储在数据库中，环境变量仅负责启动连接。</CardDescription></CardHeader><CardContent className="space-y-5"><div className="space-y-2"><Label>系统名称</Label><Input defaultValue={settings.find((item) => item.key === "system.name")?.value ?? "Tsumugi Industry"} /></div><div className="space-y-2"><Label>时区</Label><Select defaultValue="Asia/Shanghai"><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="Asia/Shanghai">Asia/Shanghai</SelectItem><SelectItem value="UTC">UTC</SelectItem></SelectContent></Select></div><Button><Save />保存设置</Button></CardContent></Card><Card className="border-border/70 shadow-none"><CardHeader><CardTitle className="flex items-center gap-2"><Database className="size-4" />数据库设置</CardTitle><CardDescription>当前实例的非敏感运行信息</CardDescription></CardHeader><CardContent className="space-y-3">{settings.map((setting) => <div className="flex justify-between gap-4 border-b border-border/60 py-3 last:border-0" key={setting.key}><span className="font-mono text-xs text-muted-foreground">{setting.key}</span><span className="text-right text-sm">{setting.value}</span></div>)}</CardContent></Card></div> }
