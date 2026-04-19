import axios from 'axios'
import type { AxiosError } from 'axios'
import { useAuthStore } from '../stores/authStore'

const apiClient = axios.create({
  baseURL: 'http://localhost:9003',
  timeout: 10000
})

apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

apiClient.interceptors.response.use(
  (response) => {
    return response
  },
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      const authStore = useAuthStore.getState()
      authStore.logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default apiClient