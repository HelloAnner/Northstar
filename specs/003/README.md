# Northstar V3 - 简化架构设计

## 变更概述

| 维度 | V2 (当前) | V3 (目标) |
|------|----------|----------|
| 项目管理 | 多项目制 | 单项目（无项目概念） |
| 数据存储 | 内存 + JSON 持久化 | SQLite 文件数据库 |
| 导入流程 | 4步向导 + 字段映射 | 一键导入 + 自动解析 |
| 首页 | ProjectHub 项目列表 | Dashboard 仪表板 |
| 复杂度 | 高（多项目、多 Sheet 映射） | 低（单数据源、自动识别） |

## 核心设计原则

1. **数据库优先**：预设计好表结构，Excel 只是数据来源
2. **自动解析**：智能识别 Excel 字段，无需用户手动映射
3. **单一入口**：启动即仪表板，右上角导入数据
4. **稳定可控**：数据库结构固定，业务逻辑清晰

## 目录结构

```
specs/003/
├── README.md             # 本文件 - 总览
├── 00_SUMMARY.md         # 设计总结
├── 01_database.md        # 数据库表结构设计（含补充字段）
├── 02_excel_parser.md    # Excel 自动解析策略
├── 03_frontend.md        # 前端页面设计
├── 04_backend.md         # 后端 API 设计
├── 05_migration.md       # 从 V2 迁移指南 (待创建)
├── 06_field_coverage_check.md  # 字段覆盖度检查报告
└── 07_validation_report.md     # 表结构完整性验证报告 ✅
```

## 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端 (React)                             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Topbar     │  │ ImportModal │  │      Dashboard          │  │
│  │  - 导入按钮 │  │ - 进度显示  │  │  - 指标卡片 (16项)      │  │
│  │  - 导出按钮 │  │ - 日志输出  │  │  - 企业数据表格         │  │
│  │  - 设置     │  │             │  │  - 筛选/排序/编辑       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         后端 (Go)                                │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Handlers   │  │  Services   │  │      Calculator         │  │
│  │  - Import   │  │  - Parser   │  │  - Engine (DAG)         │  │
│  │  - Company  │  │  - Store    │  │  - 16项指标计算         │  │
│  │  - Export   │  │  - Config   │  │  - 增量更新             │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     SQLite (data/northstar.db)                   │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  companies  │  │   config    │  │     import_logs         │  │
│  │  (企业数据) │  │  (系统配置) │  │    (导入历史)           │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## 用户流程

### 首次使用
```
启动应用 → 仪表板(空数据提示) → 点击"导入数据" → 选择Excel文件
    → 自动解析进度 → 完成 → 仪表板显示数据
```

### 日常使用
```
启动应用 → 仪表板(显示已有数据) → 查看/编辑企业数据 → 指标实时联动
    → 导出定稿Excel
```

### 数据更新
```
仪表板 → 点击"导入数据" → 选择新Excel → 自动比对更新 → 完成
```

## 技术选型

| 组件 | 选型 | 理由 |
|------|------|------|
| 数据库 | SQLite | 文件数据库、无需安装、Go 原生支持 |
| ORM | go-sqlite3 + 原生 SQL | 轻量、性能好 |
| 前端 | React + shadcn/ui | 现有基础、无需迁移 |
| Excel | excelize | 现有基础、功能完备 |

## 关键变更清单

### 后端
- [ ] 移除 `internal/model/project.go`
- [ ] 新增 `internal/store/sqlite.go` - SQLite 存储层
- [ ] 重构 `internal/service/excel/parser.go` - 自动解析逻辑 + 月份灵活识别
- [ ] 新增 `internal/service/excel/recognizer.go` - Sheet 类型自动识别
- [ ] 新增 `internal/service/excel/mapper.go` - 动态字段映射
- [ ] 新增 `internal/service/calculator/engine.go` - DAG 计算引擎
- [ ] 简化 `internal/server/handlers/import.go` - 一键导入 SSE 流式响应
- [ ] 新增 `internal/server/handlers/indicators.go` - 指标查询 API
- [ ] 新增 `internal/server/handlers/companies.go` - 企业数据 CRUD API
- [ ] 新增 `internal/server/handlers/config.go` - 配置管理 API
- [ ] 移除项目相关 API

### 前端
- [ ] 移除 `pages/ProjectHub.tsx`
- [ ] 移除 `pages/ImportWizard.tsx` (4步向导)
- [ ] 新增 `components/ImportModal.tsx` - 简单导入对话框
- [ ] 修改 `pages/Dashboard.tsx` - 作为首页，增加月份显示
- [ ] 新增 `components/IndicatorCards.tsx` - 16项指标卡片
- [ ] 新增 `components/CompanyTable.tsx` - 企业数据表格（可编辑）
- [ ] 新增 `components/EditableCell.tsx` - 可编辑单元格组件
- [ ] 新增 `components/FilterBar.tsx` - 筛选栏
- [ ] 新增 `components/ConfigPanel.tsx` - 配置面板
- [ ] 移除 `store/projectStore.ts`
- [ ] 新增 `store/dataStore.ts` - 统一数据状态管理

### 数据
- [ ] 新增 `data/` 目录
- [ ] 数据库文件 `data/northstar.db`
- [ ] 初始化数据库 schema（7张表）
- [ ] 移除 `data/projects/` 目录结构

---

## 核心改进点

### 1. 月份灵活性 ✨

**问题**: V2 硬编码 12 月数据，无法处理 1 月、2 月等其他月份

**解决**:
- 主表添加 `data_year` 和 `data_month` 字段标识数据月份
- 字段名使用相对时间概念（当月、上月、去年同期）
- 解析器自动从列名和 Sheet 名提取年月信息
- 配置表记录当前操作月份

**示例**:
```sql
-- 12月数据
INSERT INTO wholesale_retail (data_year, data_month, sales_current_month, ...)
VALUES (2025, 12, 1234.56, ...);

-- 1月数据
INSERT INTO wholesale_retail (data_year, data_month, sales_current_month, ...)
VALUES (2026, 1, 1300.00, ...);
```

### 2. 全量数据解析 📊

**问题**: V2 只解析映射字段，Excel 中的历史数据和其他 Sheet 被忽略

**解决**:
- 遍历所有 Sheet，自动识别类型（批零/住餐/快照/汇总）
- 快照表存储历史月份数据（如 "2024年12月批零"）
- sheets_meta 表记录所有 Sheet 的元信息
- 支持未来新增 Sheet 类型的扩展

**识别逻辑**:
```go
// 自动识别 Sheet 类型
func (r *Recognizer) Recognize(f *excelize.File, sheetName string) (string, float64) {
    header := f.GetRows(sheetName)[0]

    // 特征匹配
    wrScore := matchWRFields(header)      // 批零特征得分
    acScore := matchACFields(header)      // 住餐特征得分
    snapshotScore := matchSnapshot(header) // 快照特征得分

    // 返回最高得分的类型
    if wrScore > 0.8 {
        return "wholesale_retail", wrScore
    }
    // ...
}
```

### 3. 智能字段映射 🧠

**问题**: V2 需要用户手动映射字段，容错性差

**解决**:
- 基于正则表达式的模糊匹配
- 动态识别列名中的年月信息
- 根据年月关系自动判断字段类型（当月/上月/去年同期/累计）
- 优先级机制处理多列匹配同一字段的情况

**映射示例**:
```
Excel 列名: "2026年1月销售额"
  ↓ 提取年月: year=2026, month=1
  ↓ 判断: 当前数据月份 = 2026年1月
  ↓ 结论: 本月数据
  ↓ 映射: sales_current_month

Excel 列名: "2025年12月销售额"
  ↓ 提取年月: year=2025, month=12
  ↓ 判断: 当前月份的上一个月
  ↓ 结论: 上月数据
  ↓ 映射: sales_prev_month
```

### 4. 实时联动计算 ⚡

**问题**: V2 修改数据后需要手动刷新指标

**解决**:
- DAG 计算引擎追踪字段依赖关系
- 修改企业数据后自动触发增量计算
- 只重算受影响的指标，提升性能
- API 返回更新后的指标列表，前端实时更新

**DAG 示例**:
```
修改企业 A 的 retail_current_month
  ↓
1. 重算企业 A 的 retail_month_rate
  ↓
2. 重算零售业的 sum_retail_current
  ↓
3. 重算限上社零额 limit_above_retail_month
  ↓
4. 重算社零总额 total_retail_cumulative
  ↓
返回: {
  "limit_above_retail_month": 125678.90,
  "total_retail_cumulative": 240000.00
}
```

### 5. 简化用户流程 🚀

**V2 流程 (复杂)**:
```
启动 → 项目列表 → 创建项目 → 4步导入向导
  → 选择文件 → 识别Sheet → 映射字段 → 确认导入
  → 返回项目列表 → 点击项目 → 进入仪表板
```

**V3 流程 (简化)**:
```
启动 → 仪表板 → 点击"导入数据" → 选择文件 → 自动解析 → 完成
```

---

## 实现路径

### 阶段 1: 数据库 + 基础 API (3-5 天)

1. 创建 SQLite 数据库和表结构
2. 实现 Store 层（数据库操作）
3. 实现基础 API（状态、配置查询）
4. 编写单元测试

### 阶段 2: Excel 解析器 (5-7 天)

1. 实现 Sheet 类型识别器
2. 实现动态字段映射器
3. 实现批零/住餐主表解析
4. 实现历史快照解析
5. 编写解析器测试（多个月份的 Excel 样本）

### 阶段 3: 导入流程 + DAG 计算 (4-6 天)

1. 实现导入协调器
2. 实现 SSE 流式进度反馈
3. 实现 DAG 计算引擎（全量 + 增量）
4. 实现企业数据修改 API
5. 测试联动计算准确性

### 阶段 4: 前端改造 (5-7 天)

1. 移除项目管理相关页面和组件
2. 实现新的 Dashboard 首页
3. 实现 ImportModal 导入对话框
4. 实现 IndicatorCards 指标卡片
5. 实现 CompanyTable 可编辑表格
6. 实现 ConfigPanel 配置面板
7. 集成 API 和状态管理

### 阶段 5: 导出 + 测试 (3-5 天)

1. 实现定稿 Excel 导出功能
2. 端到端测试（导入-修改-导出）
3. 测试多月份数据切换
4. 性能优化
5. 文档完善

**总计**: 约 20-30 天

---
