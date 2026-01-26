# 前端组件设计文档

## 1 技术栈

- **框架**: React 18 + TypeScript
- **构建**: Vite 5
- **样式**: Tailwind CSS 3 + shadcn/ui
- **状态管理**: Zustand
- **表格**: TanStack Table (react-table)
- **图标**: Material Symbols (Outlined)
- **字体**: Inter

---

## 2 设计规范 (基于 UI 稿)

### 2.1 颜色系统

```typescript
// tailwind.config.js
const colors = {
  primary: "#13a4ec",           // 主色调 - 蓝色
  background: "#f8fafc",        // 页面背景
  surface: "#ffffff",           // 卡片/面板背景
  "border-color": "#e2e8f0",    // 边框颜色
  "text-primary": "#1e293b",    // 主要文字
  "text-secondary": "#64748b",  // 次要文字
  "text-tertiary": "#94a3b8",   // 辅助文字
  success: "#38A169",           // 正增长
  error: "#E53E3E",             // 负增长/错误
}
```

### 2.2 圆角与阴影

```typescript
const borderRadius = {
  DEFAULT: "0.25rem",  // 4px
  lg: "0.5rem",        // 8px
  xl: "0.75rem",       // 12px
  full: "9999px",
}

const boxShadow = {
  sm: "0 1px 2px 0 rgb(0 0 0 / 0.05)",
  md: "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)",
}
```

### 2.3 字体

```css
font-family: 'Inter', sans-serif;
```

---

## 3 页面结构

```
┌─────────────────────────────────────────────────────────────────┐
│ App                                                             │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Router                                                       │ │
│ │                                                              │ │
│ │   /import  →  ImportWizard (导入向导页面)                    │ │
│ │   /        →  Dashboard    (主控制面板)                      │ │
│ │                                                              │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4 导入向导页面 (/import)

### 4.1 页面布局

```
┌────────────────────────────────────────────────────────────────┐
│ Header                                                          │
│ [Logo] 数据导入与配置                  [帮助文档] [返回主面板]   │
├────────────────────────────────────────────────────────────────┤
│ StepIndicator                                                   │
│ ○ 文件上传 ── ● 字段映射 ── ○ 生成规则 ── ○ 执行导入           │
├────────────────────────────────────────────────────────────────┤
│ Content (根据当前步骤显示)                                       │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Step1: FileUploadSection                                   │ │
│ │ Step2: FieldMappingSection                                 │ │
│ │ Step3: GenerationRulesSection                              │ │
│ │ Step4: ImportProgressSection                               │ │
│ └────────────────────────────────────────────────────────────┘ │
├────────────────────────────────────────────────────────────────┤
│ Footer                                                          │
│                                    [上一步]  [下一步: xxx]       │
└────────────────────────────────────────────────────────────────┘
```

### 4.2 组件树

```
ImportWizard
├── Header
│   ├── Logo
│   ├── Title
│   └── ActionButtons
│       ├── HelpButton
│       └── BackButton
├── StepIndicator
│   └── Step (×4)
├── ImportContent
│   ├── FileUploadSection (Step 1)
│   │   ├── FileDropzone
│   │   └── SheetSelector
│   ├── FieldMappingSection (Step 2)
│   │   └── MappingTable
│   │       └── MappingRow (×n)
│   ├── GenerationRulesSection (Step 3)
│   │   └── RuleCard (×4)
│   └── ImportProgressSection (Step 4)
│       ├── ProgressBar
│       └── ImportLog
└── StepNavigator
    ├── PrevButton
    └── NextButton
```

### 4.3 关键组件接口

```typescript
// StepIndicator
interface StepIndicatorProps {
  currentStep: number
  steps: Array<{
    label: string
    completed: boolean
  }>
}

// FileDropzone
interface FileDropzoneProps {
  onFileSelect: (file: File) => void
  accept: string[]
  maxSize: number
  uploading: boolean
}

// MappingTable
interface MappingTableProps {
  systemFields: SystemField[]
  excelColumns: string[]
  mapping: Record<string, string>
  onMappingChange: (field: string, column: string) => void
}

// RuleCard
interface RuleCardProps {
  industryType: IndustryType
  rule: GenerationRule
  onRuleChange: (rule: GenerationRule) => void
}
```

---

## 5 主控制面板 (/dashboard)

### 5.1 页面布局

```
┌────────────────────────────────────────────────────────────────────┐
│ Header (固定)                                                       │
│ [Logo] 数据管理与模拟平台 │ 限上社零增速(当月): [3.45%]              │
│                          │ 限上社零增速(累计): [15.82%]             │
│                                    [智能调整] [重置] [导出]         │
├────────────────────────────────────────────────────────────────────┤
│ IndicatorPanels (4列布局)                                           │
│ ┌──────────────┬─────────────┬────────────────┬──────────────────┐ │
│ │ 限上社零额   │ 专项增速    │ 四大行业增速   │ 社零总额估算     │ │
│ │ 当期零售额   │ 吃穿用增速  │ 批发 零售      │ 社零总额(估算)   │ │
│ │ 同期零售额   │ 小微增速    │ 住宿 餐饮      │ 同比增速         │ │
│ │ 当月增速     │             │                │ 累计增速         │ │
│ │ 累计增速     │             │                │                  │ │
│ └──────────────┴─────────────┴────────────────┴──────────────────┘ │
├────────────────────────────────────────────────────────────────────┤
│ CompanyTable                                                        │
│ ┌────────────────────────────────────────────────────────────────┐ │
│ │ 企业数据微调                      [搜索] [筛选] [排序]          │ │
│ ├────────────────────────────────────────────────────────────────┤ │
│ │ 企业名称 │ 总销售额 │ 本期零售额 │ 同期零售额 │ 增速           │ │
│ │ 企业A    │ 1,250,000│ [1,285,000]│ 1,100,000 │ +16.82%        │ │
│ │ 企业B    │ 3,400,000│ [3,400,000]│ 3,550,000 │ -4.23%         │ │
│ │ ...                                                            │ │
│ └────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────┘
```

### 5.2 组件树

```
Dashboard
├── DashboardHeader
│   ├── Logo
│   ├── Title
│   ├── HeaderIndicators
│   │   ├── HeaderIndicatorInput (当月增速)
│   │   └── HeaderIndicatorInput (累计增速)
│   └── ActionButtons
│       ├── SmartAdjustButton
│       ├── ResetButton
│       └── ExportButton
├── IndicatorPanels
│   ├── LimitAbovePanel (col-span-3)
│   │   └── IndicatorRow (×4)
│   ├── SpecialRatesPanel (col-span-2)
│   │   └── IndicatorRow (×2)
│   ├── IndustryRatesPanel (col-span-4)
│   │   └── IndustryColumn (×4)
│   │       └── RateInput (×2)
│   └── TotalSocialPanel (col-span-3)
│       └── IndicatorRow (×3)
└── CompanyTable
    ├── TableHeader
    │   ├── SearchInput
    │   ├── FilterButton
    │   └── SortButton
    ├── TableHead
    └── TableBody
        └── CompanyRow (×n)
            ├── NameCell
            ├── SalesCell (只读)
            ├── RetailCell (可编辑)
            ├── LastYearCell (只读)
            └── GrowthRateCell (计算)
```

### 5.3 关键组件接口

```typescript
// IndicatorPanel
interface IndicatorPanelProps {
  title: string
  children: React.ReactNode
  className?: string
}

// IndicatorRow
interface IndicatorRowProps {
  label: string
  value: number | string
  editable?: boolean
  suffix?: string
  onChange?: (value: number) => void
  validation?: ValidationState
}

// IndustryColumn
interface IndustryColumnProps {
  industryName: string
  monthRate: number
  cumulativeRate: number
  onMonthRateChange: (value: number) => void
  onCumulativeRateChange: (value: number) => void
}

// CompanyRow
interface CompanyRowProps {
  company: Company
  onRetailChange: (value: number) => void
  highlighted?: boolean
}

// EditableCell
interface EditableCellProps {
  value: number
  onChange: (value: number) => void
  min?: number
  max?: number
  format?: 'number' | 'percent' | 'currency'
  validation?: ValidationState
}
```

---

## 6 状态管理 (Zustand)

### 6.1 Store 结构

```typescript
// stores/dataStore.ts
interface DataStore {
  // 企业数据
  companies: Company[]
  originalCompanies: Company[]  // 原始数据，用于重置
  setCompanies: (companies: Company[]) => void
  updateCompanyRetail: (id: string, value: number) => void
  resetCompanies: (ids?: string[]) => void

  // 指标数据
  indicators: Indicators
  setIndicators: (indicators: Indicators) => void

  // 配置
  config: Config
  setConfig: (config: Partial<Config>) => void

  // 导入状态
  importState: ImportState
  setImportState: (state: Partial<ImportState>) => void

  // 加载状态
  loading: boolean
  setLoading: (loading: boolean) => void
}

interface ImportState {
  step: number
  fileId: string | null
  fileName: string | null
  sheets: SheetInfo[]
  selectedSheet: string | null
  columns: string[]
  mapping: Record<string, string>
  generationRules: GenerationRule[]
  progress: number
  status: 'idle' | 'uploading' | 'mapping' | 'generating' | 'importing' | 'done' | 'error'
  error: string | null
}
```

### 6.2 计算逻辑 Hook

```typescript
// hooks/useIndicatorCalculation.ts
export function useIndicatorCalculation() {
  const { companies, config, setIndicators } = useDataStore()

  // 当企业数据变化时，重新计算指标
  useEffect(() => {
    const newIndicators = calculateIndicators(companies, config)
    setIndicators(newIndicators)
  }, [companies, config])

  return null
}

// 纯函数计算逻辑
function calculateIndicators(companies: Company[], config: Config): Indicators {
  // 预聚合各分组数据
  const sums = aggregateByGroups(companies)

  // 计算各指标
  return {
    limitAbove: calculateLimitAbove(sums),
    specialRates: calculateSpecialRates(sums),
    industryRates: calculateIndustryRates(sums),
    totalSocial: calculateTotalSocial(sums, config),
  }
}
```

---

## 7 UI 组件样式规范

### 7.1 输入框样式

```tsx
// 可编辑输入框 (有数据变更时高亮)
<input
  className="form-input w-32 text-right bg-primary/10 border border-primary/50
             rounded-md p-2 text-primary font-semibold
             focus:ring-1 focus:ring-primary focus:outline-none"
/>

// 普通输入框
<input
  className="form-input w-32 text-right bg-slate-100 border border-transparent
             rounded-md p-2 text-text-primary font-semibold
             focus:ring-1 focus:ring-primary focus:outline-none
             hover:border-border-color transition-colors"
/>

// 错误状态输入框
<input
  className="form-input w-32 text-right bg-error/10 border border-error/50
             rounded-md p-2 text-error font-semibold
             focus:ring-1 focus:ring-error focus:outline-none"
/>

// 只读显示 (非输入框)
<div className="inline-block w-32 text-right bg-slate-100/60 border-transparent
               rounded-md p-2 text-text-primary font-semibold">
  1,250,000
</div>
```

### 7.2 卡片样式

```tsx
// 指标面板卡片
<div className="bg-surface border border-border-color rounded-xl p-4
               flex flex-col gap-3 shadow-sm">
  <h3 className="font-semibold text-text-primary text-base">标题</h3>
  {/* 内容 */}
</div>
```

### 7.3 按钮样式

```tsx
// 主要按钮
<button className="flex min-w-[84px] items-center justify-center
                  rounded-lg h-10 px-4 bg-primary text-white
                  text-sm font-bold gap-2 hover:bg-primary/90 transition-colors">
  <span className="material-symbols-outlined">download</span>
  导出
</button>

// 次要按钮 (强调)
<button className="flex min-w-[84px] items-center justify-center
                  rounded-lg h-10 px-4 bg-primary/20 text-primary
                  text-sm font-bold gap-2 hover:bg-primary/30 transition-colors">
  <span className="material-symbols-outlined">auto_fix_high</span>
  智能调整
</button>

// 普通按钮
<button className="flex min-w-[84px] items-center justify-center
                  rounded-lg h-10 px-4 bg-slate-200 text-text-primary
                  text-sm font-bold hover:bg-slate-300 transition-colors">
  重置
</button>
```

### 7.4 增速显示

```tsx
// 正增长
<td className="text-right text-success">+16.82%</td>

// 负增长
<td className="text-right text-error">-4.23%</td>
```

---

## 8 响应式设计

### 8.1 断点

```typescript
// tailwind.config.js (默认)
screens: {
  'sm': '640px',
  'md': '768px',
  'lg': '1024px',
  'xl': '1280px',
  '2xl': '1536px',
}
```

### 8.2 指标面板响应式

```tsx
// 桌面: 12列网格
<div className="grid grid-cols-12 gap-4">
  <div className="col-span-3">限上社零额</div>
  <div className="col-span-2">专项增速</div>
  <div className="col-span-4">四大行业</div>
  <div className="col-span-3">社零总额</div>
</div>

// 平板: 堆叠布局
<div className="grid grid-cols-12 md:grid-cols-6 gap-4">
  <div className="col-span-12 md:col-span-3">限上社零额</div>
  ...
</div>
```

---

## 9 交互细节

### 9.1 输入防抖

```typescript
const debouncedUpdate = useDebouncedCallback(
  (id: string, value: number) => {
    updateCompanyRetail(id, value)
  },
  300
)
```

### 9.2 错误提示 (Tooltip)

```tsx
<div className="relative group">
  <input className="... bg-error/10 border-error/50" value={value} />
  <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2
                 w-max bg-error text-white text-xs rounded py-1 px-2
                 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
    零售额不能超过总销售额
  </div>
</div>
```

### 9.3 数值格式化

```typescript
// 金额格式化 (千分位)
function formatCurrency(value: number): string {
  return new Intl.NumberFormat('zh-CN').format(value)
}

// 百分比格式化
function formatPercent(value: number): string {
  const sign = value >= 0 ? '+' : ''
  return `${sign}${(value * 100).toFixed(2)}%`
}
```
