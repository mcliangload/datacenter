import apiClient from './api'
import type { ApiResponse, PaginatedResponse } from '../types'

export interface BusinessData {
  _id: string
  module: string
  description: string
  created_at: string
  [key: string]: any
}

export interface SearchParams {
  module: string
  page: number
  pageSize: number
  jql: string
}

export const businessService = {
  // 按模块查询业务数据
  search: async (params: SearchParams): Promise<PaginatedResponse<BusinessData>> => {
    const response = await apiClient.get(`/api/business/module/${params.module}`, {
      params: {
        page: params.page,
        pageSize: params.pageSize,
        jql: params.jql
      }
    })
    return response.data
  },

  // 获取业务数据详情
  getById: async (id: string): Promise<ApiResponse<BusinessData>> => {
    const response = await apiClient.get(`/api/business/${id}`)
    return response.data
  },

  // 更新业务数据
  update: async (id: string, data: Partial<BusinessData>): Promise<ApiResponse<BusinessData>> => {
    const response = await apiClient.put(`/api/business/${id}`, data)
    return response.data
  },

  // 获取模块列表
  getModules: async (): Promise<ApiResponse<Array<{ name: string, description: string }>>> => {
    const response = await apiClient.get('/api/business/modules')
    return response.data
  },

  // 获取模块的自定义字段
  getFieldsByModule: async (module: string): Promise<ApiResponse<Array<any>>> => {
    const response = await apiClient.get(`/api/fields/module/${module}`)
    return response.data
  }
}

export default businessService