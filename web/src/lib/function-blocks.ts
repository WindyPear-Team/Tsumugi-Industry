import * as Blockly from "blockly"
import "blockly/blocks"

export function defineFunctionBlocks() {
  if (Blockly.Blocks.function_start) return
  Blockly.common.defineBlocks({
    function_start: { init(this: Blockly.Block) { this.appendDummyInput().appendField("开始"); this.setNextStatement(true); this.setColour(210) } },
    function_end: { init(this: Blockly.Block) { this.appendDummyInput().appendField("结束"); this.setPreviousStatement(true); this.setColour(210) } },
    function_param_get: { init(this: Blockly.Block) { this.appendDummyInput().appendField("读取参数").appendField(new Blockly.FieldVariable("参数"), "VARIABLE"); this.setOutput(true); this.setColour(330); this.setTooltip("读取函数调用时传入的参数值") } },
    function_set: { init(this: Blockly.Block) { this.appendDummyInput().appendField("设置 PLC 变量").appendField(new Blockly.FieldTextInput("变量名"), "VARIABLE"); this.appendValueInput("VALUE").appendField("值"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(5) } },
    function_delay: { init(this: Blockly.Block) { this.appendDummyInput().appendField("延时").appendField(new Blockly.FieldNumber(1, 0), "SECONDS").appendField("秒"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(25) } },
    function_return: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("返回"); this.setPreviousStatement(true); this.setColour(290) } },
    function_if: { init(this: Blockly.Block) { this.appendValueInput("CONDITION").appendField("如果"); this.appendStatementInput("TRUE").appendField("满足"); this.appendStatementInput("FALSE").appendField("否则"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(195) } },
    function_to_number: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转数字"); this.setOutput(true, "Number"); this.setColour(45) } },
    function_to_string: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("转字符串"); this.setOutput(true, "String"); this.setColour(45) } },
  })
}

export function syncFunctionParameters(workspace: Blockly.WorkspaceSvg, parameters: string[]) {
  for (const name of parameters) {
    const trimmed = name.trim()
    if (trimmed && !workspace.getVariableMap().getVariable(trimmed)) workspace.getVariableMap().createVariable(trimmed)
  }
}

export function functionToolbox() {
  return { kind: "categoryToolbox", contents: [
    { kind: "category", name: "流程控制", colour: "210", contents: [{ kind: "block", type: "function_start" }, { kind: "block", type: "function_end" }, { kind: "block", type: "function_if" }, { kind: "block", type: "controls_repeat_ext" }, { kind: "block", type: "controls_whileUntil" }] },
    { kind: "category", name: "函数参数", colour: "330", contents: [{ kind: "block", type: "function_param_get" }, { kind: "block", type: "variables_get" }, { kind: "block", type: "variables_set" }] },
    { kind: "category", name: "设备动作", colour: "5", contents: [{ kind: "block", type: "function_set" }, { kind: "block", type: "function_delay" }] },
    { kind: "category", name: "函数结果", colour: "290", contents: [{ kind: "block", type: "function_return" }] },
    { kind: "category", name: "逻辑", colour: "210", contents: [{ kind: "block", type: "logic_compare" }, { kind: "block", type: "logic_operation" }, { kind: "block", type: "logic_negate" }, { kind: "block", type: "logic_boolean" }, { kind: "block", type: "logic_ternary" }] },
    { kind: "category", name: "数学", colour: "230", contents: [{ kind: "block", type: "math_number" }, { kind: "block", type: "math_arithmetic" }, { kind: "block", type: "math_single" }, { kind: "block", type: "math_round" }, { kind: "block", type: "math_modulo" }, { kind: "block", type: "math_constrain" }] },
    { kind: "category", name: "文本", colour: "160", contents: [{ kind: "block", type: "text" }, { kind: "block", type: "text_join" }, { kind: "block", type: "text_length" }, { kind: "block", type: "text_isEmpty" }, { kind: "block", type: "text_print" }] },
    { kind: "category", name: "列表", colour: "260", contents: [{ kind: "block", type: "lists_create_with" }, { kind: "block", type: "lists_getIndex" }, { kind: "block", type: "lists_setIndex" }, { kind: "block", type: "lists_length" }, { kind: "block", type: "lists_isEmpty" }] },
    { kind: "category", name: "转换", colour: "45", contents: [{ kind: "block", type: "function_to_number" }, { kind: "block", type: "function_to_string" }] },
  ] }
}
