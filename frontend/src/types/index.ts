export interface User {
  id: string
  username: string
  email: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface SearchRequest {
  jql: string
}

export interface SearchResponse {
  issues: Array<{
    id: string
    key: string
    summary: string
    status: string
    assignee?: User
    reporter?: User
  }>
  total: number
  page: number
  pageSize: number
}

export interface SearchParams {
  keyword: string
  page?: number
  pageSize?: number
}