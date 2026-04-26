import { Button, Card, Table, message, Space, Modal, Form, Input, Select, Popconfirm } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { ReloadOutlined, PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface Collection {
  _id: string
  module: string
  description: string
  datatype_owner: string
  collection_name: string
  created_by: string
  created_at: string
  updated_by: string
  updated_at: string
}

interface UserOption {
  _id: string
  username: string
  email: string
}

const CollectionQueryPage: React.FC = () => {
  const [collections, setCollections] = useState<Collection[]>([])
  const [loading, setLoading] = useState(false)
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [editingCollection, setEditingCollection] = useState<Collection | null>(null)
  const [users, setUsers] = useState<UserOption[]>([])
  const [form] = Form.useForm()
  const [editForm] = Form.useForm()
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  const fetchCollections = useCallback(async (pageNum: number = 1, pageSizeNum: number = 10) => {
    setLoading(true)
    try {
      const response = await apiClient.get('/api/collections', {
        params: { page: pageNum, pageSize: pageSizeNum }
      })
      setCollections(response.data.data || [])
      setTotal(response.data.total || 0)
      setPage(response.data.page || 1)
      setPageSize(response.data.pageSize || 10)
    } catch (error: any) {
      console.error('获取集合列表失败', error)
      message.error(error?.response?.data?.error || '获取集合列表失败')
      setCollections([])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchUsers = useCallback(async () => {
    try {
      const response = await apiClient.get('/api/users', {
        params: { page: 1, pageSize: 100 }
      })
      const userData = response.data.data || []
      const mapped = userData.map((u: any) => ({
        _id: u._id,
        username: u.username,
        email: u.email
      }))
      setUsers(mapped)
    } catch (error) {
      console.error('获取用户列表失败', error)
    }
  }, [])

  useEffect(() => {
    fetchCollections(page, pageSize)
  }, [page, pageSize, fetchCollections])

  const handleTableChange = (pagination: any) => {
    setPage(pagination.current)
    setPageSize(pagination.pageSize)
  }

  const handleRefresh = () => {
    fetchCollections(page, pageSize)
  }

  const handleCreateCollection = async (values: any) => {
    setLoading(true)
    try {
      await apiClient.post('/api/collections', values)
      message.success('集合创建成功')
      setCreateModalVisible(false)
      form.resetFields()
      fetchCollections(page, pageSize)
    } catch (error: any) {
      console.error('创建集合失败', error)
      message.error(error?.response?.data?.error || '创建集合失败')
    } finally {
      setLoading(false)
    }
  }

  const handleEditCollection = (record: Collection) => {
    setEditingCollection(record)
    editForm.setFieldsValue({
      module: record.module,
      description: record.description,
      datatype_owner: record.datatype_owner
    })
    fetchUsers()
    setEditModalVisible(true)
  }

  const handleUpdateCollection = async (values: any) => {
    if (!editingCollection) return
    setLoading(true)
    try {
      await apiClient.put(`/api/collections/${editingCollection.module}`, values)
      message.success('集合更新成功')
      setEditModalVisible(false)
      editForm.resetFields()
      setEditingCollection(null)
      fetchCollections(page, pageSize)
    } catch (error: any) {
      console.error('更新集合失败', error)
      message.error(error?.response?.data?.error || '更新集合失败')
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteCollection = async (module: string) => {
    setLoading(true)
    try {
      await apiClient.delete(`/api/collections/${module}`)
      message.success('集合删除成功，已级联删除所有关联数据')
      fetchCollections(page, pageSize)
    } catch (error: any) {
      console.error('删除集合失败', error)
      message.error(error?.response?.data?.error || '删除集合失败')
    } finally {
      setLoading(false)
    }
  }

  const openCreateModal = () => {
    fetchUsers()
    setCreateModalVisible(true)
  }

  const filterOption = (input: string, option?: { label: string; value: string }) =>
    (option?.label ?? '').toLowerCase().includes(input.toLowerCase())

  const columns = [
    {
      title: '模块名称',
      dataIndex: 'module',
      key: 'module',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '集合管理员',
      dataIndex: 'datatype_owner',
      key: 'datatype_owner',
    },
    {
      title: '集合名称',
      dataIndex: 'collection_name',
      key: 'collection_name',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Collection) => (
        <Space>
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEditCollection(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定要删除此集合吗？将级联删除所有关联数据（角色、权限、字段定义、业务数据）"
            onConfirm={() => handleDeleteCollection(record.module)}
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
            <span>集合查询</span>
            <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
              刷新
            </Button>
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建集合
          </Button>
        }
        style={{ marginBottom: 16 }}
      >
        <Table
          columns={columns}
          dataSource={collections}
          loading={loading}
          rowKey="_id"
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            pageSizeOptions: ['10', '20', '50', '100'],
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条记录`,
            showQuickJumper: true
          }}
          onChange={handleTableChange}
        />
      </Card>

      <Modal
        title="创建集合"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false)
          form.resetFields()
        }}
        footer={null}
        width={500}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateCollection}
        >
          <Form.Item
            name="module"
            label="模块名称"
            rules={[{ required: true, message: '请输入模块名称' }]}
          >
            <Input placeholder="请输入模块名称，如：movie" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={3} placeholder="请输入集合描述" />
          </Form.Item>
          <Form.Item
            name="datatype_owner"
            label="集合管理员（必选）"
            rules={[{ required: true, message: '请选择集合管理员' }]}
            tooltip="被选中的用户将自动获得集合管理员权限"
          >
            <Select
              showSearch
              placeholder="请搜索并选择用户"
              filterOption={filterOption}
              options={users.map(u => ({
                label: `${u.username} (${u.email})`,
                value: u.username
              }))}
            />
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
        title="编辑集合"
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false)
          editForm.resetFields()
          setEditingCollection(null)
        }}
        footer={null}
        width={500}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={handleUpdateCollection}
        >
          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea rows={3} placeholder="请输入集合描述" />
          </Form.Item>
          <Form.Item
            name="datatype_owner"
            label="集合管理员"
          >
            <Select
              showSearch
              placeholder="请搜索并选择用户"
              filterOption={filterOption}
              options={users.map(u => ({
                label: `${u.username} (${u.email})`,
                value: u.username
              }))}
            />
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

export default CollectionQueryPage
