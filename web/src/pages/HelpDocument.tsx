import { useMemo, useState } from 'react'
import { cn } from '@/lib/utils'

const catalog = ['快速入门', '数据导入', '指标计算', '智能调整', '常见问题'] as const

export default function HelpDocument() {
  const [active, setActive] = useState<(typeof catalog)[number]>('快速入门')

  const content = useMemo(() => {
    switch (active) {
      case '数据导入':
        return {
          title: '数据导入',
          lines: [
            '1. 点击“导入”进入导入页面，选择 Excel 文件后开始导入。',
            '2. 系统自动识别批发、零售、住宿、餐饮相关 Sheet。',
            '3. 导入完成后会记录每个 Sheet 状态与详细日志。',
          ],
        }
      case '指标计算':
        return {
          title: '指标计算',
          lines: [
            '1. 所有指标计算都在当前年月口径下执行。',
            '2. 支持指标定义落库，可维护公式、浮动范围和显示顺序。',
            '3. 修改企业数据后，系统自动重算并刷新指标卡片。',
          ],
        }
      case '智能调整':
        return {
          title: '智能调整',
          lines: [
            '1. 在指标卡输入目标值后，点击“智能调整”触发反推。',
            '2. 调整结果会遵循规则中心中的默认规则和约束。',
            '3. 无法调整时会输出统一提示，说明原因与建议。',
          ],
        }
      case '常见问题':
        return {
          title: '常见问题',
          lines: [
            '1. 看不到数据：请确认当前月份已导入数据。',
            '2. 指标不更新：检查企业编辑是否保存成功。',
            '3. 导出失败：检查模板路径与导出接口日志。',
          ],
        }
      default:
        return {
          title: '快速入门',
          lines: [
            'Northstar 是一个经济数据统计与分析平台，支持批发、零售、住宿、餐饮四大行业的数据管理。',
            '主要功能',
            '1. 数据导入：支持 Excel 批量导入企业数据',
            '2. 指标计算：自动计算 16 项核心指标',
            '3. 智能调整：根据目标值反推企业数据',
            '4. 数据导出：生成标准化报表',
          ],
        }
    }
  }, [active])

  return (
    <div className="min-h-full bg-[#F8FAFC] px-8 py-6">
      <div className="space-y-5">
        <header>
          <h1 className="text-[20px] font-semibold leading-[28px] text-[#0F172A]">帮助文档</h1>
          <p className="mt-0.5 text-[13px] text-[#64748B]">系统使用指南与常见问题</p>
        </header>

        <div className="grid min-h-[calc(100vh-190px)] grid-cols-[200px_1fr] gap-5">
          <aside className="space-y-2">
            <div className="text-sm font-semibold text-[#0F172A]">目录</div>
            <div className="space-y-2">
              {catalog.map((item) => {
                const activeItem = item === active
                return (
                  <button
                    key={item}
                    onClick={() => setActive(item)}
                    className={cn(
                      'flex h-9 w-full items-center rounded-md px-3 text-left text-sm transition-colors',
                      activeItem
                        ? 'bg-[#EFF6FF] text-[#3B82F6]'
                        : 'text-[#64748B] hover:bg-[#F8FAFC] hover:text-[#334155]',
                    )}
                  >
                    {item}
                  </button>
                )
              })}
            </div>
          </aside>

          <article className="rounded-xl border border-[#E2E8F0] bg-white p-6">
            <h2 className="text-[18px] font-semibold text-[#0F172A]">{content.title}</h2>
            <div className="mt-4 space-y-2 text-sm leading-7 text-[#475569]">
              {content.lines.map((line) => (
                <p key={line}>{line}</p>
              ))}
            </div>
          </article>
        </div>
      </div>
    </div>
  )
}
