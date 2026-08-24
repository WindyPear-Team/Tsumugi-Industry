import { useEffect, useRef, useState, type FormEvent } from "react"
import * as Blockly from "blockly"
import * as ZhHans from "blockly/msg/zh-hans"
import { ArrowLeft, Check, GripVertical } from "lucide-react"
import { useNavigate, useParams } from "react-router-dom"
import { api, type FlowFunction, type FlowParameter } from "@/lib/api"
import { defineFunctionBlocks, functionToolbox, syncFunctionParameters } from "@/lib/function-blocks"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

type Form = { code: string; name: string; description: string; return_type: FlowFunction["return_type"] }
const empty: Form = { code: "", name: "", description: "", return_type: "none" }
const parameterTypes: { type: FlowParameter["type"]; label: string; description: string; symbol: string }[] = [
  { type: "label", label: "说明文本", description: "在积木中显示一段可自定义文字", symbol: "文本" },
  { type: "string", label: "数字字符", description: "输入数字或文本值", symbol: "123" },
  { type: "select", label: "下拉菜单", description: "自定义参数名和下拉选项", symbol: "下拉" },
  { type: "boolean", label: "布尔值", description: "真 / 假值输入", symbol: "布尔" },
]
function parametersFrom(value: FlowFunction["parameters"]): FlowParameter[] { if (Array.isArray(value)) return value; try { return JSON.parse(value) as FlowParameter[] } catch { return [] } }
function newParameter(type: FlowParameter["type"], index: number): FlowParameter { if (type === "label") return { name: "说明文本" + index, type, required: false, default_value: "文本" }; if (type === "select") return { name: "下拉参数" + index, type, required: true, options: ["选项1", "选项2"] }; return { name: (type === "boolean" ? "布尔参数" : type === "number" ? "数字参数" : "文本参数") + index, type, required: true, default_value: "" } }

export function FunctionEditorPage() {
  const { id = "new" } = useParams()
  const navigate = useNavigate()
  const [form, setForm] = useState<Form>(empty)
  const [parameters, setParameters] = useState<FlowParameter[]>([])
  const [draggedParameter, setDraggedParameter] = useState<number | null>(null)
  const [definition, setDefinition] = useState<unknown>(null)
  const [error, setError] = useState("")
  const hostRef = useRef<HTMLDivElement>(null)
  const workspaceRef = useRef<Blockly.WorkspaceSvg | null>(null)

  useEffect(() => {
    if (id === "new") return
    api<{ function: FlowFunction }>("/api/flow-functions/" + id).then((result) => {
      setForm({ code: result.function.code, name: result.function.name, description: result.function.description ?? "", return_type: result.function.return_type })
      setParameters(parametersFrom(result.function.parameters))
      if (result.function.definition) { try { setDefinition(JSON.parse(result.function.definition)) } catch { setDefinition(null) } }
    }).catch((err) => setError(err instanceof Error ? err.message : "加载函数失败"))
  }, [id])

  useEffect(() => {
    if (!hostRef.current || workspaceRef.current) return
    defineFunctionBlocks()
    Blockly.setLocale(ZhHans as unknown as Record<string, string>)
    const workspace = Blockly.inject(hostRef.current, { toolbox: functionToolbox(), trashcan: true, grid: { spacing: 24, length: 3, colour: "#c6d4ec", snap: true }, zoom: { controls: true, wheel: true }, move: { scrollbars: true, drag: true, wheel: true } })
    workspaceRef.current = workspace
    syncFunctionParameters(workspace, parameters.filter((parameter) => parameter.type !== "label").map((parameter) => parameter.name))
    return () => { workspace.dispose(); workspaceRef.current = null }
  }, [])
  useEffect(() => { const workspace = workspaceRef.current; if (workspace) syncFunctionParameters(workspace, parameters.filter((parameter) => parameter.type !== "label").map((parameter) => parameter.name)) }, [parameters])
  useEffect(() => { const workspace = workspaceRef.current; if (workspace && definition && workspace.getAllBlocks(false).length === 0) Blockly.serialization.workspaces.load(definition as never, workspace) }, [definition])

  function addParameter(type: FlowParameter["type"]) { setParameters((current) => [...current, newParameter(type, current.length + 1)]) }
  function updateParameter(index: number, update: Partial<FlowParameter>) { setParameters((current) => current.map((parameter, parameterIndex) => parameterIndex === index ? { ...parameter, ...update } : parameter)) }
  function removeParameter(index: number) { setParameters((current) => current.filter((_, parameterIndex) => parameterIndex !== index)) }
  function moveParameter(from: number, to: number) { if (from === to) return; setParameters((current) => { const next = [...current]; const [item] = next.splice(from, 1); next.splice(to, 0, item); return next }) }
  async function save(event: FormEvent) {
    event.preventDefault()
    try {
      const definitionValue = workspaceRef.current ? Blockly.serialization.workspaces.save(workspaceRef.current) : {}
      await api(id === "new" ? "/api/flow-functions" : "/api/flow-functions/" + id, { method: id === "new" ? "POST" : "PUT", body: JSON.stringify({ ...form, parameters, definition: definitionValue }) })
      navigate("/flow-functions")
    } catch (err) { setError(err instanceof Error ? err.message : "保存函数失败") }
  }

  return <div className="space-y-6">
    <Button variant="ghost" onClick={() => navigate("/flow-functions")}><ArrowLeft />返回函数列表</Button>
    <form onSubmit={save} className="space-y-6">
      <Card className="border-border/70 shadow-none">
        <CardHeader><CardTitle>{id === "new" ? "新建自定义积木" : "编辑自定义积木"}</CardTitle><CardDescription>按照参考积木编辑器定义：编辑积木名称，点击底部类型添加参数；所有文本、参数名和下拉项都可以自定义。</CardDescription></CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2"><Label>积木名称</Label><Input required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="例如：执行类" /></div>
            <div className="space-y-2"><Label>积木编码</Label><Input placeholder="留空自动生成" value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} /></div>
            <div className="space-y-2"><Label>返回值类型</Label><select className="h-9 w-full rounded-md border bg-background px-3 text-sm" value={form.return_type} onChange={(event) => setForm({ ...form, return_type: event.target.value as FlowFunction["return_type"] })}><option value="none">无返回值</option><option value="number">数字</option><option value="string">字符串</option><option value="boolean">布尔值</option></select></div>
            <div className="space-y-2"><Label>说明</Label><Input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></div>
          </div>
          <div className="rounded-3xl bg-violet-600 p-5 text-white shadow-lg">
            <div className="flex min-h-24 flex-wrap items-center gap-3 rounded-r-full border-b-4 border-violet-800 bg-violet-600 px-5 py-4">
              <Input className="h-11 w-40 border-violet-300 bg-violet-700 text-xl font-semibold text-white placeholder:text-violet-200" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="积木名称" />
              {parameters.map((parameter, index) => <div draggable key={parameter.name + "-" + index} onDragStart={() => setDraggedParameter(index)} onDragOver={(event) => event.preventDefault()} onDrop={() => { if (draggedParameter !== null) moveParameter(draggedParameter, index); setDraggedParameter(null) }} className="flex cursor-grab items-center gap-1 rounded-xl border border-violet-300 bg-violet-700 p-1 shadow-sm active:cursor-grabbing">
                <GripVertical className="size-4 shrink-0 text-violet-200" /><select className="h-8 max-w-24 rounded-md border border-violet-300 bg-violet-600 px-1 text-xs text-white" value={parameter.type} onChange={(event) => updateParameter(index, { type: event.target.value as FlowParameter["type"] })}><option value="label">文本</option><option value="string">字符</option><option value="number">数字</option><option value="boolean">布尔</option><option value="select">下拉</option><option value="device">设备</option></select>
                {parameter.type === "label" ? <><Input className="h-8 w-28 border-violet-300 bg-violet-600 text-white" value={parameter.default_value ?? ""} onChange={(event) => updateParameter(index, { default_value: event.target.value })} placeholder="说明文本" /><Input className="h-8 w-24 border-violet-300 bg-violet-600 text-xs text-white" value={parameter.name} onChange={(event) => updateParameter(index, { name: event.target.value })} placeholder="参数名" /></> : <><Input className="h-8 w-24 border-violet-300 bg-violet-600 text-xs text-white" value={parameter.name} onChange={(event) => updateParameter(index, { name: event.target.value })} placeholder="参数名" />{parameter.type === "select" || parameter.type === "device" ? <><select className="h-8 max-w-28 rounded-md border border-violet-300 bg-violet-600 px-2 text-sm text-white" value={parameter.options?.[0] ?? ""} onChange={(event) => updateParameter(index, { options: [event.target.value, ...(parameter.options ?? []).filter((option) => option !== event.target.value)] })}>{(parameter.options?.length ? parameter.options : ["选项"]).map((option) => <option value={option} key={option}>{option}</option>)}</select><Input className="h-8 w-24 border-violet-300 bg-violet-600 text-xs text-white" value={(parameter.options ?? []).join(", ")} onChange={(event) => updateParameter(index, { options: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) })} placeholder="下拉项" /></> : <Input className="h-8 w-24 border-violet-300 bg-violet-600 text-white" value={parameter.default_value ?? ""} onChange={(event) => updateParameter(index, { default_value: event.target.value })} placeholder={parameter.type === "boolean" ? "真/假" : parameter.type === "number" ? "数字" : "文本"} />}</>}
                <button type="button" className="px-1 text-violet-100 hover:text-white" onClick={() => removeParameter(index)} aria-label="删除参数">×</button>
              </div>)}
            </div>
            <p className="mt-3 text-sm text-violet-100">直接在上方编辑参数；拖动参数胶囊可调整顺序。</p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {parameterTypes.map((item) => <button type="button" className="rounded-2xl border bg-background p-5 text-left shadow-sm transition hover:border-primary hover:bg-primary/5" onClick={() => addParameter(item.type)} key={item.type}><div className="mb-3 flex items-center justify-between text-2xl text-muted-foreground"><span>＋</span><span>{item.symbol}</span></div><p className="font-medium">{item.label}</p><p className="mt-1 text-xs text-muted-foreground">{item.description}</p></button>)}
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
        </CardContent>
      </Card>
      <Card className="border-border/70 shadow-none"><CardHeader><CardTitle>函数内部积木</CardTitle><CardDescription>函数参数会出现在“函数参数”分类中，可以直接读取并参与基本运算。</CardDescription></CardHeader><CardContent><div ref={hostRef} className="blockly-editor" aria-label="函数 Blockly 编辑器" /></CardContent></Card>
      <Button type="submit"><Check />确认并保存</Button>
    </form>
  </div>
}
