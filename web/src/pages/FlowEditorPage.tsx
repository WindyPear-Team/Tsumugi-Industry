import { useEffect, useState, type DragEvent } from "react"
import {
  ArrowLeft,
  Check,
  GripVertical,
  Loader2,
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
import { Badge } from "@/components/ui/badge"
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
const nodeColors: Record<string, string> = {
  START: "bg-blue-500 border-blue-600",
  END: "bg-blue-500 border-blue-600",
  SET: "bg-red-500 border-red-600",
  GET: "bg-cyan-500 border-cyan-600",
  WAIT: "bg-amber-500 border-amber-600",
  IF: "bg-violet-500 border-violet-600",
  DELAY: "bg-orange-500 border-orange-600",
  MANUAL_CONFIRM: "bg-pink-500 border-pink-600",
  ALARM: "bg-red-600 border-red-700",
  LOOP: "bg-indigo-500 border-indigo-600",
  PARALLEL: "bg-teal-500 border-teal-600",
  SUBFLOW: "bg-fuchsia-500 border-fuchsia-600",
}
const emptyDocument: FlowDocument = {
  nodes: [
    { id: "start", type: "START", label: "开始", x: 80, y: 40, config: {} },
    { id: "end", type: "END", label: "结束", x: 80, y: 160, config: {} },
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
function configFor(type: string): Record<string, unknown> {
  if (type === "LOOP") return { max_iterations: 3 }
  if (type === "DELAY") return { seconds: 5 }
  if (["SET", "GET", "WAIT", "IF"].includes(type))
    return {
      variable: "",
      operator: "==",
      expected: true,
      timeout_seconds: 10,
      timeout_action: "FAIL",
      max_retries: 0,
    }
  return {}
}

function chainDocument(nodes: FlowNode[]): FlowDocument {
  const start = nodes.find((node) => node.type === "START") ?? {
    id: "start",
    type: "START",
    label: "开始",
    x: 80,
    y: 40,
    config: {},
  }
  const end = nodes.find((node) => node.type === "END") ?? {
    id: "end",
    type: "END",
    label: "结束",
    x: 80,
    y: 160,
    config: {},
  }
  const middle = nodes.filter((node) => !["START", "END"].includes(node.type))
  const ordered = [start, ...middle, end].map((node, index) => ({
    ...node,
    x: 80,
    y: 40 + index * 128,
  }))
  return {
    nodes: ordered,
    edges: ordered.slice(0, -1).map((node, index) => ({
      id: `${node.id}-magnet-${ordered[index + 1].id}`,
      source: node.id,
      target: ordered[index + 1].id,
    })),
  }
}

function isMagneticDocument(flowDocument: FlowDocument) {
  const start = flowDocument.nodes.find((node) => node.type === "START")
  const end = flowDocument.nodes.find((node) => node.type === "END")
  if (!start || !end || flowDocument.nodes.length < 2) return false
  const outgoing = new Map<string, string>()
  const incoming = new Set<string>()
  for (const edge of flowDocument.edges) {
    if (outgoing.has(edge.source) || incoming.has(edge.target)) return false
    outgoing.set(edge.source, edge.target)
    incoming.add(edge.target)
  }

  const visited = new Set<string>()
  let current: string | undefined = start.id
  while (current && !visited.has(current)) {
    visited.add(current)
    current = outgoing.get(current)
  }
  if (!visited.has(end.id)) return false
  return flowDocument.edges.every(
    (edge) => visited.has(edge.source) && visited.has(edge.target)
  )
}

function linearNodes(flowDocument: FlowDocument) {
  const outgoing = new Map(
    flowDocument.edges.map((edge) => [edge.source, edge.target])
  )
  const byID = new Map(flowDocument.nodes.map((node) => [node.id, node]))
  const ordered: FlowNode[] = []
  const visited = new Set<string>()
  let current = flowDocument.nodes.find((node) => node.type === "START")?.id
  while (current && !visited.has(current)) {
    const node = byID.get(current)
    if (!node) break
    ordered.push(node)
    visited.add(current)
    current = outgoing.get(current)
  }
  return ordered
}

function rebuildChain(flowDocument: FlowDocument, middle: FlowNode[]) {
  const start = flowDocument.nodes.find((node) => node.type === "START")
  const end = flowDocument.nodes.find((node) => node.type === "END")
  const chain = chainDocument(
    [start, ...middle, end].filter((node): node is FlowNode => Boolean(node))
  )
  const chainIDs = new Set(chain.nodes.map((node) => node.id))
  return {
    nodes: [
      ...chain.nodes,
      ...flowDocument.nodes.filter((node) => !chainIDs.has(node.id)),
    ],
    edges: chain.edges,
  }
}

function DropSlot({
  active,
  onDragOver,
  onDrop,
}: {
  active: boolean
  onDragOver: (event: DragEvent) => void
  onDrop: (event: DragEvent) => void
}) {
  return (
    <div
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={`-my-px h-1 rounded-sm border border-dashed transition-all ${active ? "my-1 h-12 rounded-r-full border-slate-400 bg-slate-400/50 dark:border-slate-500 dark:bg-slate-500/50" : "border-transparent"}`}
    />
  )
}

function Block({
  node,
  selected,
  draggable,
  onSelect,
  onDragStart,
  onDragEnd,
}: {
  node?: FlowNode
  selected: boolean
  draggable: boolean
  onSelect: () => void
  onDragStart?: () => void
  onDragEnd?: () => void
}) {
  if (!node) return null
  return (
    <div
      draggable={draggable}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onClick={onSelect}
      className={`relative mx-auto w-full max-w-md rounded-l-md rounded-r-[2rem] border-2 p-4 text-white shadow-sm transition-shadow ${nodeColors[node.type] ?? "border-slate-600 bg-slate-500"} ${selected ? "ring-2 ring-foreground/40" : ""} ${draggable ? "cursor-grab active:cursor-grabbing" : "cursor-default"}`}
    >
      <div className="flex items-center gap-3">
        <GripVertical className="size-4 text-muted-foreground" />
        <div>
          <p className="font-semibold">{nodeLabels[node.type] ?? node.type}</p>
          <p className="text-sm text-muted-foreground">{node.label}</p>
        </div>
        <Badge
          className="ml-auto border-white/40 bg-black/10 text-white"
          variant="outline"
        >
          {node.type}
        </Badge>
      </div>
    </div>
  )
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
  const [newType, setNewType] = useState("SET")
  const [newLabel, setNewLabel] = useState(nodeLabels.SET)
  const [draggingID, setDraggingID] = useState<string | null>(null)
  const [dropIndex, setDropIndex] = useState<number | null>(null)
  const [issues, setIssues] = useState<string[]>([])
  const [error, setError] = useState("")
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(!isNew)
  const editable = !flow || flow.status === "draft"
  const selectedNode = document.nodes.find((node) => node.id === selectedNodeID)
  const magnetic = isMagneticDocument(document)
  const blocks = magnetic ? linearNodes(document) : document.nodes
  const chainIDs = new Set(blocks.map((node) => node.id))
  const looseBlocks = magnetic
    ? document.nodes.filter((node) => !chainIDs.has(node.id))
    : []
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
      .finally(() => setLoading(false))
  }, [id, isNew])
  function updateNode(
    patch: Partial<FlowNode>,
    config?: Record<string, unknown>
  ) {
    if (!selectedNode || !editable) return
    setDocument((current) => ({
      ...current,
      nodes: current.nodes.map((node) =>
        node.id === selectedNode.id
          ? {
              ...node,
              ...patch,
              config: config ? { ...node.config, ...config } : node.config,
            }
          : node
      ),
    }))
  }
  function openNodeDialog() {
    setNewType("SET")
    setNewLabel(nodeLabels.SET)
    setDialogOpen(true)
  }
  function addNode() {
    if (!isMagneticDocument(document)) {
      setError("当前流程包含分支或无效连线，磁吸积木仅支持线性流程。")
      return
    }
    const node: FlowNode = {
      id: newID(newType.toLowerCase()),
      type: newType,
      label: newLabel || nodeLabels[newType],
      x: 80,
      y: 0,
      config: configFor(newType),
    }
    setDocument((current) => ({
      ...current,
      nodes: [...current.nodes, node],
    }))
    setSelectedNodeID(node.id)
    setDialogOpen(false)
  }
  function removeNode() {
    if (
      !selectedNode ||
      !editable ||
      ["START", "END"].includes(selectedNode.type)
    )
      return
    if (!isMagneticDocument(document)) {
      setError("当前流程包含分支或无效连线，磁吸排序暂不可用。")
      return
    }
    setDocument((current) => {
      const remaining = current.nodes.filter(
        (node) => node.id !== selectedNode.id
      )
      return rebuildChain(
        { ...current, nodes: remaining },
        blocks.filter(
          (node) =>
            !["START", "END"].includes(node.type) && node.id !== selectedNode.id
        )
      )
    })
    setSelectedNodeID("start")
  }
  function moveNode(fromID: string, targetIndex: number) {
    if (!editable || !isMagneticDocument(document)) {
      setError("当前流程包含分支或无效连线，磁吸排序暂不可用。")
      setDraggingID(null)
      setDropIndex(null)
      return
    }
    const middle = blocks.filter(
      (node) => !["START", "END"].includes(node.type)
    )
    const from = middle.findIndex((node) => node.id === fromID)
    const node = document.nodes.find((item) => item.id === fromID)
    if (!node || ["START", "END"].includes(node.type)) return
    if (from >= 0) middle.splice(from, 1)
    middle.splice(Math.max(0, Math.min(targetIndex, middle.length)), 0, node)
    setDocument(rebuildChain(document, middle))
    setSelectedNodeID(node.id)
    setDraggingID(null)
    setDropIndex(null)
  }
  function onDrop(event: DragEvent, targetIndex: number) {
    event.preventDefault()
    if (draggingID) moveNode(draggingID, targetIndex)
  }
  function onDragOver(event: DragEvent, targetIndex: number) {
    if (!editable || !draggingID) return
    event.preventDefault()
    setDropIndex(targetIndex)
  }
  function detachNode(nodeID: string) {
    if (!editable || !isMagneticDocument(document)) return
    const middle = blocks.filter(
      (node) => !["START", "END"].includes(node.type) && node.id !== nodeID
    )
    setDocument(rebuildChain(document, middle))
    setDraggingID(null)
    setDropIndex(null)
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
  if (loading)
    return (
      <div className="grid min-h-96 place-items-center">
        <Loader2 className="animate-spin" />
      </div>
    )
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
              流程积木编辑器：拖动积木到灰色吸附位即可接入或调整执行顺序，未吸附的积木可以独立放置。
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
            <div className="rounded-2xl border border-dashed bg-muted/20 p-6">
              <div className="mx-auto max-w-xl">
                {!magnetic && (
                  <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">
                    当前流程包含分支或非线性连线，已保留原定义。磁吸排序仅对线性流程开放，避免误改生产流程拓扑。
                  </div>
                )}
                {magnetic ? (
                  <>
                    <Block
                      node={blocks[0]}
                      selected={selectedNodeID === blocks[0]?.id}
                      draggable={false}
                      onSelect={() =>
                        setSelectedNodeID(blocks[0]?.id ?? "start")
                      }
                    />
                    {blocks.slice(1, -1).map((node, index) => (
                      <div key={node.id}>
                        <DropSlot
                          active={dropIndex === index}
                          onDragOver={(event) => onDragOver(event, index)}
                          onDrop={(event) => onDrop(event, index)}
                        />
                        <Block
                          node={node}
                          selected={selectedNodeID === node.id}
                          draggable={editable}
                          onSelect={() => setSelectedNodeID(node.id)}
                          onDragStart={() => {
                            setDraggingID(node.id)
                            setSelectedNodeID(node.id)
                          }}
                          onDragEnd={() => {
                            setDraggingID(null)
                            setDropIndex(null)
                          }}
                        />
                      </div>
                    ))}
                    <DropSlot
                      active={dropIndex === Math.max(0, blocks.length - 2)}
                      onDragOver={(event) =>
                        onDragOver(event, Math.max(0, blocks.length - 2))
                      }
                      onDrop={(event) =>
                        onDrop(event, Math.max(0, blocks.length - 2))
                      }
                    />
                    <Block
                      node={blocks.at(-1)}
                      selected={selectedNodeID === blocks.at(-1)?.id}
                      draggable={false}
                      onSelect={() =>
                        setSelectedNodeID(blocks.at(-1)?.id ?? "end")
                      }
                    />
                  </>
                ) : (
                  <div className="space-y-3">
                    {blocks.map((node) => (
                      <Block
                        key={node.id}
                        node={node}
                        selected={selectedNodeID === node.id}
                        draggable={false}
                        onSelect={() => setSelectedNodeID(node.id)}
                      />
                    ))}
                  </div>
                )}
                {magnetic && (
                  <div
                    onDragOver={(event) => {
                      if (!editable || !draggingID) return
                      event.preventDefault()
                    }}
                    onDrop={(event) => {
                      event.preventDefault()
                      if (draggingID) detachNode(draggingID)
                    }}
                    className="mt-8 min-h-28 rounded-xl border-2 border-dashed border-muted-foreground/25 bg-muted/20 p-4"
                  >
                    <p className="mb-3 text-xs text-muted-foreground">
                      零散流程块 · 拖到灰色吸附位即可接入流程
                    </p>
                    <div className="flex flex-wrap gap-3">
                      {looseBlocks.map((node) => (
                        <div key={node.id} className="w-full max-w-md">
                          <Block
                            node={node}
                            selected={selectedNodeID === node.id}
                            draggable={editable}
                            onSelect={() => setSelectedNodeID(node.id)}
                            onDragStart={() => {
                              setDraggingID(node.id)
                              setSelectedNodeID(node.id)
                            }}
                            onDragEnd={() => {
                              setDraggingID(null)
                              setDropIndex(null)
                            }}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                <div className="mt-5 flex justify-center">
                  <Button
                    disabled={!editable || !magnetic}
                    onClick={openNodeDialog}
                  >
                    <Plus />
                    新增流程
                  </Button>
                </div>
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
          </CardContent>
        </Card>
        <Card className="border-border/70 shadow-none">
          <CardHeader>
            <CardTitle>积木属性</CardTitle>
            <CardDescription>
              {selectedNode
                ? `${nodeLabels[selectedNode.type] ?? selectedNode.type} · ${selectedNode.id}`
                : "请选择积木"}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {selectedNode && (
              <>
                <div className="space-y-2">
                  <Label>积木名称</Label>
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
                  删除积木
                </Button>
              </>
            )}
          </CardContent>
        </Card>
      </div>
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增流程</DialogTitle>
            <DialogDescription>
              选择流程节点类型。创建后会放入零散流程块区域，再拖到灰色吸附位接入流程。
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>积木类型</Label>
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
              <Label>积木名称</Label>
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
              创建流程
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
