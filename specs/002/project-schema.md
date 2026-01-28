# projects.json / meta.json 结构设计（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- `projects.json` 用于 Project Hub 列表展示与最近访问排序
- `meta.json` 用于项目目录内自描述（脱离索引也能识别）
- 结构尽量简单，便于人工排查和未来迁移

## projects.json（全局索引）

路径：`data/projects.json`

```json
{
  "schemaVersion": 1,
  "lastActiveProjectId": "p_xxx",
  "items": [
    {
      "projectId": "p_xxx",
      "name": "2026年1月社零测算",
      "createdAt": "2026-01-27T00:00:00Z",
      "updatedAt": "2026-01-27T00:00:00Z",
      "lastOpenedAt": "2026-01-27T00:00:00Z",
      "hasData": true
    }
  ]
}
```

字段说明：
- `lastActiveProjectId`：仅用于默认高亮/快捷进入（即使启动必到 Hub，也需要一个“当前选择项目”）
- `hasData`：`latest.xlsx && state.json` 是否存在（Hub 用于显示“需要导入/可进入”）

## meta.json（项目元信息）

路径：`data/{projectId}/meta.json`

```json
{
  "schemaVersion": 1,
  "projectId": "p_xxx",
  "name": "2026年1月社零测算",
  "createdAt": "2026-01-27T00:00:00Z",
  "updatedAt": "2026-01-27T00:00:00Z"
}
```

说明：
- `meta.json` 和 `projects.json.items[*]` 有冗余字段（name/时间），目的是支持“目录自包含”和“索引快速展示”
- `updatedAt`：任何一次 state 保存成功后都应该推进（用于 Hub 排序/提醒）

