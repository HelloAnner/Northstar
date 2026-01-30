# Northstar V3 设计总结

## 用户需求回顾

### 核心诉求

1. **从多项目制改为单项目制**
   - 不再维护多个项目
   - 系统启动后直接是仪表盘
   - 简化用户操作流程

2. **使用文件数据库（SQLite）存储在 data/ 下**
   - 替代内存 + JSON 的不稳定方案
   - 预设计好表结构，Excel 只是数据来源
   - 维护一套稳定的数据

3. **数据导入自动化**
   - 解析 Excel 数据到数据库
   - 自动识别 Sheet 类型和字段映射
   - 不需要手动字段映射
   - 一个进度页面完成导入

4. **月份灵活性** ✨ **（核心需求）**
   - 现在的数据是 12 月，后面可能是 1 月、2 月等
   - 表结构需要支持不同月份的数据
   - Excel 文件的全部数据都可以解析到表里面操作
   - Excel 可能还有其他 sheet

5. **保留核心业务逻辑**
   - 联动计算（DAG）
   - 微调功能
   - 字段业务含义
   - 指标计算

---

## 设计方案对应

### ✅ 1. 单项目制 (01_database.md + 03_frontend.md)

**实现**:
- 移除 `projects` 表和相关代码
- `wholesale_retail` 和 `accommodation_catering` 表直接存储企业数据
- 前端直接进入 Dashboard，无项目列表页面

### ✅ 2. SQLite 文件数据库 (01_database.md)

**实现**:
- 数据库文件: `data/northstar.db`
- 7 张表设计完成:
  - `wholesale_retail` - 批发零售主表
  - `accommodation_catering` - 住宿餐饮主表
  - `wr_snapshot` - 批零历史快照
  - `ac_snapshot` - 住餐历史快照
  - `sheets_meta` - Sheet 元信息
  - `config` - 系统配置
  - `import_logs` - 导入日志
- 完整的索引设计，保证查询性能

### ✅ 3. 自动化导入 (02_excel_parser.md + 04_backend.md)

**实现**:
- **Sheet 自动识别**: 基于特征字段匹配，支持批零/住餐/快照/汇总等多种类型
- **字段自动映射**: 正则表达式模糊匹配，支持多种列名格式
- **动态年月识别**: 从列名和 Sheet 名提取年月信息
- **SSE 流式反馈**: 实时显示导入进度和日志
- **导入流程简化**:
  ```
  点击"导入数据" → 选择 Excel → 自动解析 → 显示结果 → 完成
  ```

### ✅ 4. 月份灵活性 (01_database.md + 02_excel_parser.md) ⭐

**核心设计**:

#### 4.1 表结构支持月份灵活性

```sql
-- 主表添加月份标识
CREATE TABLE wholesale_retail (
    data_year INTEGER NOT NULL,   -- 数据年份
    data_month INTEGER NOT NULL,  -- 数据月份

    -- 字段使用相对时间概念
    sales_current_month REAL,     -- 本月销售额
    sales_prev_month REAL,        -- 上月销售额
    sales_last_year_month REAL,   -- 去年同期
    sales_current_cumulative REAL,-- 本年累计
    -- ...
);

-- 快照表存储历史数据
CREATE TABLE wr_snapshot (
    snapshot_year INTEGER,   -- 快照年份
    snapshot_month INTEGER,  -- 快照月份
    -- ...
);
```

#### 4.2 智能年月识别

```go
// 从列名提取年月: "2026年1月销售额" → year=2026, month=1
func extractYearMonth(columnName string) (year, month int, found bool)

// 判断字段时间类型
func inferFieldTimeType(columnName string, currentYear, currentMonth int) FieldTimeType

// 动态字段映射
func mapColumnToDBField(columnName string, currentYear, currentMonth int) string
```

#### 4.3 支持任意月份

| Excel | 12月数据 | 1月数据 | 2月数据 |
|-------|---------|--------|--------|
| 列名示例 | 2025年12月销售额 | 2026年1月销售额 | 2026年2月销售额 |
| data_year | 2025 | 2026 | 2026 |
| data_month | 12 | 1 | 2 |
| sales_current_month | 12月销售额 | 1月销售额 | 2月销售额 |
| sales_prev_month | 11月销售额 | 12月销售额 | 1月销售额 |

#### 4.4 处理特殊情况

- **1月数据**: `sales_prev_month` = 上年12月，`sales_prev_cumulative` 留空
- **跨年累计**: 自动识别年份边界
- **历史快照**: 每个 sheet 独立识别年月，存入 snapshot 表

### ✅ 5. 全量数据解析 (02_excel_parser.md)

**实现**:
- **遍历所有 Sheet**: 不跳过任何 sheet
- **识别并分类**:
  - 主表 (批发/零售/住宿/餐饮) → `wholesale_retail` / `accommodation_catering`
  - 历史快照 (如 "2024年12月批零") → `wr_snapshot` / `ac_snapshot`
  - 汇总表 (限上零售额/小微/吃穿用) → 记录到 `sheets_meta`，暂时跳过
  - 未知类型 → 记录到 `sheets_meta`，标记为 unknown
- **元信息记录**: `sheets_meta` 表记录每个 sheet 的类型、行数、列数、识别置信度

### ✅ 6. 保留核心业务逻辑 (04_backend.md + 05_dag.md)

**实现**:
- **DAG 计算引擎**: `internal/service/calculator/engine.go`
  - 全量计算: 导入后计算所有衍生字段和指标
  - 增量计算: 修改单个企业后只重算受影响的部分
- **字段联动**: 参考 `prd/05_字段联动DAG.md` 的依赖关系
- **微调功能**: API 支持修改企业数据，自动触发联动计算
- **指标计算**: 16 项指标实时计算并返回

---

## 文档清单

| 文档 | 说明 | 状态 |
|------|------|------|
| `README.md` | 总览、架构图、变更清单、实现路径 | ✅ 已完成 |
| `00_SUMMARY.md` | 本文档 - 设计总结 | ✅ 已完成 |
| `01_database.md` | 数据库表结构设计 + 月份灵活性设计 + 补充字段 | ✅ 已完成 |
| `02_excel_parser.md` | Excel 解析策略 + 动态年月识别 | ✅ 已完成 |
| `03_frontend.md` | 前端页面和组件设计 | ✅ 已完成 |
| `04_backend.md` | 后端 API 设计 + 核心逻辑实现 | ✅ 已完成 |
| `05_migration.md` | 从 V2 迁移指南 | ⏳ 待创建 |
| `06_field_coverage_check.md` | 字段覆盖度检查报告 | ✅ 已完成 |
| `07_validation_report.md` | 表结构完整性验证报告 | ✅ 已完成 |

---

## 核心技术亮点

### 1. 智能 Sheet 识别

通过特征字段匹配自动识别 Sheet 类型，支持：
- 批发/零售主表识别
- 住宿/餐饮主表识别
- 历史快照识别（任意年月）
- 汇总表识别
- 置信度评分机制

### 2. 动态字段映射

不依赖硬编码的列名，通过正则表达式和年月推断实现：
- 支持多种列名格式（"2026年1月销售额" / "销售额;2026年1月"）
- 自动判断字段时间类型（当月/上月/去年同期/累计）
- 优先级机制处理多列匹配
- 容错性强，适应各种 Excel 格式变化

### 3. 月份自适应存储

主表字段使用相对时间概念，配合 `data_year`/`data_month` 标识：
- 同一张表可以存储不同月份的数据
- 查询时根据 `data_year`/`data_month` 过滤
- 支持多月份数据对比分析
- 配置表记录当前操作月份

### 4. DAG 增量计算

修改企业数据后，只重算受影响的指标：
```
修改 retail_current_month
  ↓ 追踪依赖关系
  ↓ 只重算 4 个受影响的指标
  ↓ 返回更新后的指标列表
  ↓ 前端实时更新卡片
```

性能优化：
- 避免全量重算
- 批量更新时合并计算
- 结果缓存

---

## 与用户需求的对应关系

| 用户需求 | 设计方案 | 文档位置 |
|---------|---------|---------|
| 单项目制 | 移除项目概念，直接进入仪表板 | README.md, 03_frontend.md |
| 文件数据库 | SQLite 存储在 data/northstar.db | 01_database.md |
| 自动解析 | Sheet 识别 + 字段映射自动化 | 02_excel_parser.md |
| 月份灵活 | data_year/data_month + 动态识别 | 01_database.md, 02_excel_parser.md |
| 全量解析 | 遍历所有 Sheet，快照表存历史 | 02_excel_parser.md |
| 保留逻辑 | DAG 计算引擎，联动微调 | 04_backend.md |

---

## 下一步

### 开发优先级

1. **Phase 1**: 数据库 + Store 层（基础）
2. **Phase 2**: Excel 解析器（核心）
3. **Phase 3**: 导入流程 + 计算引擎（核心）
4. **Phase 4**: 前端改造（用户体验）
5. **Phase 5**: 导出 + 测试（完整闭环）

### 需要确认的点

1. **商品分类字段**: 粮油食品类、饮料类等 6 个分类字段是否需要全部导入？
2. **吃穿用标记**: 自动判定逻辑是否已明确？（根据行业代码/商品分类？）
3. **导出模板**: 定稿 Excel 的 11 个 Sheet 模板是否已提供？
4. **测试数据**: 是否有 1月、2月 等其他月份的测试 Excel 样本？

---

## 总结

V3 设计方案已经完整覆盖了用户的核心需求，特别是**月份灵活性**这个关键点通过以下机制实现：

1. **表结构灵活**: `data_year`/`data_month` 标识 + 相对时间字段
2. **解析器智能**: 自动提取年月信息并动态映射
3. **全量存储**: 主表存当前数据，快照表存历史数据
4. **配置管理**: 系统记录当前操作月份

这样的设计既保证了对任意月份数据的支持，又保持了数据结构的简洁性和查询性能。

**方案设计完成度**: 95%

**已完成**:
- ✅ 数据库表结构设计（7 张表）
- ✅ Excel 解析策略（Sheet 识别 + 字段映射）
- ✅ 前端页面设计（Dashboard + ImportModal）
- ✅ 后端 API 设计（7 个核心接口）
- ✅ 字段覆盖度验证（输入/输出完全兼容）
- ✅ 添加 5 个补充字段（输出定稿所需）

**待补充**:
- ⏳ 迁移指南 (05_migration.md) - 如何从 V2 平滑过渡到 V3
- ⏳ 更详细的测试用例文档

---

## 表结构验证结果 ✅

**日期**: 2026-01-30

### 输入兼容性: 100% ✅
- 批零主表: 32/32 字段完全覆盖
- 住餐主表: 34/34 字段完全覆盖
- 历史快照: 完全支持

### 输出兼容性: 100% ✅
- 批零总表: 17/17 列（含 IP 字段）
- 住餐总表: 21/21 列（含 IP 字段）
- 吃穿用: 26/26 列（含网络销售额、开业时间）
- 小微: 6/6 列
- 其他 Sheet: 全部支持

### 新增补充字段
为满足输出定稿需求，新增 5 个字段：

| 字段 | 类型 | 用途 | 表 |
|------|------|------|-----|
| `first_report_ip` | TEXT | 第一次上报的IP | wholesale_retail, accommodation_catering |
| `fill_ip` | TEXT | 填报IP | wholesale_retail, accommodation_catering |
| `network_sales` | REAL | 网络销售额 | wholesale_retail, accommodation_catering |
| `opening_year` | INTEGER | 开业年份 | wholesale_retail, accommodation_catering |
| `opening_month` | INTEGER | 开业月份 | wholesale_retail, accommodation_catering |

详见: `specs/003/06_field_coverage_check.md` 和 `specs/003/07_validation_report.md`
