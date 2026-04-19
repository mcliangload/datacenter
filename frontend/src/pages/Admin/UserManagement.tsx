import { Button, Table, Modal, Form, Input, Space, message, Popconfirm } from 'antd'
import { useState, useEffect, useCallback } from 'react'
import type { UserWithRoles } from '../../services/user'
import { userService } from '../../services'
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined } from '@ant-design/icons'
import type { ApiResponse } from '../../types'

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<UserWithRoles[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [editingUser, setEditingUser] = useState<UserWithRoles | null>(null)
  const [searchText, setSearchText] = useState('')
  const [pagination, setPagination] = useState({ current: 1, pageSize: 20, total: 0 })

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const skip = (pagination.current - 1) * pagination.pageSize
      const response = await userService.getUsers({ skip, limit: pagination.pageSize, keyword: searchText })
      setUsers(response.data || [])
      setPagination(prev => ({ ...prev, total: response.total || 0 }))
    } catch (error: any) {
      console.error('获取用户列表失败', error)
      message.error(error?.response?.data?.error || '获取用户列表失败')
    } finally {
      setLoading(false)
    }
  }, [pagination.current, pagination.pageSize, searchText])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  const handleAddUser = () => {
    setEditingUser(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEditUser = (user: UserWithRoles) => {
    setEditingUser(user)
    form.setFieldsValue({
      ...user,
      password: ''
    })
    setModalVisible(true)
  }

  const handleDeleteUser = async (userId: string) => {
    try {
      const response: ApiResponse = await userService.deleteUser(userId)
      message.success(response.message || '用户删除成功')
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
        await userService.updateUser(editingUser.id, values)
        message.success('用户更新成功')
      } else {
        await userService.createUser(values)
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

  const filteredUsers = users.filter(user => {
    return user.username.includes(searchText) || user.email.includes(searchText)
  })

  const columns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      sorter: (a: UserWithRoles, b: UserWithRoles) => a.username.localeCompare(b.username),
      filterSearch: true,
      filters: [
        { text: 'admin', value: 'admin' },
        { text: 'user1', value: 'user1' },
        { text: 'liangminchuan', value: 'liangminchuan' },
      ],
      onFilter: (value: any, record: UserWithRoles) => record.username.includes(value),
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      sorter: (a: UserWithRoles, b: UserWithRoles) => a.email.localeCompare(b.email),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <span style={{ color: status === 'active' ? 'green' : 'red' }}>
          {status === 'active' ? '活跃' : '禁用'}
        </span>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: UserWithRoles) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditUser(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此用户吗？"
            onConfirm={() => handleDeleteUser(record.id)}
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
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={editingUser || {}}
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

    </div>
  )
}

export default UserManagement