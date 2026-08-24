import { useEffect, useRef, useState } from "react"
import * as Blockly from "blockly"
import "blockly/blocks"
import * as ZhHans from "blockly/msg/zh-hans"
import { ArrowLeft, Check, Loader2, Save, Send, Settings2 } from "lucide-react"
import { useNavigate, useParams } from "react-router-dom"
import { api, type FlowDefinition, type FlowDocument, type FlowFunction, type FlowNode, type PLC, type PLCVariable, type FlowParameter } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

const nodeTypes = ["SET", "GET", "WAIT", "IF", "SWITCH", "VAR_SET", "CALCULATE", "DELAY", "MANUAL_CONFIRM", "ALARM", "LOOP", "PARALLEL", "SUBFLOW", "FUNCTION_CALL"] as const
const nodeLabels: Record<string, string> = {
  START: "开始", END: "结束", SET: "设置变量", GET: "读取变量", WAIT: "等待条件", IF: "如果",
  DELAY: "延时", MANUAL_CONFIRM: "人工确认", ALARM: "报警", LOOP: "循环", PARALLEL: "并行", SUBFLOW: "子流程", SWITCH: "多路分支", VAR_SET: "内部变量赋值", CALCULATE: "数学计算", FUNCTION_CALL: "调用无返回值函数",
}
const nodeColours: Record<string, number> = { START: 210, END: 210, SET: 5, GET: 190, WAIT: 35, IF: 195, SWITCH: 265, VAR_SET: 125, CALCULATE: 45, DELAY: 25, MANUAL_CONFIRM: 320, ALARM: 5, LOOP: 230, PARALLEL: 165, SUBFLOW: 275, FUNCTION_CALL: 290 }
let plcOptions: [string, string][] = [["请先配置 PLC", ""]]
let variablesByPLC = new Map<string, PLCVariable[]>()
let flowOptions: [string, string][] = [["请先配置已发布子流程", ""]]
let functionOptions: [string, string][] = [["请先新建流程函数", ""]]
let functionsByCode = new Map<string, FlowFunction>()
const emptyDocument: FlowDocument = {
  nodes: [
    { id: "start", type: "START", label: "开始", x: 80, y: 40, config: {} },
    { id: "end", type: "END", label: "结束", x: 80, y: 180, config: {} },
  ],
  edges: [{ id: "start-end", source: "start", target: "end" }],
}
type FlowForm = { code: string; name: string; description: string; timeout_seconds: string }

function clone<T>(value: T): T { return JSON.parse(JSON.stringify(value)) as T }
function blockTypeForNode(type: string) { return type === "IF" ? "flow_if" : `flow_${type.toLowerCase()}` }
function nodeTypeForBlock(type: string) { return type === "flow_if" ? "IF" : type.replace(/^flow_/, "").toUpperCase() }
function plcFieldOptions(): [string, string][] { return plcOptions.length > 0 ? plcOptions : [["请先配置 PLC", ""]] }
function variableFieldOptions(field: Blockly.Field<string>): [string, string][] {
  const block = field.getSourceBlock()
  const plcID = block?.getFieldValue("PLC_ID") ?? ""
  const variables = variablesByPLC.get(plcID) ?? []
  const filtered = variables.filter((variable) => {
    if (block?.type === "flow_set") return variable.flow_write_allowed && variable.access_mode !== "read"
    if (block?.type === "flow_wait" || block?.type === "flow_if" || block?.type === "flow_switch") return variable.condition_allowed
    return true
  })
  return filtered.length > 0
    ? filtered.map((variable) => [`${variable.name} · ${variable.data_type}`, variable.name])
    : [[plcID ? "该 PLC 暂无可用变量" : "请先选择 PLC", ""]]
}
function selectedVariable(name: string | null) {
  for (const variables of variablesByPLC.values()) {
    const variable = variables.find((item) => item.name === name)
    if (variable) return variable
  }
  return undefined
}
function flowFieldOptions(): [string, string][] {
  return flowOptions.length > 0 ? flowOptions : [["请先配置已发布子流程", ""]]
}
function customFunctionOptions(): [string, string][] { return functionOptions.length > 0 ? functionOptions : [["请先新建流程函数", ""]] }

function functionParameters(block: Blockly.Block): FlowParameter[] {
  const code = String(block.getFieldValue("FUNCTION") ?? "")
  const fn = functionsByCode.get(code)
  if (!fn) return code ? [1, 2, 3, 4].map((index) => ({ name: `参数${index}`, type: "string" as const })) : []
  if (Array.isArray(fn.parameters)) return fn.parameters
  try { return JSON.parse(fn.parameters) as FlowParameter[] } catch { return [] }
}

function functionArgumentOptions(parameter: FlowParameter): [string, string][] {
  return parameter.options?.length ? parameter.options.map((option) => [option, option]) : [["请配置下拉项", ""]]
}

function updateFunctionCallShape(block: Blockly.Block) {
  const parameters = functionParameters(block)
  for (let index = 1; index <= 20; index++) { block.removeInput(`ARG_${index}`, true); block.removeInput(`ARG_VALUE_${index}`, true) }
  parameters.forEach((parameter, index) => {
    const inputName = `ARG_${index + 1}`
    if (parameter.type === "select" || parameter.type === "device" || parameter.type === "option") {
      block.appendDummyInput(inputName).appendField(parameter.name).appendField(new Blockly.FieldDropdown(() => functionArgumentOptions(parameter)), `ARG_VALUE_${index + 1}`)
    } else {
      block.appendValueInput(inputName).appendField(parameter.name)
    }
  })
  if (block.rendered && block instanceof Blockly.BlockSvg) block.render()
}

function setFunctionBlockChangeHandler(block: Blockly.Block) {
  block.setOnChange((event) => { if (event instanceof Blockly.Events.BlockChange && event.blockId === block.id && event.name === "FUNCTION") updateFunctionCallShape(block) })
}

function functionArgumentsFromBlock(block: Blockly.Block): unknown[] {
  return functionParameters(block).map((parameter, index) => parameter.type === "select" || parameter.type === "device" || parameter.type === "option"
    ? block.getFieldValue(`ARG_VALUE_${index + 1}`) ?? ""
    : expressionFromBlock(block.getInput(`ARG_${index + 1}`)?.connection?.targetBlock()) ?? "")
}

function defineFlowBlocks() {
  if (Blockly.Blocks.flow_if) return
  const definitions: Record<string, unknown> = {
    flow_start: { init(this: Blockly.Block) { this.appendDummyInput().appendField("开始"); this.setNextStatement(true); this.setColour(nodeColours.START); this.setTooltip("流程开始") } },
    flow_end: { init(this: Blockly.Block) { this.appendDummyInput().appendField("结束"); this.setPreviousStatement(true); this.setColour(nodeColours.END); this.setTooltip("流程结束") } },
    flow_var_get: { init(this: Blockly.Block) { this.appendDummyInput().appendField("内部变量").appendField(new Blockly.FieldVariable("counter"), "VARIABLE"); this.setOutput(true); this.setColour(nodeColours.VAR_SET); this.setTooltip("读取内部变量，可插入计算或赋值输入") } },
    flow_plc_get: { init(this: Blockly.Block) { this.appendDummyInput().appendField("PLC变量").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE"); this.setOutput(true); this.setColour(nodeColours.GET); this.setTooltip("读取 PLC 语义变量，可插入计算或内部变量赋值") } },
    flow_math: { init(this: Blockly.Block) { this.appendValueInput("LEFT").appendField("计算"); this.appendDummyInput().appendField(new Blockly.FieldDropdown([["+", "+"], ["−", "-"], ["×", "*"], ["÷", "/"]]), "OP"); this.appendValueInput("RIGHT"); this.setOutput(true); this.setColour(nodeColours.CALCULATE); this.setTooltip("可嵌套变量、数字和其他数学计算") } },
    flow_compare: { init(this: Blockly.Block) { this.appendValueInput("LEFT").appendField("比较"); this.appendDummyInput().appendField(new Blockly.FieldDropdown([["等于", "=="], ["不等于", "!="], ["大于", ">"], ["小于", "<"], ["大于等于", ">="], ["小于等于", "<="]]), "OP"); this.appendValueInput("RIGHT"); this.setOutput(true, "Boolean"); this.setColour(195); this.setTooltip("比较两个数字或字符串") } },
    flow_to_number: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转数字"); this.setOutput(true, "Number"); this.setColour(45) } },
    flow_to_string: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转字符串"); this.setOutput(true, "String"); this.setColour(45) } },
    flow_function_value: { init(this: Blockly.Block) { this.appendDummyInput().appendField("函数返回值").appendField(new Blockly.FieldDropdown(() => customFunctionOptions()), "FUNCTION"); this.setOutput(true); this.setColour(290); this.setTooltip("调用有返回值的流程函数"); setFunctionBlockChangeHandler(this); updateFunctionCallShape(this) } },
    flow_if: { init(this: Blockly.Block) {
      this.appendDummyInput().appendField("如果").appendField(new Blockly.FieldDropdown([["PLC变量", "plc"], ["内部变量", "internal"]]), "SOURCE").appendField("PLC").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField("变量").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE").appendField("内部").appendField(new Blockly.FieldVariable("counter"), "INTERNAL_VARIABLE").appendField(new Blockly.FieldDropdown([["等于", "=="], ["不等于", "!="], ["大于", ">"], ["小于", "<"], ["大于等于", ">="], ["小于等于", "<="]]), "OP").appendField(new Blockly.FieldTextInput("true"), "EXPECTED")
      this.appendStatementInput("TRUE").appendField("满足")
      this.appendDummyInput().appendField("否则")
      this.appendStatementInput("FALSE")
      this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(nodeColours.IF); this.setTooltip("条件分支。配置变量和比较值，并在两个分支中继续连接 Blockly 积木")
    } },
  }
  for (const type of nodeTypes.filter((item) => item !== "IF")) {
    const blockType = blockTypeForNode(type)
    definitions[blockType] = { init(this: Blockly.Block) {
      if (type === "SET") {
        this.appendDummyInput().appendField("设置 PLC").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField("变量").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE")
        this.appendDummyInput().appendField("写入值").appendField(new Blockly.FieldTextInput("值"), "VALUE")
      } else if (type === "GET") {
        this.appendDummyInput().appendField("读取 PLC").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField("变量").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE")
      } else if (type === "WAIT") {
        this.appendDummyInput().appendField("等待 PLC").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField("变量").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE").appendField(new Blockly.FieldDropdown([["等于", "=="], ["不等于", "!="], ["大于", ">"], ["小于", "<"], ["大于等于", ">="], ["小于等于", "<="]]), "OP").appendField(new Blockly.FieldTextInput("true"), "EXPECTED")
        this.appendDummyInput().appendField("超时秒数").appendField(new Blockly.FieldNumber(10, 1), "TIMEOUT")
        this.appendStatementInput("TIMEOUT_BRANCH").appendField("超时")
      } else if (type === "SWITCH") {
        this.appendDummyInput().appendField("多路 PLC").appendField(new Blockly.FieldDropdown(() => plcFieldOptions()), "PLC_ID").appendField("变量").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variableFieldOptions(this) }), "VARIABLE")
        this.appendDummyInput().appendField("分支值 1").appendField(new Blockly.FieldTextInput("1"), "CASE_1").appendField("分支值 2").appendField(new Blockly.FieldTextInput("2"), "CASE_2")
        this.appendStatementInput("CASE_1_BRANCH").appendField("值 1")
        this.appendStatementInput("CASE_2_BRANCH").appendField("值 2")
        this.appendStatementInput("DEFAULT_BRANCH").appendField("默认")
      } else if (type === "VAR_SET") {
        this.appendDummyInput().appendField("内部变量").appendField(new Blockly.FieldVariable("counter"), "VARIABLE")
        this.appendValueInput("VALUE").appendField("赋值")
      } else if (type === "CALCULATE") {
        this.appendDummyInput().appendField("结果变量").appendField(new Blockly.FieldVariable("result"), "TARGET")
        this.appendValueInput("LEFT").appendField("=")
        this.appendDummyInput().appendField(new Blockly.FieldDropdown([["+", "+"], ["−", "-"], ["×", "*"], ["÷", "/"]]), "OP")
        this.appendValueInput("RIGHT")
      } else if (type === "DELAY") {
        this.appendDummyInput().appendField("延时秒数").appendField(new Blockly.FieldNumber(5, 1), "SECONDS")
      } else if (type === "LOOP") {
        this.appendDummyInput().appendField("循环次数").appendField(new Blockly.FieldNumber(3, 1), "MAX_ITERATIONS")
        this.appendStatementInput("BODY").appendField("循环体")
        this.appendStatementInput("EXIT").appendField("结束后")
      } else if (type === "SUBFLOW") {
        this.appendDummyInput().appendField("子流程").appendField(new Blockly.FieldDropdown(() => flowFieldOptions()), "FLOW_CODE")
        this.appendDummyInput().appendField("超时秒数").appendField(new Blockly.FieldNumber(60, 1), "TIMEOUT")
        this.appendStatementInput("TIMEOUT_BRANCH").appendField("超时")
      } else if (type === "ALARM") {
        this.appendDummyInput().appendField("报警消息").appendField(new Blockly.FieldTextInput("流程报警"), "MESSAGE").appendField("级别").appendField(new Blockly.FieldDropdown([["提示", "info"], ["警告", "warning"], ["严重", "critical"]]), "LEVEL")
      } else if (type === "MANUAL_CONFIRM") {
        this.appendDummyInput().appendField("确认提示").appendField(new Blockly.FieldTextInput("请确认"), "MESSAGE")
      } else if (type === "PARALLEL") {
        this.appendDummyInput().appendField("并行执行")
        this.appendStatementInput("BRANCH_A").appendField("分支一")
        this.appendStatementInput("BRANCH_B").appendField("分支二")
      } else if (type === "FUNCTION_CALL") {
        this.appendDummyInput().appendField("调用函数").appendField(new Blockly.FieldDropdown(() => customFunctionOptions()), "FUNCTION")
        setFunctionBlockChangeHandler(this); updateFunctionCallShape(this)
      } else {
        this.appendDummyInput().appendField(nodeLabels[type])
      }
      if (type === "LOOP" || type === "PARALLEL" || type === "SWITCH") { this.setPreviousStatement(true); this.setNextStatement(true) }
      else { this.setPreviousStatement(true); this.setNextStatement(true) }
      this.setColour(nodeColours[type]); this.setTooltip(nodeLabels[type])
    } }
  }
  Blockly.common.defineBlocks(definitions)
}

function toolboxDefinition() {
  const category = (name: string, colour: number, types: string[]) => ({
    kind: "category",
    name,
    colour: String(colour),
    contents: types.map((type) => ({ kind: "block", type: blockTypeForNode(type) })),
  })
  return {
    kind: "categoryToolbox",
    contents: [
      { kind: "category", name: "流程控制", colour: String(nodeColours.START), contents: ["START", "END", "IF", "SWITCH", "LOOP", "PARALLEL"].map((type) => ({ kind: "block", type: blockTypeForNode(type) })) },
      { kind: "category", name: "PLC 操作", colour: String(nodeColours.SET), contents: [{ kind: "block", type: "flow_set" }, { kind: "block", type: "flow_get" }, { kind: "block", type: "flow_wait" }, { kind: "block", type: "flow_plc_get" }] },
      { kind: "category", name: "内部变量", colour: String(nodeColours.VAR_SET), contents: [{ kind: "block", type: "flow_var_set" }, { kind: "block", type: "flow_var_get" }] },
      { kind: "category", name: "数学运算", colour: String(nodeColours.CALCULATE), contents: [{ kind: "block", type: "flow_math" }, { kind: "block", type: "math_number" }] },
      { kind: "category", name: "比较与转换", colour: "195", contents: [{ kind: "block", type: "flow_compare" }, { kind: "block", type: "flow_to_number" }, { kind: "block", type: "flow_to_string" }, { kind: "block", type: "logic_boolean" }, { kind: "block", type: "text" }] },
      { kind: "category", name: "流程函数", colour: "290", contents: [{ kind: "block", type: "flow_function_value" }, { kind: "block", type: "flow_function_call" }] },
      category("流程动作", nodeColours.DELAY, ["DELAY", "MANUAL_CONFIRM", "ALARM", "SUBFLOW"]),
      { kind: "category", name: "变量管理", colour: String(nodeColours.VAR_SET), contents: [{ kind: "button", text: "创建内部变量", callbackkey: "CREATE_INTERNAL_VARIABLE" }] },
    ],
  }
}

function configFor(type: string): Record<string, unknown> {
  if (type === "LOOP") return { max_iterations: 3 }
  if (type === "DELAY") return { seconds: 5 }
  if (type === "VAR_SET") return { variable: "counter", value: 0 }
  if (type === "CALCULATE") return { target: "result", left: 0, operator: "+", right: 0 }
  if (["SET", "GET", "WAIT", "IF", "SWITCH"].includes(type)) return { variable: "", operator: "==", expected: true, timeout_seconds: 10, timeout_action: "FAIL", max_retries: 0, retry_interval_seconds: 1 }
  if (type === "FUNCTION_CALL") return { function: "", args: [] }
  return {}
}

function parseFieldValue(value: string | null): unknown {
  const text = String(value ?? "").trim()
  if (text.toLowerCase() === "true") return true
  if (text.toLowerCase() === "false") return false
  if (text !== "" && /^-?\d+(\.\d+)?$/.test(text)) return Number(text)
  return text
}

function expressionFromBlock(block?: Blockly.Block | null): unknown {
  if (!block) return undefined
  if (block.type === "flow_var_get") return { kind: "internal", name: block.getFieldValue("VARIABLE") }
  if (block.type === "flow_plc_get") return { kind: "plc_variable", plc_id: Number(block.getFieldValue("PLC_ID") ?? 0), name: block.getFieldValue("VARIABLE") }
  if (block.type === "flow_math") {
    return {
      kind: "math",
      operator: block.getFieldValue("OP") || "+",
      left: expressionFromBlock(block.getInput("LEFT")?.connection?.targetBlock()) ?? 0,
      right: expressionFromBlock(block.getInput("RIGHT")?.connection?.targetBlock()) ?? 0,
    }
  }
  if (block.type === "flow_compare") return { kind: "compare", operator: block.getFieldValue("OP") || "==", left: expressionFromBlock(block.getInput("LEFT")?.connection?.targetBlock()) ?? 0, right: expressionFromBlock(block.getInput("RIGHT")?.connection?.targetBlock()) ?? 0 }
  if (block.type === "flow_to_number" || block.type === "flow_to_string") return { kind: block.type === "flow_to_number" ? "to_number" : "to_string", value: expressionFromBlock(block.getInput("VALUE")?.connection?.targetBlock()) ?? "" }
  if (block.type === "flow_function_value") return { kind: "function", code: block.getFieldValue("FUNCTION") ?? "", args: functionArgumentsFromBlock(block) }
  if (block.type === "math_number") return Number(block.getFieldValue("NUM") ?? 0)
  if (block.type === "logic_boolean") return block.getFieldValue("BOOL") === "TRUE"
  if (block.type === "text") return block.getFieldValue("TEXT") ?? ""
  return undefined
}

function expressionFromInput(block: Blockly.Block, inputName: string, fallback: unknown) {
  return expressionFromBlock(block.getInput(inputName)?.connection?.targetBlock()) ?? fallback
}

function configFromBlock(block: Blockly.Block, type: string) {
  let config = configFor(type)
  try { config = { ...config, ...(block.data ? JSON.parse(block.data) : {}) } } catch { /* old or malformed metadata */ }
  const field = (name: string) => block.getFieldValue(name)
  if (["SET", "GET", "WAIT", "IF", "SWITCH"].includes(type)) {
    config.plc_id = Number(field("PLC_ID") ?? config.plc_id ?? 0) || undefined
    config.variable = field("VARIABLE") ?? config.variable
    const mapped = selectedVariable(String(config.variable ?? ""))
    if (!config.plc_id && mapped) config.plc_id = mapped.plc_id
  }
  if (["WAIT", "IF"].includes(type)) {
    config.operator = field("OP") ?? config.operator
    config.expected = parseFieldValue(field("EXPECTED"))
  }
  if (type === "IF") {
    config.source = field("SOURCE") ?? config.source ?? "plc"
    config.internal_variable = field("INTERNAL_VARIABLE") ?? config.internal_variable
  }
  if (type === "SET") config.value = parseFieldValue(field("VALUE"))
  if (type === "WAIT") {
    config.timeout_seconds = Number(field("TIMEOUT") ?? config.timeout_seconds)
  }
  if (type === "SWITCH") {
    config.case_1 = parseFieldValue(field("CASE_1"))
    config.case_2 = parseFieldValue(field("CASE_2"))
  }
  if (type === "VAR_SET") {
    config.variable = field("VARIABLE") ?? config.variable
    config.value = expressionFromInput(block, "VALUE", config.value)
  }
  if (type === "CALCULATE") {
    config.target = field("TARGET") ?? config.target
    config.left = expressionFromInput(block, "LEFT", config.left)
    config.operator = field("OP") ?? "+"
    config.right = expressionFromInput(block, "RIGHT", config.right)
  }
  if (type === "DELAY") config.seconds = Number(field("SECONDS") ?? config.seconds)
  if (type === "LOOP") config.max_iterations = Number(field("MAX_ITERATIONS") ?? config.max_iterations)
  if (type === "SUBFLOW") {
    config.flow_code = field("FLOW_CODE") ?? ""
    config.timeout_seconds = Number(field("TIMEOUT") ?? 60)
  }
  if (type === "ALARM") {
    config.message = field("MESSAGE") ?? "流程报警"
    config.level = field("LEVEL") ?? "warning"
  }
  if (type === "MANUAL_CONFIRM") config.message = field("MESSAGE") ?? "请确认"
  if (type === "FUNCTION_CALL") {
    config.function = field("FUNCTION") ?? ""
    config.args = functionArgumentsFromBlock(block)
  }
  return config
}

function readBlockNode(block: Blockly.Block): FlowNode {
  const type = nodeTypeForBlock(block.type)
  const position = block.getRelativeToSurfaceXY()
  return { id: block.id, type, label: nodeLabels[type], x: position.x, y: position.y, config: configFromBlock(block, type) }
}

function documentFromWorkspace(workspace: Blockly.WorkspaceSvg, baseDocument: FlowDocument): FlowDocument {
  const blocks = workspace.getAllBlocks(false).filter((block) => !block.outputConnection)
  const nodes = blocks.map(readBlockNode)
  const edges: FlowDocument["edges"] = []
  for (const block of blocks) {
    const next = block.getNextBlock()
    if (next) edges.push({ id: `${block.id}-${next.id}`, source: block.id, target: next.id })
    if (block.type === "flow_if" || block.type === "flow_parallel" || block.type === "flow_loop" || block.type === "flow_wait" || block.type === "flow_subflow" || block.type === "flow_switch") {
      const branchInputs = block.type === "flow_if"
        ? [["TRUE", "true"], ["FALSE", "false"]] as const
        : block.type === "flow_parallel"
          ? [["BRANCH_A", "parallel_1"], ["BRANCH_B", "parallel_2"]] as const
          : block.type === "flow_loop"
            ? [["BODY", "loop"], ["EXIT", "exit"]] as const
            : block.type === "flow_switch"
              ? [["CASE_1_BRANCH", "case_1"], ["CASE_2_BRANCH", "case_2"], ["DEFAULT_BRANCH", "default"]] as const
              : [["TIMEOUT_BRANCH", "timeout"]] as const
      for (const [inputName, condition] of branchInputs) {
        const child = block.getInput(inputName)?.connection?.targetBlock()
        if (child) edges.push({ id: `${block.id}-${condition}-${child.id}`, source: block.id, target: child.id, condition })
      }
    }
  }
  return { ...baseDocument, nodes, edges }
}

function hydrateBlockConfig(block: Blockly.Block, node: FlowNode) {
  const config = node.config ?? {}
  const set = (field: string, value: unknown) => {
    if (block.getField(field) && value !== undefined && value !== null) block.setFieldValue(String(value), field)
  }
  set("VARIABLE", config.variable)
  set("PLC_ID", config.plc_id ?? selectedVariable(String(config.variable ?? ""))?.plc_id)
  set("VALUE", config.value)
  set("OP", config.operator)
  set("EXPECTED", config.expected)
  set("SOURCE", config.source)
  set("INTERNAL_VARIABLE", config.internal_variable)
  set("CASE_1", config.case_1)
  set("CASE_2", config.case_2)
  set("VARIABLE", config.variable)
  set("VALUE", config.value)
  set("TARGET", config.target)
  set("LEFT", config.left)
  set("OP", config.operator)
  set("RIGHT", config.right)
  set("TIMEOUT", config.timeout_seconds)
  set("SECONDS", config.seconds)
  set("MAX_ITERATIONS", config.max_iterations)
  set("FLOW_CODE", config.flow_code)
  set("MESSAGE", config.message)
  set("LEVEL", config.level)
  set("FUNCTION", config.function)
  if (block.type === "flow_function_call" || block.type === "flow_function_value") {
    updateFunctionCallShape(block)
    const parameters = functionParameters(block)
    const args = Array.isArray(config.args) ? config.args : []
    parameters.forEach((parameter, index) => {
      const value = args[index]
      if (parameter.type === "select" || parameter.type === "device" || parameter.type === "option") set(`ARG_VALUE_${index + 1}`, value)
    })
  }
}

function hydrateExpression(workspace: Blockly.WorkspaceSvg, input: Blockly.Input | null, expression: unknown) {
  if (!input?.connection || expression === undefined || expression === null) return
  const create = (value: unknown): Blockly.Block | null => {
    if (typeof value === "number") {
      const numberBlock = workspace.newBlock("math_number")
      numberBlock.initSvg(); numberBlock.render(); numberBlock.setFieldValue(String(value), "NUM")
      return numberBlock
    }
    if (typeof value === "boolean") {
      const booleanBlock = workspace.newBlock("logic_boolean")
      booleanBlock.initSvg(); booleanBlock.render(); booleanBlock.setFieldValue(value ? "TRUE" : "FALSE", "BOOL")
      return booleanBlock
    }
    if (typeof value === "string") {
      const textBlock = workspace.newBlock("text")
      textBlock.initSvg(); textBlock.render(); textBlock.setFieldValue(value, "TEXT")
      return textBlock
    }
    if (!value || typeof value !== "object") return null
    const object = value as Record<string, unknown>
    if (object.kind === "internal") {
      const variableBlock = workspace.newBlock("flow_var_get")
      variableBlock.initSvg(); variableBlock.render(); variableBlock.setFieldValue(String(object.name ?? "counter"), "VARIABLE")
      return variableBlock
    }
    if (object.kind === "plc_variable") {
      const plcBlock = workspace.newBlock("flow_plc_get")
      plcBlock.initSvg(); plcBlock.render(); plcBlock.setFieldValue(String(object.plc_id ?? ""), "PLC_ID"); plcBlock.setFieldValue(String(object.name ?? ""), "VARIABLE")
      return plcBlock
    }
    if (object.kind === "math") {
      const mathBlock = workspace.newBlock("flow_math")
      mathBlock.initSvg(); mathBlock.render(); mathBlock.setFieldValue(String(object.operator ?? "+"), "OP")
      hydrateExpression(workspace, mathBlock.getInput("LEFT"), object.left)
      hydrateExpression(workspace, mathBlock.getInput("RIGHT"), object.right)
      return mathBlock
    }
    if (object.kind === "function") {
      const functionBlock = workspace.newBlock("flow_function_value")
      functionBlock.setFieldValue(String(object.code ?? ""), "FUNCTION")
      updateFunctionCallShape(functionBlock)
      functionBlock.initSvg(); functionBlock.render()
      const args = Array.isArray(object.args) ? object.args : []
      functionParameters(functionBlock).forEach((parameter, index) => {
        if (parameter.type === "select" || parameter.type === "device" || parameter.type === "option") functionBlock.setFieldValue(String(args[index] ?? ""), `ARG_VALUE_${index + 1}`)
        else hydrateExpression(workspace, functionBlock.getInput(`ARG_${index + 1}`), args[index])
      })
      return functionBlock
    }
    return null
  }
  const child = create(expression)
  if (child?.outputConnection) input.connection.connect(child.outputConnection)
}

function collectExpressionVariables(value: unknown, names: Set<string>) {
  if (!value || typeof value !== "object") return
  const object = value as Record<string, unknown>
  if (object.kind === "internal" && typeof object.name === "string") names.add(object.name)
  collectExpressionVariables(object.left, names)
  collectExpressionVariables(object.right, names)
}

function workspaceFromDocument(workspace: Blockly.WorkspaceSvg, document: FlowDocument) {
  workspace.clear()
  const blocks = new Map<string, Blockly.Block>()
  const ifEdgeIndex = new Map<string, number>()
  const internalNames = new Set<string>()
  for (const node of document.nodes) {
    const config = node.config ?? {}
    const names = node.type === "VAR_SET" ? [config.variable] : node.type === "CALCULATE" ? [config.target] : node.type === "IF" ? [config.internal_variable] : []
    for (const name of names) {
      if (typeof name === "string" && name.trim()) internalNames.add(name.trim())
    }
    collectExpressionVariables(config.value, internalNames)
    collectExpressionVariables(config.left, internalNames)
    collectExpressionVariables(config.right, internalNames)
  }
  const variableMap = workspace.getVariableMap()
  for (const name of internalNames) {
    if (!variableMap.getVariable(name)) variableMap.createVariable(name)
  }
  for (const node of document.nodes) {
    const block = workspace.newBlock(blockTypeForNode(node.type))
    block.data = JSON.stringify(node.config ?? {})
    hydrateBlockConfig(block, node)
    block.initSvg(); block.render(); blocks.set(node.id, block)
    hydrateExpression(workspace, block.getInput("VALUE"), node.config?.value)
    hydrateExpression(workspace, block.getInput("LEFT"), node.config?.left)
    hydrateExpression(workspace, block.getInput("RIGHT"), node.config?.right)
    if (node.type === "FUNCTION_CALL" && Array.isArray(node.config?.args)) {
      const args = node.config.args
      functionParameters(block).forEach((parameter, index) => { if (parameter.type !== "select" && parameter.type !== "device" && parameter.type !== "option") hydrateExpression(workspace, block.getInput(`ARG_${index + 1}`), args[index]) })
    }
  }
  for (const edge of document.edges) {
    const source = blocks.get(edge.source); const target = blocks.get(edge.target)
    if (!source || !target || !target.previousConnection) continue
    if ((source.type === "flow_if" || source.type === "flow_parallel" || source.type === "flow_loop" || source.type === "flow_switch") && !edge.condition && source.nextConnection && !source.nextConnection.targetBlock()) {
      source.nextConnection.connect(target.previousConnection)
    } else if (source.type === "flow_if" || source.type === "flow_parallel" || source.type === "flow_loop" || source.type === "flow_switch") {
      const index = ifEdgeIndex.get(source.id) ?? 0
      ifEdgeIndex.set(source.id, index + 1)
      const condition = source.type === "flow_if"
        ? (edge.condition === "false" || (!edge.condition && index > 0) ? "FALSE" : "TRUE")
        : source.type === "flow_parallel"
          ? (edge.condition === "parallel_2" || (!edge.condition && index > 0) ? "BRANCH_B" : "BRANCH_A")
        : source.type === "flow_loop"
          ? (edge.condition === "exit" || (!edge.condition && index > 0) ? "EXIT" : "BODY")
          : edge.condition === "case_2" ? "CASE_2_BRANCH" : edge.condition === "default" ? "DEFAULT_BRANCH" : "CASE_1_BRANCH"
      const input = source.getInput(condition)
      if (input?.connection && !input.connection.targetBlock()) input.connection.connect(target.previousConnection)
    } else if ((source.type === "flow_wait" || source.type === "flow_subflow") && edge.condition === "timeout") {
      const input = source.getInput("TIMEOUT_BRANCH")
      if (input?.connection && !input.connection.targetBlock()) input.connection.connect(target.previousConnection)
    } else if (source.nextConnection && !source.nextConnection.targetBlock()) source.nextConnection.connect(target.previousConnection)
  }
  for (const node of document.nodes) {
    const block = blocks.get(node.id)
    if (!block || block.getParent()) continue
    block.moveBy(node.x || 40, node.y || 40)
  }
}

export function FlowEditorPage() {
  const { id = "new" } = useParams(); const navigate = useNavigate(); const isNew = id === "new"
  const [flow, setFlow] = useState<FlowDefinition | null>(null)
  const [document, setDocument] = useState<FlowDocument>(clone(emptyDocument))
  const [form, setForm] = useState<FlowForm>({ code: "FLOW-001", name: "新流程", description: "", timeout_seconds: "0" })
  const [resourceMode, setResourceMode] = useState<"variables" | "options" | null>(null)
  const [variableDraft, setVariableDraft] = useState({ name: "", type: "string", default_value: "" })
  const [optionDraft, setOptionDraft] = useState("")
  const [issues, setIssues] = useState<string[]>([]); const [error, setError] = useState(""); const [saving, setSaving] = useState(false); const [loading, setLoading] = useState(!isNew); const [workspaceReady, setWorkspaceReady] = useState(isNew); const [referencesReady, setReferencesReady] = useState(false)
  const hostRef = useRef<HTMLDivElement>(null); const workspaceRef = useRef<Blockly.WorkspaceSvg | null>(null); const documentRef = useRef(document); const hydratingRef = useRef(false)
  const editable = !flow || flow.status === "draft"
  useEffect(() => { documentRef.current = document }, [document])
  useEffect(() => {
    Promise.all([
      api<{ items: PLC[] }>("/api/plcs?page_size=100"),
      api<{ items: PLCVariable[] }>("/api/variables?page_size=100"),
      api<{ items: FlowDefinition[] }>("/api/flows?page_size=100&filter_status=published"),
      api<{ items: FlowFunction[] }>("/api/flow-functions"),
    ]).then(([plcResult, variableResult, flowResult, functionResult]) => {
      plcOptions = plcResult.items.map((plc) => [`${plc.name} · ${plc.code}`, String(plc.id)])
      variablesByPLC = new Map<string, PLCVariable[]>()
      for (const variable of variableResult.items) {
        const key = String(variable.plc_id)
        variablesByPLC.set(key, [...(variablesByPLC.get(key) ?? []), variable])
      }
      flowOptions = flowResult.items.map((item) => [`${item.name} · ${item.code}`, item.code])
      functionOptions = functionResult.items.map((item) => [`${item.name} · ${item.code}`, item.code])
      functionsByCode = new Map(functionResult.items.map((item) => [item.code, item]))
    }).catch(() => {
      plcOptions = [["暂无可用 PLC", ""]]
    }).finally(() => setReferencesReady(true))
  }, [])
  useEffect(() => {
    if (isNew) return
    api<{ flow: FlowDefinition }>(`/api/flows/${id}`).then((result) => {
      const loaded = JSON.parse(result.flow.definition) as FlowDocument
      setFlow(result.flow); setForm({ code: result.flow.code, name: result.flow.name, description: result.flow.description, timeout_seconds: String(result.flow.timeout_seconds) }); setDocument(loaded); documentRef.current = loaded; setWorkspaceReady(true)
    }).catch((err) => setError(err instanceof Error ? err.message : "加载流程失败")).finally(() => setLoading(false))
  }, [id, isNew])
  function updateDocumentMetadata(update: Partial<FlowDocument>) { setDocument((current) => { const next = { ...current, ...update }; documentRef.current = next; return next }) }
  function saveVariable() { if (!variableDraft.name.trim()) return; const variables = [...(documentRef.current.variables ?? []).filter((item) => item.name !== variableDraft.name.trim()), { name: variableDraft.name.trim(), type: variableDraft.type, default_value: variableDraft.default_value }]; updateDocumentMetadata({ variables }); setVariableDraft({ name: "", type: "string", default_value: "" }) }
  function editVariable(name: string) { const item = (documentRef.current.variables ?? []).find((entry) => entry.name === name); if (item) setVariableDraft({ name: item.name, type: item.type, default_value: item.default_value ?? "" }) }
  function removeVariable(name: string) { updateDocumentMetadata({ variables: (documentRef.current.variables ?? []).filter((item) => item.name !== name) }) }
  function saveOption() { const value = optionDraft.trim(); if (!value) return; const options = [...new Set([...(documentRef.current.options ?? []), value])]; updateDocumentMetadata({ options }); setOptionDraft("") }
  function removeOption(value: string) { updateDocumentMetadata({ options: (documentRef.current.options ?? []).filter((item) => item !== value) }) }
  useEffect(() => {
    if (!workspaceReady || !referencesReady || !hostRef.current || workspaceRef.current) return
    defineFlowBlocks(); Blockly.setLocale(ZhHans as unknown as Record<string, string>)
    const workspace = Blockly.inject(hostRef.current, { toolbox: toolboxDefinition(), trashcan: true, grid: { spacing: 24, length: 3, colour: "#c6d4ec", snap: true }, zoom: { controls: true, wheel: true, startScale: 1, maxScale: 1.4, minScale: 0.65 }, move: { scrollbars: true, drag: true, wheel: true } })
    workspace.registerButtonCallback("CREATE_INTERNAL_VARIABLE", () => Blockly.Variables.createVariableButtonHandler(workspace))
    workspaceRef.current = workspace; hydratingRef.current = true; workspaceFromDocument(workspace, documentRef.current); hydratingRef.current = false
    const listener = (event: Blockly.Events.Abstract) => {
      if (hydratingRef.current) return
      if (event.type === Blockly.Events.BLOCK_CHANGE && event instanceof Blockly.Events.BlockChange && event.name === "PLC_ID") {
        const block = event.blockId ? workspace.getBlockById(event.blockId) : undefined
        const variable = block?.getField("VARIABLE")
        const options = variable ? variableFieldOptions(variable) : []
        const firstVariable = options[0]?.[1]
        if (variable && firstVariable !== undefined) variable.setValue(firstVariable)
      }
      const next = documentFromWorkspace(workspace, documentRef.current); documentRef.current = next; setDocument(next)
    }
    workspace.addChangeListener(listener)
    return () => { workspace.removeChangeListener(listener); workspace.dispose(); workspaceRef.current = null }
  }, [workspaceReady, referencesReady])

  function syncDocument() { const workspace = workspaceRef.current; if (!workspace) return documentRef.current; const next = documentFromWorkspace(workspace, documentRef.current); documentRef.current = next; setDocument(next); return next }
  async function save() {
    setSaving(true)
    try { const definition = syncDocument(); const result = await api<{ flow: FlowDefinition }>(isNew ? "/api/flows" : `/api/flows/${id}`, { method: isNew ? "POST" : "PUT", body: JSON.stringify({ ...form, timeout_seconds: Number(form.timeout_seconds), definition }) }); setFlow(result.flow); navigate(`/flows/${result.flow.id}/edit`, { replace: true }); setError("") }
    catch (err) { setError(err instanceof Error ? err.message : "保存流程失败") } finally { setSaving(false) }
  }
  async function validate() {
    if (!flow) { setIssues(["请先保存流程"]); return }
    try { const result = await api<{ issues: string[] }>(`/api/flows/${flow.id}/validate`, { method: "POST", body: "{}" }); setIssues(result.issues) }
    catch (err) { setError(err instanceof Error ? err.message : "流程校验失败") }
  }
  async function publish() {
    if (!flow) return
    try { const result = await api<{ flow: FlowDefinition }>(`/api/flows/${flow.id}/publish`, { method: "POST", body: "{}" }); setFlow(result.flow) }
    catch (err) { setError(err instanceof Error ? err.message : "发布流程失败") }
  }
  if (loading) return <div className="grid min-h-96 place-items-center"><Loader2 className="animate-spin" /></div>
  return <div className="space-y-6">
    <div className="flex items-center justify-between"><Button variant="ghost" onClick={() => navigate("/flows")}><ArrowLeft />返回流程清单</Button><div className="flex gap-2"><Button variant="outline" disabled={!editable || saving} onClick={() => void save()}><Save />保存</Button>{flow && editable && <><Button variant="outline" onClick={() => void validate()}><Check />校验</Button><Button onClick={() => void publish()}><Send />发布</Button></>}</div></div>
    <Card className="border-border/70 shadow-none"><CardHeader><CardTitle>{form.name || "新流程"}{flow && ` · v${flow.version}`}</CardTitle><CardDescription>Blockly 流程编辑器：从左侧工具箱拖入积木，连接点会自动吸附；条件块的“满足”和“否则”分支支持继续嵌套。</CardDescription></CardHeader><CardContent>
      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4"><div className="space-y-2"><Label>流程编码</Label><Input disabled={!editable} value={form.code} onChange={(event) => setForm({ ...form, code: event.target.value })} /></div><div className="space-y-2"><Label>流程名称</Label><Input disabled={!editable} value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} /></div><div className="space-y-2"><Label>总超时（秒）</Label><Input disabled={!editable} type="number" min="0" value={form.timeout_seconds} onChange={(event) => setForm({ ...form, timeout_seconds: event.target.value })} /></div><div className="space-y-2"><Label>描述</Label><Input disabled={!editable} value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></div></div>
      <div className="mb-4 grid gap-3 sm:grid-cols-3"><Button type="button" variant="outline" disabled={!editable} onClick={() => setResourceMode("variables")}><Settings2 />管理流程变量（{document.variables?.length ?? 0}）</Button><Button type="button" variant="outline" disabled={!editable} onClick={() => setResourceMode("options")}><Settings2 />管理下拉项（{document.options?.length ?? 0}）</Button><Button type="button" variant="outline" onClick={() => navigate("/flow-functions")}><Settings2 />管理全局流程函数</Button></div>
      <div ref={hostRef} className={`blockly-editor ${editable ? "" : "pointer-events-none opacity-75"}`} aria-label="Blockly 流程工作区" />
      {issues.length > 0 && <div className="mt-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"><ul className="list-disc pl-5">{issues.map((issue) => <li key={issue}>{issue}</li>)}</ul></div>}{error && <p className="mt-3 text-sm text-destructive">{error}</p>}
    </CardContent></Card>
    <Dialog open={resourceMode !== null} onOpenChange={(open) => !open && setResourceMode(null)}><DialogContent><DialogHeader><DialogTitle>{resourceMode === "variables" ? "管理流程变量" : "管理流程下拉项"}</DialogTitle><DialogDescription>这里定义的项会保存到当前流程，工单和流程函数积木使用字符串值。</DialogDescription></DialogHeader>{resourceMode === "variables" ? <div className="space-y-4"><div className="grid gap-2 sm:grid-cols-3"><Input placeholder="变量名" value={variableDraft.name} onChange={(event) => setVariableDraft({ ...variableDraft, name: event.target.value })} /><select className="h-9 rounded-md border bg-background px-3 text-sm" value={variableDraft.type} onChange={(event) => setVariableDraft({ ...variableDraft, type: event.target.value })}><option value="string">string</option><option value="select">select</option><option value="number">number</option><option value="boolean">boolean</option></select><Input placeholder="默认值" value={variableDraft.default_value} onChange={(event) => setVariableDraft({ ...variableDraft, default_value: event.target.value })} /></div><Button type="button" onClick={saveVariable}>添加/保存变量</Button><div className="space-y-2">{(document.variables ?? []).map((item) => <div className="flex items-center justify-between rounded border p-2 text-sm" key={item.name}><span>{item.name} · {item.type}</span><span className="flex gap-1"><Button type="button" size="sm" variant="ghost" onClick={() => editVariable(item.name)}>编辑</Button><Button type="button" size="sm" variant="ghost" onClick={() => removeVariable(item.name)}>删除</Button></span></div>)}</div></div> : <div className="space-y-4"><div className="flex gap-2"><Input placeholder="下拉项值" value={optionDraft} onChange={(event) => setOptionDraft(event.target.value)} /><Button type="button" onClick={saveOption}>添加</Button></div><div className="flex flex-wrap gap-2">{(document.options ?? []).map((item) => <Button type="button" size="sm" variant="outline" key={item} onClick={() => removeOption(item)}>{item} ×</Button>)}</div></div>}<DialogFooter><Button variant="outline" onClick={() => setResourceMode(null)}>完成</Button></DialogFooter></DialogContent></Dialog>
  </div>
}
