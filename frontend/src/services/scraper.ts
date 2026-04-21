import apiClient from './api'
import type { ApiResponse } from '../types'

export interface ScrapeTask {
  _id: string
  created_by: string
  created_at: string
  updated_by: string
  updated_at: string
  module: string
  data_path: string
  scraper_path: string
  status: string
  result: Array<{
    Key: string
    Value: any
  }>
  error_message: string
  started_at: string
  completed_at: string
}

export interface ScrapeTaskQuery {
  skip?: number
  limit?: number
  keyword?: string
  status?: string
}

export interface ScrapeTaskResponse {
  data: ScrapeTask[]
  total: number
}

const scraperService = {
  // 获取刮削任务列表
  getScrapeTasks: async (query: ScrapeTaskQuery = {}): Promise<ScrapeTaskResponse> => {
    const { skip, limit, keyword, status } = query
    const page = skip ? Math.floor(skip / (limit || 20)) + 1 : 1
    const pageSize = limit || 20
    const params = {
      page,
      pageSize,
      status
    }
    if (keyword) {
      ;(params as any).module = keyword
    }
    const response = await apiClient.get('/api/scraper/tasks', { params })
    return {
      data: response.data.data || [],
      total: response.data.total || 0
    }
  },

  // 获取已删除的刮削任务
  getDeletedScrapeTasks: async (query: ScrapeTaskQuery = {}): Promise<ScrapeTaskResponse> => {
    const { skip, limit, keyword } = query
    const page = skip ? Math.floor(skip / (limit || 20)) + 1 : 1
    const pageSize = limit || 20
    const module = keyword || 'all'
    const response = await apiClient.get(`/api/deleted-scraper/module/${module}`, { 
      params: { page, pageSize } 
    })
    return {
      data: response.data.data || [],
      total: response.data.total || 0
    }
  },

  // 重试刮削任务
  retryScrapeTask: async (taskId: string, scraperPath?: string): Promise<ApiResponse> => {
    const data: any = {}
    if (scraperPath) {
      data.scraper_path = scraperPath
    }
    const response = await apiClient.post(`/api/scraper/tasks/${taskId}/retry`, data)
    return response.data
  },

  // 恢复已删除的刮削任务
  recoverScrapeTask: async (taskId: string): Promise<ApiResponse> => {
    const response = await apiClient.post(`/api/deleted-scraper/${taskId}/recover`)
    return response.data
  },

  // 创建刮削任务
  createScrapeTask: async (data: { module: string; data_path: string; scraper_path: string; description?: string }): Promise<ApiResponse> => {
    const response = await apiClient.post('/api/scraper/upload', {
      module: data.module,
      data_path: data.data_path,
      scraper_path: data.scraper_path,
      description: data.description || ''
    })
    return response.data
  },

  // 批量删除刮削任务
  batchDeleteScrapeTasks: async (ids: string[]): Promise<ApiResponse> => {
    const response = await apiClient.post('/api/scraper/tasks/batch-delete', { ids })
    return response.data
  }
}

export default scraperService