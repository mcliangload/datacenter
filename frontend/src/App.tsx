import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { ConfigProvider, Spin } from 'antd'
import { lazy, Suspense, useEffect } from 'react'
import { useAuthStore } from './stores/authStore'
import themeConfig from './theme/themeConfig'

const LoginPage = lazy(() => import('./pages/Login/LoginPage'))
const AdminLayout = lazy(() => import('./pages/Admin/AdminLayout'))
const AdminSearchPage = lazy(() => import('./pages/Admin/SearchPage'))
const UserManagement = lazy(() => import('./pages/Admin/UserManagement'))
const RoleManagement = lazy(() => import('./pages/Admin/RoleManagement'))
const PermissionManagement = lazy(() => import('./pages/Admin/PermissionManagement'))
const ScraperCenter = lazy(() => import('./pages/Admin/ScraperCenter'))
const DeletedScraperPage = lazy(() => import('./pages/Admin/DeletedScraperPage'))
const CollectionQueryPage = lazy(() => import('./pages/Admin/CollectionQueryPage'))
const CustomFieldsPage = lazy(() => import('./pages/Admin/CustomFieldsPage'))
const TestResultsPage = lazy(() => import('./pages/Admin/TestResultsPage'))

const ProtectedRoute: React.FC = () => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const checkAuth = useAuthStore((state) => state.checkAuth)
  
  // 检查认证状态
  if (!isAuthenticated) {
    const isAuth = checkAuth()
    if (!isAuth) {
      return <Navigate to="/login" replace />
    }
  }
  return <Outlet />
}

function App() {
  const checkAuth = useAuthStore((state) => state.checkAuth)
  
  useEffect(() => {
    // 应用启动时检查认证状态
    checkAuth()
  }, [checkAuth])
  
  return (
    <ConfigProvider theme={themeConfig}>
      <BrowserRouter>
        <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}><Spin size="large" /></div>}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<ProtectedRoute />}>
              <Route path="/admin" element={<AdminLayout />}>
                <Route index element={<Navigate to="/admin/scraper" replace />} />
                <Route path="scraper" element={<ScraperCenter />} />
                <Route path="deleted-scraper" element={<DeletedScraperPage />} />
                <Route path="search" element={<AdminSearchPage />} />
                <Route path="collection-query" element={<CollectionQueryPage />} />
                <Route path="custom-fields" element={<CustomFieldsPage />} />
                <Route path="users" element={<UserManagement />} />
                <Route path="roles" element={<RoleManagement />} />
                <Route path="permissions" element={<PermissionManagement />} />
                <Route path="test-results" element={<TestResultsPage />} />
              </Route>
            </Route>
            <Route path="/" element={<Navigate to="/login" replace />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ConfigProvider>
  )
}

export default App