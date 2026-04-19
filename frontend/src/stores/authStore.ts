import { create } from 'zustand'
import type { User } from '../types'

interface AuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
  login: (token: string, user: User) => void
  logout: () => void
  checkAuth: () => boolean
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  user: null,
  token: null,
  login: (token, user) => {
    localStorage.setItem('token', token)
    set({ token, user, isAuthenticated: true })
  },
  logout: () => {
    localStorage.removeItem('token')
    set({ token: null, user: null, isAuthenticated: false })
  },
  checkAuth: () => {
    const token = localStorage.getItem('token')
    if (token) {
      set({ token, isAuthenticated: true })
      return true
    }
    return false
  }
}))