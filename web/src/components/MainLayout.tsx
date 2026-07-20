// 主框架：侧边栏 + 顶栏 + 内容区（Outlet 渲染子路由）。
//
// 侧边栏菜单按场景（B2B/B2C）+ 用户角色动态显示。
// 架构文档 §1.2: 五角色适配两场景。

import { useState } from "react";
import { Layout, Menu, Avatar, Dropdown, Typography, Space } from "antd";
import type { MenuProps } from "antd";
import {
  DashboardOutlined,
  ShoppingOutlined,
  ShopOutlined,
  UserOutlined,
  TeamOutlined,
  RobotOutlined,
  SettingOutlined,
  FundOutlined,
  LogoutOutlined,
  DownOutlined,
  FileTextOutlined,
  BookOutlined,
} from "@ant-design/icons";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "../store/auth";
import type { UserRole } from "../types";

const { Header, Sider, Content } = Layout;
const { Text } = Typography;

// 菜单项定义。roles 字段控制可见性，未配置则全角色可见。
interface AppMenuItem {
  key: string;
  icon?: React.ReactNode;
  label: string;
  roles?: UserRole[];
  children?: AppMenuItem[];
}

const allMenuItems: AppMenuItem[] = [
  { key: "dashboard", icon: <DashboardOutlined />, label: "工作台" },
  { key: "products", icon: <ShoppingOutlined />, label: "商品中心" },
  { key: "suppliers", icon: <ShopOutlined />, label: "供应商" },
  {
    key: "b2b-group",
    icon: <TeamOutlined />,
    label: "外贸 B2B",
    roles: ["admin", "boss", "sales"],
    children: [
      { key: "customers", label: "客户管理" },
      { key: "inquiries", label: "询盘管理" },
      { key: "quotations", label: "报价单" },
    ],
  },
  {
    key: "b2c-group",
    icon: <FundOutlined />,
    label: "跨境 B2C",
    roles: ["admin", "boss", "operator"],
    children: [
      { key: "stores", label: "店铺管理" },
      { key: "listings", label: "上架商品" },
      { key: "orders", label: "订单管理" },
    ],
  },
  { key: "agents", icon: <RobotOutlined />, label: "Agent 任务中心", roles: ["admin", "boss"] },
  { key: "agents-special", icon: <RobotOutlined />, label: "专用 Agent", roles: ["admin", "boss", "sales", "operator"] },
  { key: "daily-reports", icon: <FileTextOutlined />, label: "老板日报", roles: ["admin", "boss"] },
  { key: "knowledge", icon: <BookOutlined />, label: "RAG 知识库", roles: ["admin", "boss"] },
  { key: "cockpit", icon: <DashboardOutlined />, label: "老板驾驶舱", roles: ["admin", "boss"] },
  { key: "settings", icon: <SettingOutlined />, label: "系统设置", roles: ["admin"] },
];

// 按角色过滤，转换为 AntD Menu 的 props.items 格式
function buildMenuItems(items: AppMenuItem[], userRole?: UserRole): MenuProps["items"] {
  return items
    .filter((it) => !it.roles || (userRole && it.roles.includes(userRole)))
    .map((it) => {
      if (it.children) {
        return {
          key: it.key,
          icon: it.icon,
          label: it.label,
          children: buildMenuItems(it.children, userRole),
        };
      }
      return { key: it.key, icon: it.icon, label: it.label };
    });
}

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();

  // 按角色过滤菜单
  const menuItems = buildMenuItems(allMenuItems, user?.role);

  // 当前选中的菜单项（从 URL 提取）
  const selectedKey = location.pathname.split("/")[1] || "dashboard";

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(`/${key}`);
  };

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const userMenu = {
    items: [
      {
        key: "profile",
        icon: <UserOutlined />,
        label: `${user?.nickname || user?.username} (${user?.role})`,
        disabled: true,
      },
      { type: "divider" as const },
      {
        key: "logout",
        icon: <LogoutOutlined />,
        label: "退出登录",
        onClick: handleLogout,
      },
    ],
  };

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        theme="light"
        style={{ boxShadow: "2px 0 8px rgba(0,0,0,0.06)" }}
      >
        <div
          style={{
            height: 64,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            borderBottom: "1px solid #f0f0f0",
          }}
        >
          <Text strong style={{ fontSize: collapsed ? 14 : 18, color: "#1677ff" }}>
            {collapsed ? "TM" : "TradeMind AI"}
          </Text>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={["b2b-group", "b2c-group"]}
          items={menuItems}
          onClick={handleMenuClick}
          style={{ borderRight: 0 }}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            background: "#fff",
            padding: "0 24px",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            boxShadow: "0 1px 4px rgba(0,0,0,0.06)",
          }}
        >
          <Text type="secondary">企业级 AI 外贸智能操作系统</Text>
          <Dropdown menu={userMenu}>
            <Space style={{ cursor: "pointer" }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <Text>{user?.nickname || user?.username}</Text>
              <DownOutlined style={{ fontSize: 10 }} />
            </Space>
          </Dropdown>
        </Header>

        <Content style={{ margin: 24, padding: 24, background: "#fff", borderRadius: 8 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
