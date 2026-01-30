# 前端页面设计

## 设计目标

1. **简洁高效**：启动即仪表板，一键导入数据
2. **实时联动**：修改数据立即反映到指标
3. **专业美观**：shadcn/ui 组件，商务风格

---

## 页面结构

```
┌─────────────────────────────────────────────────────────────────┐
│  Topbar                                                         │
│  ┌─────────┐                              ┌────────┬────────┐  │
│  │ Logo    │                              │ 导入   │ 导出   │  │
│  └─────────┘                              └────────┴────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Dashboard (主内容区)                                            │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  指标卡片区 (4 行 4 列)                                   │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │   │
│  │  │限上社零 │ │ 当月增速│ │ 累计值  │ │累计增速 │        │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘        │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │   │
│  │  │吃穿用   │ │ 小微    │ │ 批发增速│ │零售增速 │        │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘        │   │
│  │  ...                                                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Tab 切换: [批发零售] [住宿餐饮] [配置]                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  筛选栏                                                   │   │
│  │  [行业类型 ▼] [小微 ▼] [吃穿用 ▼] [搜索企业...]          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  企业数据表格 (可编辑)                                    │   │
│  │  ┌────┬──────────┬────────┬────────┬────────┬────────┐  │   │
│  │  │序号│ 企业名称  │ 本月销售│ 本月零售│ 当月增速│ 累计增速│  │   │
│  │  ├────┼──────────┼────────┼────────┼────────┼────────┤  │   │
│  │  │ 1  │ XX公司    │ 1234.5 │ 1000.0 │ 5.2%   │ 3.1%   │  │   │
│  │  │ 2  │ YY公司    │ 2345.6 │ 2000.0 │ 8.1%   │ 6.5%   │  │   │
│  │  │... │ ...       │ ...    │ ...    │ ...    │ ...    │  │   │
│  │  └────┴──────────┴────────┴────────┴────────┴────────┘  │   │
│  │  [← 上一页] 第 1/10 页 [下一页 →]                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 组件设计

### 1. Topbar 顶部栏

```tsx
// components/Topbar.tsx
export function Topbar() {
  return (
    <header className="h-14 border-b bg-background flex items-center justify-between px-4">
      <div className="flex items-center gap-2">
        <span className="font-semibold text-lg">Northstar</span>
        <span className="text-muted-foreground text-sm">经济数据统计</span>
      </div>
      <div className="flex items-center gap-2">
        <ImportButton />
        <ExportButton />
      </div>
    </header>
  )
}
```

### 2. ImportButton 导入按钮

```tsx
// components/ImportButton.tsx
export function ImportButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Upload className="w-4 h-4 mr-2" />
        导入数据
      </Button>
      <ImportModal open={open} onOpenChange={setOpen} />
    </>
  )
}
```

### 3. ImportModal 导入对话框

简化的一键导入流程，替代原来的 4 步向导。

```tsx
// components/ImportModal.tsx
interface ImportModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ImportModal({ open, onOpenChange }: ImportModalProps) {
  const [file, setFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState(0)
  const [logs, setLogs] = useState<string[]>([])
  const [result, setResult] = useState<ImportResult | null>(null)

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (f) setFile(f)
  }

  const handleImport = async () => {
    if (!file) return
    setImporting(true)
    setLogs([])

    try {
      // 上传并解析
      const res = await api.import(file, (msg, pct) => {
        setLogs(prev => [...prev, msg])
        setProgress(pct)
      })
      setResult(res)
    } catch (err) {
      setLogs(prev => [...prev, `错误: ${err.message}`])
    } finally {
      setImporting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>导入 Excel 数据</DialogTitle>
          <DialogDescription>
            选择预估表 Excel 文件，系统将自动识别并解析所有 Sheet
          </DialogDescription>
        </DialogHeader>

        {!importing && !result && (
          <div className="space-y-4">
            <div className="border-2 border-dashed rounded-lg p-8 text-center">
              <input
                type="file"
                accept=".xlsx,.xls"
                onChange={handleFileSelect}
                className="hidden"
                id="file-input"
              />
              <label htmlFor="file-input" className="cursor-pointer">
                <FileSpreadsheet className="w-12 h-12 mx-auto text-muted-foreground" />
                <p className="mt-2 text-sm text-muted-foreground">
                  点击或拖拽文件到此处
                </p>
                {file && (
                  <p className="mt-2 text-sm font-medium">{file.name}</p>
                )}
              </label>
            </div>
            <Button
              className="w-full"
              onClick={handleImport}
              disabled={!file}
            >
              开始导入
            </Button>
          </div>
        )}

        {importing && (
          <div className="space-y-4">
            <Progress value={progress} />
            <div className="h-48 overflow-auto bg-muted rounded p-2 text-xs font-mono">
              {logs.map((log, i) => (
                <div key={i}>{log}</div>
              ))}
            </div>
          </div>
        )}

        {result && (
          <div className="space-y-4">
            <div className="flex items-center gap-2 text-green-600">
              <CheckCircle className="w-5 h-5" />
              <span>导入完成</span>
            </div>
            <ImportResultTable result={result} />
            <Button className="w-full" onClick={() => {
              onOpenChange(false)
              setResult(null)
              setFile(null)
            }}>
              完成
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
```

### 4. IndicatorCards 指标卡片

```tsx
// components/IndicatorCards.tsx
interface Indicator {
  key: string
  label: string
  value: number
  unit: string
  trend?: 'up' | 'down' | 'flat'
}

const indicatorGroups = [
  {
    title: '限上社零额',
    items: [
      { key: 'limit_above_retail_month', label: '当月值', unit: '万元' },
      { key: 'limit_above_retail_month_rate', label: '当月增速', unit: '%' },
      { key: 'limit_above_retail_cumulative', label: '累计值', unit: '万元' },
      { key: 'limit_above_retail_cumulative_rate', label: '累计增速', unit: '%' },
    ]
  },
  {
    title: '专项增速',
    items: [
      { key: 'eat_wear_use_month_rate', label: '吃穿用', unit: '%' },
      { key: 'small_micro_month_rate', label: '小微', unit: '%' },
    ]
  },
  {
    title: '四大行业 (当月/累计)',
    items: [
      { key: 'wholesale_sales_month_rate', label: '批发', unit: '%' },
      { key: 'retail_sales_month_rate', label: '零售', unit: '%' },
      { key: 'accommodation_revenue_month_rate', label: '住宿', unit: '%' },
      { key: 'catering_revenue_month_rate', label: '餐饮', unit: '%' },
    ]
  },
  {
    title: '社零总额',
    items: [
      { key: 'total_retail_cumulative', label: '累计值', unit: '万元' },
      { key: 'total_retail_cumulative_rate', label: '累计增速', unit: '%' },
    ]
  },
]

export function IndicatorCards({ indicators }: { indicators: Record<string, number> }) {
  return (
    <div className="space-y-4">
      {indicatorGroups.map(group => (
        <div key={group.title}>
          <h3 className="text-sm font-medium text-muted-foreground mb-2">
            {group.title}
          </h3>
          <div className="grid grid-cols-4 gap-3">
            {group.items.map(item => (
              <Card key={item.key} className="p-3">
                <div className="text-xs text-muted-foreground">{item.label}</div>
                <div className="text-lg font-semibold mt-1">
                  {formatNumber(indicators[item.key])}
                  <span className="text-sm font-normal ml-1">{item.unit}</span>
                </div>
              </Card>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
```

### 5. CompanyTable 企业数据表格

```tsx
// components/CompanyTable.tsx
interface Column {
  key: string
  label: string
  editable?: boolean
  width?: number
  render?: (value: any, row: Company) => React.ReactNode
}

const wrColumns: Column[] = [
  { key: 'row_no', label: '序号', width: 60 },
  { key: 'name', label: '企业名称', width: 200 },
  { key: 'industry_code', label: '行业代码', width: 100 },
  { key: 'sales_current_month', label: '本月销售额', editable: true, width: 120 },
  { key: 'sales_month_rate', label: '销售额增速', width: 100,
    render: (v) => v ? `${v.toFixed(2)}%` : '-' },
  { key: 'retail_current_month', label: '本月零售额', editable: true, width: 120 },
  { key: 'retail_month_rate', label: '零售额增速', width: 100,
    render: (v) => v ? `${v.toFixed(2)}%` : '-' },
  { key: 'retail_ratio', label: '零销比', width: 80,
    render: (v) => v ? `${v.toFixed(1)}%` : '-' },
  { key: 'is_small_micro', label: '小微', width: 60,
    render: (v) => v ? '是' : '-' },
  { key: 'is_eat_wear_use', label: '吃穿用', width: 60,
    render: (v) => v ? '是' : '-' },
]

export function CompanyTable({ type }: { type: 'wr' | 'ac' }) {
  const { companies, pagination, updateCompany, fetchCompanies } = useDataStore()
  const columns = type === 'wr' ? wrColumns : acColumns

  const handleCellEdit = async (id: number, field: string, value: number) => {
    await updateCompany(id, { [field]: value })
    // 更新后自动刷新指标
  }

  return (
    <div className="border rounded-lg">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map(col => (
              <TableHead key={col.key} style={{ width: col.width }}>
                {col.label}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {companies.map(company => (
            <TableRow key={company.id}>
              {columns.map(col => (
                <TableCell key={col.key}>
                  {col.editable ? (
                    <EditableCell
                      value={company[col.key]}
                      onChange={(v) => handleCellEdit(company.id, col.key, v)}
                    />
                  ) : col.render ? (
                    col.render(company[col.key], company)
                  ) : (
                    company[col.key]
                  )}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Pagination
        current={pagination.page}
        total={pagination.total}
        pageSize={pagination.pageSize}
        onChange={(page) => fetchCompanies({ page })}
      />
    </div>
  )
}
```

### 6. EditableCell 可编辑单元格

```tsx
// components/EditableCell.tsx
interface EditableCellProps {
  value: number
  onChange: (value: number) => void
}

export function EditableCell({ value, onChange }: EditableCellProps) {
  const [editing, setEditing] = useState(false)
  const [inputValue, setInputValue] = useState(String(value))
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (editing) {
      inputRef.current?.select()
    }
  }, [editing])

  const handleBlur = () => {
    setEditing(false)
    const newValue = parseFloat(inputValue)
    if (!isNaN(newValue) && newValue !== value) {
      onChange(newValue)
    } else {
      setInputValue(String(value))
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleBlur()
    } else if (e.key === 'Escape') {
      setEditing(false)
      setInputValue(String(value))
    }
  }

  if (editing) {
    return (
      <Input
        ref={inputRef}
        type="number"
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={handleKeyDown}
        className="h-8 w-24"
      />
    )
  }

  return (
    <div
      className="cursor-pointer hover:bg-muted px-2 py-1 rounded"
      onClick={() => setEditing(true)}
    >
      {formatNumber(value)}
    </div>
  )
}
```

### 7. FilterBar 筛选栏

```tsx
// components/FilterBar.tsx
export function FilterBar() {
  const { filters, setFilters } = useDataStore()

  return (
    <div className="flex items-center gap-3 py-3">
      <Select
        value={filters.industryType}
        onValueChange={(v) => setFilters({ industryType: v })}
      >
        <SelectTrigger className="w-32">
          <SelectValue placeholder="行业类型" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部</SelectItem>
          <SelectItem value="wholesale">批发</SelectItem>
          <SelectItem value="retail">零售</SelectItem>
        </SelectContent>
      </Select>

      <Select
        value={filters.isSmallMicro}
        onValueChange={(v) => setFilters({ isSmallMicro: v })}
      >
        <SelectTrigger className="w-24">
          <SelectValue placeholder="小微" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部</SelectItem>
          <SelectItem value="1">是</SelectItem>
          <SelectItem value="0">否</SelectItem>
        </SelectContent>
      </Select>

      <Select
        value={filters.isEatWearUse}
        onValueChange={(v) => setFilters({ isEatWearUse: v })}
      >
        <SelectTrigger className="w-24">
          <SelectValue placeholder="吃穿用" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部</SelectItem>
          <SelectItem value="1">是</SelectItem>
          <SelectItem value="0">否</SelectItem>
        </SelectContent>
      </Select>

      <div className="flex-1">
        <Input
          placeholder="搜索企业名称..."
          value={filters.search}
          onChange={(e) => setFilters({ search: e.target.value })}
          className="max-w-xs"
        />
      </div>
    </div>
  )
}
```

---

## 页面组件

### Dashboard 主页面

```tsx
// pages/Dashboard.tsx
export function Dashboard() {
  const { indicators, hasData, fetchIndicators } = useDataStore()

  useEffect(() => {
    fetchIndicators()
  }, [])

  if (!hasData) {
    return <EmptyState />
  }

  return (
    <div className="flex flex-col h-screen">
      <Topbar />
      <main className="flex-1 overflow-auto p-4 space-y-4">
        <IndicatorCards indicators={indicators} />

        <Tabs defaultValue="wr">
          <TabsList>
            <TabsTrigger value="wr">批发零售</TabsTrigger>
            <TabsTrigger value="ac">住宿餐饮</TabsTrigger>
            <TabsTrigger value="config">配置</TabsTrigger>
          </TabsList>

          <TabsContent value="wr">
            <FilterBar />
            <CompanyTable type="wr" />
          </TabsContent>

          <TabsContent value="ac">
            <FilterBar />
            <CompanyTable type="ac" />
          </TabsContent>

          <TabsContent value="config">
            <ConfigPanel />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}
```

### EmptyState 空数据提示

```tsx
// components/EmptyState.tsx
export function EmptyState() {
  const [importOpen, setImportOpen] = useState(false)

  return (
    <div className="flex flex-col h-screen">
      <Topbar />
      <main className="flex-1 flex items-center justify-center">
        <div className="text-center space-y-4">
          <FileSpreadsheet className="w-16 h-16 mx-auto text-muted-foreground" />
          <h2 className="text-xl font-semibold">暂无数据</h2>
          <p className="text-muted-foreground">
            请导入 Excel 预估表开始使用
          </p>
          <Button onClick={() => setImportOpen(true)}>
            <Upload className="w-4 h-4 mr-2" />
            导入数据
          </Button>
        </div>
      </main>
      <ImportModal open={importOpen} onOpenChange={setImportOpen} />
    </div>
  )
}
```

### ConfigPanel 配置面板

```tsx
// components/ConfigPanel.tsx
export function ConfigPanel() {
  const { config, updateConfig } = useDataStore()

  const configGroups = [
    {
      title: '时间配置',
      items: [
        { key: 'current_year', label: '当前年份', type: 'number' },
        { key: 'current_month', label: '当前月份', type: 'number' },
      ]
    },
    {
      title: '社零额(定) 手工输入',
      items: [
        { key: 'small_micro_rate_month', label: '本月小微增速', type: 'number' },
        { key: 'eat_wear_use_rate_month', label: '本月吃穿用增速', type: 'number' },
        { key: 'last_year_limit_below_cumulative', label: '上年累计限下社零额', type: 'number' },
      ]
    },
    {
      title: '权重设置',
      items: [
        { key: 'weight_small_micro', label: '小微权重', type: 'number' },
        { key: 'weight_eat_wear_use', label: '吃穿用权重', type: 'number' },
        { key: 'weight_sample', label: '抽样权重', type: 'number' },
      ]
    },
  ]

  return (
    <div className="space-y-6 max-w-2xl">
      {configGroups.map(group => (
        <Card key={group.title}>
          <CardHeader>
            <CardTitle className="text-base">{group.title}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {group.items.map(item => (
              <div key={item.key} className="flex items-center justify-between">
                <Label>{item.label}</Label>
                <Input
                  type={item.type}
                  value={config[item.key] || ''}
                  onChange={(e) => updateConfig(item.key, e.target.value)}
                  className="w-32"
                />
              </div>
            ))}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
```

---

## 状态管理

### dataStore

```tsx
// store/dataStore.ts
interface DataState {
  // 数据
  hasData: boolean
  indicators: Record<string, number>
  wrCompanies: WRCompany[]
  acCompanies: ACCompany[]
  config: Record<string, string>

  // 分页
  pagination: { page: number; pageSize: number; total: number }

  // 筛选
  filters: {
    industryType: string
    isSmallMicro: string
    isEatWearUse: string
    search: string
  }

  // Actions
  fetchIndicators: () => Promise<void>
  fetchCompanies: (params?: any) => Promise<void>
  updateCompany: (id: number, data: Partial<Company>) => Promise<void>
  fetchConfig: () => Promise<void>
  updateConfig: (key: string, value: string) => Promise<void>
  setFilters: (filters: Partial<DataState['filters']>) => void
}

export const useDataStore = create<DataState>((set, get) => ({
  hasData: false,
  indicators: {},
  wrCompanies: [],
  acCompanies: [],
  config: {},
  pagination: { page: 1, pageSize: 50, total: 0 },
  filters: {
    industryType: 'all',
    isSmallMicro: 'all',
    isEatWearUse: 'all',
    search: '',
  },

  fetchIndicators: async () => {
    const res = await api.getIndicators()
    set({ indicators: res.data, hasData: res.data.limit_above_retail_month > 0 })
  },

  fetchCompanies: async (params) => {
    const { filters, pagination } = get()
    const res = await api.getCompanies({
      ...filters,
      ...pagination,
      ...params,
    })
    set({
      wrCompanies: res.data.items,
      pagination: { ...pagination, total: res.data.total },
    })
  },

  updateCompany: async (id, data) => {
    await api.updateCompany(id, data)
    // 重新获取指标
    get().fetchIndicators()
    get().fetchCompanies()
  },

  // ...
}))
```

---

## 路由配置

```tsx
// App.tsx
import { Dashboard } from './pages/Dashboard'

function App() {
  // 简化为单页面，无需路由
  return <Dashboard />
}
```

移除原有的 `ProjectHub`、`ImportWizard`、`ProjectDetail` 等页面。
