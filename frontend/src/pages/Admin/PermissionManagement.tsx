import { Button, Table, Modal, Form, Input, Space, message, Tag, Select, Popconfirm } from 'antd'
import { useState, useEffect, useCallback } from 'react'
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons'
import { permissionService } from '../../services/rbac'
import type { Permission, ApiResponse } from '../../types'

const modules = [
  { value: 'user', label: '用户管理' },
  { value: 'role', label: '角色管理' },
  { value: 'permission', label: '权限管理' },
  { value: 'data', label: '数据管理' },
]

const PermissionManagement: React.FC = () => {
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [editingPermission, setEditingPermission] = useState<Permission | null>(null)
  const [searchText, setSearchText] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  const fetchPermissions = useCallback(async () => {
    setLoading(true)
    try {
      const skip = (pagination.current - 1) * pagination.pageSize
      const response = await permissionService.getPermissions({ skip, limit: pagination.pageSize })
      setPermissions(response.data || [])
      setPagination(prev => ({ ...prev, total: response.total || 0 }))
    } catch (error: any) {
      console.error('获取权限列表失败', error)
      message.error(error?.response?.data?.error || '获取权限列表失败')
    } finally {
      setLoading(false)
    }
  }, [pagination.current, pagination.pageSize])

  useEffect(() => {
    fetchPermissions()
  }, [fetchPermissions])

  const handleAddPermission = () => {
    setEditingPermission(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEditPermission = (permission: Permission) => {
    setEditingPermission(permission)
    form.setFieldsValue(permission)
    setModalVisible(true)
  }

  const handleDeletePermission = async (permissionId: string) => {
    try {
      const response: ApiResponse = await permissionService.deletePermission(permissionId)
      message.success(response.message || '权限删除成功')
      fetchPermissions()
    } catch (error: any) {
      console.error('删除权限失败', error)
      message.error(error?.response?.data?.error || '删除权限失败')
    }
  }

  const handleSubmit = async (values: any) => {
    setLoading(true)
    try {
      if (editingPermission) {
        await permissionService.updatePermission(editingPermission.id, values)
        message.success('权限更新成功')
      } else {
        await permissionService.createPermission(values)
        message.success('权限创建成功')
      }
      setModalVisible(false)
      fetchPermissions()
    } catch (error: any) {
      console.error('保存权限失败', error)
      message.error(error?.response?.data?.error || '保存权限失败')
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

  const handleRefresh = () => {
    fetchPermissions()
  }

  const filteredPermissions = permissions.filter(permission => {
    return permission.name.includes(searchText) || permission.code.includes(searchText) || permission.description.includes(searchText)
  })

  const columns = [
    {
      title: '权限名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: Permission, b: Permission) => a.name.localeCompare(b.name),
      filterSearch: true,
      filters: [
        { text: '用户管理', value: '用户管理' },
        { text: '角色管理', value: '角色管理' },
        { text: '权限管理', value: '权限管理' },
        { text: '数据查询', value: '数据查询' },
        { text: '数据管理', value: '数据管理' },
      ],
      onFilter: (value: any, record: Permission) => record.name.includes(value),
    },
    {
      title: '权限代码',
      dataIndex: 'code',
      key: 'code',
      sorter: (a: Permission, b: Permission) => a.code.localeCompare(b.code),
    },
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      sorter: (a: Permission, b: Permission) => (a.module || '').localeCompare(b.module || ''),
      render: (module: string) => {
        const moduleInfo = modules.find(m => m.value === module)
        return moduleInfo ? <Tag color="blue">{moduleInfo.label}</Tag> : module
      },
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Permission) => (
        <Space>
          <Button type="link" icon={<EyeOutlined />}>
            详情
          </Button>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditPermission(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此权限吗？"
            onConfirm={() => handleDeletePermission(record.id)}
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
    <div>
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Space>
          <h2>权限管理</h2>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddPermission}>
            添加权限
          </Button>
          <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
            刷新
          </Button>
        </Space>
        <Space wrap>
          <Input
            placeholder="搜索权限名称、代码或描述"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 200 }}
            prefix={<SearchOutlined />}
          />
        </Space>
      </Space>
      <Table
        columns={columns}
        dataSource={filteredPermissions}
        rowKey="id"
        loading={loading}
        pagination={{ 
          ...pagination, 
          pageSize: pagination.pageSize,
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`
        }}
        onChange={handleTableChange}
        rowSelection={{}}
      />

      <Modal
        title={editingPermission ? '编辑权限' : '添加权限'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={editingPermission || {}}
        >
          <Form.Item
            name="name"
            label="权限名称"
            rules={[{ required: true, message: '请输入权限名称' }]}
          >
            <Input placeholder="请输入权限名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="权限代码"
            rules={[{ required: true, message: '请输入权限代码' }]}
          >
            <Input placeholder="请输入权限代码，格式：模块:操作" disabled={!!editingPermission} />
          </Form.Item>
          <Form.Item
            name="module"
            label="模块"
            rules={[{ required: true, message: '请选择模块' }]}
          >
            <Select
              placeholder="请选择模块"
              options={modules}
            />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                保存
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default PermissionManagement