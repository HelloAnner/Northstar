import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useDataStore } from '@/store'
import { formatPercent, formatCurrency, debounce } from '@/lib/utils'
import { optimizeApi, exportApi } from '@/services/api'
import type { IndustryType } from '@/types'
import {
  Download,
  Upload,
  RotateCcw,
  Sparkles,
  Search,
  Filter,
  ArrowUpDown,
} from 'lucide-react'

// Logo 组件
function Logo() {
  return (
    <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
      <path fillRule="evenodd" clipRule="evenodd" d="M12.0799 24L4 19.2479L9.95537 8.75216L18.04 13.4961L18.0446 4H29.9554L29.96 13.4961L38.0446 8.75216L44 19.2479L35.92 24L44 28.7521L38.0446 39.2479L29.96 34.5039L29.9554 44H18.0446L18.04 34.5039L9.95537 39.2479L4 28.7521L12.0799 24Z" fill="currentColor"/>
    </svg>
  )
}

// 指标面板组件
function IndicatorPanel({ title, children, className = '' }: { title: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={`bg-white border border-border-color rounded-xl p-4 flex flex-col gap-3 shadow-sm ${className}`}>
      <h3 className="font-semibold text-text-primary text-base">{title}</h3>
      {children}
    </div>
  )
}

// 指标行组件
function IndicatorRow({ label, value, editable = false, suffix = '', onChange }: {
  label: string
  value: string | number
  editable?: boolean
  suffix?: string
  onChange?: (value: number) => void
}) {
  const displayValue = typeof value === 'number' ? formatCurrency(value) : value

  return (
    <div className="flex justify-between items-center text-sm">
      <p className="text-text-secondary">{label}</p>
      {editable ? (
        <input
          type="text"
          className="w-24 text-right bg-slate-100 border-transparent rounded p-1 text-text-primary font-semibold focus:ring-1 focus:ring-primary focus:outline-none hover:border-border-color transition-colors"
          defaultValue={displayValue}
          onChange={(e) => onChange?.(parseFloat(e.target.value.replace(/,/g, '')) || 0)}
        />
      ) : (
        <div className="w-24 text-right bg-slate-100/60 border-transparent rounded p-1 text-text-primary font-semibold">
          {displayValue}{suffix}
        </div>
      )}
    </div>
  )
}

// 行业列组件
function IndustryColumn({ name, monthRate, cumulativeRate }: {
  name: string
  monthRate: number
  cumulativeRate: number
}) {
  return (
    <div className="flex flex-col justify-around gap-1">
      <p className="font-medium text-sm text-text-primary text-center">{name}</p>
      <div className="flex items-center gap-1.5 text-xs">
        <label className="text-text-secondary">当月:</label>
        <div className="flex-1 min-w-0 text-right bg-slate-100 border-transparent rounded p-1 text-text-primary font-semibold text-sm">
          {formatPercent(monthRate)}
        </div>
      </div>
      <div className="flex items-center gap-1.5 text-xs">
        <label className="text-text-secondary">累计:</label>
        <div className="flex-1 min-w-0 text-right bg-slate-100 border-transparent rounded p-1 text-text-primary font-semibold text-sm">
          {formatPercent(cumulativeRate)}
        </div>
      </div>
    </div>
  )
}

// 企业表格组件
function CompanyTable() {
  const {
    companies,
    totalCompanies,
    currentPage,
    pageSize,
    searchKeyword,
    loading,
    fetchCompanies,
    updateCompanyRetail,
    setSearchKeyword,
    setPage,
  } = useDataStore()

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')

  useEffect(() => {
    fetchCompanies()
  }, [currentPage, searchKeyword, fetchCompanies])

  const handleSearch = useCallback(
    debounce((value: string) => {
      setSearchKeyword(value)
    }, 300),
    [setSearchKeyword]
  )

  const handleEditStart = (id: string, value: number) => {
    setEditingId(id)
    setEditValue(formatCurrency(value))
  }

  const handleEditEnd = async (id: string) => {
    const numValue = parseFloat(editValue.replace(/,/g, '')) || 0
    await updateCompanyRetail(id, numValue)
    setEditingId(null)
  }

  const totalPages = Math.ceil(totalCompanies / pageSize)

  return (
    <div className="flex-1 flex flex-col bg-white rounded-xl border border-border-color overflow-hidden shadow-md">
      <div className="p-6 border-b border-border-color flex justify-between items-center">
        <h3 className="text-lg font-bold text-text-primary">企业数据微调</h3>
        <div className="flex items-center gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary w-4 h-4 pointer-events-none" />
            <input
              type="text"
              className="w-64 bg-background border border-border-color rounded-lg h-10 pl-9 pr-3 text-sm text-text-primary placeholder:text-text-tertiary focus:ring-1 focus:ring-primary focus:outline-none"
              placeholder="按企业名称搜索..."
              onChange={(e) => handleSearch(e.target.value)}
            />
          </div>
          <button className="flex items-center gap-2 h-10 px-4 rounded-lg bg-slate-200 text-sm text-text-primary font-medium hover:bg-slate-300 transition-colors">
            <Filter className="w-4 h-4" />
            筛选
          </button>
          <button className="flex items-center gap-2 h-10 px-4 rounded-lg bg-slate-200 text-sm text-text-primary font-medium hover:bg-slate-300 transition-colors">
            <ArrowUpDown className="w-4 h-4" />
            排序
          </button>
        </div>
      </div>

      <div className="overflow-x-auto flex-1">
        {loading ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-text-secondary">加载中...</div>
          </div>
        ) : companies.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-2">
            <p className="text-text-secondary">暂无数据，请先导入数据</p>
          </div>
        ) : (
          <table className="w-full text-sm text-left text-text-secondary">
            <thead className="text-xs text-text-secondary uppercase bg-background">
              <tr>
                <th className="px-6 py-3">企业名称</th>
                <th className="px-6 py-3 text-right">总销售额</th>
                <th className="px-6 py-3 text-right">本期零售额</th>
                <th className="px-6 py-3 text-right">同期零售额</th>
                <th className="px-6 py-3 text-right">增速</th>
              </tr>
            </thead>
            <tbody>
              {companies.map((company) => {
                const isEditing = editingId === company.id
                const hasError = company.validation?.hasError
                const growthRate = company.monthGrowthRate || 0
                const isPositive = growthRate >= 0

                return (
                  <tr key={company.id} className="border-b border-border-color">
                    <th className="px-6 py-4 font-medium text-text-primary whitespace-nowrap">
                      {company.name}
                    </th>
                    <td className="px-6 py-4 text-right">
                      <div className="inline-block w-32 text-right bg-slate-100/60 border-transparent rounded-md p-2 text-text-primary font-semibold">
                        {formatCurrency(company.salesCurrentMonth)}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-right">
                      {isEditing ? (
                        <input
                          type="text"
                          className={`w-32 text-right rounded-md p-2 font-semibold focus:ring-1 focus:outline-none ${
                            hasError
                              ? 'bg-error/10 border border-error/50 text-error focus:ring-error'
                              : 'bg-primary/10 border border-primary/50 text-primary focus:ring-primary'
                          }`}
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          onBlur={() => handleEditEnd(company.id)}
                          onKeyDown={(e) => e.key === 'Enter' && handleEditEnd(company.id)}
                          autoFocus
                        />
                      ) : (
                        <input
                          type="text"
                          className={`w-32 text-right rounded-md p-2 font-semibold cursor-pointer transition-colors ${
                            hasError
                              ? 'bg-error/10 border border-error/50 text-error'
                              : company.retailCurrentMonth !== company.salesCurrentMonth
                              ? 'bg-primary/10 border border-primary/50 text-primary'
                              : 'bg-slate-100 border border-transparent text-text-primary hover:border-border-color'
                          }`}
                          value={formatCurrency(company.retailCurrentMonth)}
                          onClick={() => handleEditStart(company.id, company.retailCurrentMonth)}
                          readOnly
                        />
                      )}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="inline-block w-32 text-right bg-slate-100/60 border-transparent rounded-md p-2 text-text-primary font-semibold">
                        {formatCurrency(company.retailLastYearMonth)}
                      </div>
                    </td>
                    <td className={`px-6 py-4 text-right font-semibold ${isPositive ? 'text-success' : 'text-error'}`}>
                      {formatPercent(growthRate)}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {totalPages > 1 && (
        <div className="p-4 border-t border-border-color flex justify-between items-center">
          <span className="text-sm text-text-secondary">
            共 {totalCompanies} 条数据
          </span>
          <div className="flex gap-2">
            <button
              className="px-3 py-1 rounded border border-border-color text-sm disabled:opacity-50"
              disabled={currentPage === 1}
              onClick={() => setPage(currentPage - 1)}
            >
              上一页
            </button>
            <span className="px-3 py-1 text-sm">
              {currentPage} / {totalPages}
            </span>
            <button
              className="px-3 py-1 rounded border border-border-color text-sm disabled:opacity-50"
              disabled={currentPage === totalPages}
              onClick={() => setPage(currentPage + 1)}
            >
              下一页
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// 主面板页面
export default function Dashboard() {
  const navigate = useNavigate()
  const { indicators, fetchIndicators, resetCompanies } = useDataStore()
  const [optimizing, setOptimizing] = useState(false)

  useEffect(() => {
    fetchIndicators()
  }, [fetchIndicators])

  const handleOptimize = async () => {
    const target = prompt('请输入目标累计增速（如 0.075 表示 7.5%）：', '0.075')
    if (!target) return

    setOptimizing(true)
    try {
      await optimizeApi.run(parseFloat(target))
      await fetchIndicators()
      alert('智能调整完成！')
    } catch (e) {
      alert('调整失败：' + (e as Error).message)
    } finally {
      setOptimizing(false)
    }
  }

  const handleReset = async () => {
    if (!confirm('确定要重置所有企业数据吗？')) return
    await resetCompanies()
  }

  const handleExport = async () => {
    try {
      const result = await exportApi.export({
        format: 'xlsx',
        includeIndicators: true,
        includeChanges: true,
      })
      window.open(result.downloadUrl, '_blank')
    } catch (e) {
      alert('导出失败：' + (e as Error).message)
    }
  }

  const industryNames: Record<IndustryType, string> = {
    wholesale: '批发业',
    retail: '零售业',
    accommodation: '住宿业',
    catering: '餐饮业',
  }

  return (
    <div className="flex h-screen w-full flex-col">
      {/* Header */}
      <header className="flex flex-none items-center justify-between whitespace-nowrap border-b border-border-color bg-white/80 px-8 py-3 backdrop-blur-sm sticky top-0 z-10 shadow-sm">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-3">
            <Logo />
            <h1 className="text-text-primary text-base font-medium">数据管理与模拟平台</h1>
          </div>
          <div className="w-px h-8 bg-border-color"></div>
          <div className="flex flex-col gap-1 items-start">
            <p className="text-text-secondary text-xs font-medium">限上社零额增速 (当月)</p>
            <div className="w-28 text-center bg-background border-transparent rounded-md py-1 text-text-primary font-semibold text-lg">
              {formatPercent(indicators.limitAboveMonthRate)}
            </div>
          </div>
          <div className="flex flex-col gap-1 items-start">
            <p className="text-text-secondary text-xs font-medium">限上社零额增速 (累计)</p>
            <div className="w-28 text-center bg-background border-transparent rounded-md py-1 text-text-primary font-semibold text-lg">
              {formatPercent(indicators.limitAboveCumulativeRate)}
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-3">
          <button
            className="flex items-center gap-2 h-10 px-4 rounded-lg bg-primary/20 text-primary text-sm font-bold hover:bg-primary/30 transition-colors"
            onClick={handleOptimize}
            disabled={optimizing}
          >
            <Sparkles className="w-4 h-4" />
            {optimizing ? '调整中...' : '智能调整'}
          </button>
          <button
            className="flex items-center gap-2 h-10 px-4 rounded-lg bg-slate-200 text-text-primary text-sm font-bold hover:bg-slate-300 transition-colors"
            onClick={handleReset}
          >
            <RotateCcw className="w-4 h-4" />
            重置
          </button>
          <div className="w-px h-6 bg-border-color"></div>
          <button
            className="flex items-center gap-2 h-10 px-4 rounded-lg bg-slate-200 text-text-primary text-sm font-bold hover:bg-slate-300 transition-colors"
            onClick={() => navigate('/import')}
          >
            <Upload className="w-4 h-4" />
            导入
          </button>
          <button
            className="flex items-center gap-2 h-10 px-4 rounded-lg bg-primary text-white text-sm font-bold hover:bg-primary/90 transition-colors"
            onClick={handleExport}
          >
            <Download className="w-4 h-4" />
            导出
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 flex-col overflow-y-auto">
        <div className="flex-1 p-8">
          <div className="flex flex-col gap-6">
            {/* Indicator Panels */}
            <div className="grid grid-cols-12 gap-4">
              {/* 限上社零额 */}
              <IndicatorPanel title="限上社零额" className="col-span-3">
                <IndicatorRow label="当期零售额 (万元)" value={indicators.limitAboveMonthValue} />
                <IndicatorRow label="当月增速 (%)" value={formatPercent(indicators.limitAboveMonthRate)} />
                <IndicatorRow label="累计零售额 (万元)" value={indicators.limitAboveCumulativeValue} />
                <IndicatorRow label="累计增速 (%)" value={formatPercent(indicators.limitAboveCumulativeRate)} />
              </IndicatorPanel>

              {/* 专项增速 */}
              <IndicatorPanel title="专项增速" className="col-span-2">
                <div className="flex flex-col gap-4 text-sm flex-grow justify-around">
                  <div>
                    <p className="text-text-secondary mb-1">吃穿用增速 (当月)</p>
                    <div className="w-full text-right bg-slate-100 border-transparent rounded p-1 text-text-primary font-semibold">
                      {formatPercent(indicators.eatWearUseMonthRate)}
                    </div>
                  </div>
                  <div>
                    <p className="text-text-secondary mb-1">小微企业增速 (当月)</p>
                    <div className="w-full text-right bg-slate-100 border-transparent rounded p-1 text-text-primary font-semibold">
                      {formatPercent(indicators.microSmallMonthRate)}
                    </div>
                  </div>
                </div>
              </IndicatorPanel>

              {/* 四大行业增速 */}
              <IndicatorPanel title="四大行业增速" className="col-span-4">
                <div className="grid grid-cols-4 gap-x-4 h-full">
                  {(['wholesale', 'retail', 'accommodation', 'catering'] as IndustryType[]).map((type) => (
                    <IndustryColumn
                      key={type}
                      name={industryNames[type]}
                      monthRate={indicators.industryRates[type]?.monthRate || 0}
                      cumulativeRate={indicators.industryRates[type]?.cumulativeRate || 0}
                    />
                  ))}
                </div>
              </IndicatorPanel>

              {/* 社零总额估算 */}
              <IndicatorPanel title="社零总额估算" className="col-span-3">
                <IndicatorRow label="社零总额 (估算)" value={indicators.totalSocialCumulativeValue} />
                <div className="h-full flex flex-col justify-around">
                  <IndicatorRow label="累计增速 (%)" value={formatPercent(indicators.totalSocialCumulativeRate)} />
                </div>
              </IndicatorPanel>
            </div>

            {/* Company Table */}
            <CompanyTable />
          </div>
        </div>
      </main>
    </div>
  )
}
