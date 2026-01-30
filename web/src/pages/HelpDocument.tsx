import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { useNavigate } from 'react-router-dom'
import { useState } from 'react'

export default function HelpDocument() {
  const navigate = useNavigate()
  const [activeSection, setActiveSection] = useState<'flow' | 'indicator' | 'linkage'>('flow')

  const sections = [
    { id: 'flow' as const, title: '推荐流程', desc: '导入与使用的最佳路径' },
    { id: 'indicator' as const, title: '指标含义', desc: '企业字段与指标口径' },
    { id: 'linkage' as const, title: '联动逻辑', desc: '指标计算与保存机制' },
  ]

  return (
    <div className="min-h-screen bg-[#1A1A1A] text-white">
      {/* 顶部栏 */}
      <div className="flex h-14 items-center justify-between border-b border-white/10 px-6">
        <div className="text-sm text-[#D4D4D4]">帮助文档</div>
        <Button
          variant="outline"
          className="border-white/10 bg-[#2D2D2D] text-white hover:bg-white/10"
          onClick={() => navigate('/')}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          返回首页
        </Button>
      </div>

      {/* 内容区域 */}
      <div className="w-full px-6 py-6">
        <div className="mx-auto w-full max-w-6xl">
          <div className="flex flex-col gap-6 lg:flex-row">
            <aside className="w-full lg:w-64">
              <div className="sticky top-20 space-y-2 rounded-xl border border-white/10 bg-[#0D0D0D] p-3">
                <div className="px-2 text-xs font-semibold text-[#A3A3A3]">章节导航</div>
                {sections.map((section) => {
                  const active = section.id === activeSection
                  return (
                    <Button
                      key={section.id}
                      variant="ghost"
                      className={`h-auto w-full items-start justify-start rounded-lg px-3 py-2 text-left ${
                        active ? 'bg-white/10 text-white' : 'text-[#D4D4D4] hover:bg-white/5'
                      }`}
                      onClick={() => setActiveSection(section.id)}
                    >
                      <div>
                        <div className="text-sm font-semibold">{section.title}</div>
                        <div className="text-xs text-[#A3A3A3]">{section.desc}</div>
                      </div>
                    </Button>
                  )
                })}
              </div>
            </aside>

            <main className="min-w-0 flex-1">
              <div className="mb-6">
                <h1 className="text-2xl font-bold">帮助文档</h1>
                <p className="mt-1 text-[#D4D4D4]">系统使用指南与指标计算规则</p>
              </div>

              {activeSection === 'flow' && (
                <Card className="mb-6 border-white/10 bg-[#0D0D0D] p-6">
                  <h2 className="text-lg font-semibold">推荐流程</h2>
                  <p className="mt-1 text-sm text-[#D4D4D4]">导入 → 仪表盘调整 → 导出</p>

                  <div className="mt-6 space-y-4">
                    {[
                      { n: 1, title: '导入 Excel 数据', desc: '上传企业数据 Excel，系统会自动识别表格结构并导入数据。' },
                      { n: 2, title: '在仪表盘查看与调整', desc: '查看16项指标和企业数据；所有数据均可编辑，任意字段变更后会自动重新计算全部指标并保存。' },
                      { n: 3, title: '导出数据', desc: '导出包含企业数据与指标汇总的 Excel 文件。' },
                    ].map((it) => (
                      <div key={it.n} className="flex items-start gap-4">
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-[#FF6B35] text-sm font-bold text-black">
                          {it.n}
                        </div>
                        <div>
                          <div className="font-semibold">{it.title}</div>
                          <div className="text-sm text-[#D4D4D4]">{it.desc}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </Card>
              )}

              {activeSection === 'indicator' && (
                <Card className="mb-6 border-white/10 bg-[#0D0D0D] p-6">
                  <h2 className="text-lg font-semibold">指标含义（企业层）</h2>
                  <p className="mt-1 text-sm text-[#D4D4D4]">仪表盘「企业数据微调」表格字段口径与约束</p>

                  <div className="mt-5 space-y-3 text-sm text-[#D4D4D4]">
                    <div>
                      <span className="font-semibold text-white">企业名称</span>：企业展示名称；支持修改（影响排序与检索）。
                    </div>
                    <div>
                      <span className="font-semibold text-white">总销售额（本期）</span>：企业当期销售/营业额（单位：万元）。
                    </div>
                    <div>
                      <span className="font-semibold text-white">本期零售额</span>：企业当期零售额（单位：万元）。
                    </div>
                    <div>
                      <span className="font-semibold text-white">同期零售额</span>：上年同期零售额（单位：万元）。
                    </div>
                    <div>
                      <span className="font-semibold text-white">增速</span>：当月增速（%），按公式计算：
                      <span className="font-mono text-white">（本期零售额 - 同期零售额）/ 同期零售额</span>。
                    </div>
                    <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4 text-xs text-[#D4D4D4]">
                      约束：<span className="font-mono text-white">本期零售额 ≤ 总销售额（本期）</span>
                      （当总销售额为 0 时不强制）；同时数值不允许为负数。
                    </div>
                  </div>
                </Card>
              )}

              {activeSection === 'linkage' && (
                <>
                  <Card className="mb-6 border-white/10 bg-[#0D0D0D] p-6">
                    <h2 className="text-lg font-semibold">指标含义与联动公式（系统层）</h2>
                    <p className="mt-1 text-sm text-[#D4D4D4]">任意企业数据变化后，以下指标会自动重新计算</p>

                    <div className="mt-6 space-y-4">
                      <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4">
                        <div className="font-semibold text-[#FF6B35]">限上社零额（当月 / 增速）</div>
                        <div className="mt-2 text-sm text-[#D4D4D4]">
                          当月值：<span className="font-mono text-white">Σ(企业.本期零售额)</span>
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          当月增速：<span className="font-mono text-white">(Σ本期零售额 - Σ同期零售额) / Σ同期零售额</span>
                        </div>
                      </div>

                      <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4">
                        <div className="font-semibold text-[#FF6B35]">限上社零额（累计 / 增速）</div>
                        <div className="mt-2 text-sm text-[#D4D4D4]">
                          累计值：<span className="font-mono text-white">Σ(企业.本年累计零售额)</span>
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          累计增速：<span className="font-mono text-white">(Σ本年累计零售额 - Σ上年累计零售额) / Σ上年累计零售额</span>
                        </div>
                      </div>

                      <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4">
                        <div className="font-semibold text-[#FF6B35]">专项增速（吃穿用 / 小微）</div>
                        <div className="mt-2 text-sm text-[#D4D4D4]">
                          吃穿用（当月）：对<span className="font-mono text-white">IsEatWearUse=true</span>企业汇总零售额后按增速公式计算。
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          小微（当月）：对企业规模为 <span className="font-mono text-white">3/4</span> 的企业汇总零售额后按增速公式计算。
                        </div>
                      </div>

                      <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4">
                        <div className="font-semibold text-[#FF6B35]">四大行业增速（当月 / 累计）</div>
                        <div className="mt-2 text-sm text-[#D4D4D4]">
                          按行业分组（批发/零售/住宿/餐饮），使用<span className="font-mono text-white">销售额</span>口径计算：
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          当月增速：<span className="font-mono text-white">(Σ本期销售额 - Σ上年同期销售额) / Σ上年同期销售额</span>
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          累计增速：<span className="font-mono text-white">(Σ本年累计销售额 - Σ上年累计销售额) / Σ上年累计销售额</span>
                        </div>
                      </div>

                      <div className="rounded-lg border border-white/10 bg-[#2D2D2D] p-4">
                        <div className="font-semibold text-[#FF6B35]">社零总额（估算）</div>
                        <div className="mt-2 text-sm text-[#D4D4D4]">
                          估算本年累计限下社零额：<span className="font-mono text-white">上年累计限下 × (1 + 小微企业增速)</span>
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          社零总额（估算）：<span className="font-mono text-white">限上累计 + 估算限下累计</span>
                        </div>
                        <div className="mt-1 text-sm text-[#D4D4D4]">
                          累计增速：<span className="font-mono text-white">(本年社零总额 - 上年社零总额) / 上年社零总额</span>
                        </div>
                      </div>
                    </div>
                  </Card>

                  <Card className="border-white/10 bg-[#0D0D0D] p-6">
                    <h2 className="text-lg font-semibold">编辑、保存、重置与撤销</h2>
                    <div className="mt-4 space-y-3 text-sm text-[#D4D4D4]">
                      <div>
                        <span className="font-semibold text-white">实时联动</span>：输入框内容变化后会自动提交（带短暂防抖），并触发后端重新计算指标。
                      </div>
                      <div>
                        <span className="font-semibold text-white">自动保存</span>：后端以防抖方式持久化（约 1000ms），页面右上角展示“上次保存时间”。
                      </div>
                      <div>
                        <span className="font-semibold text-white">重置</span>：将企业数据恢复为导入 Excel 的原始值（原始快照）。
                      </div>
                      <div>
                        <span className="font-semibold text-white">撤销</span>：支持多步撤销，恢复到上一版快照。
                      </div>
                    </div>
                  </Card>
                </>
              )}
            </main>
          </div>
        </div>
      </div>
    </div>
  )
}
