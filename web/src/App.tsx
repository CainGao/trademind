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
import Cockpit from "./pages/cockpit/Cockpit";
import Placeholder from "./components/Placeholder";
import { RobotOutlined } from "@ant-design/icons";

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
              <Route path="customers" element={<Placeholder title="客户管理（B2B）" />} />
              <Route path="inquiries" element={<Placeholder title="询盘管理（B2B）" />} />
              <Route path="quotations" element={<Placeholder title="报价单（B2B）" />} />
              <Route path="stores" element={<Placeholder title="店铺管理（B2C）" />} />
              <Route path="listings" element={<Placeholder title="上架商品（B2C）" />} />
              <Route path="orders" element={<Placeholder title="订单管理（B2C）" />} />
              <Route path="agents" element={<Placeholder title="AI Agent" icon={<RobotOutlined />} />} />
              <Route path="cockpit" element={<Cockpit />} />
              <Route path="settings" element={<Placeholder title="系统设置" />} />
              <Route path="*" element={<Navigate to="/dashboard" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </AntApp>
    </ConfigProvider>
  );
}
