import { Button, Table, Modal, Form, Input, Space, message, Checkbox, Divider, Badge, Popconfirm } from 'antd'
import { useState, useEffect, useCallback } from 'react'
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import { roleService, permissionService, type RoleWithPermissions } from '../../services/rbac'
import type { Permission, ApiResponse } from '../../types'

const RoleManagement: React.FC = () => {
  const [roles, setRoles] = useState<RoleWithPermissions[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [editingRole, setEditingRole] = useState<RoleWithPermissions | null>(null)
  const [searchText, setSearchText] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [permissionGroups, setPermissionGroups] = useState<{ [key: string]: Permission[] }>({})

  const fetchRoles = useCallback(async () => {
    setLoading(true)
    try {
      const skip = (pagination.current - 1) * pagination.pageSize
      const response = await roleService.getRoles({ skip, limit: pagination.pageSize })
      setRoles(response.data || [])
      setPagination(prev => ({ ...prev, total: response.total || 0 }))
    } catch (error: any) {
      console.error('获取角色列表失败', error)
      message.error(error?.response?.data?.error || '获取角色列表失败')
    } finally {
      setLoading(false)
    }
  }, [pagination.current, pagination.pageSize])

  const fetchPermissions = useCallback(async () => {
    try {
      const response = await permissionService.getPermissions({ limit: 100 })
      setPermissions(response.data || [])
    } catch (error: any) {
      console.error('获取权限列表失败', error)
      message.error(error?.response?.data?.error || '获取权限列表失败')
    }
  }, [])

  useEffect(() => {
    fetchRoles()
    fetchPermissions()
  }, [fetchRoles, fetchPermissions])

  useEffect(() => {
    const groups: { [key: string]: Permission[] } = {}
    permissions.forEach(permission => {
      const group = permission.code.split(':')[0]
      if (!groups[group]) {
        groups[group] = []
      }
      groups[group].push(permission)
    })
    setPermissionGroups(groups)
  }, [permissions])

  const handleAddRole = () => {
    setEditingRole(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEditRole = (role: RoleWithPermissions) => {
    setEditingRole(role)
    form.setFieldsValue({
      ...role,
      permission_ids: role.permission_ids || []
    })
    setModalVisible(true)
  }

  const handleDeleteRole = async (roleId: string) => {
    try {
      const response: ApiResponse = await roleService.deleteRole(roleId)
      message.success(response.message || '角色删除成功')
      fetchRoles()
    } catch (error: any) {
      console.error('删除角色失败', error)
      message.error(error?.response?.data?.error || '删除角色失败')
    }
  }

  const handleSubmit = async (values: any) => {
    setLoading(true)
    try {
      if (editingRole) {
        await roleService.updateRole(editingRole.id, values)
        message.success('角色更新成功')
      } else {
        await roleService.createRole(values)
        message.success('角色创建成功')
      }
      setModalVisible(false)
      fetchRoles()
    } catch (error: any) {
      console.error('保存角色失败', error)
      message.error(error?.response?.data?.error || '保存角色失败')
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
    fetchRoles()
  }

  const filteredRoles = roles.filter(role => {
    return role.name.includes(searchText) || role.code.includes(searchText) || role.description.includes(searchText)
  })

  const columns = [
    {
      title: '角色名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: RoleWithPermissions, b: RoleWithPermissions) => a.name.localeCompare(b.name),
      filterSearch: true,
      filters: [
        { text: '超级管理员', value: '超级管理员' },
        { text: '普通用户', value: '普通用户' },
        { text: '数据管理员', value: '数据管理员' },
      ],
      onFilter: (value: any, record: RoleWithPermissions) => record.name.includes(value),
    },
    {
      title: '角色代码',
      dataIndex: 'code',
      key: 'code',
      sorter: (a: RoleWithPermissions, b: RoleWithPermissions) => a.code.localeCompare(b.code),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '权限数量',
      key: 'permission_count',
      render: (_: any, record: RoleWithPermissions) => (
        <Badge count={record.permission_ids?.length || 0} showZero />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: RoleWithPermissions) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditRole(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此角色吗？"
            onConfirm={() => handleDeleteRole(record.id)}
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
          <h2>角色管理</h2>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddRole}>
            添加角色
          </Button>
          <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
            刷新
          </Button>
        </Space>
        <Space wrap>
          <Input
            placeholder="搜索角色名称、代码或描述"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 200 }}
            prefix={<SearchOutlined />}
          />
        </Space>
      </Space>
      <Table
        columns={columns}
        dataSource={filteredRoles}
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
        title={editingRole ? '编辑角色' : '添加角色'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={editingRole || {}}
        >
          <Form.Item
            name="name"
            label="角色名称"
            rules={[{ required: true, message: '请输入角色名称' }]}
          >
            <Input placeholder="请输入角色名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="角色代码"
            rules={[{ required: true, message: '请输入角色代码' }]}
          >
            <Input placeholder="请输入角色代码" disabled={!!editingRole} />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
          <Form.Item
            name="permission_ids"
            label="权限"
            rules={[{ required: true, message: '请选择权限' }]}
          >
            <div>
              {Object.entries(permissionGroups).map(([group, groupPermissions]) => (
                <div key={group} style={{ marginBottom: 16 }}>
                  <h4>{group}</h4>
                  <Checkbox.Group>
                    <Space direction="vertical">
                      {groupPermissions.map(permission => (
                        <Checkbox key={permission.id} value={permission.id}>
                          {permission.name} ({permission.description})
                        </Checkbox>
                      ))}
                    </Space>
                  </Checkbox.Group>
                  <Divider style={{ margin: '12px 0' }} />
                </div>
              ))}
            </div>
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

export default RoleManagement