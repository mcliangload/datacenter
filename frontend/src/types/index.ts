export interface User {
  id: string
  username: string
  email: string
  phone?: string
  address?: string
  avatar?: string
  role_ids?: string[]
  created_by?: string
  updated_by?: string
  created_at?: string
  updated_at?: string
}

export interface Role {
  id: string
  name: string
  code: string
  description: string
  permission_ids: string[]
  created_by?: string
  updated_by?: string
  created_at?: string
  updated_at?: string
}

export interface Permission {
  id: string
  name: string
  code: string
  description: string
  module?: string
  created_by?: string
  updated_by?: string
  created_at?: string
  updated_at?: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface CreateDataRequest {
  module: string
  description: string
  custom_fields?: Record<string, any>
}

export interface CreateUserRequest {
  username: string
  email: string
  password: string
  phone?: string
  address?: string
  role_ids?: string[]
}

export interface UpdateUserRequest {
  email?: string
  password?: string
  phone?: string
  address?: string
  role_ids?: string[]
}

export interface CreateRoleRequest {
  name: string
  code: string
  description: string
  permission_ids?: string[]
}

export interface UpdateRoleRequest {
  name?: string
  code?: string
  description?: string
  permission_ids?: string[]
}

export interface ApiResponse<T = any> {
  data?: T
  message?: string
  error?: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  pageSize: number
}