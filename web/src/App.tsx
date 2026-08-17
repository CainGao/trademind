// 应用根组件：路由 + 路由守卫。
//
// 路由结构：
//   /login         公开
//   /              MainLayout，需登录
//     /dashboard   工作台
//     /products    商品中心
//     /suppliers   供应商
//     /cockpit     老板驾驶舱
//     /settings    系统设置
//     ...          其他占位

import { HashRouter, Routes, Route, Navigate } from "react-router-dom";
import { ConfigProvider, App as AntApp } from "antd";
import zhCN from "antd/locale/zh_CN";
import { useEffect, useState } from "react";
import { useAuthStore } from "./store/auth";
import { setupApi, type SetupStatus } from "./api/setup";
import LoginPage from "./pages/login/LoginPage";
import SetupPage from "./pages/setup/SetupPage";
import MainLayout from "./components/MainLayout";
import DashboardPage from "./pages/dashboard/DashboardPage";
import ProductList from "./pages/products/ProductList";
import SupplierList from "./pages/suppliers/SupplierList";
import CustomerList from "./pages/customers/CustomerList";
import InquiryList from "./pages/inquiries/InquiryList";
import QuotationList from "./pages/quotations/QuotationList";
import Cockpit from "./pages/cockpit/Cockpit";
import AgentList from "./pages/agents/AgentList";
import SpecialAgents from "./pages/agents/SpecialAgents";
import DailyReports from "./pages/reports/DailyReports";
import StoreList from "./pages/stores/StoreList";
import OrderList from "./pages/orders/OrderList";
import Knowledge from "./pages/knowledge/Knowledge";
import SettingsPage from "./pages/system/SettingsPage";
import Placeholder from "./components/Placeholder";

// 需登录的路由守卫
function PrivateRoute({ children }: { children: React.ReactNode }) {
  const isAuthed = useAuthStore((s) => s.isAuthenticated());
  return isAuthed ? <>{children}</> : <Navigate to="/login" replace />;
}

// 首启守卫：未完成首启则强制跳 /setup
function SetupGuard({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setupApi
      .status()
      .then((s) => setStatus(s))
      .catch(() => {
        // 查询失败也放行（让正常路由处理）
        setStatus({ completed: true } as SetupStatus);
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return null;
  if (status && !status.completed) {
    return <Navigate to="/setup" replace />;
  }
  return <>{children}</>;
}

export default function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: "#1677ff",
          borderRadius: 8,
        },
      }}
    >
      <AntApp>
        <HashRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/setup" element={<SetupPage />} />
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <SetupGuard>
                    <MainLayout />
                  </SetupGuard>
                </PrivateRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<DashboardPage />} />
              <Route path="products" element={<ProductList />} />
              <Route path="suppliers" element={<SupplierList />} />
              <Route path="customers" element={<CustomerList />} />
              <Route path="inquiries" element={<InquiryList />} />
              <Route path="quotations" element={<QuotationList />} />
              <Route path="stores" element={<StoreList />} />
              <Route path="listings" element={<Placeholder title="上架商品（B2C）" />} />
              <Route path="orders" element={<OrderList />} />
              <Route path="agents" element={<AgentList />} />
              <Route path="agents-special" element={<SpecialAgents />} />
              <Route path="daily-reports" element={<DailyReports />} />
              <Route path="knowledge" element={<Knowledge />} />
              <Route path="cockpit" element={<Cockpit />} />
              <Route path="settings" element={<SettingsPage />} />
              <Route path="*" element={<Navigate to="/dashboard" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </AntApp>
    </ConfigProvider>
  );
}
