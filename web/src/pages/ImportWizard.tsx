import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useImportStore, useDataStore } from '@/store'
import { importApi } from '@/services/api'
import type { FieldMapping } from '@/types'
import { Upload, Check, ArrowLeft, HelpCircle, FileSpreadsheet } from 'lucide-react'

// Logo 组件
function Logo() {
  return (
    <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
      <path fillRule="evenodd" clipRule="evenodd" d="M24 4C12.9543 4 4 12.9543 4 24C4 35.0457 12.9543 44 24 44C35.0457 44 44 35.0457 44 24C44 12.9543 35.0457 4 24 4ZM24 38C16.268 38 10 31.732 10 24C10 16.268 16.268 10 24 10C31.732 10 38 16.268 38 24C38 31.732 31.732 38 24 38Z" fill="currentColor"/>
      <path d="M22 18H26V30H22V18Z" fill="currentColor"/>
      <path d="M16 22H20V30H16V22Z" fill="currentColor"/>
      <path d="M28 14H32V30H28V14Z" fill="currentColor"/>
    </svg>
  )
}

// 步骤指示器
function StepIndicator({ currentStep }: { currentStep: number }) {
  const steps = [
    { label: '文件上传', step: 1 },
    { label: '字段映射', step: 2 },
    { label: '生成规则', step: 3 },
    { label: '执行导入', step: 4 },
  ]

  return (
    <div className="flex items-center mb-12">
      {steps.map((s, index) => {
        const isCompleted = currentStep > s.step
        const isActive = currentStep === s.step
        const isLast = index === steps.length - 1

        return (
          <div key={s.step} className={`flex items-center gap-4 border-b-2 pb-4 ${isLast ? 'flex-none pl-4' : 'flex-1'} ${
            isCompleted || isActive ? 'border-primary text-primary' : 'border-gray-300 text-gray-400'
          }`}>
            <div className={`flex items-center justify-center w-8 h-8 rounded-full text-sm font-bold ${
              isCompleted || isActive ? 'bg-primary text-white' : 'bg-gray-200 text-gray-500'
            }`}>
              {isCompleted ? <Check className="w-5 h-5" /> : s.step}
            </div>
            <span className="font-semibold">{s.label}</span>
            {!isLast && <div className={`h-0.5 flex-1 ${isCompleted ? 'bg-primary' : 'bg-gray-300'}`} />}
          </div>
        )
      })}
    </div>
  )
}

// 文件上传步骤
function FileUploadStep() {
  const { fileId, fileName, sheets, selectedSheet, setFileInfo, setSelectedSheet, setStep } = useImportStore()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')

  const handleFileSelect = async (file: File) => {
    setError('')
    setUploading(true)
    try {
      const result = await importApi.upload(file)
      setFileInfo(result.fileId, result.fileName, result.sheets)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    const file = e.dataTransfer.files[0]
    if (file) handleFileSelect(file)
  }

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">1. 文件上传与选择工作表</h2>
      <p className="mt-1 text-sm text-gray-600">请上传包含企业数据的 Excel 文件（.xlsx, .xls），并选择要使用的工作表。</p>

      <div className="mt-6 grid grid-cols-1 gap-8 md:grid-cols-2">
        <div>
          <label className="mb-2 block text-sm font-medium text-gray-700">上传文件</label>
          <div
            className="flex flex-col items-center justify-center w-full h-48 border-2 border-gray-300 border-dashed rounded-lg cursor-pointer bg-gray-50 hover:bg-gray-100 transition"
            onClick={() => fileInputRef.current?.click()}
            onDrop={handleDrop}
            onDragOver={(e) => e.preventDefault()}
          >
            <div className="flex flex-col items-center justify-center pt-5 pb-6">
              <Upload className="w-10 h-10 text-gray-500 mb-2" />
              <p className="my-2 text-sm text-gray-500">
                <span className="font-semibold">点击上传</span> 或拖拽文件到此区域
              </p>
              <p className="text-xs text-gray-500">支持 XLS, XLSX (最大 10MB)</p>
            </div>
            <input
              ref={fileInputRef}
              type="file"
              className="hidden"
              accept=".xlsx,.xls"
              onChange={(e) => e.target.files?.[0] && handleFileSelect(e.target.files[0])}
            />
          </div>
          {uploading && <p className="mt-2 text-sm text-primary">上传中...</p>}
          {error && <p className="mt-2 text-sm text-error">{error}</p>}
        </div>

        <div>
          <label className="mb-2 block text-sm font-medium text-gray-700">选择工作表</label>
          {fileId ? (
            <>
              <div className="flex items-center gap-3">
                <FileSpreadsheet className="w-5 h-5 text-primary" />
                <select
                  className="flex-1 rounded-md border-gray-300 shadow-sm focus:border-primary focus:ring-primary text-sm"
                  value={selectedSheet || ''}
                  onChange={(e) => setSelectedSheet(e.target.value)}
                >
                  {sheets.map((s) => (
                    <option key={s.name} value={s.name}>
                      {s.name} ({s.rowCount} 行)
                    </option>
                  ))}
                </select>
              </div>
              <div className="mt-4 rounded-md bg-green-50 p-4 border border-green-200">
                <div className="flex">
                  <Check className="w-5 h-5 text-green-500 flex-shrink-0" />
                  <div className="ml-3">
                    <h3 className="text-sm font-medium text-green-800">文件已成功上传</h3>
                    <div className="mt-2 text-sm text-green-700">
                      <p>文件名: <span className="font-mono">{fileName}</span></p>
                    </div>
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="flex items-center justify-center h-32 border border-gray-200 rounded-lg bg-gray-50">
              <p className="text-sm text-gray-400">请先上传文件</p>
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

// 字段映射步骤
function FieldMappingStep() {
  const { fileId, selectedSheet, columns, previewRows, mapping, setColumns, setMapping, setStep } = useImportStore()
  const [loading, setLoading] = useState(false)

  useState(() => {
    if (fileId && selectedSheet && columns.length === 0) {
      setLoading(true)
      importApi.getColumns(fileId, selectedSheet)
        .then((result) => setColumns(result.columns, result.previewRows))
        .finally(() => setLoading(false))
    }
  })

  const systemFields: { key: keyof FieldMapping; label: string; required: boolean; description: string }[] = [
    { key: 'companyName', label: '企业名称', required: true, description: '企业的唯一官方名称' },
    { key: 'creditCode', label: '统一社会信用代码', required: false, description: '18位法人和其他组织身份代码' },
    { key: 'industryCode', label: '行业代码', required: false, description: '国民经济行业分类代码' },
    { key: 'companyScale', label: '单位规模', required: false, description: '企业规模（1/2/3/4）' },
    { key: 'retailCurrentMonth', label: '本期零售额', required: true, description: '报告期内的零售总额' },
    { key: 'retailLastYearMonth', label: '上年同期零售额', required: true, description: '去年同期的零售总额' },
    { key: 'retailCurrentCumulative', label: '本年累计零售额', required: false, description: '今年1月至当月累计' },
    { key: 'retailLastYearCumulative', label: '上年累计零售额', required: false, description: '去年1月至当月累计' },
    { key: 'salesCurrentMonth', label: '本期销售额', required: false, description: '报告期内的销售总额' },
    { key: 'salesLastYearMonth', label: '上年同期销售额', required: false, description: '去年同期的销售总额' },
  ]

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">2. 配置关键数据字段映射</h2>
      <p className="mt-1 text-sm text-gray-600">请将您的 Excel 列名与系统所需的字段进行匹配，确保数据正确导入。</p>

      {loading ? (
        <div className="mt-6 text-center py-8">
          <p className="text-gray-500">加载列信息中...</p>
        </div>
      ) : (
        <div className="mt-6 overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-300">
            <thead>
              <tr>
                <th className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900">系统字段</th>
                <th className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">描述</th>
                <th className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900">Excel 列名</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {systemFields.map((field) => (
                <tr key={field.key}>
                  <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900">
                    {field.label} {field.required && <span className="text-red-500">*</span>}
                  </td>
                  <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500">{field.description}</td>
                  <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
                    <select
                      className="w-full rounded-md border-gray-300 shadow-sm focus:border-primary focus:ring-primary text-sm"
                      value={mapping[field.key]}
                      onChange={(e) => setMapping(field.key, e.target.value)}
                    >
                      <option value="">-- 选择列 --</option>
                      {columns.map((col) => (
                        <option key={col} value={col}>{col}</option>
                      ))}
                    </select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

// 生成规则步骤
function GenerationRulesStep() {
  const { generateHistory, currentMonth, setGenerateHistory, setCurrentMonth } = useImportStore()

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">3. 配置历史数据生成规则</h2>
      <p className="mt-1 text-sm text-gray-600">对于新入库的企业，系统可以自动生成模拟的历史数据。</p>

      <div className="mt-6 space-y-6">
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="generateHistory"
            className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            checked={generateHistory}
            onChange={(e) => setGenerateHistory(e.target.checked)}
          />
          <label htmlFor="generateHistory" className="text-sm font-medium text-gray-700">
            为缺失历史数据的企业自动生成模拟数据
          </label>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">当前操作月份</label>
          <select
            className="w-48 rounded-md border-gray-300 shadow-sm focus:border-primary focus:ring-primary text-sm"
            value={currentMonth}
            onChange={(e) => setCurrentMonth(parseInt(e.target.value))}
          >
            {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => (
              <option key={m} value={m}>{m} 月</option>
            ))}
          </select>
        </div>
      </div>
    </section>
  )
}

// 执行导入步骤
function ImportExecuteStep() {
  const { fileId, selectedSheet, mapping, generateHistory, currentMonth, importing, importResult, setImporting, setImportResult } = useImportStore()
  const { setIndicators, fetchCompanies } = useDataStore()
  const navigate = useNavigate()

  const handleImport = async () => {
    if (!fileId || !selectedSheet) return

    // 先设置映射
    await importApi.setMapping(fileId, selectedSheet, mapping)

    setImporting(true)
    try {
      const result = await importApi.execute(fileId, selectedSheet, generateHistory, currentMonth)
      setImportResult(result)
      setIndicators(result.indicators)
      await fetchCompanies()
    } catch (e) {
      alert('导入失败：' + (e as Error).message)
    } finally {
      setImporting(false)
    }
  }

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-8 shadow-sm">
      <h2 className="text-lg font-semibold text-gray-900">4. 执行导入</h2>
      <p className="mt-1 text-sm text-gray-600">确认配置无误后，点击下方按钮开始导入数据。</p>

      <div className="mt-6">
        {importResult ? (
          <div className="rounded-md bg-green-50 p-6 border border-green-200">
            <div className="flex items-start">
              <Check className="w-6 h-6 text-green-500 flex-shrink-0" />
              <div className="ml-4">
                <h3 className="text-lg font-medium text-green-800">导入成功！</h3>
                <div className="mt-2 text-sm text-green-700 space-y-1">
                  <p>成功导入 <strong>{importResult.importedCount}</strong> 家企业数据</p>
                  {importResult.generatedHistoryCount > 0 && (
                    <p>自动生成 <strong>{importResult.generatedHistoryCount}</strong> 家企业的历史数据</p>
                  )}
                </div>
                <button
                  className="mt-4 px-4 py-2 bg-primary text-white rounded-lg font-medium hover:bg-primary/90 transition-colors"
                  onClick={() => navigate('/')}
                >
                  进入主面板
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="text-center py-8">
            <button
              className="px-6 py-3 bg-primary text-white rounded-lg font-medium hover:bg-primary/90 transition-colors disabled:opacity-50"
              onClick={handleImport}
              disabled={importing || !fileId}
            >
              {importing ? '导入中...' : '开始导入'}
            </button>
          </div>
        )}
      </div>
    </section>
  )
}

// 导入向导页面
export default function ImportWizard() {
  const navigate = useNavigate()
  const { step, fileId, setStep, reset } = useImportStore()

  const canGoNext = () => {
    switch (step) {
      case 1: return !!fileId
      case 2: return true
      case 3: return true
      default: return false
    }
  }

  const handleBack = () => {
    if (step === 1) {
      navigate('/')
    } else {
      setStep(step - 1)
    }
  }

  return (
    <div className="flex h-screen w-full flex-col">
      {/* Header */}
      <header className="flex flex-none items-center justify-between border-b border-gray-200 bg-white px-8 py-4">
        <div className="flex items-center gap-4">
          <Logo />
          <h1 className="text-xl font-semibold text-gray-900">数据导入与配置</h1>
        </div>
        <div className="flex items-center gap-3">
          <button className="flex items-center gap-2 rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 transition">
            <HelpCircle className="w-4 h-4" />
            帮助文档
          </button>
          <button
            className="flex items-center gap-2 rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 transition"
            onClick={() => { reset(); navigate('/') }}
          >
            <ArrowLeft className="w-4 h-4" />
            返回主面板
          </button>
        </div>
      </header>

      {/* Main */}
      <main className="flex-1 overflow-y-auto bg-white">
        <div className="mx-auto max-w-5xl px-4 py-10">
          <StepIndicator currentStep={step} />

          <div className="space-y-10">
            {step === 1 && <FileUploadStep />}
            {step === 2 && <FieldMappingStep />}
            {step === 3 && <GenerationRulesStep />}
            {step === 4 && <ImportExecuteStep />}
          </div>

          {/* Navigation */}
          <div className="mt-12 flex justify-end gap-4 border-t border-gray-200 pt-6">
            <button
              className="rounded-lg bg-gray-200 px-6 py-2.5 text-sm font-semibold text-gray-700 hover:bg-gray-300 transition"
              onClick={handleBack}
            >
              {step === 1 ? '取消' : '上一步'}
            </button>
            {step < 4 && (
              <button
                className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-primary/90 transition disabled:opacity-50"
                onClick={() => setStep(step + 1)}
                disabled={!canGoNext()}
              >
                下一步
              </button>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
