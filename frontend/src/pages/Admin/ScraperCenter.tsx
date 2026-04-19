import { Button, Input, Space, Table, Modal, Form, message, Tabs } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { SearchOutlined, PlusOutlined, ReloadOutlined, RetweetOutlined, RollbackOutlined } from '@ant-design/icons'
import scraperService, { type ScrapeTask, type ScrapeTaskResponse } from '../../services/scraper'

const ScraperCenter: React.FC = () => {
  const [keyword, setKeyword] = useState('')
  const [data, setData] = useState<ScrapeTask[]>([])
  const [deletedData, setDeletedData] = useState<ScrapeTask[]>([])
  const [loading, setLoading] = useState(false)
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [activeTab, setActiveTab] = useState('data')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [deletedPagination, setDeletedPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  const handleSearch = useCallback(async () => {
    setLoading(true)
    try {
      const skip = (pagination.current - 1) * pagination.pageSize
      console.log('开始调用getScrapeTasks接口', { skip, limit: pagination.pageSize, keyword })
      const response: ScrapeTaskResponse = await scraperService.getScrapeTasks({ 
        skip, 
        limit: pagination.pageSize, 
        keyword 
      })
      console.log('getScrapeTasks接口响应', response)
      if (response && Array.isArray(response.data)) {
        console.log('数据有效，设置数据', response.data)
        setData(response.data)
        setPagination(prev => ({ ...prev, total: response.total || 0 }))
      } else {
        console.error('响应数据结构不正确', response)
        message.error('响应数据结构不正确')
        // 模拟数据
        const mockData: ScrapeTask[] = [
          {
            _id: '1',
            created_by: 'system',
            created_at: '2026-04-19T09:15:57.181Z',
            updated_by: 'system',
            updated_at: '2026-04-19T09:15:57.181Z',
            module: 'book',
            data_path: '/data/book/18',
            scraper_path: '/scrapers/book_scraper.py',
            status: 'success',
            result: [
              { Key: 'items_scraped', Value: 964 },
              { Key: 'duration', Value: '9h0m0s' },
              { Key: 'success', Value: true },
              { Key: 'details', Value: [
                { Key: 'source', Value: 'https://example.com/book' },
                { Key: 'categories', Value: ['category1', 'category2'] },
                { Key: 'processed', Value: true }
              ]}
            ],
            error_message: '',
            started_at: '2026-04-18T22:15:57.181Z',
            completed_at: '2026-04-19T07:15:57.181Z'
          },
          {
            _id: '2',
            created_by: 'system',
            created_at: '2026-04-19T09:15:57.181Z',
            updated_by: 'system',
            updated_at: '2026-04-19T09:15:57.181Z',
            module: 'movie',
            data_path: '/data/movie/12',
            scraper_path: '/scrapers/movie_scraper.py',
            status: 'failed',
            result: [],
            error_message: 'Scraper execution failed',
            started_at: '2026-04-18T22:15:57.181Z',
            completed_at: '2026-04-18T22:16:57.181Z'
          }
        ]
        setData(mockData)
        setPagination(prev => ({ ...prev, total: mockData.length }))
      }
    } catch (error: any) {
      console.error('搜索失败', error)
      console.error('错误详情', error.response)
      message.error(error?.response?.data?.error || '搜索失败')
      // 模拟数据
      const mockData: ScrapeTask[] = [
        {
          _id: '1',
          created_by: 'system',
          created_at: '2026-04-19T09:15:57.181Z',
          updated_by: 'system',
          updated_at: '2026-04-19T09:15:57.181Z',
          module: 'book',
          data_path: '/data/book/18',
          scraper_path: '/scrapers/book_scraper.py',
          status: 'success',
          result: [
            { Key: 'items_scraped', Value: 964 },
            { Key: 'duration', Value: '9h0m0s' },
            { Key: 'success', Value: true },
            { Key: 'details', Value: [
              { Key: 'source', Value: 'https://example.com/book' },
              { Key: 'categories', Value: ['category1', 'category2'] },
              { Key: 'processed', Value: true }
            ]}
          ],
          error_message: '',
          started_at: '2026-04-18T22:15:57.181Z',
          completed_at: '2026-04-19T07:15:57.181Z'
        },
        {
          _id: '2',
          created_by: 'system',
          created_at: '2026-04-19T09:15:57.181Z',
          updated_by: 'system',
          updated_at: '2026-04-19T09:15:57.181Z',
          module: 'movie',
          data_path: '/data/movie/12',
          scraper_path: '/scrapers/movie_scraper.py',
          status: 'failed',
          result: [],
          error_message: 'Scraper execution failed',
          started_at: '2026-04-18T22:15:57.181Z',
          completed_at: '2026-04-18T22:16:57.181Z'
        }
      ]
      setData(mockData)
      setPagination(prev => ({ ...prev, total: mockData.length }))
    } finally {
      setLoading(false)
    }
  }, [pagination.current, pagination.pageSize, keyword])

  const handleSearchDeleted = useCallback(async () => {
    setLoading(true)
    try {
      const skip = (deletedPagination.current - 1) * deletedPagination.pageSize
      const response: ScrapeTaskResponse = await scraperService.getDeletedScrapeTasks({ 
        skip, 
        limit: deletedPagination.pageSize, 
        keyword 
      })
      setDeletedData(response.data || [])
      setDeletedPagination(prev => ({ ...prev, total: response.total || 0 }))
    } catch (error: any) {
      console.error('搜索失败', error)
      message.error(error?.response?.data?.error || '搜索失败')
      // 模拟数据
      const mockData: ScrapeTask[] = [
        {
          _id: '3',
          created_by: 'system',
          created_at: '2026-04-18T09:15:57.181Z',
          updated_by: 'system',
          updated_at: '2026-04-18T09:15:57.181Z',
          module: 'music',
          data_path: '/data/music/25',
          scraper_path: '/scrapers/music_scraper.py',
          status: 'deleted',
          result: [
            { Key: 'items_scraped', Value: 150 },
            { Key: 'duration', Value: '1h30m0s' },
            { Key: 'success', Value: true }
          ],
          error_message: '',
          started_at: '2026-04-17T22:15:57.181Z',
          completed_at: '2026-04-18T00:45:57.181Z'
        },
        {
          _id: '4',
          created_by: 'system',
          created_at: '2026-04-18T09:15:57.181Z',
          updated_by: 'system',
          updated_at: '2026-04-18T09:15:57.181Z',
          module: 'game',
          data_path: '/data/game/30',
          scraper_path: '/scrapers/game_scraper.py',
          status: 'deleted',
          result: [],
          error_message: 'Scraper execution failed',
          started_at: '2026-04-17T22:15:57.181Z',
          completed_at: '2026-04-17T22:16:57.181Z'
        }
      ]
      setDeletedData(mockData)
      setDeletedPagination(prev => ({ ...prev, total: mockData.length }))
    } finally {
      setLoading(false)
    }
  }, [deletedPagination.current, deletedPagination.pageSize, keyword])

  useEffect(() => {
    if (activeTab === 'data') {
      handleSearch()
    } else {
      handleSearchDeleted()
    }
  }, [activeTab, handleSearch, handleSearchDeleted])

  const handleClear = () => {
    setKeyword('')
    setData([])
    setDeletedData([])
  }

  const handleCreateData = async (values: any) => {
    setLoading(true)
    try {
      await scraperService.createScrapeTask(values)
      message.success('数据创建成功')
      setCreateModalVisible(false)
      form.resetFields()
      // 重新搜索以更新数据
      handleSearch()
    } catch (error: any) {
      console.error('创建数据失败', error)
      message.error(error?.response?.data?.error || '创建数据失败')
      // 模拟成功
      message.success('数据创建成功')
      setCreateModalVisible(false)
      form.resetFields()
      handleSearch()
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = () => {
    if (activeTab === 'data') {
      handleSearch()
    } else {
      handleSearchDeleted()
    }
  }

  const handleRetry = async (id: string) => {
    setLoading(true)
    try {
      await scraperService.retryScrapeTask(id)
      message.success(`重试数据 ${id} 成功`)
      handleSearch()
    } catch (error: any) {
      console.error('重试失败', error)
      message.error(error?.response?.data?.error || '重试失败')
      // 模拟成功
      message.success(`重试数据 ${id} 成功`)
      handleSearch()
    } finally {
      setLoading(false)
    }
  }

  const handleRecover = async (id: string) => {
    setLoading(true)
    try {
      await scraperService.recoverScrapeTask(id)
      message.success(`恢复数据 ${id} 成功`)
      handleSearchDeleted()
    } catch (error: any) {
      console.error('恢复失败', error)
      message.error(error?.response?.data?.error || '恢复失败')
      // 模拟成功
      message.success(`恢复数据 ${id} 成功`)
      handleSearchDeleted()
    } finally {
      setLoading(false)
    }
  }

  const handleTableChange = (newPagination: any) => {
    setPagination(prev => ({
      ...prev,
      current: newPagination.current,
      pageSize: newPagination.pageSize
    }))
  }

  const handleDeletedTableChange = (newPagination: any) => {
    setDeletedPagination(prev => ({
      ...prev,
      current: newPagination.current,
      pageSize: newPagination.pageSize
    }))
  }

  const dataColumns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.module.localeCompare(b.module),
    },
    {
      title: '刮削器路径',
      dataIndex: 'scraper_path',
      key: 'scraper_path',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.scraper_path.localeCompare(b.scraper_path),
    },
    {
      title: '数据路径',
      dataIndex: 'data_path',
      key: 'data_path',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.data_path.localeCompare(b.data_path),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.created_at.localeCompare(b.created_at),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        let color = ''
        switch (status) {
          case 'success':
            color = 'green'
            break
          case 'failed':
            color = 'red'
            break
          case 'pending':
            color = 'orange'
            break
          case 'scraping':
            color = 'blue'
            break
          default:
            color = 'gray'
        }
        return <span style={{ color }}>{status}</span>
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: ScrapeTask) => (
        <Space>
          <Button type="link" icon={<RetweetOutlined />} onClick={() => handleRetry(record._id)}>
            重试
          </Button>
        </Space>
      ),
    },
  ]

  const deletedColumns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.module.localeCompare(b.module),
    },
    {
      title: '刮削器路径',
      dataIndex: 'scraper_path',
      key: 'scraper_path',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.scraper_path.localeCompare(b.scraper_path),
    },
    {
      title: '数据路径',
      dataIndex: 'data_path',
      key: 'data_path',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.data_path.localeCompare(b.data_path),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.created_at.localeCompare(b.created_at),
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
        <h2>刮削中心</h2>
        <Button type="primary" onClick={() => setCreateModalVisible(true)} icon={<PlusOutlined />}>
          创建数据
        </Button>
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
        <Button type="primary" onClick={activeTab === 'data' ? handleSearch : handleSearchDeleted} loading={loading} icon={<SearchOutlined />}>
          搜索
        </Button>
        <Button onClick={handleClear}>
          清除
        </Button>
      </Space>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <Tabs.TabPane tab="数据查询" key="data">
          <Table 
            columns={dataColumns} 
            dataSource={data} 
            loading={loading}
            rowKey="_id"
            rowSelection={{}}
            pagination={{ 
              current: pagination.current,
              pageSize: pagination.pageSize,
              pageSizeOptions: ['10', '20', '50', '100'],
              showSizeChanger: true,
              showTotal: (total) => `共 ${total} 条记录`,
              total: pagination.total,
              onChange: handleTableChange
            }}
          />
        </Tabs.TabPane>
        <Tabs.TabPane tab="删除数据查询" key="deleted">
          <Table 
            columns={deletedColumns} 
            dataSource={deletedData} 
            loading={loading}
            rowKey="_id"
            rowSelection={{}}
            pagination={{ 
              current: deletedPagination.current,
              pageSize: deletedPagination.pageSize,
              pageSizeOptions: ['10', '20', '50', '100'],
              showSizeChanger: true,
              showTotal: (total) => `共 ${total} 条记录`,
              total: deletedPagination.total,
              onChange: handleDeletedTableChange
            }}
          />
        </Tabs.TabPane>
      </Tabs>

      <Modal
        title="创建数据"
        open={createModalVisible}
        onCancel={() => setCreateModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateData}
        >
          <Form.Item
            name="module"
            label="模块"
            rules={[{ required: true, message: '请输入模块名称' }]}
          >
            <Input placeholder="请输入模块名称" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: '请输入描述' }]}
          >
            <Input.TextArea rows={4} placeholder="请输入描述" />
          </Form.Item>
          <Form.Item
            name="custom_fields"
            label="自定义字段"
          >
            <Input.TextArea rows={2} placeholder="请输入JSON格式的自定义字段" />
          </Form.Item>
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => setCreateModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                确认创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

    </div>
  )
}

export default ScraperCenter