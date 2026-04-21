import { Button, Card, Table, message, Space, Modal, Form, Input, Select, Popconfirm, InputNumber } from 'antd'
import { useState, useEffect } from 'react'
import { ReloadOutlined, PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface Constraint {
  type?: string
  min?: number
  max?: number
  pattern?: string
  enum_values?: string[]
  max_length?: number
}

interface FieldDefinition {
  _id: string
  module: string
  field_name: string
  field_type: string
  description: string
  required: boolean
  default_value?: any
  constraints: Constraint
  created_at: string
  updated_at: string
}

interface Collection {
  module: string
  description: string
  datatype_owner: string
  collection_name: string
}

const FIELD_TYPES = [
  { value: 'string', label: '字符串' },
  { value: 'number', label: '数字' },
  { value: 'boolean', label: '布尔值' },
  { value: 'date', label: '日期' },
  { value: 'array', label: '数组' },
  { value: 'object', label: '对象' },
]

const CONSTRAINT_TYPES = [
  { value: 'string', label: '字符串' },
  { value: 'number', label: '数字' },
  { value: 'enum', label: '枚举' },
  { value: 'pattern', label: '正则表达式' },
]

const CustomFieldsPage: React.FC = () => {
  const [fields, setFields] = useState<FieldDefinition[]>([])
  const [loading, setLoading] = useState(false)
  const [module, setModule] = useState<string>('movie')
  const [modules, setModules] = useState<Collection[]>([])
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingField, setEditingField] = useState<FieldDefinition | null>(null)
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()

  useEffect(() => {
    const fetchCollections = async () => {
      try {
        const response = await apiClient.get('/api/collections')
        const collections = response.data.data || []
        setModules(collections)
        if (collections.length > 0 && !module) {
          setModule(collections[0].module)
        }
      } catch (error) {
        console.error('获取集合列表失败', error)
      }
    }
    fetchCollections()
  }, [])

  useEffect(() => {
    const fetchFields = async () => {
      if (!module) return
      setLoading(true)
      try {
        const response = await apiClient.get(`/api/fields/module/${module}`)
        setFields(response.data.data || [])
      } catch (error: any) {
        console.error('获取字段定义失败', error)
        message.error(error?.response?.data?.error || '获取字段定义失败')
        setFields([])
      } finally {
        setLoading(false)
      }
    }
    fetchFields()
  }, [module])

  const handleRefresh = () => {
    if (module) {
      setLoading(true)
      apiClient.get(`/api/fields/module/${module}`)
        .then(response => {
          setFields(response.data.data || [])
        })
        .catch(error => {
          message.error(error?.response?.data?.error || '获取字段定义失败')
        })
        .finally(() => {
          setLoading(false)
        })
    }
  }

  const handleModuleChange = (newModule: string) => {
    setModule(newModule)
  }

  const handleCreateField = async (values: any) => {
    if (!module) {
      message.error('请先选择模块')
      return
    }
    setLoading(true)
    try {
      const payload: any = {
        module,
        field_name: values.field_name,
        field_type: values.field_type,
        description: values.description || '',
        required: values.required || false,
        default_value: values.default_value,
        constraints: {
          type: values.constraints?.type,
          min: values.constraints?.min,
          max: values.constraints?.max,
          max_length: values.constraints?.max_length,
          pattern: values.constraints?.pattern,
        }
      }

      if (values.constraints?.enum_values) {
        const enumStr = values.constraints.enum_values
        if (typeof enumStr === 'string') {
          payload.constraints.enum_values = enumStr.split(',').map((s: string) => s.trim()).filter((s: string) => s)
        } else {
          payload.constraints.enum_values = enumStr
        }
      }

      await apiClient.post('/api/fields', payload)
      message.success('字段创建成功')
      setCreateModalVisible(false)
      form.resetFields()
      handleRefresh()
    } catch (error: any) {
      console.error('创建字段失败', error)
      message.error(error?.response?.data?.error || '创建字段失败')
    } finally {
      setLoading(false)
    }
  }

  const handleEditField = (record: FieldDefinition) => {
    setEditingField(record)
    const editValues: any = {
      field_name: record.field_name,
      field_type: record.field_type,
      description: record.description,
      required: record.required,
      default_value: record.default_value,
      constraints: {
        type: record.constraints?.type,
        min: record.constraints?.min,
        max: record.constraints?.max,
        max_length: record.constraints?.max_length,
        pattern: record.constraints?.pattern,
      }
    }
    if (record.constraints?.enum_values) {
      editValues.constraints.enum_values = record.constraints.enum_values.join(', ')
    }
    editForm.setFieldsValue(editValues)
    setEditModalVisible(true)
  }

  const handleUpdateField = async (values: any) => {
    if (!editingField) return
    setLoading(true)
    try {
      const payload: any = {
        field_type: values.field_type,
        description: values.description || '',
        required: values.required || false,
        default_value: values.default_value,
        constraints: {
          type: values.constraints?.type,
          min: values.constraints?.min,
          max: values.constraints?.max,
          max_length: values.constraints?.max_length,
          pattern: values.constraints?.pattern,
        }
      }

      if (values.constraints?.enum_values) {
        const enumStr = values.constraints.enum_values
        if (typeof enumStr === 'string') {
          payload.constraints.enum_values = enumStr.split(',').map((s: string) => s.trim()).filter((s: string) => s)
        } else {
          payload.constraints.enum_values = enumStr
        }
      }

      await apiClient.put(`/api/fields/${editingField._id}`, payload)
      message.success('字段更新成功')
      setEditModalVisible(false)
      editForm.resetFields()
      setEditingField(null)
      handleRefresh()
    } catch (error: any) {
      console.error('更新字段失败', error)
      message.error(error?.response?.data?.error || '更新字段失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteField = async (id: string) => {
    setLoading(true)
    try {
      await apiClient.delete(`/api/fields/${id}`)
      message.success('字段删除成功')
      handleRefresh()
    } catch (error: any) {
      console.error('删除字段失败', error)
      message.error(error?.response?.data?.error || '删除字段失败')
    } finally {
      setLoading(false)
    }
  }

  const expandedRowRender = (record: FieldDefinition) => {
    return (
      <Card size="small" title="字段约束详情">
        <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
          {JSON.stringify(record.constraints, null, 2)}
        </pre>
      </Card>
    )
  }

  const columns = [
    {
      title: '字段名称',
      dataIndex: 'field_name',
      key: 'field_name',
    },
    {
      title: '字段类型',
      dataIndex: 'field_type',
      key: 'field_type',
      render: (type: string) => FIELD_TYPES.find(t => t.value === type)?.label || type
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '必填',
      dataIndex: 'required',
      key: 'required',
      render: (required: boolean) => required ? '是' : '否'
    },
    {
      title: '默认值',
      dataIndex: 'default_value',
      key: 'default_value',
      render: (val: any) => val !== undefined ? String(val) : '-'
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: FieldDefinition) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditField(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此字段吗？"
            onConfirm={() => handleDeleteField(record._id)}
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

  return (
    <>
      <Card
        title={
          <Space>
            <span>自定义字段定义</span>
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
            <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
              刷新
            </Button>
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModalVisible(true)} disabled={!module}>
            创建字段
          </Button>
        }
        style={{ marginBottom: 16 }}
      >
        <Table
          columns={columns}
          dataSource={fields}
          loading={loading}
          rowKey="_id"
          expandable={{
            expandedRowRender,
            rowExpandable: () => true
          }}
          pagination={{
            pageSizeOptions: ['10', '20', '50', '100'],
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条记录`
          }}
        />
      </Card>

      <Modal
        title="创建字段"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false)
          form.resetFields()
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateField}
        >
          <Form.Item
            name="field_name"
            label="字段名称"
            rules={[{ required: true, message: '请输入字段名称' }]}
          >
            <Input placeholder="请输入字段名称，如：title" />
          </Form.Item>
          <Form.Item
            name="field_type"
            label="字段类型"
            rules={[{ required: true, message: '请选择字段类型' }]}
          >
            <Select placeholder="请选择字段类型" options={FIELD_TYPES} />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={2} placeholder="请输入字段描述" />
          </Form.Item>
          <Form.Item
            name="required"
            label="是否必填"
            valuePropName="checked"
          >
            <Select placeholder="是否必填">
              <Select.Option value={true}>是</Select.Option>
              <Select.Option value={false}>否</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="default_value"
            label="默认值"
          >
            <Input placeholder="请输入默认值" />
          </Form.Item>
          <Form.Item label="约束条件">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Form.Item name={['constraints', 'type']} label="约束类型" style={{ marginBottom: 8 }}>
                <Select placeholder="约束类型" options={CONSTRAINT_TYPES} />
              </Form.Item>
              <Form.Item name={['constraints', 'min']} label="最小值" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最小值" />
              </Form.Item>
              <Form.Item name={['constraints', 'max']} label="最大值" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最大值" />
              </Form.Item>
              <Form.Item name={['constraints', 'max_length']} label="最大长度" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最大长度" />
              </Form.Item>
              <Form.Item name={['constraints', 'pattern']} label="正则表达式" style={{ marginBottom: 8 }}>
                <Input placeholder="请输入正则表达式" />
              </Form.Item>
              <Form.Item name={['constraints', 'enum_values']} label="枚举值（逗号分隔）" style={{ marginBottom: 8 }}>
                <Input.TextArea rows={2} placeholder="如：value1,value2,value3" />
              </Form.Item>
            </Space>
          </Form.Item>
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
        title="编辑字段"
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false)
          editForm.resetFields()
          setEditingField(null)
        }}
        footer={null}
        width={600}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={handleUpdateField}
        >
          <Form.Item
            name="field_name"
            label="字段名称"
            rules={[{ required: true, message: '请输入字段名称' }]}
          >
            <Input placeholder="请输入字段名称" disabled />
          </Form.Item>
          <Form.Item
            name="field_type"
            label="字段类型"
            rules={[{ required: true, message: '请选择字段类型' }]}
          >
            <Select placeholder="请选择字段类型" options={FIELD_TYPES} />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={2} placeholder="请输入字段描述" />
          </Form.Item>
          <Form.Item
            name="required"
            label="是否必填"
            valuePropName="checked"
          >
            <Select placeholder="是否必填">
              <Select.Option value={true}>是</Select.Option>
              <Select.Option value={false}>否</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="default_value"
            label="默认值"
          >
            <Input placeholder="请输入默认值" />
          </Form.Item>
          <Form.Item label="约束条件">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Form.Item name={['constraints', 'type']} label="约束类型" style={{ marginBottom: 8 }}>
                <Select placeholder="约束类型" options={CONSTRAINT_TYPES} />
              </Form.Item>
              <Form.Item name={['constraints', 'min']} label="最小值" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最小值" />
              </Form.Item>
              <Form.Item name={['constraints', 'max']} label="最大值" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最大值" />
              </Form.Item>
              <Form.Item name={['constraints', 'max_length']} label="最大长度" style={{ marginBottom: 8 }}>
                <InputNumber style={{ width: '100%' }} placeholder="最大长度" />
              </Form.Item>
              <Form.Item name={['constraints', 'pattern']} label="正则表达式" style={{ marginBottom: 8 }}>
                <Input placeholder="请输入正则表达式" />
              </Form.Item>
              <Form.Item name={['constraints', 'enum_values']} label="枚举值（逗号分隔）" style={{ marginBottom: 8 }}>
                <Input.TextArea rows={2} placeholder="如：value1,value2,value3" />
              </Form.Item>
            </Space>
          </Form.Item>
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

export default CustomFieldsPage