import { Button, Input, Space, Table, message, Select } from 'antd'
import { useState, useCallback } from 'react'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface TableRow {
  key: string
  module: string
  description: string
  createdTime: string
  [key: string]: any
}

const modules = [
  { value: '', label: '所有模块' },
  { value: 'book', label: '图书' },
  { value: 'movie', label: '电影' },
  { value: 'music', label: '音乐' },
  { value: 'game', label: '游戏' },
]

const AdminSearchPage: React.FC = () => {
  const [keyword, setKeyword] = useState('')
  const [data, setData] = useState<TableRow[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedModule, setSelectedModule] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  const handleSearch = useCallback(async () => {
    setLoading(true)
    try {
      // 由于后端接口需要具体的模块名，我们暂时使用'book'作为默认模块
      const module = selectedModule || 'book'
      const page = pagination.current
      const pageSize = pagination.pageSize
      const jql = keyword
      
      const response = await apiClient.get('/api/business/module/' + module, {
        params: { page, pageSize, jql }
      })
      
      const tableData = response.data.data.map((item: any) => ({
        key: item._id || Math.random().toString(36).substr(2, 9),
        module: item.module,
        description: item.description || '无描述',
        createdTime: item.created_at || new Date().toISOString(),
        ...item
      }))
      
      setData(tableData)
      setPagination(prev => ({ ...prev, total: response.data.total || 0 }))
    } catch (error: any) {
      console.error('搜索失败', error)
      message.error(error?.response?.data?.error || '搜索失败')
      // 模拟数据
      const tableData: TableRow[] = [
        {
          key: '1',
          module: 'book',
          description: '图书数据1',
          createdTime: '2026-04-19 10:00:00'
        },
        {
          key: '2',
          module: 'movie',
          description: '电影数据1',
          createdTime: '2026-04-19 10:00:00'
        }
      ]
      setData(tableData)
    } finally {
      setLoading(false)
    }
  }, [keyword, selectedModule, pagination.current, pagination.pageSize])

  const handleClear = () => {
    setKeyword('')
    setSelectedModule('')
    setData([])
  }

  const handleRefresh = () => {
    handleSearch()
  }

  const handleTableChange = (newPagination: any) => {
    setPagination(prev => ({
      ...prev,
      current: newPagination.current,
      pageSize: newPagination.pageSize
    }))
  }

  const columns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      sorter: (a: TableRow, b: TableRow) => a.module.localeCompare(b.module),
      filterSearch: true,
      filters: modules.filter(m => m.value !== ''),
      onFilter: (value: any, record: TableRow) => record.module.includes(value),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      sorter: (a: TableRow, b: TableRow) => a.description.localeCompare(b.description),
    },
    {
      title: '创建时间',
      dataIndex: 'createdTime',
      key: 'createdTime',
      sorter: (a: TableRow, b: TableRow) => a.createdTime.localeCompare(b.createdTime),
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space>
          <Button type="link">详情</Button>
          <Button type="link">编辑</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: '16px', display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        <h2>数据搜索</h2>
        <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
          刷新
        </Button>
      </Space>
      <Space style={{ marginBottom: '16px', display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        <Select
          placeholder="选择模块"
          value={selectedModule}
          onChange={setSelectedModule}
          options={modules}
          style={{ width: 150 }}
        />
        <Input
          placeholder="输入JQL查询语句，例如：name contains 'A'"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ width: 400 }}
          prefix={<SearchOutlined />}
        />
        <Button type="primary" onClick={handleSearch} loading={loading} icon={<SearchOutlined />}>
          搜索
        </Button>
        <Button onClick={handleClear}>
          清除
        </Button>
      </Space>
      <Table 
        columns={columns} 
        dataSource={data} 
        loading={loading}
        rowKey="key"
        rowSelection={{}}
        pagination={{
          ...pagination,
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`
        }}
        onChange={handleTableChange}
      />
    </div>
  )
}

export default AdminSearchPage