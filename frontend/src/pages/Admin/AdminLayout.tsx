import { Button, Layout, Menu, Space, Breadcrumb, Dropdown, Avatar } from 'antd'
import { useState, useEffect } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../../stores/authStore'
import { SettingOutlined, UserOutlined, LogoutOutlined, UserAddOutlined, MenuFoldOutlined, MenuUnfoldOutlined, DatabaseOutlined } from '@ant-design/icons'

const { Header, Content, Sider } = Layout

const AdminLayout: React.FC = () => {
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore((state) => ({ user: state.user, logout: state.logout }))
  const [collapsed, setCollapsed] = useState(false)
  const [breadcrumbItems, setBreadcrumbItems] = useState<{ title: string }[]>([])

  useEffect(() => {
    const path = location.pathname
    const segments = path.split('/').filter(Boolean)

    // 处理刮削中心的特殊情况，不显示子标题
    if (segments.includes('scraper')) {
      setBreadcrumbItems([
        { title: '管理后台' },
        { title: '刮削中心' }
      ])
      return
    }

    const items: { title: string }[] = segments.map((segment) => {
      let title = segment

      const titleMap: Record<string, string> = {
        'admin': '管理后台',
        'search': '数据搜索',
        'collection-query': '集合查询',
        'custom-fields': '自定义字段',
        'user-center': '用户中心',
        'users': '用户管理',
        'roles': '角色管理',
        'permissions': '权限管理',
        'scraper': '刮削中心',
        'data-management': '数据管理'
      }

      if (titleMap[segment]) {
        title = titleMap[segment]
      }

      return { title }
    })

    setBreadcrumbItems(items)
  }, [location.pathname])

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const handleGoToSettings = () => {
    navigate('/settings')
  }

  const userMenuItems = [
    {
      key: 'profile',
      label: '个人设置',
      icon: <SettingOutlined />,
      onClick: handleGoToSettings
    },
    {
      key: 'logout',
      label: '退出登录',
      icon: <LogoutOutlined />,
      onClick: handleLogout,
      danger: true
    }
  ]

  const menuItems = [
    {
          key: 'scraper',
          label: '刮削中心',
          icon: <DatabaseOutlined />,
          children: [
            {
              key: 'data-query',
              label: '数据查询',
              onClick: () => navigate('/admin/scraper'),
            },
            {
              key: 'deleted-data-query',
              label: '删除数据查询',
              onClick: () => navigate('/admin/deleted-scraper'),
            },
          ],
        },
    {
      key: 'data-management',
      label: '数据管理',
      icon: <DatabaseOutlined />,
      children: [
        {
          key: 'search',
          label: '数据搜索',
          onClick: () => navigate('/admin/search'),
        },
        {
          key: 'collection-query',
          label: '集合查询',
          onClick: () => navigate('/admin/collection-query'),
        },
        {
          key: 'custom-fields',
          label: '自定义字段',
          onClick: () => navigate('/admin/custom-fields'),
        },
      ],
    },
    {
      key: 'user-center',
      label: '用户中心',
      icon: <UserAddOutlined />,
      children: [
        {
          key: 'users',
          label: '用户管理',
          onClick: () => navigate('/admin/users'),
        },
        {
          key: 'roles',
          label: '角色管理',
          onClick: () => navigate('/admin/roles'),
        },
        {
          key: 'permissions',
          label: '权限管理',
          onClick: () => navigate('/admin/permissions'),
        },
      ],
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: '#001529', padding: '0 24px' }}>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
            style={{ color: 'white', marginRight: 16 }}
          />
          <div style={{ color: 'white', fontSize: '18px', fontWeight: 'bold', marginRight: 24 }}>
            数据中心管理
          </div>
        </div>
        <Space>
          <Dropdown menu={{ items: userMenuItems }}>
            <Button type="text" style={{ color: 'white' }}>
              <Space>
                <Avatar size="small" icon={<UserOutlined />} />
                <span>{user?.username}</span>
              </Space>
            </Button>
          </Dropdown>
        </Space>
      </Header>
      <Layout>
        <Sider
          width={200}
          collapsible
          collapsed={collapsed}
          onCollapse={(value) => setCollapsed(value)}
          theme="dark"
          breakpoint="lg"
          collapsedWidth="80"
          style={{ position: 'relative' }}
        >
          <Menu
            mode="inline"
            defaultSelectedKeys={['scraper']}
            style={{ height: '100%', borderRight: 0 }}
            items={menuItems}
            theme="dark"
          />
        </Sider>
        <Content style={{ padding: '24px', minHeight: 280 }}>
          <Breadcrumb items={breadcrumbItems} style={{ marginBottom: 16 }} />
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

export default AdminLayout