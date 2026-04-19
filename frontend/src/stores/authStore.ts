import { create } from 'zustand'
import type { User } from '../types'

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
  login: (token: string, user: User) => void
  logout: () => void
  checkAuth: () => boolean
  updateUser: (userData: Partial<User>) => Promise<void>
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>
}

export const useAuthStore = create<AuthState>((set, get) => ({
  isAuthenticated: false,
  user: null,
  token: null,
  login: (token, user) => {
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
    set({ token, user, isAuthenticated: true })
  },
  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    set({ token: null, user: null, isAuthenticated: false })
  },
  checkAuth: () => {
    const token = localStorage.getItem('token')
    if (token) {
      // 从localStorage中获取用户信息
      const userStr = localStorage.getItem('user')
      let user: User | null = null
      if (userStr) {
        try {
          user = JSON.parse(userStr)
        } catch (error) {
          console.error('解析用户信息失败', error)
        }
      }
      set({ token, user, isAuthenticated: true })
      return true
    }
    return false
  },
  updateUser: async (userData: Partial<User>) => {
    const currentUser = get().user
    if (currentUser) {
      // 这里应该调用API更新用户信息
      // 暂时使用模拟数据
      const updatedUser = { ...currentUser, ...userData }
      set({ user: updatedUser })
    }
  },
  changePassword: async (currentPassword: string, newPassword: string) => {
    // 这里应该调用API修改密码
    // 暂时使用模拟实现
    console.log('修改密码:', currentPassword, newPassword)
  }
}))