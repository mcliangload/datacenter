import apiClient from './api'
import type {
  User,
  Role,
  CreateUserRequest,
  UpdateUserRequest,
  ApiResponse,
  PaginatedResponse
} from '../types'

export interface UserListParams {
  skip?: number
  limit?: number
  keyword?: string
}

export interface UserWithRoles extends User {
  roles?: Role[]
  status?: 'active' | 'inactive'
  created_at?: string
  updated_at?: string
}

export const userService = {
  async getUsers(params: UserListParams = {}): Promise<PaginatedResponse<UserWithRoles>> {
    const { skip = 0, limit = 10, keyword } = params
    const page = Math.floor(skip / limit) + 1
    const pageSize = limit
    const response = await apiClient.get<any>('/api/users', {
      params: { page, pageSize, keyword }
    })
    
    // 处理后端返回的字段映射
    const mappedData = response.data.data.map((user: any) => ({
      ...user,
      id: user._id,
      status: 'active' // 假设所有用户都是活跃状态
    }))
    
    return {
      data: mappedData,
      total: response.data.total || 0,
      page: response.data.page || page,
      pageSize: response.data.pageSize || pageSize
    }
  },

  async getUserById(id: string): Promise<UserWithRoles> {
    const response = await apiClient.get<any>(`/api/users/${id}`)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id,
      status: 'active' // 假设所有用户都是活跃状态
    }
  },

  async createUser(data: CreateUserRequest): Promise<UserWithRoles> {
    const response = await apiClient.post<any>('/api/users', data)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id,
      status: 'active' // 假设所有用户都是活跃状态
    }
  },

  async updateUser(id: string, data: UpdateUserRequest): Promise<UserWithRoles> {
    const response = await apiClient.put<any>(`/api/users/${id}`, data)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id,
      status: 'active' // 假设所有用户都是活跃状态
    }
  },

  async deleteUser(id: string): Promise<ApiResponse> {
    const response = await apiClient.delete<ApiResponse>(`/api/users/${id}`)
    return response.data
  },

  async assignRoleToUser(userId: string, roleId: string): Promise<ApiResponse> {
    const response = await apiClient.post<ApiResponse>(`/api/users/${userId}/roles`, {
      role_id: roleId
    })
    return response.data
  },

  async removeRoleFromUser(userId: string, roleId: string): Promise<ApiResponse> {
    const response = await apiClient.delete<ApiResponse>(`/api/users/${userId}/roles/${roleId}`)
    return response.data
  },

  async getUserRoles(userId: string): Promise<Role[]> {
    const response = await apiClient.get<Role[]>(`/api/users/${userId}/roles`)
    return response.data
  }
}

export default userService