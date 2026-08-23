import { Activity, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function DevicesPage() { return <Card className="min-h-[420px] border-dashed border-border/80 bg-transparent shadow-none"><CardHeader><div className="flex items-center justify-between"><div><CardTitle>设备接入</CardTitle><CardDescription>监控现场设备与连接状态</CardDescription></div><Button><Plus />接入设备</Button></div></CardHeader><CardContent className="grid place-items-center"><div className="max-w-md text-center"><div className="mx-auto mb-4 flex size-12 items-center justify-center rounded-2xl bg-muted"><Activity className="size-5" /></div><h2 className="text-lg font-semibold">设备模型已预留</h2><p className="mt-2 text-sm leading-6 text-muted-foreground">后端可继续接入 OPC UA、Modbus 或 MQTT 适配器，并在此处展示设备状态与遥测数据。</p></div></CardContent></Card> }
