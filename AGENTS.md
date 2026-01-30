# Northstar 项目 Agent 指南

## 项目概述

Northstar 是一个经济数据统计系统，用于处理批发零售、住宿餐饮四大行业的月度数据。

**核心功能**:
1. Excel 数据导入（预估表）
2. 企业数据管理和微调
3. 指标联动计算（16 项指标）
4. Excel 导出（定稿表）

---

## 架构版本

### V2 (当前)
- 多项目制
- 内存 + JSON 持久化
- 4 步导入向导 + 手动字段映射
- 月份硬编码（仅支持 12 月）

### V3 (目标 - specs/003/)
- 单项目制
- SQLite 文件数据库
- 一键自动导入
- **月份灵活支持（核心改进）**

---

## 关键设计决策

### 1. 月份灵活性 ⭐ (2026-01-30 设计)

**问题**: 用户的 Excel 数据月份是动态的（12月、1月、2月...），V2 硬编码了 12 月，无法适应

**解决方案**:

#### 数据库设计
```sql
-- 主表添加月份标识
CREATE TABLE wholesale_retail (
    data_year INTEGER NOT NULL,   -- 数据年份 (如 2026)
    data_month INTEGER NOT NULL,  -- 数据月份 (如 1)

    -- 字段使用相对时间概念
    sales_current_month REAL,     -- 本月销售额 (如 2026年1月)
    sales_prev_month REAL,        -- 上月销售额 (如 2025年12月)
    sales_last_year_month REAL,   -- 去年同期 (如 2025年1月)
    sales_current_cumulative REAL,-- 本年累计 (如 2026年1-1月)
    -- ...
);
```

**关键字段说明**:
- `sales_current_month`: 不是"12月销售额"，而是"本月销售额"
- `data_year`, `data_month`: 标识这批数据对应的年月
- 字段名使用相对概念，通过 `data_year`/`data_month` 确定具体月份

#### Excel 解析策略

```go
// 1. 从列名提取年月
"2026年1月销售额" → year=2026, month=1

// 2. 确定当前数据月份
dataYear, dataMonth = findMaxYearMonth(allColumns)

// 3. 判断字段时间类型
if year == dataYear && month == dataMonth:
    → CurrentMonth (本月)
else if year == dataYear && month == dataMonth-1:
    → PrevMonth (上月)
else if year == dataYear-1 && month == dataMonth:
    → LastYearMonth (去年同期)

// 4. 映射到数据库字段
CurrentMonth + "销售额" → sales_current_month
```

#### 查询数据

```sql
-- 查询 1 月数据
SELECT * FROM wholesale_retail
WHERE data_year = 2026 AND data_month = 1;

-- 查询 12 月数据
SELECT * FROM wholesale_retail
WHERE data_year = 2025 AND data_month = 12;
```

### 2. 全量数据解析 (2026-01-30 设计)

**问题**: Excel 中有大量历史 Sheet（如 "2024年12月批零"、"2025年11月批零"），V2 只解析主表

**解决方案**:
- 遍历所有 Sheet
- 自动识别类型：主表 / 历史快照 / 汇总表
- 主表数据 → `wholesale_retail` / `accommodation_catering`
- 历史快照 → `wr_snapshot` / `ac_snapshot`
- 所有 Sheet 元信息 → `sheets_meta`

### 3. 自动字段映射 (2026-01-30 设计)

**问题**: V2 需要用户手动映射字段，操作繁琐且容易出错

**解决方案**:
- 正则表达式模糊匹配列名
- 动态识别年月信息
- 自动推断字段时间类型
- 优先级机制处理重复匹配

---

## 代码规范要点

### Java 代码
- public 函数必须注释
- 单个 class 不超过 300 行
- 单个函数不超过 30 行
- 禁止全限定名，使用 import
- 单行 if/for 也需要大括号
- 注释优先使用中文

### 日志
- 优先使用英文
- 结构化日志格式

### 对话
- 优先使用中文与用户沟通

---

## 常见问题

### Q1: 为什么不用时间序列表？

**候选方案**:
```sql
-- 方案 A: 时间序列表
CREATE TABLE company_metrics (
    company_id INT,
    metric_type VARCHAR, -- sales/retail
    year INT,
    month INT,
    value REAL
);
```

**不采用原因**:
- 查询复杂度高（需要多次 JOIN）
- 不符合业务模型（用户操作的是"当前月份"的数据）
- 计算联动时效率低

**当前方案优势**:
- 查询简单（WHERE data_year=X AND data_month=Y）
- 符合业务语义（"当前工作数据"概念明确）
- 计算联动效率高

### Q2: 如何处理 1 月的特殊情况？

1 月数据的特点：
- `sales_prev_month`: 上年 12 月 ✅
- `sales_prev_cumulative`: 无意义（1月无"本年累计到上月"）→ 设为 0 或 NULL
- `sales_current_cumulative`: 等于 `sales_current_month` ✅

解析器会自动识别 `data_month=1` 并正确处理。

### Q3: 能否同时存储多个月份的数据？

可以！主表支持存储多个月份：
```sql
-- 12 月数据
INSERT INTO wholesale_retail (data_year, data_month, ...) VALUES (2025, 12, ...);

-- 1 月数据
INSERT INTO wholesale_retail (data_year, data_month, ...) VALUES (2026, 1, ...);

-- 查询时按 data_year/data_month 过滤
```

但仪表盘默认只显示当前操作月份（`config` 表中的 `current_year`/`current_month`）。

---

## 下次 Session 注意事项

1. **月份灵活性是核心需求**
   - 所有设计必须支持任意月份（1-12月）
   - 字段名不能硬编码具体月份
   - 解析器必须从列名动态提取年月

2. **全量数据解析**
   - 不能遗漏任何 Sheet
   - 无法识别的 Sheet 记录到 `sheets_meta` 并标记为 unknown

3. **数据库设计已完成**
   - 参考 `specs/003/01_database.md`
   - 7 张表，索引完整
   - 支持月份灵活性

4. **Excel 解析策略已完成**
   - 参考 `specs/003/02_excel_parser.md`
   - Sheet 识别、字段映射、年月推断逻辑完整

5. **实现时优先顺序**
   - 数据库 + Store 层
   - Excel 解析器（核心）
   - 导入流程 + 计算引擎
   - 前端改造
   - 导出功能

---

## 相关文档

| 文档 | 说明 |
|------|------|
| `prd/` | 业务需求文档 |
| `specs/003/` | V3 设计方案 |
| `specs/003/00_SUMMARY.md` | 设计总结 |
| `specs/003/01_database.md` | 数据库设计 |
| `specs/003/02_excel_parser.md` | Excel 解析策略 |
| `specs/003/03_frontend.md` | 前端设计 |
| `specs/003/04_backend.md` | 后端 API 设计 |

---

## 更新日志

- 2026-01-30: 完成 V3 设计方案，重点解决月份灵活性问题
- 2026-01-30: 字段覆盖度检查，发现并添加 5 个输出定稿所需字段
  - `first_report_ip` - 第一次上报的IP
  - `fill_ip` - 填报IP
  - `network_sales` - 网络销售额
  - `opening_year` - 开业年份
  - `opening_month` - 开业月份
