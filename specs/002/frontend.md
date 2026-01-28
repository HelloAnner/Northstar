# 前端页面与自动保存落地建议（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- 新增 `Project Hub` 作为启动首页，统一“创建/选择项目 + 进入导入/进入仪表盘”
- 不破坏现有 Dashboard 的视觉与核心交互，仅增加“当前项目展示 + 切换入口”
- 自动保存：用户每次改动数据后 1000ms 内写入 `state.json`（由后端保证），并在关键节点强制 flush

## 路由建议（与“启动必进 Hub”一致）

- `/`：Project Hub（项目列表 + 新建项目）
- `/import`：Import Wizard（沿用现有导入向导 UI）
- `/dashboard`：主 Dashboard（沿用现有主页面）

说明：
- Hub 点击“进入项目”后先调用 `POST /projects/:id/select`，再根据 `hasData` 跳转：
  - `hasData=false` → `/import`
  - `hasData=true` → `/dashboard`

## Project Hub（页面结构建议）

### 主要组件（建议用 shadcn）

- 顶部：应用标题 + “新建项目”按钮（`Dialog + Input + Button`）
- 中部：项目列表（`Card` 或 `Table`）
  - 字段：项目名、最近打开、最近更新、数据状态（未导入/可进入）
  - 操作：进入（进入导入或 dashboard）、重命名（可后置）

### 关键交互

- 新建项目成功后：自动 `select` 为当前项目并跳转 `/import`
- 进入项目：先 `select`，再按 `hasData` 跳转

## Dashboard 顶部“项目切换入口”

位置建议：现有 Dashboard 右上角按钮组左侧（不破坏主视觉）

交互建议（两级）：
- 一级：显示当前项目名 + 下拉/弹窗入口（`DropdownMenu` 或 `Dialog`）
- 二级：弹窗中展示项目列表（简版），支持：
  - 切换项目（触发 `select` → 若 hasData=false 跳 `/import`，否则刷新 dashboard 数据）
  - “更多管理”跳转到 `/`（Project Hub）

## 自动保存（debounce=1000ms）建议

### 原则

- **自动保存由后端兜底**：任何会改变数据的 API（导入执行/改单值/批量改/重置/更新配置/优化写回）在服务端触发一次“保存调度”
- 前端只负责在关键节点显式 flush（提高确定性），但即使 flush 失败也不影响最终一致性

### 前端触发 flush 的节点（建议）

- 切换项目：在调用 `select` 前先调用 `POST /projects/current/save` 并等待返回
- 从 Dashboard 跳回 Hub：同上（先 save 再 navigate）
- 导入执行成功：后端已经保存，但前端可不额外 flush（避免重复）
- 浏览器关闭/刷新：可尝试 `navigator.sendBeacon` 调用 save（可选，不保证）

### UI 提示（可选）

- Dashboard 顶部显示一个轻量保存状态：`已保存` / `保存中...` / `保存失败可重试`
- 保存失败仅提示，不阻断编辑（后端下次成功写入会自动恢复）

