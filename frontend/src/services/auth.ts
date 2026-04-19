import apiClient from './api'
import type { LoginRequest, User } from '../types'

export interface AuthResponse {
  token: string
  user: {
    id: string
    username: string
    email: string
    roles?: string[]
  }
}

export const authService = {
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>('/api/auth/login', credentials)
    return response.data
  },

  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<User>('/api/auth/me')
    return response.data
  }
}

export default authService