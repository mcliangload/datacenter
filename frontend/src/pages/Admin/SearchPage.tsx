import { Button, Input, Space, Table, message, Select, Modal, Form, Card, Descriptions } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface TableRow {
  key: string
  module: string
  description: string
  createdTime: string
  [key: string]: any
}

interface ModuleOption {
  value: string
  label: string
}

const AdminSearchPage: React.FC = () => {
  const [keyword, setKeyword] = useState('')
  const [data, setData] = useState<TableRow[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedModule, setSelectedModule] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [modules, setModules] = useState<ModuleOption[]>([])
  const [detailModalVisible, setDetailModalVisible] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [selectedRecord, setSelectedRecord] = useState<any>(null)
  const [editForm] = Form.useForm()
  const [detailLoading, setDetailLoading] = useState(false)
  const [editLoading, setEditLoading] = useState(false)

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

  const handleSearch = useCallback(async () => {
    // 没有选择模块时不调用接口
    if (!selectedModule) {
      setData([])
      setPagination(prev => ({ ...prev, total: 0 }))
      return
    }
    
    setLoading(true)
    try {
      const module = selectedModule
      const page = pagination.current
      const pageSize = pagination.pageSize
      const jql = keyword
      
      console.log('开始搜索', { module, page, pageSize, jql })
      
      const response = await apiClient.get('/api/business/module/' + module, {
        params: { page, pageSize, jql }
      })
      
      const tableData = (response.data.data || []).map((item: any) => ({
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
      setData([])
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
    // 翻页时重新搜索
    handleSearch()
  }

  // 查看详情
  const handleViewDetail = useCallback(async (record: any) => {
    setSelectedRecord(record)
    setDetailLoading(true)
    setEditMode(false)
    try {
      // 调用后端接口获取详情，路径为 /api/business/module/:module/:id
      const response = await apiClient.get(`/api/business/module/${record.module}/${record._id || record.key}`)
      setSelectedRecord(response.data.data)
    } catch (error: any) {
      console.error('获取详情失败', error)
      // 使用原记录作为 fallback
      setSelectedRecord(record)
    } finally {
      setDetailLoading(false)
      setDetailModalVisible(true)
    }
  }, [])

  // 切换到编辑模式
  const handleSwitchToEdit = useCallback(() => {
    if (selectedRecord) {
      const editableData = { ...selectedRecord }
      // 移除不需要展示的字段
      delete editableData._id
      delete editableData.key
      // 移除不需要修改的字段
      delete editableData.createdTime
      delete editableData.created_by
      delete editableData.created_at
      delete editableData.updated_by
      delete editableData.updated_at
      editForm.setFieldsValue({ jsonData: JSON.stringify(editableData, null, 2) })
      setEditMode(true)
    }
  }, [selectedRecord, editForm])

  // 提交编辑
  const handleSubmitEdit = useCallback(async () => {
    try {
      const values = await editForm.validateFields()
      setEditLoading(true)
      
      // 解析 JSON 数据
      const jsonData = JSON.parse(values.jsonData)
      
      // 调用后端接口更新数据，路径为 PUT /api/business/module/:module/:id
      const response = await apiClient.put(`/api/business/module/${selectedRecord.module}/${selectedRecord._id || selectedRecord.key}`, jsonData)
      
      message.success('更新成功')
      // 切换回查看模式
      setEditMode(false)
      // 重新获取详情以刷新数据
      const detailResponse = await apiClient.get(`/api/business/module/${selectedRecord.module}/${selectedRecord._id || selectedRecord.key}`)
      setSelectedRecord(detailResponse.data.data)
    } catch (error: any) {
      console.error('更新失败', error)
      message.error(error?.response?.data?.error || '更新失败')
    } finally {
      setEditLoading(false)
    }
  }, [editForm, selectedRecord])

  // 当模块或分页变化时自动搜索（仅在已选择模块时）
  useEffect(() => {
    if (selectedModule) {
      handleSearch()
    }
  }, [selectedModule, pagination.current, pagination.pageSize])

  // 页面加载时获取模块列表
  useEffect(() => {
    fetchModules()
  }, [fetchModules])

  const columns = [
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      sorter: (a: TableRow, b: TableRow) => a.module.localeCompare(b.module),
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
      render: (_: any, record: any) => (
        <Space>
          <Button type="link" onClick={() => handleViewDetail(record)}>数据详情</Button>
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

      {/* 数据详情模态框 */}
      <Modal
        title={
          <Space>
            数据详情
            {!editMode && (
              <Button 
                type="link" 
                size="small" 
                onClick={handleSwitchToEdit}
              >
                编辑
              </Button>
            )}
          </Space>
        }
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false)
          setEditMode(false)
        }}
        footer={editMode ? [
          <Button key="cancel" onClick={() => setEditMode(false)}>
            取消
          </Button>,
          <Button 
            key="submit" 
            type="primary" 
            onClick={handleSubmitEdit}
            loading={editLoading}
          >
            提交
          </Button>
        ] : [
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={800}
      >
        {editMode ? (
          <Form form={editForm} layout="vertical">
            <Form.Item 
              name="jsonData" 
              label="数据 (JSON格式)"
              rules={[
                { required: true, message: '请输入JSON数据' },
                {
                  validator: (_, value) => {
                    try {
                      if (value) JSON.parse(value)
                      return Promise.resolve()
                    } catch (error) {
                      return Promise.reject(new Error('JSON格式错误'))
                    }
                  }
                }
              ]}
            >
              <Input.TextArea 
                rows={15} 
                style={{ fontFamily: 'monospace' }}
                placeholder="请输入JSON格式的数据"
              />
            </Form.Item>
          </Form>
        ) : (
          <Card loading={detailLoading}>
            <pre style={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
              {selectedRecord && JSON.stringify(selectedRecord, null, 2)}
            </pre>
          </Card>
        )}
      </Modal>
    </div>
  )
}

export default AdminSearchPage