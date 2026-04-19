import apiClient from './api'
import type { LoginRequest, LoginResponse, User } from '../types'

export const authService = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>('/api/auth/login', credentials)
    return response.data
  },

  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<User>('/api/auth/me')
    return response.data
  }
}

export default authService