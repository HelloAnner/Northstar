# Northstar 系统架构设计文档

## 1 项目概述

Northstar 是一个经济数据统计分析工具，用于解决社零额增长目标计算中的反复调整问题。该工具部署为独立的二进制文件，启动后自动打开浏览器访问本地 Web 页面。

### 1.1 部署形态

| 平台    | 产物                  | 运行方式                    |
| ------- | --------------------- | --------------------------- |
| Windows | `northstar.exe`       | 双击运行，托盘图标 + 浏览器 |
| macOS   | `northstar` (binary)  | 终端运行或 .app 封装        |
| Linux   | `northstar` (binary)  | 终端运行                    |

### 1.2 技术栈

- **后端**: Go 1.21+
- **前端**: React + Vite + Tailwind CSS + shadcn/ui
- **打包**: Go embed 将前端静态资源嵌入二进制文件
- **数据处理**: excelize (Excel 读写)

---

## 2 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户浏览器                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    React SPA                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │ 导入向导页面 │  │ 主控制面板  │  │ 企业数据表格    │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │  │
│  │                          │                                │  │
│  │              ┌───────────┴───────────┐                   │  │
│  │              │   状态管理 (Zustand)   │                   │  │
│  │              │   - 企业数据状态       │                   │  │
│  │              │   - 指标计算状态       │                   │  │
│  │              │   - UI 状态           │                   │  │
│  │              └───────────────────────┘                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│                              │ HTTP/WebSocket                    │
└──────────────────────────────┼───────────────────────────────────┘
                               │
┌──────────────────────────────┼───────────────────────────────────┐
│                         Go 后端                                   │
│  ┌───────────────────────────┴───────────────────────────────┐  │
│  │                    HTTP Server (Gin/Echo)                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │  API 路由   │  │ 静态资源服务 │  │ WebSocket Hub   │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│  ┌───────────────────────────┴───────────────────────────────┐  │
│  │                    业务逻辑层                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │ Excel 导入  │  │ 指标计算引擎 │  │ 智能调整算法    │   │  │
│  │  │   服务      │  │              │  │                 │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│  ┌───────────────────────────┴───────────────────────────────┐  │
│  │                    数据层                                  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │               内存数据存储 (企业数据)                │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│  ┌───────────────────────────┴───────────────────────────────┐  │
│  │              embed.FS (前端静态资源)                       │  │
│  └───────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3 目录结构

```
northstar/
├── cmd/
│   └── northstar/
│       └── main.go              # 程序入口
├── internal/
│   ├── server/
│   │   ├── server.go            # HTTP 服务器
│   │   ├── routes.go            # 路由定义
│   │   └── handlers/
│   │       ├── import.go        # 导入相关处理
│   │       ├── data.go          # 数据 CRUD
│   │       ├── indicator.go     # 指标计算
│   │       └── export.go        # 导出处理
│   ├── service/
│   │   ├── excel/
│   │   │   ├── parser.go        # Excel 解析
│   │   │   ├── generator.go     # 历史数据生成
│   │   │   └── exporter.go      # Excel 导出
│   │   ├── calculator/
│   │   │   ├── engine.go        # 计算引擎
│   │   │   ├── indicators.go    # 16个指标计算
│   │   │   └── optimizer.go     # 智能调整算法
│   │   └── store/
│   │       └── memory.go        # 内存数据存储
│   ├── model/
│   │   ├── company.go           # 企业数据模型
│   │   ├── indicator.go         # 指标数据模型
│   │   └── config.go            # 配置模型
│   └── util/
│       ├── browser.go           # 浏览器启动
│       └── decimal.go           # 精度处理
├── web/                          # 前端源码
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui 组件
│   │   │   ├── layout/
│   │   │   │   └── Header.tsx
│   │   │   ├── import/
│   │   │   │   ├── FileUpload.tsx
│   │   │   │   ├── FieldMapping.tsx
│   │   │   │   ├── GenerationRules.tsx
│   │   │   │   └── ImportProgress.tsx
│   │   │   ├── dashboard/
│   │   │   │   ├── IndicatorPanel.tsx
│   │   │   │   ├── IndustryPanel.tsx
│   │   │   │   └── TotalPanel.tsx
│   │   │   └── table/
│   │   │       ├── CompanyTable.tsx
│   │   │       └── EditableCell.tsx
│   │   ├── hooks/
│   │   │   ├── useIndicators.ts
│   │   │   └── useCompanyData.ts
│   │   ├── store/
│   │   │   └── dataStore.ts     # Zustand store
│   │   ├── services/
│   │   │   └── api.ts           # API 调用
│   │   └── types/
│   │       └── index.ts         # TypeScript 类型
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   └── package.json
├── static/                       # 构建后的前端静态资源 (被 embed)
│   └── dist/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 4 构建流程

### 4.1 开发模式

```bash
# 终端 1: 前端开发服务器
cd web && npm run dev

# 终端 2: 后端开发服务器 (代理前端)
go run ./cmd/northstar -dev
```

### 4.2 生产构建

```bash
# Makefile 目标
make build         # 构建当前平台
make build-all     # 构建所有平台
make build-windows # 构建 Windows
make build-darwin  # 构建 macOS
make build-linux   # 构建 Linux
```

构建步骤:
1. `npm run build` - 构建前端到 `static/dist/`
2. `go build` - 使用 `//go:embed` 嵌入静态资源

### 4.3 Embed 实现

```go
package server

import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed all:dist
var staticFiles embed.FS

func ServeStatic() http.Handler {
    sub, _ := fs.Sub(staticFiles, "dist")
    return http.FileServer(http.FS(sub))
}
```

---

## 5 启动流程

```
main.go
    │
    ├── 1. 解析命令行参数
    │
    ├── 2. 初始化内存数据存储
    │
    ├── 3. 初始化计算引擎
    │
    ├── 4. 创建 HTTP 服务器
    │       ├── 注册 API 路由 (/api/*)
    │       └── 注册静态资源 (embed.FS)
    │
    ├── 5. 查找可用端口 (默认 8080，冲突时递增)
    │
    ├── 6. 启动 HTTP 服务器 (goroutine)
    │
    ├── 7. 打开默认浏览器
    │       └── browser.OpenURL("http://localhost:{port}")
    │
    └── 8. 等待信号 (SIGINT/SIGTERM) 优雅关闭
```

---

## 6 核心模块设计

### 6.1 计算引擎 (Calculator Engine)

计算引擎负责所有 16 个指标的实时计算，采用响应式设计。

```go
// internal/service/calculator/engine.go

type Engine struct {
    store      *store.MemoryStore
    indicators *Indicators
    mu         sync.RWMutex
}

// RecalculateAll 重新计算所有指标
func (e *Engine) RecalculateAll() *model.IndicatorResult

// RecalculateAffected 根据变更的企业重新计算受影响的指标
func (e *Engine) RecalculateAffected(companyID string, changes []string) *model.IndicatorResult
```

### 6.2 智能调整算法 (Optimizer)

使用线性规划或启发式算法实现目标增速的反向推算。

```go
// internal/service/calculator/optimizer.go

type Optimizer struct {
    engine      *Engine
    constraints *OptimizeConstraints
}

type OptimizeConstraints struct {
    TargetGrowthRate   float64  // 目标增速
    MaxIndividualRate  float64  // 单个企业最大增速 (默认 50%)
    MinIndividualRate  float64  // 单个企业最小增速 (默认 0%)
    PriorityIndustries []string // 优先调整的行业
}

// Optimize 执行智能调整
func (o *Optimizer) Optimize(target float64) (*OptimizeResult, error)
```

### 6.3 Excel 处理 (Excel Service)

```go
// internal/service/excel/parser.go

type Parser struct {
    fieldMapping map[string]string
}

// Parse 解析 Excel 文件
func (p *Parser) Parse(file io.Reader) ([]*model.Company, error)

// GetSheets 获取工作表列表
func (p *Parser) GetSheets(file io.Reader) ([]string, error)

// GetColumns 获取列名列表
func (p *Parser) GetColumns(file io.Reader, sheet string) ([]string, error)
```

---

## 7 API 设计

详见 [api.md](./api.md)

---

## 8 前端状态管理

使用 Zustand 进行轻量级状态管理:

```typescript
// web/src/store/dataStore.ts

interface DataStore {
  // 企业数据
  companies: Company[]
  setCompanies: (companies: Company[]) => void
  updateCompany: (id: string, field: string, value: number) => void

  // 指标数据
  indicators: Indicators
  setIndicators: (indicators: Indicators) => void

  // 配置
  config: Config
  setConfig: (config: Partial<Config>) => void

  // 导入状态
  importState: ImportState
  setImportState: (state: Partial<ImportState>) => void
}
```

---

## 9 性能考虑

1. **前端计算**: 简单的增速计算在前端完成，减少网络往返
2. **批量更新**: 多个企业数据变更时，合并计算请求
3. **防抖**: 输入框使用 debounce，避免频繁重算
4. **虚拟滚动**: 企业表格使用虚拟滚动，支持大量数据

---

## 10 安全考虑

1. **本地运行**: 仅监听 localhost，不暴露到网络
2. **输入验证**: 所有数值输入进行范围和类型校验
3. **文件校验**: Excel 文件大小和格式限制
