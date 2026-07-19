// 登录页。
// 默认账号 admin / admin123（首启注入），登录成功跳 dashboard。

import { useState } from "react";
import { Card, Form, Input, Button, Typography, App } from "antd";
import { LockOutlined, UserOutlined } from "@ant-design/icons";
import { authApi } from "../../api/auth";
import { useAuthStore } from "../../store/auth";
import type { LoginInput } from "../../types";

const { Title, Text } = Typography;

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const setTokens = useAuthStore((s) => s.setTokens);
  const { message } = App.useApp();

  const onFinish = async (values: LoginInput) => {
    setLoading(true);
    try {
      const data = await authApi.login(values);
      setTokens(data);
      message.success(`欢迎回来，${data.user.nickname || data.user.username}`);
      window.location.hash = "#/dashboard";
    } catch (err) {
      message.error((err as Error).message || "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
      }}
    >
      <Card style={{ width: 400, boxShadow: "0 10px 40px rgba(0,0,0,0.15)" }}>
        <div style={{ textAlign: "center", marginBottom: 32 }}>
          <Title level={2} style={{ marginBottom: 4 }}>
            TradeMind AI
          </Title>
          <Text type="secondary">企业级 AI 外贸智能操作系统</Text>
        </div>

        <Form
          name="login"
          size="large"
          onFinish={onFinish}
          autoComplete="off"
          initialValues={{ username: "admin", password: "admin123" }}
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: "请输入用户名" }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: "请输入密码" }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" block loading={loading}>
              登录
            </Button>
          </Form.Item>
        </Form>

        <div style={{ marginTop: 16, textAlign: "center" }}>
          <Text type="secondary" style={{ fontSize: 12 }}>
            默认管理员：admin / admin123（首次登录后请修改密码）
          </Text>
        </div>
      </Card>
    </div>
  );
}
