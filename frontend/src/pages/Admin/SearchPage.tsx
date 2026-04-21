import { Button, Card, Table, message, Space, Modal, Form, Input, Select, Popconfirm, Tag, Tooltip } from 'antd'
import { useState, useCallback, useEffect, useMemo } from 'react'
import { ReloadOutlined, SearchOutlined, DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'
import { jqlExamples } from '../../services/jql'
import './SearchPage.css'

interface BusinessData {
  _id: string
  module: string
  description?: string
  custom_fields?: Record<string, any>
  created_at: string
  updated_at: string
  created_by: string
}

interface FieldDefinition {
  _id: string
  module: string
  field_name: string
  field_type: string
  description: string
  required: boolean
  default_value?: any
  constraints: {
    type?: string
    min?: number
    max?: number
    min_length?: number
    max_length?: number
    pattern?: string
    enum_values?: string[]
  }
}

interface Collection {
  module: string
  description: string
  datatype_owner: string
  collection_name: string
}

interface DynamicField {
  name: string
  type: string
  required: boolean
  constraints: {
    type?: string
    min?: number
    max?: number
    min_length?: number
    max_length?: number
    pattern?: string
    enum_values?: string[]
  }
  isDefinition: boolean
}

const SearchPage: React.FC = () => {
  const [data, setData] = useState<BusinessData[]>([])
  const [loading, setLoading] = useState(false)
  const [module, setModule] = useState<string>('')
  const [modules, setModules] = useState<Collection[]>([])
  const [jqlQuery, setJqlQuery] = useState<string>('')
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingData, setEditingData] = useState<BusinessData | null>(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  const [fieldDefinitions, setFieldDefinitions] = useState<FieldDefinition[]>([])
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })

  const fetchCollections = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/collections')
      const collections = response.data.data || []
      setModules(collections)
      if (collections.length > 0 && !module) {
        setModule(collections[0].module)
      }
    } catch (error) {
      console.error('获取集合列表失败', error)
      setModules([])
    }
  }, [module])

  const fetchFieldDefinitions = useCallback(async () => {
    if (!module) {
      setFieldDefinitions([])
      return
    }
    try {
      const response = await apiClient.get(`/api/fields/module/${module}`)
      const fields = response.data.data as FieldDefinition[]
      setFieldDefinitions(fields || [])
    } catch (error) {
      console.error('获取字段定义失败', error)
      setFieldDefinitions([])
    }
  }, [module])

  const fetchData = useCallback(async () => {
    if (!module) return
    setLoading(true)
    try {
      let url = `/api/business/module/${module}?page=${pagination.current}&pageSize=${pagination.pageSize}`
      if (jqlQuery) {
        url += `&jql=${encodeURIComponent(jqlQuery)}`
      }
      const response = await apiClient.get(url)
      setData(response.data.data || [])
      if (response.data.total !== undefined) {
        setPagination(prev => ({ ...prev, total: response.data.total }))
      }
    } catch (error: any) {
      console.error('获取业务数据失败', error)
      message.error(error?.response?.data?.error || '获取业务数据失败')
      setData([])
    } finally {
      setLoading(false)
    }
  }, [module, jqlQuery, pagination.current, pagination.pageSize])

  useEffect(() => {
    fetchCollections()
  }, [])

  useEffect(() => {
    if (module) {
      fetchFieldDefinitions()
    }
  }, [module, fetchFieldDefinitions])

  useEffect(() => {
    if (module) {
      fetchData()
    }
  }, [module, fetchData])

  const dynamicFields = useMemo((): DynamicField[] => {
    if (fieldDefinitions.length > 0) {
      return fieldDefinitions.map(f => ({
        name: f.field_name,
        type: f.field_type,
        required: f.required,
        constraints: f.constraints,
        isDefinition: true
      }))
    }

    if (data.length === 0) return []

    const allKeys = new Set<string>()
    data.forEach(item => {
      if (item.custom_fields) {
        Object.keys(item.custom_fields).forEach(key => {
          if (!['data_path', 'module', 'scrape_path', 'scraped_at', 'task_id'].includes(key)) {
            allKeys.add(key)
          }
        })
      }
    })

    return Array.from(allKeys).map(key => ({
      name: key,
      type: 'string',
      required: false,
      constraints: {},
      isDefinition: false
    }))
  }, [fieldDefinitions, data])

  const handleSearch = () => {
    setPagination(prev => ({ ...prev, current: 1 }))
    fetchData()
  }

  const handleRefresh = () => {
    fetchData()
  }

  const handleTableChange = (paginationConfig: any) => {
    setPagination(prev => ({
      ...prev,
      current: paginationConfig.current,
      pageSize: paginationConfig.pageSize
    }))
  }

  const handleModuleChange = (newModule: string) => {
    setModule(newModule)
    setJqlQuery('')
    setPagination(prev => ({ ...prev, current: 1 }))
  }

  const handleCreateData = async (values: any) => {
    if (!module) {
      message.error('请先选择模块')
      return
    }
    setLoading(true)
    try {
      const dataPayload: Record<string, any> = {}
      for (const key in values) {
        if (key !== 'description' && values[key] !== undefined) {
          dataPayload[key] = values[key]
        }
      }
      await apiClient.post('/api/business', {
        module,
        data: dataPayload,
        description: values.description || ''
      })
      message.success('数据创建成功')
      setCreateModalVisible(false)
      form.resetFields()
      fetchData()
    } catch (error: any) {
      console.error('创建数据失败', error)
      message.error(error?.response?.data?.error || '创建数据失败')
    } finally {
      setLoading(false)
    }
  }

  const handleEditData = (record: BusinessData) => {
    setEditingData(record)
    const initialValues: Record<string, any> = {}
    if (record.custom_fields) {
      for (const key in record.custom_fields) {
        if (!['data_path', 'module', 'scrape_path', 'scraped_at', 'task_id'].includes(key)) {
          initialValues[key] = record.custom_fields[key]
        }
      }
    }
    if (record.description) {
      initialValues.description = record.description
    }
    editForm.setFieldsValue(initialValues)
    setEditModalVisible(true)
  }

  const handleUpdateData = async (values: any) => {
    if (!editingData) return
    setLoading(true)
    try {
      const dataPayload: Record<string, any> = {}
      for (const key in values) {
        if (key !== 'description' && values[key] !== undefined) {
          dataPayload[key] = values[key]
        }
      }
      await apiClient.put(`/api/business/module/${editingData.module}/${editingData._id}`, {
        description: values.description,
        data: dataPayload
      })
      message.success('数据更新成功')
      setEditModalVisible(false)
      editForm.resetFields()
      setEditingData(null)
      fetchData()
    } catch (error: any) {
      console.error('更新数据失败', error)
      message.error(error?.response?.data?.error || '更新数据失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteData = async (id: string) => {
    setLoading(true)
    try {
      await apiClient.delete(`/api/business/module/${module}/${id}`)
      message.success('数据删除成功')
      fetchData()
    } catch (error: any) {
      console.error('删除数据失败', error)
      message.error(error?.response?.data?.error || '删除数据失败')
    } finally {
      setLoading(false)
    }
  }

  const renderFormItem = (field: any) => {
    const rules: any[] = []
    if (field.required) {
      rules.push({ required: true, message: `请输入${field.name}` })
    }
    if (field.constraints?.type === 'number') {
      if (field.constraints.min !== undefined) {
        rules.push({ type: 'number', min: field.constraints.min, message: `最小值为 ${field.constraints.min}` })
      }
      if (field.constraints.max !== undefined) {
        rules.push({ type: 'number', max: field.constraints.max, message: `最大值为 ${field.constraints.max}` })
      }
    }
    if (field.constraints?.enum_values && field.constraints.enum_values.length > 0) {
      return (
        <Form.Item
          key={field.name}
          name={field.name}
          label={`${field.name}${field.required ? ' *' : ''}`}
          rules={rules}
        >
          <Select placeholder={`请选择${field.name}`}>
            {field.constraints.enum_values.map((val: string) => (
              <Select.Option key={val} value={val}>{val}</Select.Option>
            ))}
          </Select>
        </Form.Item>
      )
    }

    switch (field.type) {
      case 'number':
      case 'int':
      case 'float':
        return (
          <Form.Item
            key={field.name}
            name={field.name}
            label={`${field.name}${field.required ? ' *' : ''}`}
            rules={rules}
          >
            <Input type="number" placeholder={`请输入${field.name}`} />
          </Form.Item>
        )
      case 'boolean':
        return (
          <Form.Item
            key={field.name}
            name={field.name}
            label={`${field.name}${field.required ? ' *' : ''}`}
            valuePropName="checked"
            rules={rules}
          >
            <Select placeholder={`请选择${field.name}`}>
              <Select.Option value={true}>是</Select.Option>
              <Select.Option value={false}>否</Select.Option>
            </Select>
          </Form.Item>
        )
      case 'date':
        return (
          <Form.Item
            key={field.name}
            name={field.name}
            label={`${field.name}${field.required ? ' *' : ''}`}
            rules={rules}
          >
            <Input type="date" placeholder={`请输入${field.name}`} />
          </Form.Item>
        )
      default:
        return (
          <Form.Item
            key={field.name}
            name={field.name}
            label={`${field.name}${field.required ? ' *' : ''}`}
            rules={rules}
          >
            <Input placeholder={`请输入${field.name}`} />
          </Form.Item>
        )
    }
  }

  const getColumns = () => {
    const baseColumns: any[] = [
      {
        title: 'ID',
        dataIndex: '_id',
        key: '_id',
        width: 120,
        ellipsis: true
      },
      {
        title: '描述',
        dataIndex: 'description',
        key: 'description',
        width: 150,
        ellipsis: true
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 150,
      },
      {
        title: '操作',
        key: 'action',
        width: 150,
        render: (_: any, record: BusinessData) => (
          <Space>
            <Button type="link" icon={<EditOutlined />} onClick={() => handleEditData(record)}>
              编辑
            </Button>
            <Popconfirm
              title="确定要删除此数据吗？"
              onConfirm={() => handleDeleteData(record._id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="link" danger icon={<DeleteOutlined />}>
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ]

    dynamicFields.forEach(field => {
      baseColumns.splice(baseColumns.length - 2, 0, {
        title: field.name,
        dataIndex: field.name,
        key: field.name,
        width: 120,
        ellipsis: true,
        render: (_value: any, record: BusinessData) => {
          const customFields = record.custom_fields || {}
          const fieldValue = customFields[field.name]
          if (fieldValue === null || fieldValue === undefined) {
            return <Tag color="default">-</Tag>
          }
          if (typeof fieldValue === 'boolean') {
            return <Tag color={fieldValue ? 'green' : 'red'}>{fieldValue ? '是' : '否'}</Tag>
          }
          if (field.constraints?.enum_values && field.constraints.enum_values.includes(fieldValue)) {
            return <Tag color="blue">{fieldValue}</Tag>
          }
          return <Tooltip title={String(fieldValue)}>{String(fieldValue)}</Tooltip>
        }
      })
    })

    return baseColumns
  }

  return (
    <>
      <Card style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Space wrap>
            <Select
              value={module}
              onChange={handleModuleChange}
              style={{ width: 200 }}
              placeholder="选择模块"
              loading={loading}
            >
              {modules.map(col => (
                <Select.Option key={col.module} value={col.module}>
                  {col.module} {col.description ? `- ${col.description}` : ''}
                </Select.Option>
              ))}
            </Select>
            <Input
              placeholder="输入JQL查询，如: title = '测试'"
              value={jqlQuery}
              onChange={(e) => setJqlQuery(e.target.value)}
              style={{ width: 400 }}
              onPressEnter={handleSearch}
            />
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
              刷新
            </Button>
          </Space>
          <Space wrap>
            <span style={{ color: '#666' }}>JQL示例:</span>
            {jqlExamples.slice(0, 5).map((example, index) => (
              <Tag
                key={index}
                style={{ cursor: 'pointer' }}
                onClick={() => setJqlQuery(example)}
              >
                {example.length > 30 ? example.substring(0, 30) + '...' : example}
              </Tag>
            ))}
          </Space>
        </Space>
      </Card>

      <Card
        title="业务数据"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)} disabled={!module}>
            创建数据
          </Button>
        }
      >
        <Table
          className="search-page-table"
          columns={getColumns()}
          dataSource={data}
          loading={loading}
          rowKey="_id"
          pagination={{
            ...pagination,
            pageSizeOptions: ['10', '20', '50', '100'],
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条记录`
          }}
          onChange={handleTableChange}
          scroll={{ x: 'max-content', y: 'calc(100vh - 320px)' }}
        />
      </Card>

      <Modal
        title="创建数据"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false)
          form.resetFields()
        }}
        footer={null}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateData}
        >
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={2} placeholder="请输入数据描述" />
          </Form.Item>
          {dynamicFields.map(field => renderFormItem(field))}
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => setCreateModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑数据"
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false)
          editForm.resetFields()
          setEditingData(null)
        }}
        footer={null}
        width={700}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={handleUpdateData}
        >
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={2} placeholder="请输入数据描述" />
          </Form.Item>
          {dynamicFields.map(field => renderFormItem(field))}
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => setEditModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                保存
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  )
}

export default SearchPage