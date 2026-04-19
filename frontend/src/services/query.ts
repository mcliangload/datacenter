import apiClient from './api'
import type { SearchRequest, SearchResponse } from '../types'

export const queryService = {
  async search(jql: string): Promise<SearchResponse> {
    const request: SearchRequest = { jql }
    const response = await apiClient.post<SearchResponse>('/api/query', request)
    return response.data
  }
}

export default queryService