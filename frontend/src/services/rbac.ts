import apiClient from './api'
import type {
  Role,
  Permission,
  CreateRoleRequest,
  UpdateRoleRequest,
  ApiResponse,
  PaginatedResponse
} from '../types'

export interface RoleWithPermissions extends Role {
  permissions?: Permission[]
  permission_codes?: string[]
  created_at?: string
  updated_at?: string
}

export const roleService = {
  async getRoles(params: { skip?: number; limit?: number } = {}): Promise<PaginatedResponse<RoleWithPermissions>> {
    const { skip = 0, limit = 10 } = params
    const page = Math.floor(skip / limit) + 1
    const pageSize = limit
    const response = await apiClient.get<any>('/api/roles', {
      params: { page, pageSize }
    })
    
    console.log('getRoles response:', response.data)
    
    // 处理后端返回的字段映射
    const mappedData = (response.data.data || []).map((role: any) => ({
      ...role,
      id: role._id || role.id,
      permission_ids: Array.isArray(role.permission_ids) ? role.permission_ids : [],
      permission_codes: Array.isArray(role.permission_codes) ? role.permission_codes : []
    }))
    
    console.log('Mapped roles data:', mappedData)
    
    return {
      data: mappedData,
      total: response.data.total || 0,
      page: response.data.page || page,
      pageSize: response.data.pageSize || pageSize
    }
  },

  async getRoleById(id: string): Promise<RoleWithPermissions> {
    const response = await apiClient.get<any>(`/api/roles/${id}`)
    
    console.log('Response data:', response.data)
    
    if (!response.data) {
      throw new Error('获取角色详情失败：响应数据为空')
    }
    
    let roleData = response.data.data
    if (!roleData) {
      roleData = response.data
    }
    
    if (!roleData) {
      throw new Error('获取角色详情失败：数据为空')
    }
    
    console.log('Role data:', roleData)
    
    const permissionIds = Array.isArray(roleData.permission_ids) ? roleData.permission_ids : []
    const permissionCodes = Array.isArray(roleData.permission_codes) ? roleData.permission_codes : []
    
    return {
      ...roleData,
      id: roleData._id || roleData.id,
      permission_ids: permissionIds,
      permission_codes: permissionCodes
    }
  },

  async createRole(data: CreateRoleRequest): Promise<RoleWithPermissions> {
    const response = await apiClient.post<any>('/api/roles', data)
    
    let roleData = response.data.data
    if (!roleData) {
      roleData = response.data
    }
    
    if (!roleData) {
      throw new Error('创建角色失败：响应数据为空')
    }
    
    return {
      ...roleData,
      id: roleData._id || roleData.id
    }
  },

  async updateRole(id: string, data: UpdateRoleRequest): Promise<RoleWithPermissions> {
    const response = await apiClient.put<any>(`/api/roles/${id}`, data)
    
    let roleData = response.data.data
    if (!roleData) {
      roleData = response.data
    }
    
    if (!roleData) {
      throw new Error('更新角色失败：响应数据为空')
    }
    
    return {
      ...roleData,
      id: roleData._id || roleData.id
    }
  },

  async deleteRole(id: string): Promise<ApiResponse> {
    const response = await apiClient.delete<ApiResponse>(`/api/roles/${id}`)
    return response.data
  },

  async assignPermissionToRole(roleId: string, permissionId: string): Promise<ApiResponse> {
    const response = await apiClient.post<ApiResponse>(`/api/roles/${roleId}/permissions`, {
      permission_id: permissionId
    })
    return response.data
  },

  async removePermissionFromRole(roleId: string, permissionId: string): Promise<ApiResponse> {
    const response = await apiClient.delete<ApiResponse>(`/api/roles/${roleId}/permissions/${permissionId}`)
    return response.data
  },

  async getRolePermissions(roleId: string): Promise<Permission[]> {
    const response = await apiClient.get<Permission[]>(`/api/roles/${roleId}/permissions`)
    return response.data
  }
}

export const permissionService = {
  async getPermissions(params: { skip?: number; limit?: number } = {}): Promise<PaginatedResponse<Permission>> {
    const { skip = 0, limit = 10 } = params
    const page = Math.floor(skip / limit) + 1
    const pageSize = limit
    const response = await apiClient.get<any>('/api/permissions', {
      params: { page, pageSize }
    })
    
    // 处理后端返回的字段映射
    const mappedData = response.data.data.map((permission: any) => ({
      ...permission,
      id: permission._id
    }))
    
    return {
      data: mappedData,
      total: response.data.total || 0,
      page: response.data.page || page,
      pageSize: response.data.pageSize || pageSize
    }
  },

  async getPermissionById(id: string): Promise<Permission> {
    const response = await apiClient.get<any>(`/api/permissions/${id}`)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id
    }
  },

  async createPermission(data: Omit<Permission, 'id'>): Promise<Permission> {
    const response = await apiClient.post<any>('/api/permissions', data)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id
    }
  },

  async updatePermission(id: string, data: Partial<Permission>): Promise<Permission> {
    const response = await apiClient.put<any>(`/api/permissions/${id}`, data)
    
    // 处理后端返回的字段映射
    return {
      ...response.data,
      id: response.data._id
    }
  },

  async deletePermission(id: string): Promise<ApiResponse> {
    const response = await apiClient.delete<ApiResponse>(`/api/permissions/${id}`)
    return response.data
  }
}

export default { roleService, permissionService }