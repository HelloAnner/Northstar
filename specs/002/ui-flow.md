# 多项目 UI 流程设计（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- 让“项目管理 + 数据导入”成为用户进入主 Dashboard 前的必经路径（首次/无项目时）
- 保持现有主 Dashboard 视觉与交互尽量不动，仅在顶部增加“项目切换入口”

## 路由/上下文方案（3 选 1）

### 方案 1（推荐）：后端维护 activeProject（最小改动）

- 新增项目相关 API：list/create/select/current
- 现有业务 API（/companies、/config、/indicators、/import/**）默认作用于当前 activeProject
- 前端切换项目：调用 `select` 后刷新数据（复用现有 store 的 `fetch*`）

优点：改动面最小、不会污染现有 API；缺点：API 语义隐含“当前项目”，需要在 UI 明确展示当前项目名，避免误操作。

### 方案 2：projectId 显式入路由/接口（更规范，改动更大）

- 所有 API/路由加 `projectId`：如 `/projects/:id/companies`
- 优点：语义清晰；缺点：现有前后端改动多，成本高。

### 方案 3：纯前端 localStorage（不推荐）

- 仅前端记录 projectId，后端仍单 store
- 无法满足“每项目独立持久化”的根需求

## 页面流转（推荐方案 1）

1. 启动应用：
   - **总是先展示 `Project Hub`**（项目列表 + 新建项目 + 进入导入）
   - 在 Hub 中选择“进入项目”后再进入 Dashboard（顶部显示项目名 + 切换入口）
2. Project Hub：
   - 新建项目：输入项目名 → 创建成功后进入 Import Wizard（绑定该项目）
   - 选择已有项目：
     - 若该项目缺少 `latest.xlsx/state.json`（hasData=false）→ 点击“进入”**直接进入 Import Wizard**
     - 否则点击“进入”进入 Dashboard
3. Dashboard：
   - 顶部新增“项目下拉/弹窗”：可快速切换项目、进入 Project Hub、创建新项目

---

已确认：启动后总是先显示 `Project Hub`。
