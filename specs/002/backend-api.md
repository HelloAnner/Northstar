# 后端 Project API 与持久化机制（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 推荐方案：后端维护 activeProject（最小改动）

系统在服务端维护一个“当前项目”（activeProjectId）。现有业务 API（企业/指标/配置/导入/导出/优化）默认都作用于当前项目，从而最大限度复用现有前端 store 与 API 调用方式；前端只需要在进入项目或切换项目时调用一次 `select`。

### 数据归属（单进程假设）

- 服务端进程内始终只有一份“当前项目内存态”（companies/config/导入元信息等）
- 切换项目时，服务端把新的 `state.json` 加载进内存态并替换当前内存态
- 当前阶段不考虑多个浏览器窗口并行编辑同一项目；如发生并发，以“最后一次写入 state.json”覆盖为准

### 新增 API（建议）

- `GET /api/v1/projects`：返回 Project Hub 列表（来自 `data/projects.json`）
- `POST /api/v1/projects`：新建项目（写 `meta.json`、更新 `projects.json`，并将其设为 active）
- `POST /api/v1/projects/:projectId/select`：切换 activeProject（更新 `projects.json.lastActiveProjectId` + `lastOpenedAt`）
- `GET /api/v1/projects/current`：获取当前 activeProject（用于 Dashboard 顶部显示项目名/状态）
- `POST /api/v1/projects/current/save`：触发一次“强制保存 state.json”（前端 flush 时调用）

### state.json 写入策略（与你确认的 B）

- 后端提供“保存当前项目状态”的能力：将内存中的 `companies + config`（必要时附带 `import` 元信息）写入 `data/{projectId}/state.json`
- 写入必须原子：先写 `state.json.tmp`，再 rename 覆盖 `state.json`
- `updatedAt` 需要同步写入 `meta.json` 与 `projects.json.items[*].updatedAt`
- 自动保存 debounce：`1000ms`（合并频繁变更，减少 IO）

### Excel 落盘

- 导入向导上传成功后，后端把原始 Excel 复制/落盘到 `data/{projectId}/latest.xlsx`
- 解析/预览仍可走现有内存 parser 缓存，但“最终生效文件”以 `latest.xlsx` 为准（重启可恢复）

## 自动保存触发矩阵（建议）

说明：所有“改变 state.json 内容”的动作都要触发一次 `scheduleSave(1000ms)`；关键节点触发 `saveNow()`。

### 需要 scheduleSave 的接口（后端触发）

- `POST /import/:fileId/execute`：导入完成后
  - 额外要求：写入每条企业的 `originalRetailCurrentMonth`（作为 reset 基线，见 `specs/002/state-schema.md`）
- `PATCH /companies/:id`：改单个企业
- `PATCH /companies/batch`：批量改
- `POST /companies/reset`：重置企业
- `PATCH /config`：更新配置
- `POST /optimize`：优化执行（如果该接口会写回企业数据；若只返回 preview 则不保存）

### 不需要保存的接口（只读）

- `GET /companies`、`GET /companies/:id`、`GET /indicators`、`GET /config`
- `POST /optimize/preview`

### 强制 saveNow 的接口/场景

- `POST /projects/current/save`：前端 flush 请求（切换项目/返回 Hub 前）
- `POST /projects/:projectId/select`：服务端切换项目前应先 `saveNow` 当前项目（即使前端没 flush，也兜底）

## 切换项目时序（推荐实现）

### select(projectId)（后端）

1. `saveNow()`：把当前 activeProject 的内存态落盘到 `data/{activeProjectId}/state.json`
   - 若保存失败：**返回错误并阻止切换**（保持 activeProject 不变，避免丢修改）
2. 切换 activeProjectId，并更新 `data/projects.json` 的：
   - `lastActiveProjectId`
   - `items[*].lastOpenedAt`
3. 加载目标项目：
   - 若 `data/{projectId}/state.json` 不存在：返回 `hasData=false`（前端按规则跳转到 Import）
   - 若存在：读取并替换内存态（companies/config/import），随后重算 indicators 并写回内存态
4. 返回当前项目信息（projectId/name/hasData/updatedAt）

### 进入 Dashboard（前端）

进入 `/dashboard` 后按现有逻辑调用 `fetchCompanies/fetchConfig/fetchIndicators` 即可，它们都自动指向 activeProject。

## 启动时序（你确认：启动必进 Hub）

1. 前端启动进入 `/`（Project Hub）
2. Hub 调用 `GET /projects` 渲染列表
3. 用户点击“进入”：
   - 前端先调用 `POST /projects/:id/select`
   - 后端按上面 select 时序返回 `hasData`
   - `hasData=false` → 前端跳 `/import`；`hasData=true` → 前端跳 `/dashboard`

## 失败与恢复（建议）

- `state.json` 写入失败：记录日志（英文），并返回给调用方（若由 scheduleSave 触发可吞掉并在下一次再尝试）
- 读 `state.json` 失败（损坏/旧版本）：返回错误并提示用户重新导入；同时保留原文件以便排查（可追加 `.bad` 后缀）
- `schemaVersion` 不匹配：返回“需要迁移/重新导入”的明确错误（后续再补迁移策略）

---

前端落地与合约已补充：
- `specs/002/frontend.md`
- `specs/002/api-contract.md`
