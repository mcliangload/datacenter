import { Button, Input, Space, Table, message } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { SearchOutlined, ReloadOutlined, RollbackOutlined } from '@ant-design/icons'
import scraperService, { type ScrapeTask, type ScrapeTaskResponse } from '../../services/scraper'

const DeletedScraperPage: React.FC = () => {
  const [keyword, setKeyword] = useState('')
  const [deletedData, setDeletedData] = useState<ScrapeTask[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  // 获取删除的刮削任务列表
  const fetchDeletedData = useCallback(async (page: number, pageSize: number, searchKeyword: string) => {
    setLoading(true)
    try {
      const skip = (page - 1) * pageSize
      const module = searchKeyword || 'all'
      const response: ScrapeTaskResponse = await scraperService.getDeletedScrapeTasks({
        skip,
        limit: pageSize,
        keyword: module
      })
      setDeletedData(response.data || [])
      setPagination(prev => ({ ...prev, total: response.total || 0, current: page, pageSize }))
    } catch (error: any) {
      console.error('获取已删除任务失败', error)
      message.error(error?.response?.data?.error || '获取已删除任务失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchDeletedData(pagination.current, pagination.pageSize, keyword)
  }, [])

  const handleSearch = () => {
    fetchDeletedData(1, pagination.pageSize, keyword)
  }

  const handleClear = () => {
    setKeyword('')
    setDeletedData([])
  }

  const handleRefresh = () => {
    handleSearch()
  }

  const handleRecover = async (id: string) => {
    setLoading(true)
    try {
      await scraperService.recoverScrapeTask(id)
      message.success(`恢复数据 ${id} 成功`)
      handleSearch()
    } catch (error: any) {
      console.error('恢复失败', error)
      message.error(error?.response?.data?.error || '恢复失败')
    } finally {
      setLoading(false)
    }
  }

  const handleTableChange = (pagination: any) => {
    fetchDeletedData(pagination.current, pagination.pageSize, keyword)
  }

  const deletedColumns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
    },
    {
      title: '刮削器路径',
      dataIndex: 'scraper_path',
      key: 'scraper_path',
    },
    {
      title: '数据路径',
      dataIndex: 'data_path',
      key: 'data_path',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <span style={{ color: 'red' }}>{status}</span>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: ScrapeTask) => (
        <Space>
          <Button type="link" icon={<RollbackOutlined />} onClick={() => handleRecover(record._id)}>
            恢复
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: '16px', display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        <h2>删除数据查询</h2>
        <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
          刷新
        </Button>
      </Space>
      <Space style={{ marginBottom: '16px', display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
        <Input
          placeholder="输入搜索关键词"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ width: 300 }}
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
        columns={deletedColumns}
        dataSource={deletedData}
        loading={loading}
        rowKey="_id"
        rowSelection={{}}
        pagination={{
          current: pagination.current,
          pageSize: pagination.pageSize,
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`,
          total: pagination.total
        }}
        onChange={handleTableChange}
      />
    </div>
  )
}

export default DeletedScraperPage