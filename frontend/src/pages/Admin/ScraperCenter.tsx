import { Button, Input, Space, Table, Modal, Form, message, Select, Tabs } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { SearchOutlined, PlusOutlined, ReloadOutlined, RetweetOutlined, RollbackOutlined } from '@ant-design/icons'
import scraperService, { type ScrapeTask, type ScrapeTaskResponse } from '../../services/scraper'
import apiClient from '../../services/api'

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
  const [modules, setModules] = useState<{ value: string; label: string }[]>([])
  const [retryModalVisible, setRetryModalVisible] = useState(false)
  const [retryTaskId, setRetryTaskId] = useState('')
  const [retryForm] = Form.useForm()

  const fetchData = useCallback(async (page: number, pageSize: number, searchKeyword: string) => {
    setLoading(true)
    try {
      const skip = (page - 1) * pageSize
      console.log('开始调用getScrapeTasks接口', { page, skip, limit: pageSize, keyword: searchKeyword })
      const response: ScrapeTaskResponse = await scraperService.getScrapeTasks({
        skip,
        limit: pageSize,
        keyword: searchKeyword
      })
      console.log('getScrapeTasks接口响应', response)
      if (response && Array.isArray(response.data)) {
        console.log('数据有效，设置数据', response.data)
        setData(response.data)
        setPagination(prev => ({ ...prev, total: response.total || 0, current: page, pageSize }))
      } else {
        console.error('响应数据结构不正确', response)
        message.error('响应数据结构不正确')
      }
    } catch (error: any) {
      console.error('搜索失败', error)
      console.error('错误详情', error.response)
      message.error(error?.response?.data?.error || '搜索失败')
    } finally {
      setLoading(false)
    }
  }, [])

  // 获取模块列表
  const fetchModules = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/collections')
      const moduleList = response.data.data || []
      const moduleOptions = moduleList.map((module: any) => ({
        value: module.module,
        label: module.module
      }))
      setModules(moduleOptions)
    } catch (error: any) {
      console.error('获取模块列表失败', error)
    }
  }, [])

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
      setDeletedPagination(prev => ({ ...prev, total: response.total || 0, current: page, pageSize }))
    } catch (error: any) {
      console.error('获取已删除任务失败', error)
      message.error(error?.response?.data?.error || '获取已删除任务失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchModules()
  }, [fetchModules])

  useEffect(() => {
    if (activeTab === 'data') {
      fetchData(pagination.current, pagination.pageSize, keyword)
    } else {
      fetchDeletedData(deletedPagination.current, deletedPagination.pageSize, keyword)
    }
  }, [activeTab])

  const handleSearch = () => {
    if (activeTab === 'data') {
      fetchData(1, pagination.pageSize, keyword)
    } else {
      fetchDeletedData(1, deletedPagination.pageSize, keyword)
    }
  }

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
      handleSearch()
    } catch (error: any) {
      console.error('创建数据失败', error)
      message.error(error?.response?.data?.error || '创建数据失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = () => {
    handleSearch()
  }

  const handleRetry = async (id: string) => {
    setRetryTaskId(id)
    setRetryModalVisible(true)
  }

  const handleSubmitRetry = async (values: any) => {
    setLoading(true)
    try {
      await scraperService.retryScrapeTask(retryTaskId, values.scraper_path)
      message.success(`重试数据 ${retryTaskId} 成功`)
      setRetryModalVisible(false)
      retryForm.resetFields()
      handleSearch()
    } catch (error: any) {
      console.error('重试失败', error)
      message.error(error?.response?.data?.error || '重试失败')
    } finally {
      setLoading(false)
    }
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

  const handleTableChange = (pagination: any, filters: any, sorter: any) => {
    if (activeTab === 'data') {
      fetchData(pagination.current, pagination.pageSize, keyword)
    } else {
      fetchDeletedData(pagination.current, pagination.pageSize, keyword)
    }
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
      sorter: (a: ScrapeTask, b: ScrapeTask) => a.data_path.localeCompare(b.scraper_path),
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
        <Button type="primary" onClick={handleSearch} loading={loading} icon={<SearchOutlined />}>
          搜索
        </Button>
        <Button onClick={handleClear}>
          清除
        </Button>
      </Space>
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
          total: pagination.total
        }}
        onChange={handleTableChange}
      />

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
            rules={[{ required: true, message: '请选择模块' }]}
          >
            <Select
              placeholder="选择模块"
              options={modules}
            />
          </Form.Item>
          <Form.Item
            name="data_path"
            label="数据路径"
            rules={[{ required: true, message: '请输入数据路径' }]}
          >
            <Input placeholder="请输入数据路径，例如：/data/book" />
          </Form.Item>
          <Form.Item
            name="scraper_path"
            label="刮削器路径"
            rules={[{ required: true, message: '请输入刮削器路径' }]}
          >
            <Input placeholder="请输入刮削器路径，例如：/scrapers/book_scraper.py" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={4} placeholder="请输入描述" />
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

      <Modal
        title="重试任务"
        open={retryModalVisible}
        onCancel={() => setRetryModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={retryForm}
          layout="vertical"
          onFinish={handleSubmitRetry}
        >
          <Form.Item
            name="scraper_path"
            label="刮削器路径"
            tooltip="不输入则使用默认刮削器"
          >
            <Input placeholder="请输入刮削器路径，例如：/scrapers/book_scraper.py" />
          </Form.Item>
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => setRetryModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                确认重试
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

    </div>
  )
}

export default ScraperCenter
