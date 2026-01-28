# state.json 结构设计（草案）

> 版本: 0.1（草案）
> 更新日期: 2026-01-27

## 目标

- `data/{projectId}/state.json` 能完整恢复主 Dashboard 的业务数据状态
- 支持“重置企业数据”语义：需要保留导入后的基线值（否则重启后无法 reset）
- 与现有后端 `model.*`、前端 `web/src/types` 尽量对齐，减少改动面

## 顶层结构（建议）

```json
{
  "schemaVersion": 1,
  "projectId": "p_xxx",
  "createdAt": "2026-01-27T00:00:00Z",
  "updatedAt": "2026-01-27T00:00:00Z",
  "import": {
    "fileName": "企业数据.xlsx",
    "uploadedAt": "2026-01-27T00:00:00Z",
    "sheet": "Sheet1",
    "mapping": { "companyName": "...", "creditCode": "...", "...": "..." },
    "generateHistory": true,
    "currentMonth": 6
  },
  "config": {
    "currentMonth": 6,
    "lastYearLimitBelowCumulative": 50000
  },
  "companies": [
    {
      "id": "uuid",
      "name": "xx公司",
      "creditCode": "...",
      "industryCode": "...",
      "industryType": "retail",
      "companyScale": 3,
      "isEatWearUse": false,
      "retailLastYearMonth": 0,
      "retailCurrentMonth": 0,
      "retailLastYearCumulative": 0,
      "retailCurrentCumulative": 0,
      "salesLastYearMonth": 0,
      "salesCurrentMonth": 0,
      "salesLastYearCumulative": 0,
      "salesCurrentCumulative": 0,
      "originalRetailCurrentMonth": 0
    }
  ],
  "indicators": { "limitAboveMonthValue": 0, "...": 0 }
}
```

## 字段说明

- `schemaVersion`：用于未来升级/迁移（比如补字段、改字段名）
- `import.*`：用于“这个项目当前生效的 Excel 是怎么导入出来的”
  - Excel 原文件固定保存为 `data/{projectId}/latest.xlsx`，这里不存二进制，只存元信息
- `companies[*].originalRetailCurrentMonth`：导入完成时的基线值，用于 reset（现有内存实现用 `Company.OriginalRetailCurrentMonth`，但该字段当前不进 JSON，需要显式持久化）
- `indicators`：可选缓存字段（用于打开项目时秒开）
  - 后端加载项目时仍建议以 `companies + config` 为准重新计算一次，并覆盖缓存，保证一致性

---

下一段将补充：项目索引 `projects.json` / `meta.json` 的结构，以及后端 Project API（list/create/select/save/load）与前端页面流转。

