import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Topbar from '@/components/app/Topbar'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { ChevronLeft, ChevronRight, Plus, Search, Trash2 } from 'lucide-react'
import { useProjectStore } from '@/store'

const PAGE_SIZE = 5

function formatDateTime(iso: string | undefined) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() <= 2000) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
}

export default function ProjectHub() {
  const navigate = useNavigate()
  const { index, refreshIndex, refreshCurrent, createProject, selectProject, deleteProject } = useProjectStore()
  const [keyword, setKeyword] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [projectName, setProjectName] = useState('')
  const [page, setPage] = useState(1)
  const [pendingDeleteProjectId, setPendingDeleteProjectId] = useState<string | null>(null)

  useEffect(() => {
    refreshCurrent()
    refreshIndex()
  }, [refreshCurrent, refreshIndex])

  const items = useMemo(() => index?.items ?? [], [index?.items])
  const filtered = useMemo(() => {
    const k = keyword.trim()
    if (!k) return items
    return items.filter((p) => p.projectId.includes(k) || p.name.includes(k))
  }, [items, keyword])

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE)
  const paginatedItems = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE
    return filtered.slice(start, start + PAGE_SIZE)
  }, [filtered, page])

  return (
    <>
      <Topbar title="项目中心" statusText="已保存" />

      <div className="flex-1 overflow-y-scroll p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="text-2xl font-semibold">项目中心</div>
            <div className="mt-1 text-sm text-[#A3A3A3]">新建或选择项目，然后进入仪表盘 / 导入向导</div>
          </div>

          <div className="flex items-center gap-2">
            <Dialog open={createOpen} onOpenChange={setCreateOpen}>
              <DialogTrigger asChild>
                <Button className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90">
                  <Plus className="mr-2 h-4 w-4" />
                  新建项目
                </Button>
              </DialogTrigger>
              <DialogContent className="border-white/10 bg-[#0D0D0D] text-white">
                <DialogHeader>
                  <DialogTitle>新建项目</DialogTitle>
                </DialogHeader>
                <div className="space-y-2">
                  <div className="text-sm text-[#D4D4D4]">输入项目名称</div>
                  <Input
                    value={projectName}
                    onChange={(e) => setProjectName(e.target.value)}
                    placeholder="例如：2026年1月社零测算"
                    className="border-white/10 bg-white/5 text-white"
                  />
                </div>
                <DialogFooter>
                  <Button
                    className="bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
                    onClick={async () => {
                      const created = await createProject(projectName.trim() || '新项目')
                      setCreateOpen(false)
                      setProjectName('')
                      navigate(created.hasData ? '/dashboard' : '/import')
                    }}
                  >
                    创建并进入
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>

            <Button
              variant="ghost"
              className="text-[#D4D4D4] hover:bg-white/5 hover:text-white"
              onClick={() => window.open('/help', '_blank')}
            >
              帮助文档
            </Button>
          </div>
        </div>

        <div className="mt-6 flex items-center gap-3">
          <div className="flex h-10 flex-1 items-center gap-2 rounded-lg border border-white/10 bg-[#0D0D0D] px-3 text-[12px] text-[#D4D4D4]">
            <Search className="h-4 w-4" />
            <Input
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
              placeholder="搜索项目名称..."
              className="h-8 border-0 bg-transparent p-0 text-white placeholder:text-[#D4D4D4] focus-visible:ring-0"
            />
          </div>
        </div>

        <div className="mt-6">
          <Card className="border-white/10 bg-[#0D0D0D] p-4">
            <div className="space-y-1">
              <div className="text-lg font-semibold">项目列表</div>
              <div className="text-sm text-[#D4D4D4]">选择"进入"或"导入"，系统会自动切换当前项目</div>
            </div>

            <div className="mt-4 overflow-hidden rounded-lg border border-white/10">
              <Table>
                <TableHeader>
                  <TableRow className="border-white/10">
                    <TableHead className="text-[#D4D4D4]">项目</TableHead>
                    <TableHead className="text-[#D4D4D4]">更新时间</TableHead>
                    <TableHead className="text-[#D4D4D4]">状态</TableHead>
                    <TableHead className="text-right text-[#D4D4D4]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginatedItems.length === 0 ? (
                    <TableRow className="border-white/10">
                      <TableCell colSpan={4} className="py-10 text-center text-sm text-[#D4D4D4]">
                        暂无项目
                      </TableCell>
                    </TableRow>
                  ) : (
                    paginatedItems.map((p) => (
                        <TableRow key={p.projectId} className="border-white/10">
                        <TableCell className="text-white">
                          <Button
                            variant="link"
                            className="h-auto p-0 text-left text-[14px] font-medium text-white underline-offset-4 hover:text-white"
                            onClick={() => navigate(`/projects/${encodeURIComponent(p.projectId)}`)}
                          >
                            {p.projectId}{p.name ? `（${p.name}）` : ''}
                          </Button>
                          <div className="mt-0.5 text-[12px] text-[#D4D4D4]">
                            {p.hasData ? `企业：${p.companyCount || 0}` : '未导入数据'}
                          </div>
                        </TableCell>
                        <TableCell className="text-[#D4D4D4]">{formatDateTime(p.updatedAt)}</TableCell>
                        <TableCell>
                          <Badge
                            variant="secondary"
                            className={p.hasData
                              ? "border border-[#00D4AA]/30 bg-[#00D4AA]/10 text-[#00D4AA]"
                              : "border border-white/10 bg-white/5 text-[#D4D4D4]"
                            }
                          >
                            {p.hasData ? '已就绪' : '需要导入'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              size="sm"
                              className={p.hasData
                                ? "bg-[#2D2D2D] text-white hover:bg-[#3D3D3D]"
                                : "bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90"
                              }
                              onClick={async () => {
                                const selected = await selectProject(p.projectId)
                                navigate(selected.hasData ? '/dashboard' : '/import')
                              }}
                            >
                              {p.hasData ? '进入' : '导入'}
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="border-[#EF4444]/50 bg-transparent text-[#EF4444] hover:bg-[#EF4444]/10 hover:text-[#EF4444]"
                              onClick={() => setPendingDeleteProjectId(p.projectId)}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            {/* 分页控件 */}
            {filtered.length > 0 && (
              <div className="mt-4 flex items-center justify-between">
                <div className="text-sm text-[#D4D4D4]">
                  共 {filtered.length} 个项目，每页 {PAGE_SIZE} 个
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 w-8 border-white/10 bg-[#2D2D2D] p-0 text-[#D4D4D4] hover:bg-white/10"
                    disabled={page === 1}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => i + 1).map((p) => (
                    <Button
                      key={p}
                      size="sm"
                      variant="outline"
                      className={`h-8 w-8 border-white/10 p-0 ${
                        p === page
                          ? 'bg-[#FF6B35] text-black hover:bg-[#FF6B35]/90'
                          : 'bg-[#2D2D2D] text-white hover:bg-white/10'
                      }`}
                      onClick={() => setPage(p)}
                    >
                      {p}
                    </Button>
                  ))}
                  {totalPages > 5 && <span className="text-[#D4D4D4]">...</span>}
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 w-8 border-white/10 bg-[#2D2D2D] p-0 text-white hover:bg-white/10"
                    disabled={page === totalPages || totalPages === 0}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}
          </Card>
        </div>
      </div>

      <Dialog open={pendingDeleteProjectId !== null} onOpenChange={(open) => !open && setPendingDeleteProjectId(null)}>
        <DialogContent className="border-white/10 bg-[#0D0D0D] text-white">
          <DialogHeader>
            <DialogTitle className="text-white">删除项目</DialogTitle>
            <DialogDescription className="text-white/70">
              确定要删除项目「{pendingDeleteProjectId ?? '-'}」吗？此操作不可恢复。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              className="border border-white/10 bg-white/5 text-white hover:bg-white/10"
              onClick={() => setPendingDeleteProjectId(null)}
            >
              取消
            </Button>
            <Button
              className="bg-[#EF4444] text-white hover:bg-[#EF4444]/90"
              onClick={async () => {
                if (!pendingDeleteProjectId) return
                await deleteProject(pendingDeleteProjectId)
                setPendingDeleteProjectId(null)
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
