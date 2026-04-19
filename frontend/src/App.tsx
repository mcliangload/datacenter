import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import LoginPage from './pages/Login/LoginPage'
import SearchPage from './pages/Search/SearchPage'
import { useAuthStore } from './stores/authStore'

const ProtectedRoute: React.FC = () => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}

function App() {
  return (
    <ConfigProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<ProtectedRoute />}>
            <Route path="/search" element={<SearchPage />} />
          </Route>
          <Route path="/" element={<Navigate to="/login" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}

export default App