# Excel 导出一致性 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 导出 Excel 与仪表盘数据一致，且与模板公式逻辑一致；内嵌模板与 `prd/12月月报（定）(1).xlsx` 结构一致（数值样例清空）。

**Architecture:** 以 B 模板为权威来源，生成“公式解析文档 + 计算逻辑映射列表”，导出时仅写入输入单元格，公式交给 Excel 计算；必要时调整代码计算逻辑以与公式一致。

**Tech Stack:** Go（excelize）、Python3（openpyxl，仅用于一次性解析与模板清洗）。

---

说明：仓库规则禁止 git add/commit，本计划不包含提交步骤。

### Task 1: 公式解析与差异证据

**Files:**
- Create: `docs/excel/12月月报（定）_公式解析.md`
- Create: `docs/excel/12月月报（定）_输入单元格映射.md`

**Step 1: 生成公式清单（失败用例）**
- 期望：能列出 `社零额（定）` 的 54 条公式与 `汇总表（定）` 的 4 条公式。
- 失败条件：公式为空/数量不匹配。

**Step 2: 输出公式解析文档**
- 对每条公式列出：单元格、公式、引用单元格、业务含义（简述）。

**Step 3: 输出输入单元格映射文档**
- 按 sheet 列出所有“非公式且导出需填写”的单元格与其业务来源。

### Task 2: 内嵌模板与 B 同步

**Files:**
- Modify: `internal/reporttpl/embedded_template_data.go`
- (临时) Create: `tmp/template_cleaner.py`

**Step 1: 清洗 B 模板数值（失败用例）**
- 期望：清空样例数值与公式缓存值；保留标题、合并、样式、公式、以及模板索引依赖的行业码列（列 C）。
- 失败条件：行业码被清空/公式丢失。

**Step 2: 生成 gzip+base64 并替换内嵌模板**
- 将清洗后的 xlsx 写回 `embedded_template_data.go`。

**Step 3: 核验结构一致性**
- 对比 sheet 列表、维度与公式数量。

### Task 3: 公式优先与计算逻辑对齐

**Files:**
- Modify: `internal/exporter/extra_sheets.go`
- Modify: `internal/exporter/exporter.go`
- Create: `internal/exporter/template_logic.go`
- Modify: `internal/exporter/formula_preserve_test.go`
- Create: `internal/exporter/template_logic_test.go`

**Step 1: 先写失败测试**
- 汇总表中公式单元格应保持公式，不被导出逻辑覆盖。
- 输出单元格应使用“逻辑列表”驱动写入。

**Step 2: 引入模板逻辑列表**
- 建立 `TemplateCellLogic` 列表：sheet、cell、key、calc（中文描述）与取值函数。
- 先覆盖 `社零额（定）`、`汇总表（定）` 两个 sheet 的输入单元格。

**Step 3: 调整导出逻辑**
- 改为遍历逻辑列表写入输入单元格。
- 公式相关单元格只保留模板公式（不在代码中计算）。

**Step 4: 对齐汇总文案**
- 若模板存在 A11 公式，导出时令 X3 引用 A11（或直接读取 A11 公式）以避免手写拼接偏差。

### Task 4: 结果校验

**Files:**
- Modify: `internal/exporter/formula_preserve_test.go`
- Create: `internal/exporter/template_compare_test.go`

**Step 1: 新增一致性测试**
- 校验导出后 `社零额（定）` 与 `汇总表（定）` 公式仍存在。

**Step 2: 回归导出流程**
- 运行现有导出测试 + 新增测试。

