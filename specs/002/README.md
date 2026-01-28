# Northstar 多项目管理设计草案

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- 支持“多项目”：每个项目独立存储在 `data/{projectId}/` 下
- 每个项目持久化：
  - 最新 Excel：`latest.xlsx`
  - 当前页面数据快照：`state.json`（用于重启恢复/切换项目）
- 用户使用流程：**启动先进入 Project Hub**（项目管理：创建/选择）→ 导入向导（上传/映射/执行导入）→ 主 Dashboard
- 若项目未导入数据（hasData=false），从 Hub 点击进入将直接跳转到导入向导
- 主 Dashboard 提供入口，随时切换/管理其他项目

## 非目标（当前阶段）

- 不做“多版本导入历史/回滚/对比”
- 不做多人协作与并发编辑
- 不引入数据库（先用文件原子写入，保持实现简单）

## 核心设计（第一段：项目目录与持久化语义）

### 项目目录结构

```
data/
  projects.json                 # 项目索引（列表、最近打开、显示名等）
  {projectId}/
    meta.json                   # 项目元信息（name、createdAt、updatedAt 等）
    latest.xlsx                 # 最新上传并“生效”的 Excel
    state.json                  # 当前页面数据快照（companies/config/indicators/导入元信息）
    state.json.tmp              # 写入中间文件（原子替换）
    locks/
      state.lock                # 可选：写入互斥（跨进程）
```

### 持久化语义（你确认的策略 B）

- “每一次数据改动都会自动保存”，但采用 `1000ms` debounce 合并写盘（避免频繁 IO）
- 关键节点强制 flush（不等待 debounce）：
  - 切换项目、关闭/刷新页面前
  - 导入完成（Execute Import 成功）
  - 优化完成（Optimize 写回数据）
- 写盘采用 `state.json.tmp` → rename 覆盖 `state.json`，保证断电/崩溃时不产生半截文件

---

下一段将补充：`state.json` 的字段结构（兼容现有 API/前端 store）、以及后端 Project API 与前端页面流转。

## 文档索引（002）

| 文档 | 说明 |
|---|---|
| `specs/002/project-schema.md` | `projects.json` / `meta.json` 结构 |
| `specs/002/state-schema.md` | `state.json` 结构（含 reset 基线字段） |
| `specs/002/ui-flow.md` | Project Hub → Import → Dashboard 的页面流转 |
| `specs/002/backend-api.md` | Project API 与持久化机制（activeProject 最小改动） |
| `specs/002/frontend.md` | 前端页面与自动保存落地建议 |
| `specs/002/api-contract.md` | Project API 合约（请求/响应/错误码） |
| `specs/002/persistence.md` | 持久化细节（原子写、debounce、Excel 落盘时机） |
