# Northstar 实现工作流

> 基于设计文档生成的分阶段实现计划

## 工作流概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Northstar 实现工作流                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1          Phase 2          Phase 3          Phase 4                 │
│  ┌─────────┐      ┌─────────┐      ┌─────────┐      ┌─────────┐            │
│  │ 项目初始化│ ──▶ │ 后端核心 │ ──▶ │ 前端实现 │ ──▶ │ 集成测试 │            │
│  │ 与基础设施│      │ 功能开发 │      │ 与联调  │      │ 与构建  │            │
│  └─────────┘      └─────────┘      └─────────┘      └─────────┘            │
│       │                │                │                │                  │
│       ▼                ▼                ▼                ▼                  │
│  • Go 项目结构      • 数据模型       • React 项目     • 端到端测试          │
│  • 前端脚手架       • Excel 解析     • 组件开发       • 性能优化            │
│  • 构建配置         • 计算引擎       • 状态管理       • 跨平台构建          │
│  • 开发环境         • API 接口       • API 集成       • 发布打包            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: 项目初始化与基础设施

### 任务 1.1: Go 项目结构搭建

**优先级**: P0 (阻塞后续)
**依赖**: 无

```bash
# 创建目录结构
mkdir -p cmd/northstar
mkdir -p internal/{server,service,model,util}
mkdir -p internal/server/handlers
mkdir -p internal/service/{excel,calculator,store}
mkdir -p static/dist
```

**交付物**:
- [ ] `go.mod` - Go 模块初始化
- [ ] `cmd/northstar/main.go` - 程序入口骨架
- [ ] `internal/model/*.go` - 数据模型定义
- [ ] `Makefile` - 构建脚本

**验收标准**:
- `go build ./cmd/northstar` 成功编译
- 程序启动后输出 "Server starting..."

---

### 任务 1.2: 前端项目初始化

**优先级**: P0 (阻塞后续)
**依赖**: 无 (可与 1.1 并行)

```bash
# 创建 Vite + React + TypeScript 项目
cd web
npm create vite@latest . -- --template react-ts
npm install
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

**交付物**:
- [ ] `web/package.json` - 依赖配置
- [ ] `web/vite.config.ts` - Vite 配置
- [ ] `web/tailwind.config.js` - Tailwind 配置
- [ ] `web/src/App.tsx` - 根组件骨架

**验收标准**:
- `npm run dev` 启动开发服务器
- 访问 `localhost:5173` 显示默认页面

---

### 任务 1.3: shadcn/ui 集成

**优先级**: P0
**依赖**: 1.2

```bash
npx shadcn@latest init
npx shadcn@latest add button input table select card
```

**交付物**:
- [ ] `web/components.json` - shadcn 配置
- [ ] `web/src/components/ui/*.tsx` - 基础 UI 组件
- [ ] `web/src/lib/utils.ts` - 工具函数

**验收标准**:
- 导入 Button 组件并正常渲染
- Tailwind 样式正确应用

---

### 任务 1.4: 开发环境配置

**优先级**: P1
**依赖**: 1.1, 1.2

**交付物**:
- [ ] 后端开发模式 (代理前端请求)
- [ ] 热重载配置
- [ ] 环境变量配置

**验收标准**:
- `go run ./cmd/northstar -dev` 启动后端
- 前端请求 `/api/*` 被正确代理

---

## Phase 2: 后端核心功能开发

### 任务 2.1: 数据模型实现

**优先级**: P0
**依赖**: 1.1

**交付物**:
- [ ] `internal/model/company.go` - 企业模型
- [ ] `internal/model/indicator.go` - 指标模型
- [ ] `internal/model/config.go` - 配置模型

```go
// company.go 核心结构
type Company struct {
    ID                        string
    Name                      string
    IndustryCode              string
    IndustryType              IndustryType
    CompanyScale              int
    IsEatWearUse              bool
    RetailLastYearMonth       float64
    RetailCurrentMonth        float64
    RetailLastYearCumulative  float64
    RetailCurrentCumulative   float64
    SalesLastYearMonth        float64
    SalesCurrentMonth         float64
    SalesLastYearCumulative   float64
    SalesCurrentCumulative    float64
}
```

---

### 任务 2.2: 内存存储实现

**优先级**: P0
**依赖**: 2.1

**交付物**:
- [ ] `internal/service/store/memory.go` - 内存存储

```go
type MemoryStore struct {
    companies map[string]*model.Company
    config    *model.Config
    mu        sync.RWMutex
}

func (s *MemoryStore) GetAllCompanies() []*model.Company
func (s *MemoryStore) GetCompany(id string) *model.Company
func (s *MemoryStore) UpdateCompany(id string, updates map[string]interface{}) error
func (s *MemoryStore) SetCompanies(companies []*model.Company)
```

---

### 任务 2.3: Excel 解析服务

**优先级**: P0
**依赖**: 2.1

**交付物**:
- [ ] `internal/service/excel/parser.go` - Excel 解析
- [ ] `internal/service/excel/generator.go` - 历史数据生成

```go
// 核心函数
func (p *Parser) ParseExcel(file io.Reader, sheet string, mapping FieldMapping) ([]*model.Company, error)
func (p *Parser) GetSheets(file io.Reader) ([]SheetInfo, error)
func (p *Parser) GetColumns(file io.Reader, sheet string) ([]string, error)
func (g *Generator) GenerateHistory(company *model.Company, rule GenerationRule) error
```

**依赖库**: `github.com/xuri/excelize/v2`

---

### 任务 2.4: 指标计算引擎

**优先级**: P0 (核心)
**依赖**: 2.1, 2.2

**交付物**:
- [ ] `internal/service/calculator/engine.go` - 计算引擎
- [ ] `internal/service/calculator/indicators.go` - 指标计算

```go
// 核心接口
type Engine interface {
    Calculate() *model.Indicators
    RecalculateForCompany(companyID string) *model.Indicators
}

// 16 个指标计算函数
func (e *engine) calcLimitAboveMonth() (value, rate float64)
func (e *engine) calcLimitAboveCumulative() (value, rate float64)
func (e *engine) calcEatWearUseRate() float64
func (e *engine) calcMicroSmallRate() float64
func (e *engine) calcIndustryRates() map[IndustryType]IndustryRate
func (e *engine) calcTotalSocial() (value, rate float64)
```

**验收标准**:
- 使用测试用例中的数据验证计算结果
- 指标 1-16 计算结果与预期一致

---

### 任务 2.5: 智能调整算法

**优先级**: P1
**依赖**: 2.4

**交付物**:
- [ ] `internal/service/calculator/optimizer.go` - 优化器

```go
type Optimizer interface {
    Optimize(target OptimizeTarget, constraints Constraints) (*OptimizeResult, error)
    Preview(target OptimizeTarget, constraints Constraints) (*OptimizeResult, error)
}

// 算法选择: 贪心 + 启发式
// 1. 计算达成目标需要的总增量
// 2. 按优先级行业排序企业
// 3. 在约束范围内分配增量
// 4. 验证是否达成目标
```

---

### 任务 2.6: HTTP 服务器与 API

**优先级**: P0
**依赖**: 2.2, 2.3, 2.4

**交付物**:
- [ ] `internal/server/server.go` - HTTP 服务器
- [ ] `internal/server/routes.go` - 路由注册
- [ ] `internal/server/handlers/import.go` - 导入 API
- [ ] `internal/server/handlers/data.go` - 数据 API
- [ ] `internal/server/handlers/indicator.go` - 指标 API
- [ ] `internal/server/handlers/export.go` - 导出 API

**API 端点**:
```
POST   /api/v1/import/upload
GET    /api/v1/import/{fileId}/columns
POST   /api/v1/import/{fileId}/mapping
POST   /api/v1/import/{fileId}/execute

GET    /api/v1/companies
PATCH  /api/v1/companies/{id}
PATCH  /api/v1/companies/batch
POST   /api/v1/companies/reset

GET    /api/v1/indicators
POST   /api/v1/optimize
POST   /api/v1/export
```

---

### 任务 2.7: Embed 静态资源

**优先级**: P1
**依赖**: 2.6

**交付物**:
- [ ] `internal/server/static.go` - 静态资源服务
- [ ] 前端构建输出到 `static/dist/`

```go
//go:embed all:dist
var staticFiles embed.FS
```

---

## Phase 3: 前端实现与联调

### 任务 3.1: 类型定义与 API 服务

**优先级**: P0
**依赖**: 2.6

**交付物**:
- [ ] `web/src/types/index.ts` - TypeScript 类型
- [ ] `web/src/services/api.ts` - API 调用封装

```typescript
// types/index.ts
export interface Company { ... }
export interface Indicators { ... }
export type IndustryType = 'wholesale' | 'retail' | 'accommodation' | 'catering'

// services/api.ts
export const api = {
  import: {
    upload: (file: File) => fetch('/api/v1/import/upload', ...),
    getColumns: (fileId: string, sheet: string) => ...,
    ...
  },
  companies: {
    list: (params?: ListParams) => ...,
    update: (id: string, data: Partial<Company>) => ...,
    ...
  },
  indicators: {
    get: () => ...,
  },
  optimize: {
    run: (target: OptimizeTarget, constraints: Constraints) => ...,
  }
}
```

---

### 任务 3.2: Zustand 状态管理

**优先级**: P0
**依赖**: 3.1

**交付物**:
- [ ] `web/src/store/dataStore.ts` - 数据状态
- [ ] `web/src/store/importStore.ts` - 导入状态

```typescript
// dataStore.ts
export const useDataStore = create<DataStore>((set, get) => ({
  companies: [],
  indicators: defaultIndicators,
  config: defaultConfig,

  updateCompanyRetail: (id, value) => {
    // 1. 更新本地状态
    // 2. 调用 API
    // 3. 更新指标
  },

  setIndicators: (indicators) => set({ indicators }),
}))
```

---

### 任务 3.3: 导入向导页面

**优先级**: P1
**依赖**: 3.1, 3.2

**交付物**:
- [ ] `web/src/pages/ImportWizard.tsx` - 导入向导页面
- [ ] `web/src/components/import/StepIndicator.tsx` - 步骤指示器
- [ ] `web/src/components/import/FileUpload.tsx` - 文件上传
- [ ] `web/src/components/import/FieldMapping.tsx` - 字段映射
- [ ] `web/src/components/import/GenerationRules.tsx` - 生成规则
- [ ] `web/src/components/import/ImportProgress.tsx` - 导入进度

**验收标准**:
- 4 步骤导入流程完整可用
- UI 与设计稿一致

---

### 任务 3.4: 主控制面板 - 指标卡片

**优先级**: P0
**依赖**: 3.2

**交付物**:
- [ ] `web/src/pages/Dashboard.tsx` - 主面板页面
- [ ] `web/src/components/dashboard/DashboardHeader.tsx` - 头部
- [ ] `web/src/components/dashboard/LimitAbovePanel.tsx` - 限上社零
- [ ] `web/src/components/dashboard/SpecialRatesPanel.tsx` - 专项增速
- [ ] `web/src/components/dashboard/IndustryRatesPanel.tsx` - 四大行业
- [ ] `web/src/components/dashboard/TotalSocialPanel.tsx` - 社零总额

**验收标准**:
- 4 个指标卡片正确显示
- 数值编辑触发重算

---

### 任务 3.5: 企业数据表格

**优先级**: P0
**依赖**: 3.2

**交付物**:
- [ ] `web/src/components/table/CompanyTable.tsx` - 企业表格
- [ ] `web/src/components/table/EditableCell.tsx` - 可编辑单元格
- [ ] `web/src/components/table/TableToolbar.tsx` - 工具栏

**功能**:
- 搜索、筛选、排序
- 本期零售额可编辑
- 增速颜色标识 (正绿负红)
- 错误状态提示

---

### 任务 3.6: 计算联动实现

**优先级**: P0 (核心)
**依赖**: 3.4, 3.5

**交付物**:
- [ ] `web/src/hooks/useIndicatorCalculation.ts` - 指标计算 Hook
- [ ] 前端 debounce 处理
- [ ] 后端 API 联动

**验收标准**:
- 修改企业零售额 → 指标实时更新
- 修改目标增速 → 智能调整 → 企业数据更新

---

### 任务 3.7: 智能调整对话框

**优先级**: P1
**依赖**: 3.4

**交付物**:
- [ ] `web/src/components/dialog/SmartAdjustDialog.tsx` - 智能调整对话框

**功能**:
- 目标增速输入
- 约束条件配置
- 预览调整结果
- 确认应用

---

## Phase 4: 集成测试与构建

### 任务 4.1: 后端单元测试

**优先级**: P1
**依赖**: Phase 2 完成

**交付物**:
- [ ] `internal/service/calculator/engine_test.go`
- [ ] `internal/service/excel/parser_test.go`

**测试用例**: 使用 `test-cases.md` 中的数据

---

### 任务 4.2: 前端组件测试

**优先级**: P2
**依赖**: Phase 3 完成

**交付物**:
- [ ] `web/src/__tests__/` - 组件测试

---

### 任务 4.3: 端到端测试

**优先级**: P1
**依赖**: 4.1, 4.2

**测试场景**:
1. Excel 导入完整流程
2. 企业数据编辑 → 指标更新
3. 智能调整功能
4. 数据导出

---

### 任务 4.4: 跨平台构建

**优先级**: P0
**依赖**: Phase 3 完成

**交付物**:
- [ ] `Makefile` 完善
- [ ] GitHub Actions CI/CD (可选)

```makefile
.PHONY: build build-all build-windows build-darwin build-linux

build-frontend:
	cd web && npm run build
	cp -r web/dist static/

build: build-frontend
	go build -o northstar ./cmd/northstar

build-windows: build-frontend
	GOOS=windows GOARCH=amd64 go build -o northstar.exe ./cmd/northstar

build-darwin: build-frontend
	GOOS=darwin GOARCH=amd64 go build -o northstar-darwin ./cmd/northstar

build-linux: build-frontend
	GOOS=linux GOARCH=amd64 go build -o northstar-linux ./cmd/northstar

build-all: build-windows build-darwin build-linux
```

---

### 任务 4.5: 浏览器自动打开

**优先级**: P1
**依赖**: 4.4

**交付物**:
- [ ] `internal/util/browser.go` - 跨平台浏览器打开

```go
func OpenBrowser(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "windows":
        cmd = exec.Command("cmd", "/c", "start", url)
    case "darwin":
        cmd = exec.Command("open", url)
    default:
        cmd = exec.Command("xdg-open", url)
    }
    return cmd.Start()
}
```

---

### 任务 4.6: 发布与文档

**优先级**: P2
**依赖**: 4.4

**交付物**:
- [ ] `README.md` - 使用文档
- [ ] `CHANGELOG.md` - 版本记录
- [ ] Release 产物 (各平台二进制文件)

---

## 依赖关系图

```
Phase 1                    Phase 2                    Phase 3                Phase 4
────────────────────────────────────────────────────────────────────────────────────────

1.1 Go项目结构 ──────────▶ 2.1 数据模型 ──────────────▶ 3.1 类型定义 ────────▶ 4.1 后端测试
        │                       │                            │
        │                       ▼                            ▼
        │                  2.2 内存存储 ──────────────▶ 3.2 Zustand ─────────▶ 4.2 前端测试
        │                       │                            │
        │                       │                            │
1.2 前端初始化 ────────────────│─────────────────────────────┤
        │                       │                            │
        ▼                       ▼                            ▼
1.3 shadcn/ui ──────────▶ 2.3 Excel解析                3.3 导入向导
                                │                            │
                                ▼                            ▼
                          2.4 计算引擎 ──────────────▶ 3.4 指标卡片 ────────▶ 4.3 E2E测试
                                │                            │
                                ▼                            ▼
                          2.5 智能调整              3.5 企业表格
                                │                            │
                                ▼                            ▼
                          2.6 HTTP服务 ◀─────────────▶ 3.6 计算联动
                                │
                                ▼
1.4 开发环境 ◀──────────▶ 2.7 Embed静态 ────────────────────────────────────▶ 4.4 跨平台构建
                                                                                    │
                                                                                    ▼
                                                                              4.5 浏览器打开
                                                                                    │
                                                                                    ▼
                                                                              4.6 发布文档
```

---

## 里程碑

| 里程碑 | 包含任务 | 验收标准 |
|--------|----------|----------|
| M1: 基础设施就绪 | 1.1-1.4 | 前后端开发环境可用 |
| M2: 后端功能完整 | 2.1-2.7 | 所有 API 可调用，计算正确 |
| M3: 前端功能完整 | 3.1-3.7 | UI 完整，联动正常 |
| M4: 发布就绪 | 4.1-4.6 | 跨平台二进制可用 |

---

## 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| Excel 解析复杂度 | P2 | 使用成熟的 excelize 库，限制支持的格式 |
| 计算精度问题 | P0 | 使用 decimal 库，中间结果保留 8 位小数 |
| 大数据量性能 | P1 | 虚拟滚动 + 增量计算 |
| 跨平台兼容性 | P1 | 早期在各平台测试构建 |
