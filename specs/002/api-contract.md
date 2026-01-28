# Project API 合约（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 通用响应格式（沿用现有）

```json
{ "code": 0, "message": "success", "data": {} }
```

## 数据模型（建议）

### ProjectSummary

```json
{
  "projectId": "p_xxx",
  "name": "2026年1月社零测算",
  "createdAt": "2026-01-27T00:00:00Z",
  "updatedAt": "2026-01-27T00:00:00Z",
  "lastOpenedAt": "2026-01-27T00:00:00Z",
  "hasData": true
}
```

### CurrentProject

```json
{
  "project": { "projectId": "p_xxx", "name": "..." },
  "hasData": true
}
```

## 接口合约（建议）

### 1) 获取项目列表

`GET /api/v1/projects`

Response `data`：

```json
{
  "lastActiveProjectId": "p_xxx",
  "items": [ { "projectId": "p_xxx", "name": "...", "hasData": true } ]
}
```

### 2) 新建项目

`POST /api/v1/projects`

Request：

```json
{ "name": "2026年1月社零测算" }
```

Response `data`：

```json
{ "projectId": "p_new", "name": "2026年1月社零测算", "hasData": false }
```

### 3) 选择/切换项目

`POST /api/v1/projects/:projectId/select`

Response `data`：

```json
{ "projectId": "p_xxx", "name": "...", "hasData": true }
```

说明：
- 服务端在 select 内部会对“旧 activeProject”先 `saveNow()` 再切换（见 `specs/002/backend-api.md`）
- 若 `saveNow()` 失败：**直接返回错误并阻止切换**（避免丢修改）
- 若返回 `hasData=false`，前端应直接跳转 `/import`

### 4) 获取当前项目

`GET /api/v1/projects/current`

Response `data`：

```json
{ "projectId": "p_xxx", "name": "...", "hasData": true }
```

### 5) 强制保存

`POST /api/v1/projects/current/save`

Response `data`：

```json
{ "saved": true, "savedAt": "2026-01-27T00:00:00Z" }
```

## 错误码建议

- `4001`：项目不存在
- `4002`：项目创建失败（名称非法/写入失败）
- `4003`：项目状态加载失败（state.json 损坏/版本不兼容）
- `4004`：保存失败（state.json 写入失败）
