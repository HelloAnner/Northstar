import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Upload, Database } from 'lucide-react'
import ImportDialog from '@/components/ImportDialog'

interface Indicator {
  id: string
  name: string
  value: number
  unit: string
}

interface IndicatorGroup {
  name: string
  indicators: Indicator[]
}

interface SystemStatus {
  initialized: boolean
  currentYear: number
  currentMonth: number
  totalCompanies: number
  wrCount: number
  acCount: number
}

export default function DashboardV3() {
  const [status, setStatus] = useState<SystemStatus | null>(null)
  const [groups, setGroups] = useState<IndicatorGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [showImportDialog, setShowImportDialog] = useState(false)

  // 加载系统状态
  const loadStatus = async () => {
    try {
      const res = await fetch('/api/status')
      const data = await res.json()
      setStatus(data)
    } catch (err) {
      console.error('Failed to load status:', err)
    }
  }

  // 加载指标数据
  const loadIndicators = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/indicators')
      const data = await res.json()
      setGroups(data.groups || [])
    } catch (err) {
      console.error('Failed to load indicators:', err)
    } finally {
      setLoading(false)
    }
  }

  // 初始加载
  useEffect(() => {
    loadStatus()
    loadIndicators()
  }, [])

  // 导入完成回调
  const handleImportSuccess = () => {
    setShowImportDialog(false)
    loadStatus()
    loadIndicators()
  }

  // 空状态
  if (status && !status.initialized) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center space-y-4">
          <Database className="w-16 h-16 mx-auto text-gray-400" />
          <h2 className="text-2xl font-bold text-gray-700">暂无数据</h2>
          <p className="text-gray-500">请先导入 Excel 数据文件</p>
          <Button onClick={() => setShowImportDialog(true)} size="lg">
            <Upload className="w-4 h-4 mr-2" />
            导入数据
          </Button>
        </div>

        {showImportDialog && (
          <ImportDialog
            open={showImportDialog}
            onClose={() => setShowImportDialog(false)}
            onSuccess={handleImportSuccess}
          />
        )}
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* 顶部状态栏 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">数据概览</h1>
          {status && (
            <p className="text-gray-500 mt-1">
              {status.currentYear}年{status.currentMonth}月 · 共 {status.totalCompanies} 家企业
              （批零 {status.wrCount} + 住餐 {status.acCount}）
            </p>
          )}
        </div>
        <Button onClick={() => setShowImportDialog(true)}>
          <Upload className="w-4 h-4 mr-2" />
          导入数据
        </Button>
      </div>

      {/* 指标展示 */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[...Array(16)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-4 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-24 mb-2" />
                <Skeleton className="h-3 w-16" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="space-y-8">
          {groups.map((group, groupIndex) => (
            <div key={groupIndex}>
              <h2 className="text-xl font-semibold mb-4 text-gray-700">
                {group.name}
              </h2>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                {group.indicators.map((indicator) => (
                  <IndicatorCard key={indicator.id} indicator={indicator} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 导入弹窗 */}
      {showImportDialog && (
        <ImportDialog
          open={showImportDialog}
          onClose={() => setShowImportDialog(false)}
          onSuccess={handleImportSuccess}
        />
      )}
    </div>
  )
}

// 指标卡片组件
function IndicatorCard({ indicator }: { indicator: Indicator }) {
  const isRate = indicator.unit === '%'
  const isPositive = indicator.value >= 0

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-gray-600">
          {indicator.name}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-1">
          <div className="flex items-baseline gap-1">
            <span className="text-3xl font-bold">
              {formatNumber(indicator.value)}
            </span>
            <span className="text-sm text-gray-500">{indicator.unit}</span>
          </div>
          {isRate && (
            <div
              className={`text-xs font-medium ${
                isPositive ? 'text-green-600' : 'text-red-600'
              }`}
            >
              {isPositive ? '↑' : '↓'} {Math.abs(indicator.value).toFixed(2)}%
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// 数字格式化
function formatNumber(value: number): string {
  if (Math.abs(value) >= 10000) {
    return (value / 10000).toFixed(2) + '万'
  } else if (Math.abs(value) >= 1000) {
    return (value / 1000).toFixed(2) + 'k'
  } else {
    return value.toFixed(2)
  }
}
