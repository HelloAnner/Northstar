import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import Topbar from '@/components/app/Topbar'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { exportApi } from '@/services/api'
import { useProjectStore } from '@/store'

function formatDate(iso: string | undefined) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() <= 2000) return '-'
  return d.toLocaleDateString('zh-CN')
}

function formatDateTime(iso: string | undefined) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() <= 2000) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
}

export default function ProjectDetail() {
  const navigate = useNavigate()
  const params = useParams()
  const projectId = params.id || ''

  const { detail, refreshDetail, selectProject, saveCurrent, deleteProject } = useProjectStore()
  const [deleteOpen, setDeleteOpen] = useState(false)

  useEffect(() => {
    if (!projectId) return
    ;(async () => {
      await selectProject(projectId)
      await refreshDetail(projectId)
    })()
  }, [projectId, refreshDetail, selectProject])

  const statusText = detail?.project?.hasData ? '已保存' : '等待导入'
  const subtitle = detail?.project?.hasData ? '数据状态：已导入（可进入仪表盘）' : '数据状态：未导入（需先进入导入向导）'
  const projectName = detail?.project?.name?.trim() || detail?.meta?.name?.trim() || '未命名项目'

  const history = useMemo(() => (detail?.history ?? []).slice(0, 3), [detail?.history])

  return (
    <>
      <Topbar title={`项目详情 / ${projectName}`} statusText={statusText} />

      <div className="flex-1 overflow-y-scroll p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-2xl font-semibold">项目详情：{projectName}</div>
            <div className="mt-1 text-sm text-[#A3A3A3]">{subtitle}</div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
              onClick={() => navigate('/import')}
            >
              重新导入
            </Button>
            <Button
              className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
              onClick={() => navigate(detail?.project?.hasData ? '/dashboard' : '/import')}
            >
              进入仪表盘
            </Button>
            <Button
              variant="outline"
              className="border-white/10 bg-transparent text-white hover:bg-white/5"
              onClick={async () => {
                const res = await exportApi.export({ format: 'xlsx', includeIndicators: true, includeChanges: true })
                window.location.href = res.downloadUrl
              }}
            >
              导出
            </Button>
          </div>
        </div>

        <div className="mt-6 grid grid-cols-12 gap-4">
          <div className="col-span-12 grid gap-4 lg:col-span-8">
            <Card className="border-white/10 bg-[#0D0D0D] p-4">
              <div className="text-lg font-semibold">项目概览</div>
              <div className="mt-1 text-sm text-[#A3A3A3]">基础信息与数据概况</div>

              <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                <div className="text-[#A3A3A3]">项目名称</div>
                <div className="text-white">{projectName}</div>
                <div className="text-[#A3A3A3]">企业数量</div>
                <div className="text-white">{detail?.project?.companyCount ?? 0}</div>
                <div className="text-[#A3A3A3]">创建时间</div>
                <div className="text-white">{formatDate(detail?.meta?.createdAt)}</div>
                <div className="text-[#A3A3A3]">最近更新</div>
                <div className="text-white">{formatDateTime(detail?.project?.updatedAt)}</div>
                <div className="text-[#A3A3A3]">数据状态</div>
                <div className="text-white">{detail?.project?.hasData ? '已就绪' : '需要导入'}</div>
                <div className="text-[#A3A3A3]">自动保存</div>
                <div className="text-white">开启（1000ms debounce）</div>
              </div>
            </Card>

            <Card className="border-white/10 bg-[#0D0D0D] p-4">
              <div className="text-lg font-semibold">导入记录</div>
              <div className="mt-1 text-sm text-[#A3A3A3]">最近 3 次</div>

              <div className="mt-4 overflow-hidden rounded-lg border border-white/10">
                <Table>
                  <TableHeader>
                    <TableRow className="border-white/10">
                      <TableHead className="text-[#A3A3A3]">时间</TableHead>
                      <TableHead className="text-[#A3A3A3]">文件</TableHead>
                      <TableHead className="text-[#A3A3A3]">结果</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.length === 0 ? (
                      <TableRow className="border-white/10">
                        <TableCell colSpan={3} className="py-8 text-center text-sm text-[#A3A3A3]">
                          暂无记录
                        </TableCell>
                      </TableRow>
                    ) : (
                      history.map((h) => (
                        <TableRow key={h.importedAt} className="border-white/10">
                          <TableCell className="text-[#A3A3A3]">{formatDateTime(h.importedAt)}</TableCell>
                          <TableCell className="text-white">{h.fileName || '-'}</TableCell>
                          <TableCell className="text-white">{h.importedCount > 0 ? '成功' : '-'}</TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </Card>
          </div>

          <div className="col-span-12 lg:col-span-4">
            <Card className="border-white/10 bg-[#0D0D0D] p-4">
              <div className="text-lg font-semibold">安全操作</div>
              <div className="mt-1 text-sm text-[#A3A3A3]">删除 / 归档需确认</div>
              <div className="mt-4 text-sm text-[#A3A3A3]">提示：切换项目时系统会先保存当前状态。</div>

              <div className="mt-4 flex items-center gap-2">
                <Button
                  variant="secondary"
                  className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
                  onClick={async () => {
                    await saveCurrent()
                  }}
                >
                  立即保存
                </Button>
                <Button
                  variant="destructive"
                  className="bg-[#EF4444] text-white hover:bg-[#EF4444]/90"
                  onClick={() => setDeleteOpen(true)}
                >
                  删除项目
                </Button>
              </div>
            </Card>
          </div>
        </div>
      </div>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className="border-white/10 bg-[#0D0D0D] text-white">
          <DialogHeader>
            <DialogTitle>确认删除项目？</DialogTitle>
          </DialogHeader>
          <div className="text-sm text-[#A3A3A3]">
            删除后将移除项目目录（meta/state/latest.xlsx 等），且不可恢复。
          </div>
          <DialogFooter>
            <Button variant="ghost" className="hover:bg-white/5" onClick={() => setDeleteOpen(false)}>
              取消
            </Button>
            <Button
              variant="destructive"
              className="bg-[#EF4444] text-white hover:bg-[#EF4444]/90"
              onClick={async () => {
                await deleteProject(projectId)
                setDeleteOpen(false)
                navigate('/')
              }}
            >
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
