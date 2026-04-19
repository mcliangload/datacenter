import { Button, Card, Table, message, Space } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { ReloadOutlined } from '@ant-design/icons'
import apiClient from '../../services/api'

interface Collection {
  module: string
  description: string
  datatype_owner: string
  collection_name: string
  created_by: string
  created_at: string
  updated_by: string
  updated_at: string
}

const CollectionQueryPage: React.FC = () => {
  const [collections, setCollections] = useState<Collection[]>([])
  const [loading, setLoading] = useState(false)

  // 获取集合列表
  const fetchCollections = useCallback(async () => {
    setLoading(true)
    try {
      const response = await apiClient.get('/api/collections')
      setCollections(response.data.data || [])
    } catch (error: any) {
      console.error('获取集合列表失败', error)
      message.error(error?.response?.data?.error || '获取集合列表失败')
      // 模拟数据
      const mockCollections: Collection[] = [
        {
          module: 'book',
          description: '图书数据模块',
          datatype_owner: 'admin',
          collection_name: 'book_data',
          created_by: 'admin',
          created_at: new Date().toISOString(),
          updated_by: 'admin',
          updated_at: new Date().toISOString()
        },
        {
          module: 'movie',
          description: '电影数据模块',
          datatype_owner: 'admin',
          collection_name: 'movie_data',
          created_by: 'admin',
          created_at: new Date().toISOString(),
          updated_by: 'admin',
          updated_at: new Date().toISOString()
        }
      ]
      setCollections(mockCollections)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchCollections()
  }, [fetchCollections])

  const handleRefresh = () => {
    fetchCollections()
  }

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
      title: '数据所有者',
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
  ]

  return (
    <Card 
      title={
        <Space>
          '集合查询'
          <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
            刷新
          </Button>
        </Space>
      } 
      style={{ marginBottom: 16 }}
    >
      <Table
        columns={columns}
        dataSource={collections}
        loading={loading}
        rowKey="module"
        pagination={{
          pageSizeOptions: ['10', '20', '50', '100'],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条记录`
        }}
      />
    </Card>
  )
}

export default CollectionQueryPage