import { Button, Table, Modal, Form, Input, Space, message, Popconfirm, Tag, Select } from 'antd'
import { useState, useEffect, useCallback } from 'react'
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, UserOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface User {
  _id: string
  username: string
  email: string
  role_ids: string[]
  created_at: string
  updated_at: string
}

interface Role {
  _id: string
  name: string
  code: string
  description: string
}

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [roleModalVisible, setRoleModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [roleForm] = Form.useForm()
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [searchText, setSearchText] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const response = await apiClient.get('/api/users', {
        params: {
          skip: (pagination.current - 1) * pagination.pageSize,
          limit: pagination.pageSize
        }
      })
      setUsers(response.data.data || [])
      setPagination(prev => ({ ...prev, total: response.data.total || 0 }))
    } catch (error: any) {
      console.error('获取用户列表失败', error)
      message.error(error?.response?.data?.error || '获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }, [pagination.current, pagination.pageSize])

  const fetchRoles = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/roles')
      setRoles(response.data.data || [])
    } catch (error: any) {
      console.error('获取角色列表失败', error)
      message.error(error?.response?.data?.error || '获取角色列表失败')
    }
  }, [])

  useEffect(() => {
    fetchUsers()
    fetchRoles()
  }, [fetchUsers, fetchRoles])

  const handleAddUser = () => {
    setEditingUser(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEditUser = (user: User) => {
    setEditingUser(user)
    form.setFieldsValue({
      username: user.username,
      email: user.email,
      password: ''
    })
    setModalVisible(true)
  }

  const handleDeleteUser = async (userId: string) => {
    try {
      await apiClient.delete(`/api/users/${userId}`)
      message.success('用户删除成功')
      fetchUsers()
    } catch (error: any) {
      console.error('删除用户失败', error)
      message.error(error?.response?.data?.error || '删除用户失败')
    }
  }

  const handleSubmit = async (values: any) => {
    setLoading(true)
    try {
      if (editingUser) {
        const updateData: any = {
          email: values.email
        }
        if (values.password) {
          updateData.password = values.password
        }
        await apiClient.put(`/api/users/${editingUser._id}`, updateData)
        message.success('用户更新成功')
      } else {
        await apiClient.post('/api/users', values)
        message.success('用户创建成功')
      }
      setModalVisible(false)
      fetchUsers()
    } catch (error: any) {
      console.error('保存用户失败', error)
      message.error(error?.response?.data?.error || '保存用户失败')
    } finally {
      setLoading(false)
    }
  }

  const handleOpenRoleModal = (user: User) => {
    setSelectedUser(user)
    setRoles([])
    fetchRoles().then(() => {
      roleForm.setFieldsValue({
        role_ids: user.role_ids || []
      })
      setRoleModalVisible(true)
    })
  }

  const handleAssignRoles = async (values: any) => {
    if (!selectedUser) return
    setLoading(true)
    try {
      const currentRoles = selectedUser.role_ids || []

      for (const roleId of values.assign_role_id) {
        if (!currentRoles.includes(roleId)) {
          await apiClient.post(`/api/users/${selectedUser._id}/roles`, { role_id: roleId })
        }
      }

      message.success('角色分配成功')
      setRoleModalVisible(false)
      setSelectedUser(null)
      roleForm.resetFields()
      fetchUsers()
    } catch (error: any) {
      console.error('分配角色失败', error)
      message.error(error?.response?.data?.error || '分配角色失败')
    } finally {
      setLoading(false)
    }
  }

  const handleRemoveRole = async (userId: string, roleId: string) => {
    try {
      await apiClient.delete(`/api/users/${userId}/roles/${roleId}`)
      message.success('角色移除成功')
      fetchUsers()
    } catch (error: any) {
      console.error('移除角色失败', error)
      message.error(error?.response?.data?.error || '移除角色失败')
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
    fetchUsers()
  }

  const getRoleName = (roleId: string) => {
    const role = roles.find(r => r._id === roleId)
    return role ? role.name : roleId
  }

  const filteredUsers = users.filter(user => {
    return user.username.toLowerCase().includes(searchText.toLowerCase()) ||
           user.email.toLowerCase().includes(searchText.toLowerCase())
  })

  const columns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '角色',
      dataIndex: 'role_ids',
      key: 'role_ids',
      render: (roleIds: string[], record: User) => (
        <Space wrap>
          {roleIds && roleIds.length > 0 ? (
            roleIds.map(roleId => (
              <Tag key={roleId} closable onClose={() => handleRemoveRole(record._id, roleId)}>
                {getRoleName(roleId)}
              </Tag>
            ))
          ) : (
            <Tag>未分配角色</Tag>
          )}
          <Button size="small" type="link" icon={<UserOutlined />} onClick={() => handleOpenRoleModal(record)}>
            分配角色
          </Button>
        </Space>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => text ? new Date(text).toLocaleString() : '-'
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: User) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditUser(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此用户吗？"
            onConfirm={() => handleDeleteUser(record._id)}
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
          <h2>用户管理</h2>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddUser}>
            添加用户
          </Button>
          <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
            刷新
          </Button>
        </Space>
        <Space wrap>
          <Input
            placeholder="搜索用户名或邮箱"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 200 }}
            prefix={<SearchOutlined />}
          />
        </Space>
      </Space>
      <Table
        columns={columns}
        dataSource={filteredUsers}
        rowKey="_id"
        loading={loading}
        pagination={{
          ...pagination,
          pageSize: pagination.pageSize,
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`
        }}
        onChange={handleTableChange}
      />

      <Modal
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" disabled={!!editingUser} />
          </Form.Item>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '请输入正确的邮箱格式' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={editingUser ? [] : [{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder={editingUser ? '留空则不修改密码' : '请输入密码'} />
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

      <Modal
        title={`为用户 "${selectedUser?.username}" 分配角色`}
        open={roleModalVisible}
        onCancel={() => {
          setRoleModalVisible(false)
          setSelectedUser(null)
          roleForm.resetFields()
        }}
        footer={null}
        width={500}
      >
        <Form
          form={roleForm}
          layout="vertical"
          onFinish={handleAssignRoles}
        >
          <Form.Item
            name="assign_role_id"
            label="选择要分配的角色"
            rules={[{ required: true, message: '请选择至少一个角色' }]}
          >
            <Select
              mode="multiple"
              placeholder="请选择角色"
              options={roles.map(role => ({
                value: role._id,
                label: `${role.name} (${role.code})`
              }))}
            />
          </Form.Item>
          <Form.Item>
            <Space style={{ justifyContent: 'flex-end' }}>
              <Button onClick={() => {
                setRoleModalVisible(false)
                setSelectedUser(null)
                roleForm.resetFields()
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                分配角色
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default UserManagement