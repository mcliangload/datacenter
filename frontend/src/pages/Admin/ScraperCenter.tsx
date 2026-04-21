import { Button, Input, Space, Table, Modal, Form, message, Select, Popconfirm } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { SearchOutlined, ReloadOutlined, RetweetOutlined, RollbackOutlined, PlusOutlined, DeleteOutlined } from '@ant-design/icons'
import scraperService, { type ScrapeTask, type ScrapeTaskResponse } from '../../services/scraper'
import apiClient from '../../services/api'

const ScraperCenter: React.FC = () => {
  const [keyword, setKeyword] = useState('')
  const [data, setData] = useState<ScrapeTask[]>([])
  const [deletedData, setDeletedData] = useState<ScrapeTask[]>([])
  const [loading, setLoading] = useState(false)
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [activeTab] = useState('data')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [deletedPagination, setDeletedPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [modules, setModules] = useState<{ value: string; label: string }[]>([])
  const [retryModalVisible, setRetryModalVisible] = useState(false)
  const [retryTaskId, setRetryTaskId] = useState('')
  const [retryForm] = Form.useForm()
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [scraperPaths, setScraperPaths] = useState<{ value: string; label: string }[]>([])

  const fetchData = useCallback(async (page: number, pageSize: number, searchKeyword: string) => {
    setLoading(true)
    try {
      const skip = (page - 1) * pageSize
      const response: ScrapeTaskResponse = await scraperService.getScrapeTasks({
        skip,
        limit: pageSize,
        keyword: searchKeyword
      })
      if (response && Array.isArray(response.data)) {
        setData(response.data)
        setPagination(prev => ({ ...prev, total: response.total || 0, current: page, pageSize }))
      }
    } catch (error: any) {
      message.error(error?.response?.data?.error || '获取数据失败')
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchModules = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/collections')
      const moduleList = response.data.data || []
      const moduleOptions = moduleList.map((m: any) => ({
        value: m.module,
        label: `${m.module} - ${m.description || ''}`
      }))
      setModules(moduleOptions)

      const pathSet = new Set<string>()
      moduleList.forEach((m: any) => {
        if (m.collection_name) {
          pathSet.add(`/scrapers/${m.module}_scraper.py`)
        }
      })
      const pathOptions = Array.from(pathSet).map(p => ({
        value: p,
        label: p
      }))
      setScraperPaths(pathOptions)
    } catch (error: any) {
      console.error('获取模块列表失败', error)
    }
  }, [])

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
      message.error(error?.response?.data?.error || '获取已删除数据失败')
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
    setSelectedRowKeys([])
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
      message.error(error?.response?.data?.error || '恢复失败')
    } finally {
      setLoading(false)
    }
  }

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的数据')
      return
    }
    setLoading(true)
    try {
      await scraperService.batchDeleteScrapeTasks(selectedRowKeys as string[])
      message.success(`成功删除 ${selectedRowKeys.length} 条数据`)
      setSelectedRowKeys([])
      handleSearch()
    } catch (error: any) {
      message.error(error?.response?.data?.error || '批量删除失败')
    } finally {
      setLoading(false)
    }
  }

  const handleTableChange = (pagination: any) => {
    if (activeTab === 'data') {
      fetchData(pagination.current, pagination.pageSize, keyword)
    } else {
      fetchDeletedData(pagination.current, pagination.pageSize, keyword)
    }
  }

  const onSelectChange = (newSelectedRowKeys: React.Key[]) => {
    setSelectedRowKeys(newSelectedRowKeys)
  }

  const rowSelection = {
    selectedRowKeys,
    onChange: onSelectChange
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

      {activeTab === 'data' && (
        <Space style={{ marginBottom: '16px' }}>
          <Popconfirm
            title={`确定要删除选中的 ${selectedRowKeys.length} 条数据吗？`}
            onConfirm={handleBatchDelete}
            okText="确定"
            cancelText="取消"
            disabled={selectedRowKeys.length === 0}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={selectedRowKeys.length === 0}
              loading={loading}
            >
              批量删除 {selectedRowKeys.length > 0 ? `(${selectedRowKeys.length})` : ''}
            </Button>
          </Popconfirm>
        </Space>
      )}

      <Table
        columns={activeTab === 'data' ? dataColumns : deletedColumns}
        dataSource={activeTab === 'data' ? data : deletedData}
        loading={loading}
        rowKey="_id"
        rowSelection={rowSelection}
        pagination={{
          current: activeTab === 'data' ? pagination.current : deletedPagination.current,
          pageSize: activeTab === 'data' ? pagination.pageSize : deletedPagination.pageSize,
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`,
          total: activeTab === 'data' ? pagination.total : deletedPagination.total
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
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item
            name="data_path"
            label="数据路径"
            rules={[{ required: true, message: '请输入数据路径' }]}
            tooltip="数据文件所在的目录路径，如：/data/movie"
          >
            <Input placeholder="请输入数据路径，例如：/data/movie" addonBefore="/" />
          </Form.Item>
          <Form.Item
            name="scraper_path"
            label="刮削器路径"
            rules={[{ required: true, message: '请输入刮削器路径' }]}
            tooltip="刮削器脚本的路径，如：/scrapers/movie_scraper.py"
          >
            <Select
              placeholder="选择或输入刮削器路径"
              options={scraperPaths}
              showSearch
              allowClear
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
              dropdownRender={(menu) => (
                <>
                  {menu}
                  <div style={{ padding: '8px', borderTop: '1px solid #e8e8e8' }}>
                    <Input
                      placeholder="或直接输入路径"
                      onPressEnter={(e) => {
                        const value = (e.target as HTMLInputElement).value
                        if (value) {
                          form.setFieldValue('scraper_path', '/' + value.replace(/^\/+/, ''))
                        }
                      }}
                      onBlur={(e) => {
                        const value = e.target.value
                        if (value && !value.startsWith('/')) {
                          form.setFieldValue('scraper_path', '/' + value)
                        }
                      }}
                    />
                  </div>
                </>
              )}
            />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            tooltip="可选，描述此刮削任务的用途"
          >
            <Input.TextArea rows={3} placeholder="请输入描述（可选）" />
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