import { useEffect, useRef, useState, type DragEvent } from "react"
import {
  ArrowLeft,
  Check,
  GripVertical,
  Plus,
  Save,
  Send,
  Trash2,
} from "lucide-react"
import { useNavigate, useParams } from "react-router-dom"
import {
  api,
  type FlowDefinition,
  type FlowDocument,
  type FlowNode,
  type PLCVariable,
} from "@/lib/api"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"

const nodeTypes = [
  "SET",
  "GET",
  "WAIT",
  "IF",
  "DELAY",
  "MANUAL_CONFIRM",
  "ALARM",
  "LOOP",
  "PARALLEL",
  "SUBFLOW",
]
const nodeLabels: Record<string, string> = {
  START: "开始",
  END: "结束",
  SET: "设置变量",
  GET: "读取变量",
  WAIT: "等待条件",
  IF: "条件判断",
  DELAY: "延时",
  MANUAL_CONFIRM: "人工确认",
  ALARM: "报警",
  LOOP: "循环",
  PARALLEL: "并行",
  SUBFLOW: "子流程",
}
const emptyDocument: FlowDocument = {
  nodes: [
    { id: "start", type: "START", label: "开始", x: 40, y: 80, config: {} },
    { id: "end", type: "END", label: "结束", x: 540, y: 80, config: {} },
  ],
  edges: [{ id: "start-end", source: "start", target: "end" }],
}
type FlowForm = {
  code: string
  name: string
  description: string
  timeout_seconds: string
}
function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
function newID(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
}

export function FlowEditorPage() {
  const { id = "new" } = useParams()
  const navigate = useNavigate()
  const isNew = id === "new"
  const [flow, setFlow] = useState<FlowDefinition | null>(null)
  const [variables, setVariables] = useState<PLCVariable[]>([])
  const [document, setDocument] = useState<FlowDocument>(clone(emptyDocument))
  const [selectedNodeID, setSelectedNodeID] = useState("start")
  const [form, setForm] = useState<FlowForm>({
    code: "FLOW-001",
    name: "新流程",
    description: "",
    timeout_seconds: "0",
  })
  const [dialogOpen, setDialogOpen] = useState(false)
  const [insertAnchorID, setInsertAnchorID] = useState<string | null>(null)
  const [insertSide, setInsertSide] = useState<"before" | "after">("after")
  const [newType, setNewType] = useState("SET")
  const [newLabel, setNewLabel] = useState(nodeLabels.SET)
  const [issues, setIssues] = useState<string[]>([])
  const [error, setError] = useState("")
  const [saving, setSaving] = useState(false)
  const canvasRef = useRef<HTMLDivElement>(null)
  const editable = !flow || flow.status === "draft"
  const selectedNode = document.nodes.find((node) => node.id === selectedNodeID)
  useEffect(() => {
    api<{ items: PLCVariable[] }>("/api/variables?page_size=100")
      .then((result) => setVariables(result.items))
      .catch(() => undefined)
  }, [])
  useEffect(() => {
    if (isNew) return
    api<{ flow: FlowDefinition }>(`/api/flows/${id}`)
      .then((result) => {
        setFlow(result.flow)
        setForm({
          code: result.flow.code,
          name: result.flow.name,
          description: result.flow.description,
          timeout_seconds: String(result.flow.timeout_seconds),
        })
        setDocument(JSON.parse(result.flow.definition) as FlowDocument)
      })
      .catch((err) =>
        setError(err instanceof Error ? err.message : "加载流程失败")
      )
  }, [id, isNew])
  function updateDocument(next: FlowDocument) {
    setDocument(next)
  }
  function updateNode(
    patch: Partial<FlowNode>,
    config?: Record<string, unknown>
  ) {
    if (!selectedNode || !editable) return
    updateDocument({
      ...document,
      nodes: document.nodes.map((node) =>
        node.id === selectedNode.id
          ? {
              ...node,
              ...patch,
              config: config ? { ...node.config, ...config } : node.config,
            }
          : node
      ),
    })
  }
  function openNodeDialog(
    anchorID: string | null = null,
    side: "before" | "after" = "after"
  ) {
    setInsertAnchorID(anchorID)
    setInsertSide(side)
    setNewType("SET")
    setNewLabel(nodeLabels.SET)
    setDialogOpen(true)
  }
  function addNode() {
    const end = document.nodes.find((node) => node.type === "END")
    const anchor = insertAnchorID
      ? document.nodes.find((node) => node.id === insertAnchorID)
      : undefined
    const previous =
      anchor ?? document.nodes.filter((node) => node.type !== "END").at(-1)
    const config =
      newType === "LOOP"
        ? { max_iterations: 3 }
        : newType === "DELAY"
          ? { seconds: 5 }
          : ["SET", "GET", "WAIT", "IF"].includes(newType)
            ? {
                variable: "",
                operator: "==",
                expected: true,
                timeout_seconds: 10,
                timeout_action: "FAIL",
                max_retries: 0,
              }
            : {}
    const node: FlowNode = {
      id: newID(newType.toLowerCase()),
      type: newType,
      label: newLabel || nodeLabels[newType],
      x: (previous?.x ?? 40) + 250,
      y: previous?.y ?? 80,
      config,
    }
    let edges = [...document.edges]
    if (anchor && insertSide === "before") {
      const incoming = edges.filter((edge) => edge.target === anchor.id)
      edges = edges.map((edge) =>
        edge.target === anchor.id ? { ...edge, target: node.id } : edge
      )
      if (incoming.length === 0)
        edges.push({ id: newID("edge"), source: node.id, target: anchor.id })
      else edges.push({ id: newID("edge"), source: node.id, target: anchor.id })
      node.x = anchor.x - 250
    } else if (anchor) {
      const outgoing = edges.filter((edge) => edge.source === anchor.id)
      edges = edges.map((edge) =>
        edge.source === anchor.id ? { ...edge, source: node.id } : edge
      )
      edges.push({ id: newID("edge"), source: anchor.id, target: node.id })
      if (outgoing.length === 0 && end && anchor.id !== end.id)
        edges.push({ id: newID("edge"), source: node.id, target: end.id })
      node.x = anchor.x + 250
    } else if (previous && end) {
      edges = edges.filter(
        (edge) => !(edge.source === previous.id && edge.target === end.id)
      )
      edges.push(
        { id: newID("edge"), source: previous.id, target: node.id },
        { id: newID("edge"), source: node.id, target: end.id }
      )
    }
    updateDocument({
      nodes: [
        ...document.nodes.filter((item) => item.type !== "END"),
        node,
        ...(end ? [end] : []),
      ],
      edges,
    })
    setSelectedNodeID(node.id)
    setInsertAnchorID(null)
    setDialogOpen(false)
  }
  function dragNode(event: DragEvent, nodeID: string) {
    const canvas = canvasRef.current
    if (!canvas || !editable) return
    const rect = canvas.getBoundingClientRect()
    const x = Math.max(
      0,
      Math.round((event.clientX - rect.left - 80) / 20) * 20
    )
    const y = Math.max(0, Math.round((event.clientY - rect.top - 30) / 20) * 20)
    const moved = document.nodes.map((node) =>
      node.id === nodeID ? { ...node, x, y } : node
    )
    updateDocument(
      isLinearFlow(document)
        ? reflowLinearFlow({ ...document, nodes: moved })
        : { ...document, nodes: moved }
    )
  }

  function isLinearFlow(flowDocument: FlowDocument) {
    if (
      flowDocument.nodes.some((node) =>
        ["IF", "PARALLEL", "LOOP"].includes(node.type)
      )
    )
      return false
    const incoming = new Map<string, number>()
    const outgoing = new Map<string, number>()
    for (const edge of flowDocument.edges) {
      incoming.set(edge.target, (incoming.get(edge.target) ?? 0) + 1)
      outgoing.set(edge.source, (outgoing.get(edge.source) ?? 0) + 1)
    }
    return [...incoming.values(), ...outgoing.values()].every(
      (count) => count <= 1
    )
  }

  function reflowLinearFlow(flowDocument: FlowDocument): FlowDocument {
    const start = flowDocument.nodes.find((node) => node.type === "START")
    const end = flowDocument.nodes.find((node) => node.type === "END")
    if (!start || !end) return flowDocument
    const middle = flowDocument.nodes
      .filter((node) => !["START", "END"].includes(node.type))
      .sort((a, b) => a.x - b.x || a.y - b.y)
    const ordered = [start, ...middle, end]
    const positions = new Map(
      ordered.map((node, index) => [node.id, { x: 40 + index * 220, y: 180 }])
    )
    const nodes = flowDocument.nodes.map((node) =>
      positions.has(node.id) ? { ...node, ...positions.get(node.id) } : node
    )
    const edges = ordered.slice(0, -1).map((node, index) => ({
      id: `${node.id}-auto-${ordered[index + 1].id}`,
      source: node.id,
      target: ordered[index + 1].id,
    }))
    return { nodes, edges }
  }
  function removeNode() {
    if (
      !selectedNode ||
      !editable ||
      ["START", "END"].includes(selectedNode.type)
    )
      return
    updateDocument({
      nodes: document.nodes.filter((node) => node.id !== selectedNode.id),
      edges: document.edges.filter(
        (edge) =>
          edge.source !== selectedNode.id && edge.target !== selectedNode.id
      ),
    })
    setSelectedNodeID("start")
  }
  async function save() {
    setSaving(true)
    try {
      const result = await api<{ flow: FlowDefinition }>(
        isNew ? "/api/flows" : `/api/flows/${id}`,
        {
          method: isNew ? "POST" : "PUT",
          body: JSON.stringify({
            ...form,
            timeout_seconds: Number(form.timeout_seconds),
            definition: JSON.stringify(document),
          }),
        }
      )
      setFlow(result.flow)
      navigate(`/flows/${result.flow.id}/edit`, { replace: true })
      setError("")
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存流程失败")
    } finally {
      setSaving(false)
    }
  }
  async function validate() {
    if (!flow) {
      setIssues(["请先保存流程"])
      return
    }
    try {
      const result = await api<{ issues: string[] }>(
        `/api/flows/${flow.id}/validate`,
        { method: "POST", body: "{}" }
      )
      setIssues(result.issues)
    } catch (err) {
      setError(err instanceof Error ? err.message : "流程校验失败")
    }
  }
  async function publish() {
    if (!flow) return
    try {
      const result = await api<{ flow: FlowDefinition }>(
        `/api/flows/${flow.id}/publish`,
        { method: "POST", body: "{}" }
      )
      setFlow(result.flow)
    } catch (err) {
      setError(err instanceof Error ? err.message : "发布流程失败")
    }
  }
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => navigate("/flows")}>
          <ArrowLeft />
          返回流程清单
        </Button>
        <div className="flex gap-2">
          <Button
            variant="outline"
            disabled={!editable || saving}
            onClick={() => void save()}
          >
            <Save />
            保存
          </Button>
          {flow && editable && (
            <>
              <Button variant="outline" onClick={() => void validate()}>
                <Check />
                校验
              </Button>
              <Button onClick={() => void publish()}>
                <Send />
                发布
              </Button>
            </>
          )}
        </div>
      </div>
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <Card className="min-w-0 border-border/70 shadow-none">
          <CardHeader>
            <CardTitle>
              {form.name || "新流程"}
              {flow && ` · v${flow.version}`}
            </CardTitle>
            <CardDescription>
              独立流程编辑器。线性流程拖动节点会自动排序并重建连线，分支流程保留既有连接。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="mb-4 grid gap-3 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>流程编码</Label>
                <Input
                  disabled={!editable}
                  value={form.code}
                  onChange={(event) =>
                    setForm({ ...form, code: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>流程名称</Label>
                <Input
                  disabled={!editable}
                  value={form.name}
                  onChange={(event) =>
                    setForm({ ...form, name: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>总超时（秒）</Label>
                <Input
                  disabled={!editable}
                  type="number"
                  min="0"
                  value={form.timeout_seconds}
                  onChange={(event) =>
                    setForm({ ...form, timeout_seconds: event.target.value })
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>描述</Label>
                <Input
                  disabled={!editable}
                  value={form.description}
                  onChange={(event) =>
                    setForm({ ...form, description: event.target.value })
                  }
                />
              </div>
            </div>
            <div
              ref={canvasRef}
              className="relative min-h-[520px] overflow-auto rounded-xl border border-dashed bg-muted/20 p-4"
            >
              <div className="relative h-[480px] min-w-[900px]">
                <svg
                  className="pointer-events-none absolute inset-0 z-0 size-full overflow-visible"
                  aria-hidden="true"
                >
                  {document.edges.map((edge) => {
                    const source = document.nodes.find(
                      (node) => node.id === edge.source
                    )
                    const target = document.nodes.find(
                      (node) => node.id === edge.target
                    )
                    if (!source || !target) return null
                    return (
                      <line
                        key={edge.id}
                        x1={source.x + 160}
                        y1={source.y + 30}
                        x2={target.x}
                        y2={target.y + 30}
                        stroke="currentColor"
                        strokeOpacity="0.55"
                        strokeWidth="1.5"
                        strokeDasharray="6 5"
                        strokeLinecap="round"
                        vectorEffect="non-scaling-stroke"
                      />
                    )
                  })}
                </svg>
                {document.nodes.map((node) => (
                  <div
                    key={node.id}
                    draggable={editable}
                    onDragStart={() => setSelectedNodeID(node.id)}
                    onDragEnd={(event) => dragNode(event, node.id)}
                    onClick={() => setSelectedNodeID(node.id)}
                    className={`group absolute z-10 w-40 rounded-xl border bg-background p-3 shadow-sm ${selectedNodeID === node.id ? "border-foreground ring-2 ring-foreground/10" : "border-border/70"}`}
                    style={{ left: node.x, top: node.y }}
                  >
                    {editable && node.type !== "START" && (
                      <Button
                        type="button"
                        draggable={false}
                        variant="outline"
                        size="icon-sm"
                        className="absolute top-1/2 -left-5 z-20 size-7 -translate-y-1/2 rounded-full bg-background opacity-70 shadow-sm transition-opacity group-hover:opacity-100"
                        title="在此节点之前添加"
                        onPointerDown={(event) => event.stopPropagation()}
                        onDragStart={(event) => event.stopPropagation()}
                        onClick={(event) => {
                          event.stopPropagation()
                          openNodeDialog(node.id, "before")
                        }}
                      >
                        <span className="size-3 rounded-full bg-foreground ring-4 ring-background" />
                      </Button>
                    )}
                    <div className="flex items-center gap-2">
                      <GripVertical className="size-3 text-muted-foreground" />
                      <span className="text-xs font-semibold">
                        {nodeLabels[node.type] ?? node.type}
                      </span>
                    </div>
                    <p className="mt-2 truncate text-xs text-muted-foreground">
                      {node.label}
                    </p>
                    {editable && node.type !== "END" && (
                      <Button
                        type="button"
                        draggable={false}
                        variant="outline"
                        size="icon-sm"
                        className="absolute top-1/2 -right-5 z-20 size-7 -translate-y-1/2 rounded-full bg-background opacity-70 shadow-sm transition-opacity group-hover:opacity-100"
                        title="在此节点之后添加"
                        onPointerDown={(event) => event.stopPropagation()}
                        onDragStart={(event) => event.stopPropagation()}
                        onClick={(event) => {
                          event.stopPropagation()
                          openNodeDialog(node.id, "after")
                        }}
                      >
                        <span className="size-3 rounded-full bg-foreground ring-4 ring-background" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </div>
            {issues.length > 0 && (
              <div className="mt-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                <ul className="list-disc pl-5">
                  {issues.map((issue) => (
                    <li key={issue}>{issue}</li>
                  ))}
                </ul>
              </div>
            )}
            {error && <p className="mt-3 text-sm text-destructive">{error}</p>}
            <div className="mt-4 flex flex-wrap gap-2">
              <Button disabled={!editable} onClick={() => openNodeDialog()}>
                <Plus />
                新增节点
              </Button>
            </div>
          </CardContent>
        </Card>
        <Card className="border-border/70 shadow-none">
          <CardHeader>
            <CardTitle>节点属性</CardTitle>
            <CardDescription>
              {selectedNode
                ? `${nodeLabels[selectedNode.type] ?? selectedNode.type} · ${selectedNode.id}`
                : "请选择节点"}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {selectedNode && (
              <>
                <div className="space-y-2">
                  <Label>节点名称</Label>
                  <Input
                    disabled={!editable}
                    value={selectedNode.label}
                    onChange={(event) =>
                      updateNode({ label: event.target.value })
                    }
                  />
                </div>
                {["SET", "GET", "WAIT", "IF"].includes(selectedNode.type) && (
                  <div className="space-y-2">
                    <Label>语义变量</Label>
                    <Select
                      disabled={!editable}
                      value={String(selectedNode.config.variable ?? "")}
                      onValueChange={(variable) => updateNode({}, { variable })}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="选择变量" />
                      </SelectTrigger>
                      <SelectContent>
                        {variables.map((variable) => (
                          <SelectItem key={variable.id} value={variable.name}>
                            {variable.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
                {["WAIT", "IF"].includes(selectedNode.type) && (
                  <>
                    <div className="space-y-2">
                      <Label>运算符</Label>
                      <Select
                        disabled={!editable}
                        value={String(selectedNode.config.operator ?? "==")}
                        onValueChange={(operator) =>
                          updateNode({}, { operator })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {["==", "!=", ">", "<", ">=", "<="].map(
                            (operator) => (
                              <SelectItem key={operator} value={operator}>
                                {operator}
                              </SelectItem>
                            )
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="space-y-2">
                      <Label>超时秒数</Label>
                      <Input
                        disabled={!editable}
                        type="number"
                        min="1"
                        value={String(
                          selectedNode.config.timeout_seconds ?? 10
                        )}
                        onChange={(event) =>
                          updateNode(
                            {},
                            { timeout_seconds: Number(event.target.value) }
                          )
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>超时策略</Label>
                      <Select
                        disabled={!editable}
                        value={String(
                          selectedNode.config.timeout_action ?? "FAIL"
                        )}
                        onValueChange={(timeout_action) =>
                          updateNode({}, { timeout_action })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {[
                            "FAIL",
                            "RETRY",
                            "ALARM",
                            "JUMP",
                            "MANUAL_CONFIRM",
                          ].map((item) => (
                            <SelectItem key={item} value={item}>
                              {item}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </>
                )}
                {selectedNode.type === "SET" && (
                  <div className="space-y-2">
                    <Label>写入值</Label>
                    <Input
                      disabled={!editable}
                      value={String(selectedNode.config.value ?? "")}
                      onChange={(event) =>
                        updateNode({}, { value: event.target.value })
                      }
                    />
                  </div>
                )}
                {selectedNode.type === "LOOP" && (
                  <div className="space-y-2">
                    <Label>最大循环次数</Label>
                    <Input
                      disabled={!editable}
                      type="number"
                      min="1"
                      value={String(selectedNode.config.max_iterations ?? 3)}
                      onChange={(event) =>
                        updateNode(
                          {},
                          { max_iterations: Number(event.target.value) }
                        )
                      }
                    />
                  </div>
                )}
                <Separator />
                <Button
                  disabled={
                    !editable || ["START", "END"].includes(selectedNode.type)
                  }
                  variant="outline"
                  onClick={removeNode}
                >
                  <Trash2 />
                  删除节点
                </Button>
              </>
            )}
          </CardContent>
        </Card>
      </div>
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增流程节点</DialogTitle>
            <DialogDescription>
              节点创建后会加入当前流程路径，详细参数在右侧属性栏配置。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>节点类型</Label>
              <Select
                value={newType}
                onValueChange={(type) => {
                  setNewType(type)
                  setNewLabel(nodeLabels[type])
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {nodeTypes.map((type) => (
                    <SelectItem key={type} value={type}>
                      {nodeLabels[type]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>节点名称</Label>
              <Input
                value={newLabel}
                onChange={(event) => setNewLabel(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={addNode}>
              <Plus />
              创建节点
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
