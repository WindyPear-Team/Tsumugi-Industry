import * as Blockly from "blockly"
import "blockly/blocks"

let widgetOptions: [string, string][] = [["请先添加组件", ""]]
let plcOptions: [string, string][] = [["请先配置 PLC", ""]]
let variablesByPLC = new Map<string, [string, string][]>()
function widgets() { return widgetOptions }
function plcs() { return plcOptions }
function variables(field: Blockly.Field<string>): [string, string][] { return variablesByPLC.get(field.getSourceBlock()?.getFieldValue("PLC_ID") ?? "") ?? [["请先配置变量", ""]] }

export function setDashboardWidgetOptions(options: [string, string][]) { widgetOptions = options.length ? options : [["请先添加组件", ""]] }
export function setDashboardDataOptions(plcsList: [string, string][], variablesList: { plc_id: number; name: string; data_type?: string }[]) {
  plcOptions = plcsList.length ? plcsList : [["请先配置 PLC", ""]]
  variablesByPLC = new Map<string, [string, string][]>()
  for (const item of variablesList) {
    const key = String(item.plc_id)
    variablesByPLC.set(key, [...(variablesByPLC.get(key) ?? []), [item.name + (item.data_type ? " · " + item.data_type : ""), item.name]])
  }
}

export function defineDashboardBlocks() {
  if (Blockly.Blocks.dashboard_start) return
  Blockly.common.defineBlocks({
    dashboard_start: { init(this: Blockly.Block) { this.appendDummyInput().appendField("开始"); this.setNextStatement(true); this.setColour(210) } },
    dashboard_end: { init(this: Blockly.Block) { this.appendDummyInput().appendField("结束"); this.setPreviousStatement(true); this.setColour(210) } },
    dashboard_set_text: { init(this: Blockly.Block) { this.appendDummyInput().appendField("修改文字").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendValueInput("VALUE").appendField("内容"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(160) } },
    dashboard_set_image: { init(this: Blockly.Block) { this.appendDummyInput().appendField("修改图片").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendValueInput("VALUE").appendField("地址"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(160) } },
    dashboard_plc_get: { init(this: Blockly.Block) { this.appendDummyInput().appendField("获取 PLC 数据").appendField(new Blockly.FieldDropdown(() => plcs()), "PLC_ID").appendField(new Blockly.FieldDropdown(function(this: Blockly.FieldDropdown) { return variables(this) }), "VARIABLE"); this.setOutput(true); this.setColour(190) } },
    dashboard_math: { init(this: Blockly.Block) { this.appendValueInput("LEFT").appendField("计算"); this.appendDummyInput().appendField(new Blockly.FieldDropdown([["+", "+"], ["−", "-"], ["×", "*"], ["÷", "/"]]), "OP"); this.appendValueInput("RIGHT"); this.setOutput(true); this.setColour(230) } },
    dashboard_compare: { init(this: Blockly.Block) { this.appendValueInput("LEFT").appendField("比较"); this.appendDummyInput().appendField(new Blockly.FieldDropdown([["等于", "=="], ["不等于", "!="], ["大于", ">"], ["小于", "<"], ["大于等于", ">="], ["小于等于", "<="]]), "OP"); this.appendValueInput("RIGHT"); this.setOutput(true, "Boolean"); this.setColour(195) } },
    dashboard_to_number: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转数字"); this.setOutput(true, "Number"); this.setColour(45) } },
    dashboard_to_string: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转字符串"); this.setOutput(true, "String"); this.setColour(45) } },
  })
}

export function dashboardToolbox() { return { kind: "categoryToolbox", contents: [
  { kind: "category", name: "流程", colour: "210", contents: [{ kind: "block", type: "dashboard_start" }, { kind: "block", type: "dashboard_end" }] },
  { kind: "category", name: "获取数据", colour: "190", contents: [{ kind: "block", type: "dashboard_plc_get" }, { kind: "block", type: "text" }, { kind: "block", type: "math_number" }, { kind: "block", type: "logic_boolean" }] },
  { kind: "category", name: "修改组件", colour: "160", contents: [{ kind: "block", type: "dashboard_set_text" }, { kind: "block", type: "dashboard_set_image" }] },
  { kind: "category", name: "基本运算", colour: "230", contents: [{ kind: "block", type: "dashboard_math" }, { kind: "block", type: "dashboard_compare" }, { kind: "block", type: "dashboard_to_number" }, { kind: "block", type: "dashboard_to_string" }, { kind: "block", type: "math_arithmetic" }, { kind: "block", type: "logic_compare" }] },
  { kind: "category", name: "逻辑控制", colour: "210", contents: [{ kind: "block", type: "controls_if" }, { kind: "block", type: "controls_repeat_ext" }, { kind: "block", type: "logic_operation" }, { kind: "block", type: "logic_negate" }] },
  { kind: "category", name: "文本与列表", colour: "160", contents: [{ kind: "block", type: "text_join" }, { kind: "block", type: "text_length" }, { kind: "block", type: "lists_create_with" }, { kind: "block", type: "lists_getIndex" }, { kind: "block", type: "lists_length" }] },
] } }

export function loadDashboardDefinition(workspace: Blockly.WorkspaceSvg, definition?: string) { workspace.clear(); if (!definition) return; try { Blockly.serialization.workspaces.load(JSON.parse(definition) as never, workspace) } catch { /* ignore malformed old definitions */ } }
export function saveDashboardDefinition(workspace: Blockly.WorkspaceSvg) { return JSON.stringify(Blockly.serialization.workspaces.save(workspace)) }
function expression(block: Blockly.Block | null | undefined, resolvePLC: (plcID: string, variable: string) => unknown): unknown {
  if (!block) return ""
  if (block.type === "text") return block.getFieldValue("TEXT")
  if (block.type === "math_number") return Number(block.getFieldValue("NUM") ?? 0)
  if (block.type === "logic_boolean") return block.getFieldValue("BOOL") === "TRUE"
  if (block.type === "dashboard_plc_get") return resolvePLC(block.getFieldValue("PLC_ID"), block.getFieldValue("VARIABLE"))
  if (block.type === "dashboard_to_number") return Number(expression(block.getInput("VALUE")?.connection?.targetBlock(), resolvePLC))
  if (block.type === "dashboard_to_string") return String(expression(block.getInput("VALUE")?.connection?.targetBlock(), resolvePLC))
  const target = (...names: string[]) => names.map((name) => block.getInput(name)?.connection?.targetBlock()).find(Boolean)
  if (block.type === "dashboard_math" || block.type === "math_arithmetic") { const left = Number(expression(target("LEFT", "A"), resolvePLC)); const right = Number(expression(target("RIGHT", "B"), resolvePLC)); const op = block.getFieldValue("OP") ?? block.getFieldValue("OPERATOR"); return op === "-" ? left - right : op === "*" ? left * right : op === "/" ? (right === 0 ? 0 : left / right) : left + right }
  if (block.type === "dashboard_compare" || block.type === "logic_compare") { const left = expression(target("LEFT", "A"), resolvePLC); const right = expression(target("RIGHT", "B"), resolvePLC); const op = block.getFieldValue("OP") ?? "=="; return op === "!=" ? left !== right : op === ">" ? Number(left) > Number(right) : op === "<" ? Number(left) < Number(right) : op === ">=" ? Number(left) >= Number(right) : op === "<=" ? Number(left) <= Number(right) : left === right }
  return ""
}
export function runDashboardBlocks(workspace: Blockly.WorkspaceSvg, setValue: (widgetID: string, type: string, value: string) => void, resolvePLC: (plcID: string, variable: string) => unknown = () => "") {
  let block: Blockly.Block | null | undefined = workspace.getTopBlocks(true).find((item) => item.type === "dashboard_start"); const visited = new Set<string>()
  while (block && !visited.has(block.id)) { visited.add(block.id); const widgetID = block.getFieldValue("WIDGET"); if (block.type === "dashboard_set_text") setValue(widgetID, "text", String(expression(block.getInput("VALUE")?.connection?.targetBlock(), resolvePLC))); if (block.type === "dashboard_set_image") setValue(widgetID, "image", String(expression(block.getInput("VALUE")?.connection?.targetBlock(), resolvePLC))); block = block.getNextBlock() }
}
