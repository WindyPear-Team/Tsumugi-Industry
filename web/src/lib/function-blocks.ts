import * as Blockly from "blockly"
import "blockly/blocks"

export function defineFunctionBlocks() {
  if (Blockly.Blocks.function_start) return
  Blockly.common.defineBlocks({
    function_start: { init(this: Blockly.Block) { this.appendDummyInput().appendField("开始"); this.setNextStatement(true); this.setColour(210) } },
    function_end: { init(this: Blockly.Block) { this.appendDummyInput().appendField("结束"); this.setPreviousStatement(true); this.setColour(210) } },
    function_set: { init(this: Blockly.Block) { this.appendDummyInput().appendField("设置 PLC 变量").appendField(new Blockly.FieldTextInput("变量名"), "VARIABLE"); this.appendValueInput("VALUE").appendField("值"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(5) } },
    function_delay: { init(this: Blockly.Block) { this.appendDummyInput().appendField("延时").appendField(new Blockly.FieldNumber(1, 0), "SECONDS").appendField("秒"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(25) } },
    function_return: { init(this: Blockly.Block) { this.appendValueInput("VALUE").appendField("返回"); this.setPreviousStatement(true); this.setColour(290) } },
    function_if: { init(this: Blockly.Block) { this.appendValueInput("CONDITION").appendField("如果"); this.appendStatementInput("TRUE").appendField("满足"); this.appendStatementInput("FALSE").appendField("否则"); this.setPreviousStatement(true); this.setNextStatement(true); this.setColour(195) } },
  })
}

export function functionToolbox() { return { kind: "categoryToolbox", contents: [{ kind: "category", name: "流程", colour: "210", contents: [{ kind: "block", type: "function_start" }, { kind: "block", type: "function_end" }, { kind: "block", type: "function_if" }] }, { kind: "category", name: "设备动作", colour: "5", contents: [{ kind: "block", type: "function_set" }, { kind: "block", type: "function_delay" }] }, { kind: "category", name: "函数结果", colour: "290", contents: [{ kind: "block", type: "function_return" }, { kind: "block", type: "math_number" }, { kind: "block", type: "text" }] }] } }
