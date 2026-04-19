import { Button, Card, Select, Table, message, Space } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { ReloadOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface FieldDefinition {
  _id: string
  module: string
  field_name: string
  field_type: string
  description: string
  constraints: any
  created_by: string
  created_at: string
  updated_by: string
  updated_at: string
}

interface ModuleOption {
  value: string
  label: string
}

const CustomFieldsPage: React.FC = () => {
  const [selectedModule, setSelectedModule] = useState('')
  const [fields, setFields] = useState<FieldDefinition[]>([])
  const [loading, setLoading] = useState(false)
  const [modules, setModules] = useState<ModuleOption[]>([
    { value: '', label: '选择模块' }
  ])

  // 获取模块列表
  const fetchModules = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/collections')
      const moduleList = response.data.data || []
      const moduleOptions = [
        { value: '', label: '选择模块' },
        ...moduleList.map((module: any) => ({
          value: module.module,
          label: module.module
        }))
      ]
      setModules(moduleOptions)
    } catch (error: any) {
      console.error('获取模块列表失败', error)
      // 使用默认模块列表
      setModules([
        { value: '', label: '选择模块' },
        { value: 'book', label: '图书' },
        { value: 'movie', label: '电影' },
        { value: 'music', label: '音乐' },
        { value: 'game', label: '游戏' },
      ])
    }
  }, [])

  // 获取自定义字段
  const fetchFields = useCallback(async () => {
    if (!selectedModule) return

    setLoading(true)
    try {
      const response = await apiClient.get(`/api/fields/module/${selectedModule}`)
      setFields(response.data.data || [])
    } catch (error: any) {
      console.error('获取自定义字段失败', error)
      message.error(error?.response?.data?.error || '获取自定义字段失败')
      // 模拟数据
      const mockFields: FieldDefinition[] = [
        {
          _id: '1',
          module: selectedModule,
          field_name: 'title',
          field_type: 'string',
          description: '标题',
          constraints: { min_length: 1, max_length: 200 },
          created_by: 'admin',
          created_at: new Date().toISOString(),
          updated_by: 'admin',
          updated_at: new Date().toISOString()
        },
        {
          _id: '2',
          module: selectedModule,
          field_name: 'author',
          field_type: 'string',
          description: '作者',
          constraints: { min_length: 1, max_length: 100 },
          created_by: 'admin',
          created_at: new Date().toISOString(),
          updated_by: 'admin',
          updated_at: new Date().toISOString()
        }
      ]
      setFields(mockFields)
    } finally {
      setLoading(false)
    }
  }, [selectedModule])

  useEffect(() => {
    fetchModules()
  }, [fetchModules])

  useEffect(() => {
    if (selectedModule) {
      fetchFields()
    }
  }, [selectedModule, fetchFields])

  const handleRefresh = () => {
    fetchFields()
  }

  const columns = [
    {
      title: '字段名',
      dataIndex: 'field_name',
      key: 'field_name',
    },
    {
      title: '字段类型',
      dataIndex: 'field_type',
      key: 'field_type',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '约束',
      dataIndex: 'constraints',
      key: 'constraints',
      render: (constraints: any) => JSON.stringify(constraints),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
  ]

  return (
    <Card 
      title={
        <Space>
          '自定义字段'
          <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
            刷新
          </Button>
        </Space>
      } 
      style={{ marginBottom: 16 }}
    >
      <Select
        placeholder="选择模块"
        value={selectedModule}
        onChange={setSelectedModule}
        options={modules}
        style={{ width: 200, marginBottom: 16 }}
      />
      <Table
        columns={columns}
        dataSource={fields}
        loading={loading}
        rowKey="_id"
        pagination={{
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`
        }}
      />
    </Card>
  )
}

export default CustomFieldsPage