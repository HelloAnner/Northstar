import { create } from 'zustand'
import { projectsApi } from '@/services/api'
import type { CurrentProject, ProjectDetail, ProjectSummary, ProjectsIndex } from '@/types'

interface ProjectStore {
  index: ProjectsIndex | null
  current: CurrentProject | null
  detail: ProjectDetail | null
  loading: boolean

  refreshIndex: () => Promise<ProjectsIndex | null>
  refreshCurrent: () => Promise<CurrentProject | null>
  refreshDetail: (projectId: string) => Promise<ProjectDetail | null>

  createProject: (name: string) => Promise<ProjectSummary>
  selectProject: (projectId: string) => Promise<ProjectSummary>
  saveCurrent: () => Promise<void>
  deleteProject: (projectId: string) => Promise<void>
}

export const useProjectStore = create<ProjectStore>((set, get) => ({
  index: null,
  current: null,
  detail: null,
  loading: false,

  refreshIndex: async () => {
    set({ loading: true })
    try {
      const idx = await projectsApi.list()
      set({ index: idx })
      return idx
    } finally {
      set({ loading: false })
    }
  },

  refreshCurrent: async () => {
    const cur = await projectsApi.current()
    set({ current: cur })
    return cur
  },

  refreshDetail: async (projectId) => {
    const detail = await projectsApi.detail(projectId)
    set({ detail })
    return detail
  },

  createProject: async (name) => {
    const created = await projectsApi.create(name)
    await Promise.all([get().refreshIndex(), get().refreshCurrent()])
    return created
  },

  selectProject: async (projectId) => {
    const selected = await projectsApi.select(projectId)
    await Promise.all([get().refreshIndex(), get().refreshCurrent()])
    return selected
  },

  saveCurrent: async () => {
    await projectsApi.save()
    await Promise.all([get().refreshIndex(), get().refreshCurrent()])
  },

  deleteProject: async (projectId) => {
    await projectsApi.delete(projectId)
    const cur = get().current
    set({ detail: null })
    await get().refreshIndex()
    if (cur?.project?.projectId === projectId) {
      await get().refreshCurrent()
    }
  },
}))

