import { Button, Input, Layout, Pagination, Space, Table } from 'antd'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../../stores/authStore'
import queryService from '../../services/query'
import type { SearchResponse } from '../../types'

const { Header, Content } = Layout

interface TableRow {
  key: string
  module: string
  description: string
  createdTime: string
}

const SearchPage: React.FC = () => {
  const navigate = useNavigate()
  const { user, logout } = useAuthStore((state) => ({ user: state.user, logout: state.logout }))
  const [keyword, setKeyword] = useState('')
  const [data, setData] = useState<TableRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [loading, setLoading] = useState(false)

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const handleSearch = async () => {
    setLoading(true)
    try {
      const response: SearchResponse = await queryService.search(keyword)
      const tableData: TableRow[] = response.issues.map((issue) => ({
        key: issue.id,
        module: issue.summary,
        description: issue.status,
        createdTime: issue.reporter?.username || '-',
      }))
      setData(tableData)
      setTotal(response.total)
    } catch (error) {
      console.error('搜索失败', error)
    } finally {
      setLoading(false)
    }
  }

  const handleClear = () => {
    setKeyword('')
    setData([])
    setTotal(0)
  }

  const handlePageChange = (newPage: number, newPageSize: number) => {
    setPage(newPage)
    setPageSize(newPageSize)
  }

  const columns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '创建时间',
      dataIndex: 'createdTime',
      key: 'createdTime',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space>
          <Button type="link">详情</Button>
        </Space>
      ),
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ color: 'white', fontSize: '18px', fontWeight: 'bold' }}>数据中心</span>
        <Space>
          <span style={{ color: 'white' }}>{user?.username}</span>
          <Button type="primary" danger onClick={handleLogout}>
            退出
          </Button>
        </Space>
      </Header>
      <Content style={{ padding: '24px' }}>
        <Space style={{ marginBottom: '16px' }}>
          <Input
            placeholder="输入搜索关键词"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            style={{ width: 300 }}
          />
          <Button type="primary" onClick={handleSearch} loading={loading}>
            搜索
          </Button>
          <Button onClick={handleClear}>清除</Button>
        </Space>
        <Table columns={columns} dataSource={data} loading={loading} pagination={false} />
        {total > 0 && (
          <Pagination
            current={page}
            pageSize={pageSize}
            total={total}
            onChange={handlePageChange}
            style={{ marginTop: '16px', textAlign: 'right' }}
          />
        )}
      </Content>
    </Layout>
  )
}

export default SearchPage