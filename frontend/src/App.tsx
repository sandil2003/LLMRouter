import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/layout/Layout';
import { DashboardPage } from './pages/DashboardPage';
import { PlaygroundPage } from './pages/PlaygroundPage';
import { ProvidersPage } from './pages/ProvidersPage';
import { RoutingPage } from './pages/RoutingPage';
import { UsagePage } from './pages/UsagePage';
import { LogsPage } from './pages/LogsPage';
import { SettingsPage } from './pages/SettingsPage';
import { ToastProvider } from './components/common/Toast';
import './styles/index.css';

export default function App() {
  return (
    <ToastProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<DashboardPage />} />
            <Route path="playground" element={<PlaygroundPage />} />
            <Route path="providers" element={<ProvidersPage />} />
            <Route path="routing" element={<RoutingPage />} />
            <Route path="usage" element={<UsagePage />} />
            <Route path="logs" element={<LogsPage />} />
            <Route path="settings" element={<SettingsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ToastProvider>
  );
}
