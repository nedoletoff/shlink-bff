import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';

import { AuthProvider }  from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { AppLayout }     from './components/layout/AppLayout';
import { Dashboard }     from './pages/Dashboard';
import { ShortUrls }     from './pages/ShortUrls';
import { UrlDetail }     from './pages/UrlDetail';
import { Tags }          from './pages/Tags';
import { AdminUsers }    from './pages/admin/Users';
import { UserDetail }    from './pages/admin/UserDetail';
import { AuditLogs }     from './pages/admin/AuditLogs';
import { Roles }         from './pages/admin/Roles';
import { Settings }      from './pages/admin/Settings';

export default function App() {
  return (
    <MantineProvider defaultColorScheme="dark">
      <Notifications position="top-right" />
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route
              path="/"
              element={
                <ProtectedRoute>
                  <AppLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard"          element={<Dashboard />} />
              <Route path="links"              element={<ShortUrls />} />
              <Route path="links/:shortCode"   element={<UrlDetail />} />
              <Route path="tags"               element={<Tags />} />

              {/* Admin-only маршруты */}
              <Route path="admin/users" element={
                <ProtectedRoute requiredRole="admin"><AdminUsers /></ProtectedRoute>
              } />
              <Route path="admin/users/:id" element={
                <ProtectedRoute requiredRole="admin"><UserDetail /></ProtectedRoute>
              } />
              <Route path="admin/logs" element={
                <ProtectedRoute requiredRole="admin"><AuditLogs /></ProtectedRoute>
              } />
              <Route path="admin/roles" element={
                <ProtectedRoute requiredRole="admin"><Roles /></ProtectedRoute>
              } />
              <Route path="admin/settings" element={
                <ProtectedRoute requiredRole="admin"><Settings /></ProtectedRoute>
              } />
              {/* 5.1 — детальная страница ссылки из админ-зоны */}
              <Route path="admin/urls/:shortCode" element={
                <ProtectedRoute requiredRole="admin"><UrlDetail /></ProtectedRoute>
              } />
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </MantineProvider>
  );
}
