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
import { useAuthStore } from "./store/auth";
import LoginPage from "./pages/login/LoginPage";
import MainLayout from "./components/MainLayout";
import DashboardPage from "./pages/dashboard/DashboardPage";
import Placeholder from "./components/Placeholder";
import { RobotOutlined } from "@ant-design/icons";

// 需登录的路由守卫
function PrivateRoute({ children }: { children: React.ReactNode }) {
  const isAuthed = useAuthStore((s) => s.isAuthenticated());
  return isAuthed ? <>{children}</> : <Navigate to="/login" replace />;
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
            <Route
              path="/"
              element={
                <PrivateRoute>
                  <MainLayout />
                </PrivateRoute>
              }
            >
              <Route index element={<Navigate to="/dashboard" replace />} />
              <Route path="dashboard" element={<DashboardPage />} />
              <Route path="products" element={<Placeholder title="商品中心" />} />
              <Route path="suppliers" element={<Placeholder title="供应商" />} />
              <Route path="customers" element={<Placeholder title="客户管理（B2B）" />} />
              <Route path="inquiries" element={<Placeholder title="询盘管理（B2B）" />} />
              <Route path="quotations" element={<Placeholder title="报价单（B2B）" />} />
              <Route path="stores" element={<Placeholder title="店铺管理（B2C）" />} />
              <Route path="listings" element={<Placeholder title="上架商品（B2C）" />} />
              <Route path="orders" element={<Placeholder title="订单管理（B2C）" />} />
              <Route path="agents" element={<Placeholder title="AI Agent" icon={<RobotOutlined />} />} />
              <Route path="cockpit" element={<Placeholder title="老板驾驶舱" />} />
              <Route path="settings" element={<Placeholder title="系统设置" />} />
              <Route path="*" element={<Navigate to="/dashboard" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </AntApp>
    </ConfigProvider>
  );
}
