# API 接口设计文档

## 1 API 概述

基础路径: `/api/v1`
数据格式: JSON
编码: UTF-8

### 1.1 通用响应格式

```json
{
  "code": 0,           // 0=成功, 非0=错误码
  "message": "success",
  "data": {}           // 实际数据
}
```

### 1.2 错误码定义

| 错误码 | 描述 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 文件格式错误 |
| 1003 | 文件过大 |
| 2001 | 数据不存在 |
| 2002 | 数据校验失败 |
| 3001 | 计算错误 |
| 3002 | 优化无解 |

---

## 2 导入相关 API

### 2.1 上传 Excel 文件

```http
POST /api/v1/import/upload
Content-Type: multipart/form-data
```

**请求参数:**
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| file | File | 是 | Excel 文件 (.xlsx, .xls) |

**响应:**
```json
{
  "code": 0,
  "data": {
    "fileId": "f_abc123",
    "fileName": "data_2024.xlsx",
    "fileSize": 2345678,
    "sheets": [
      { "name": "Sheet1", "rowCount": 150 },
      { "name": "Sheet2", "rowCount": 80 }
    ]
  }
}
```

### 2.2 获取工作表列信息

```http
GET /api/v1/import/{fileId}/columns?sheet={sheetName}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "columns": [
      "公司名称",
      "统一社会信用代码",
      "行业代码",
      "单位规模",
      "本期零售额",
      "上年同期零售额",
      "..."
    ],
    "previewRows": [
      ["企业A", "91110...", "5211", "2", "1250000", "1100000"],
      ["企业B", "91110...", "5212", "3", "520000", "480000"]
    ]
  }
}
```

### 2.3 配置字段映射

```http
POST /api/v1/import/{fileId}/mapping
Content-Type: application/json
```

**请求体:**
```json
{
  "sheet": "Sheet1",
  "mapping": {
    "companyName": "公司名称",
    "creditCode": "统一社会信用代码",
    "industryCode": "行业代码",
    "companyScale": "单位规模",
    "retailCurrentMonth": "本期零售额",
    "retailLastYearMonth": "上年同期零售额",
    "retailCurrentCumulative": "本年累计零售额",
    "retailLastYearCumulative": "上年累计零售额",
    "salesCurrentMonth": "本期销售额",
    "salesLastYearMonth": "上年同期销售额",
    "salesCurrentCumulative": "本年累计销售额",
    "salesLastYearCumulative": "上年累计销售额"
  }
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "validRows": 145,
    "invalidRows": 5,
    "warnings": [
      { "row": 12, "message": "行业代码无法识别: 9999" },
      { "row": 45, "message": "本期零售额为空" }
    ]
  }
}
```

### 2.4 配置历史数据生成规则

```http
POST /api/v1/import/{fileId}/generation-rules
Content-Type: application/json
```

**请求体:**
```json
{
  "rules": [
    {
      "industryType": "retail",
      "minThreshold": 5000000,
      "maxThreshold": 6000000,
      "monthlyVariance": 0.2
    },
    {
      "industryType": "wholesale",
      "minThreshold": 20000000,
      "maxThreshold": 30000000,
      "monthlyVariance": 0.15
    }
  ]
}
```

### 2.5 执行导入

```http
POST /api/v1/import/{fileId}/execute
Content-Type: application/json
```

**请求体:**
```json
{
  "generateHistory": true,
  "currentMonth": 6
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "importedCount": 145,
    "generatedHistoryCount": 12,
    "indicators": { /* 初始指标值 */ }
  }
}
```

---

## 3 数据操作 API

### 3.1 获取企业列表

```http
GET /api/v1/companies?page={page}&pageSize={size}&search={keyword}&industry={type}&scale={scale}
```

**查询参数:**
| 参数 | 类型 | 必填 | 描述 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| pageSize | int | 否 | 每页数量，默认 50，最大 200 |
| search | string | 否 | 搜索关键词 (企业名称) |
| industry | string | 否 | 行业类型筛选 |
| scale | string | 否 | 规模筛选 (支持多选: "3,4") |

**响应:**
```json
{
  "code": 0,
  "data": {
    "total": 145,
    "page": 1,
    "pageSize": 50,
    "items": [
      {
        "id": "c_001",
        "name": "大型商超 A",
        "industryCode": "5211",
        "industryType": "retail",
        "companyScale": 2,
        "isEatWearUse": true,
        "retailCurrentMonth": 1285000,
        "retailLastYearMonth": 1100000,
        "retailCurrentCumulative": 7500000,
        "retailLastYearCumulative": 6800000,
        "salesCurrentMonth": 1250000,
        "salesLastYearMonth": 1150000,
        "salesCurrentCumulative": 7200000,
        "salesLastYearCumulative": 6500000,
        "monthGrowthRate": 0.1682,
        "cumulativeGrowthRate": 0.1029,
        "validation": {
          "hasError": false,
          "errors": []
        }
      }
    ]
  }
}
```

### 3.2 更新企业数据

```http
PATCH /api/v1/companies/{id}
Content-Type: application/json
```

**请求体:**
```json
{
  "retailCurrentMonth": 1350000
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "company": {
      "id": "c_001",
      "retailCurrentMonth": 1350000,
      "retailCurrentCumulative": 7565000,
      "monthGrowthRate": 0.2273,
      "validation": {
        "hasError": false,
        "errors": []
      }
    },
    "indicators": {
      /* 更新后的所有指标 */
    }
  }
}
```

### 3.3 批量更新企业数据

```http
PATCH /api/v1/companies/batch
Content-Type: application/json
```

**请求体:**
```json
{
  "updates": [
    { "id": "c_001", "retailCurrentMonth": 1350000 },
    { "id": "c_002", "retailCurrentMonth": 3500000 }
  ]
}
```

### 3.4 重置企业数据

```http
POST /api/v1/companies/reset
Content-Type: application/json
```

**请求体:**
```json
{
  "companyIds": ["c_001", "c_002"],  // 可选，为空则重置全部
  "resetType": "all"  // all: 重置全部, month: 仅重置当月数据
}
```

---

## 4 指标 API

### 4.1 获取所有指标

```http
GET /api/v1/indicators
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "limitAbove": {
      "monthValue": 15820000,
      "monthRate": 0.0345,
      "cumulativeValue": 85000000,
      "cumulativeRate": 0.1582
    },
    "specialRates": {
      "eatWearUseMonthRate": 0.048,
      "microSmallMonthRate": 0.081
    },
    "industryRates": {
      "wholesale": {
        "monthRate": 0.031,
        "cumulativeRate": 0.029
      },
      "retail": {
        "monthRate": -0.014,
        "cumulativeRate": -0.008
      },
      "accommodation": {
        "monthRate": 0.065,
        "cumulativeRate": 0.072
      },
      "catering": {
        "monthRate": 0.092,
        "cumulativeRate": 0.101
      }
    },
    "totalSocial": {
      "cumulativeValue": 35800000,
      "monthRate": 0.041,
      "cumulativeRate": 0.058
    }
  }
}
```

### 4.2 设置目标增速

```http
POST /api/v1/indicators/target
Content-Type: application/json
```

**请求体:**
```json
{
  "targetIndicator": "limitAboveCumulativeRate",
  "targetValue": 0.075
}
```

---

## 5 智能调整 API

### 5.1 执行智能调整

```http
POST /api/v1/optimize
Content-Type: application/json
```

**请求体:**
```json
{
  "targetIndicator": "limitAboveCumulativeRate",
  "targetValue": 0.075,
  "constraints": {
    "maxIndividualRate": 0.5,
    "minIndividualRate": 0,
    "priorityIndustries": ["retail", "catering"]
  }
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "success": true,
    "achievedValue": 0.0751,
    "adjustments": [
      {
        "companyId": "c_001",
        "companyName": "大型商超 A",
        "originalValue": 1285000,
        "adjustedValue": 1350000,
        "changePercent": 0.0506
      },
      {
        "companyId": "c_005",
        "companyName": "餐饮集团 E",
        "originalValue": 575000,
        "adjustedValue": 620000,
        "changePercent": 0.0783
      }
    ],
    "summary": {
      "adjustedCount": 12,
      "totalAdjustment": 1250000,
      "averageChangePercent": 0.032
    },
    "indicators": {
      /* 调整后的所有指标 */
    }
  }
}
```

### 5.2 预览智能调整结果

```http
POST /api/v1/optimize/preview
Content-Type: application/json
```

请求体与 5.1 相同，但不会实际修改数据，仅返回预览结果。

### 5.3 应用智能调整

```http
POST /api/v1/optimize/apply
Content-Type: application/json
```

**请求体:**
```json
{
  "adjustmentId": "adj_xyz789"  // 来自 optimize 返回的ID
}
```

---

## 6 配置 API

### 6.1 获取配置

```http
GET /api/v1/config
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "currentMonth": 6,
    "lastYearLimitBelowCumulative": 20000000,
    "industryThresholds": {
      "wholesale": { "min": 20000000 },
      "retail": { "min": 5000000 },
      "accommodation": { "min": 2000000 },
      "catering": { "min": 2000000 }
    },
    "optimizeConstraints": {
      "maxIndividualRate": 0.5,
      "minIndividualRate": 0
    }
  }
}
```

### 6.2 更新配置

```http
PATCH /api/v1/config
Content-Type: application/json
```

**请求体:**
```json
{
  "lastYearLimitBelowCumulative": 25000000
}
```

---

## 7 导出 API

### 7.1 导出 Excel

```http
POST /api/v1/export
Content-Type: application/json
```

**请求体:**
```json
{
  "format": "xlsx",
  "includeIndicators": true,
  "includeChanges": true
}
```

**响应:**
```json
{
  "code": 0,
  "data": {
    "downloadUrl": "/api/v1/export/download/exp_abc123",
    "expiresAt": "2024-01-01T12:00:00Z"
  }
}
```

### 7.2 下载导出文件

```http
GET /api/v1/export/download/{exportId}
```

返回文件流，Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`

---

## 8 数据模型

### 8.1 Company (企业)

```typescript
interface Company {
  id: string
  name: string
  creditCode: string
  industryCode: string
  industryType: 'wholesale' | 'retail' | 'accommodation' | 'catering'
  companyScale: 1 | 2 | 3 | 4
  isEatWearUse: boolean

  // 零售额
  retailCurrentMonth: number
  retailLastYearMonth: number
  retailCurrentCumulative: number
  retailLastYearCumulative: number

  // 销售额/营业额
  salesCurrentMonth: number
  salesLastYearMonth: number
  salesCurrentCumulative: number
  salesLastYearCumulative: number

  // 计算字段
  monthGrowthRate: number
  cumulativeGrowthRate: number

  // 校验
  validation: {
    hasError: boolean
    errors: ValidationError[]
  }
}

interface ValidationError {
  field: string
  message: string
  severity: 'error' | 'warning'
}
```

### 8.2 Indicators (指标)

```typescript
interface Indicators {
  limitAbove: {
    monthValue: number
    monthRate: number
    cumulativeValue: number
    cumulativeRate: number
  }
  specialRates: {
    eatWearUseMonthRate: number
    microSmallMonthRate: number
  }
  industryRates: {
    [key in IndustryType]: {
      monthRate: number
      cumulativeRate: number
    }
  }
  totalSocial: {
    cumulativeValue: number
    monthRate: number
    cumulativeRate: number
  }
}
```

---

## 9 WebSocket 接口 (可选)

如果需要实时推送指标更新，可使用 WebSocket:

```
ws://localhost:20261/api/v1/ws
```

**消息格式:**
```json
{
  "type": "indicatorUpdate",
  "data": { /* Indicators */ }
}
```
