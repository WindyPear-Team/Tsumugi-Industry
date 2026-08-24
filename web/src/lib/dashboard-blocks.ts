import * as Blockly from "blockly"
import "blockly/blocks"

let widgetOptions: [string, string][] = [["请先添加组件", ""]]

export function setDashboardWidgetOptions(options: [string, string][]) { widgetOptions = options.length ? options : [["请先添加组件", ""]] }
function widgets() { return widgetOptions }

export function defineDashboardBlocks() {
  if (Blockly.Blocks.dashboard_start) return
  Blockly.common.defineBlocks({
    dashboard_start: { init(this: Blockly.Block) { this.appendDummyInput().appendField("开始"); this.setNextStatement(true); this.setColour(210) } },
    dashboard_end: { init(this: Blockly.Block) { this.appendDummyInput().appendField("结束"); this.setPreviousStatement(true); this.setColour(210) } },
    dashboard_set_text: { init(this: Blockly.Block) { this.appendDummyInput().appendField("修改文字").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendDummyInput().appendField(new Blockly.FieldTextInput("文字内容"), "VALUE"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(160) } },
    dashboard_set_image: { init(this: Blockly.Block) { this.appendDummyInput().appendField("修改图片").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendDummyInput().appendField(new Blockly.FieldTextInput("图片地址"), "VALUE"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(160) } },
    dashboard_set_variable: { init(this: Blockly.Block) { this.appendDummyInput().appendField("显示变量").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendDummyInput().appendField("变量名").appendField(new Blockly.FieldTextInput("变量"), "VARIABLE"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(120) } },
    dashboard_set_status: { init(this: Blockly.Block) { this.appendDummyInput().appendField("显示状态").appendField(new Blockly.FieldDropdown(() => widgets()), "WIDGET"); this.appendDummyInput().appendField(new Blockly.FieldDropdown([["运行中", "running"], ["空闲中", "idle"], ["报警", "alarm"]]), "STATUS"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(45) } },
  })
}

export function dashboardToolbox() { return { kind: "categoryToolbox", contents: [{ kind: "category", name: "流程", colour: "210", contents: [{ kind: "block", type: "dashboard_start" }, { kind: "block", type: "dashboard_end" }] }, { kind: "category", name: "修改组件", colour: "160", contents: [{ kind: "block", type: "dashboard_set_text" }, { kind: "block", type: "dashboard_set_image" }, { kind: "block", type: "dashboard_set_variable" }, { kind: "block", type: "dashboard_set_status" }] }] } }

export function loadDashboardDefinition(workspace: Blockly.WorkspaceSvg, definition?: string) {
  workspace.clear()
  if (!definition) return
  try { Blockly.serialization.workspaces.load(JSON.parse(definition) as never, workspace) } catch { /* ignore malformed old definitions */ }
}

export function saveDashboardDefinition(workspace: Blockly.WorkspaceSvg) { return JSON.stringify(Blockly.serialization.workspaces.save(workspace)) }

export function runDashboardBlocks(workspace: Blockly.WorkspaceSvg, setValue: (widgetID: string, type: string, value: string) => void) {
  let block: Blockly.Block | null | undefined = workspace.getTopBlocks(true).find((item) => item.type === "dashboard_start")
  const visited = new Set<string>()
  while (block && !visited.has(block.id)) {
    visited.add(block.id)
    const widgetID = block.getFieldValue("WIDGET")
    if (block.type === "dashboard_set_text") setValue(widgetID, "text", block.getFieldValue("VALUE"))
    if (block.type === "dashboard_set_image") setValue(widgetID, "image", block.getFieldValue("VALUE"))
    if (block.type === "dashboard_set_variable") setValue(widgetID, "variable", block.getFieldValue("VARIABLE"))
    if (block.type === "dashboard_set_status") setValue(widgetID, "status", block.getFieldValue("STATUS"))
    block = block.getNextBlock()
  }
}
