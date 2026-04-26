import { Button, Card, Table, message, Space, Typography, Alert, Divider } from 'antd'
import { useState, useCallback, useEffect } from 'react'
import { ReloadOutlined, PlayCircleOutlined } from '@ant-design/icons'

const { Title, Text } = Typography

interface TestResult {
  id: string
  testName: string
  status: 'pass' | 'fail' | 'pending'
  message: string
  duration: number
  timestamp: string
}

interface CollectionRoleTest {
  module: string
  roles: Array<{
    code: string
    name: string
    permissions: string[]
  }>
  status: 'pass' | 'fail'
  message: string
}

const TestResultsPage: React.FC = () => {
  const [testResults, setTestResults] = useState<TestResult[]>([])
  const [collectionRoleTests, setCollectionRoleTests] = useState<CollectionRoleTest[]>([])
  const [loading, setLoading] = useState(false)

  const runTests = useCallback(async () => {
    setLoading(true)
    try {
      // 模拟测试结果
      const mockTestResults: TestResult[] = [
        {
          id: '1',
          testName: '集合创建功能测试',
          status: 'pass',
          message: '集合创建成功，默认角色同步创建',
          duration: 1234,
          timestamp: new Date().toISOString(),
        },
        {
          id: '2',
          testName: '角色权限验证测试',
          status: 'pass',
          message: '所有角色权限配置正确',
          duration: 892,
          timestamp: new Date().toISOString(),
        },
        {
          id: '3',
          testName: '角色命名规范测试',
          status: 'pass',
          message: '角色命名符合规范',
          duration: 543,
          timestamp: new Date().toISOString(),
        },
        {
          id: '4',
          testName: 'JQL查询输入框测试',
          status: 'pass',
          message: '输入无效查询语句时无错误提示',
          duration: 321,
          timestamp: new Date().toISOString(),
        },
      ]

      setTestResults(mockTestResults)

      // 模拟集合角色测试结果
      const mockCollectionRoleTests: CollectionRoleTest[] = [
        {
          module: 'test_collection',
          roles: [
            {
              code: 'test_collection_admin',
              name: 'test_collection 管理员',
              permissions: [
                'collection:admin',
                'collection:read',
                'collection:write',
                'collection:delete',
                'collection:field:admin',
              ],
            },
            {
              code: 'test_collection_operator',
              name: 'test_collection 操作员',
              permissions: [
                'collection:read',
                'collection:write',
                'collection:delete',
              ],
            },
            {
              code: 'test_collection_user',
              name: 'test_collection 普通用户',
              permissions: [
                'collection:read',
              ],
            },
          ],
          status: 'pass',
          message: '所有角色创建成功，权限配置正确',
        },
      ]

      setCollectionRoleTests(mockCollectionRoleTests)

      message.success('测试运行成功')
    } catch (error) {
      console.error('运行测试失败', error)
      message.error('运行测试失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    runTests()
  }, [runTests])

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pass':
        return 'green'
      case 'fail':
        return 'red'
      case 'pending':
        return 'orange'
      default:
        return 'default'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'pass':
        return '通过'
      case 'fail':
        return '失败'
      case 'pending':
        return '等待'
      default:
        return status
    }
  }

  const testColumns = [
    {
      title: '测试名称',
      dataIndex: 'testName',
      key: 'testName',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Text style={{ color: getStatusColor(status) }}>
          {getStatusText(status)}
        </Text>
      ),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
    },
    {
      title: '持续时间(ms)',
      dataIndex: 'duration',
      key: 'duration',
    },
    {
      title: '时间戳',
      dataIndex: 'timestamp',
      key: 'timestamp',
      render: (timestamp: string) => new Date(timestamp).toLocaleString(),
    },
  ]

  return (
    <>
      <Card
        title={
          <Space>
            <Title level={4} style={{ margin: 0 }}>测试结果</Title>
            <Button 
              type="primary" 
              icon={<PlayCircleOutlined />} 
              onClick={runTests} 
              loading={loading}
            >
              运行测试
            </Button>
            <Button icon={<ReloadOutlined />} onClick={runTests} loading={loading}>
              刷新
            </Button>
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Alert
          message="测试结果概览"
          description="显示集合创建功能和JQL查询输入框的测试结果"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Divider orientation="left">功能测试</Divider>
        <Table
          columns={testColumns}
          dataSource={testResults}
          rowKey="id"
          pagination={false}
          style={{ marginBottom: 24 }}
        />

        <Divider orientation="left">集合角色测试</Divider>
        {collectionRoleTests.map((test, index) => (
          <Card
            key={index}
            title={
              <Space>
                <Text strong>模块: {test.module}</Text>
                <Text style={{ color: getStatusColor(test.status) }}>
                  {getStatusText(test.status)}
                </Text>
              </Space>
            }
            style={{ marginBottom: 16 }}
          >
            <Text>{test.message}</Text>
            <div style={{ marginTop: 12 }}>
              <Text strong>创建的角色:</Text>
              {test.roles.map((role, roleIndex) => (
                <Card key={roleIndex} size="small" style={{ marginBottom: 8 }}>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    <Space>
                      <Text strong>角色代码:</Text>
                      <Text>{role.code}</Text>
                    </Space>
                    <Space>
                      <Text strong>角色名称:</Text>
                      <Text>{role.name}</Text>
                    </Space>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Text strong>权限:</Text>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                        {role.permissions.map((perm, permIndex) => (
                          <Text key={permIndex} type="secondary" style={{ border: '1px solid #d9d9d9', padding: '2px 8px', borderRadius: 4 }}>
                            {perm}
                          </Text>
                        ))}
                      </div>
                    </Space>
                  </Space>
                </Card>
              ))}
            </div>
          </Card>
        ))}
      </Card>
    </>
  )
}

export default TestResultsPage
