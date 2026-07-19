// 首次启动向导页面。
//
// 4 步骤表单（Steps 组件）：
//   Step 1: 企业信息
//   Step 2: 业务场景选择（B2B/B2C/综合）
//   Step 3: AI Key 配置（至少一个）
//   Step 4: 修改默认密码 + 完成
//
// 每步保存成功后自动跳下一步。完成后跳 Dashboard。

import { useEffect, useState } from "react";
import {
  Card,
  Steps,
  Form,
  Input,
  Button,
  App,
  Typography,
  Space,
  Alert,
  Tag,
} from "antd";
import {
  ShopOutlined,
  AppstoreOutlined,
  KeyOutlined,
  LockOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
} from "@ant-design/icons";
import { setupApi, type SetupStatus, type CompanyInput, type AIKeyInput } from "../../api/setup";
import { authApi } from "../../api/auth";
import { useAuthStore } from "../../store/auth";

const { Title, Text, Paragraph } = Typography;

const SCENARIO_OPTIONS = [
  {
    value: "b2b",
    label: "传统外贸（B2B）",
    desc: "海外采购商 / 进口商 / 贸易公司。客户管理 + 询盘 + 报价单 + 邮件分析。",
  },
  {
    value: "b2c",
    label: "跨境电商（B2C）",
    desc: "Amazon / Shopify / TikTok Shop / Temu。上架 + 订单 + 库存 + 广告。",
  },
  {
    value: "both",
    label: "工贸一体 / 两者都有",
    desc: "同时启用 B2B + B2C 全部模块（推荐，覆盖最全）。",
  },
];

export default function SetupPage() {
  const [current, setCurrent] = useState(0);
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [companyForm] = Form.useForm();
  const [aiKeyForm] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const { message } = App.useApp();
  const logout = useAuthStore((s) => s.logout);

  // 加载首启状态
  useEffect(() => {
    setupApi.status().then((s) => {
      setStatus(s);
      // 跳到当前应做的步骤
      const stepMap: Record<string, number> = {
        company: 0,
        scenario: 1,
        ai_key: 2,
        password: 3,
        done: 3,
      };
      setCurrent(stepMap[s.current_step] ?? 0);
    }).catch(() => {
      message.error("读取向导状态失败");
    });
  }, []);

  // Step 1: 企业信息
  const onSubmitCompany = async (values: CompanyInput) => {
    setLoading(true);
    try {
      await setupApi.saveCompany(values);
      message.success("企业信息已保存");
      refreshStatus();
      setCurrent(1);
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // Step 2: 场景选择
  const onSubmitScenario = async (scenario: "b2b" | "b2c" | "both") => {
    setLoading(true);
    try {
      await setupApi.selectScenario({ scenario });
      message.success("业务场景已选择");
      refreshStatus();
      setCurrent(2);
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // Step 3: AI Key
  const onSubmitAIKeys = async (values: AIKeyInput) => {
    setLoading(true);
    try {
      await setupApi.saveAIKeys(values);
      message.success("AI Key 已配置");
      refreshStatus();
      setCurrent(3);
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  // Step 4: 改密码 + 完成
  const onSubmitPassword = async (values: { old_password: string; new_password: string }) => {
    setLoading(true);
    try {
      await setupApi.changePassword(values);
      message.success("密码已修改");
      await setupApi.complete();
      message.success("初始化完成，即将重新登录");
      // 用新密码重新登录
      try {
        const user = useAuthStore.getState().user;
        if (user) {
          const data = await authApi.login({
            username: user.username,
            password: values.new_password,
          });
          useAuthStore.getState().setTokens(data);
        }
      } catch {
        // 重新登录失败，强制登出
        logout();
      }
      window.location.hash = "#/dashboard";
    } catch (err) {
      message.error((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const refreshStatus = () => {
    setupApi.status().then(setStatus).catch(() => {});
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        padding: 24,
        background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <Card style={{ width: 720, boxShadow: "0 10px 40px rgba(0,0,0,0.15)" }}>
        <div style={{ textAlign: "center", marginBottom: 24 }}>
          <Title level={3} style={{ marginBottom: 4 }}>
           欢迎使用 TradeMind AI
          </Title>
          <Text type="secondary">首次启动向导 · 4 步完成初始化（约 3 分钟）</Text>
        </div>

        <Steps
          current={current}
          size="small"
          style={{ marginBottom: 32 }}
          items={[
            { title: "企业信息", icon: <ShopOutlined /> },
            { title: "业务场景", icon: <AppstoreOutlined /> },
            { title: "AI 配置", icon: <KeyOutlined /> },
            { title: "设置密码", icon: <LockOutlined /> },
          ]}
        />

        {/* Step 1: 企业信息 */}
        {current === 0 && (
          <Form
            form={companyForm}
            layout="vertical"
            onFinish={onSubmitCompany}
            initialValues={{ country: "中国" }}
          >
            <Form.Item
              name="name"
              label="企业名称"
              rules={[{ required: true, message: "请输入企业名称" }]}
            >
              <Input placeholder="如：深圳市 XXX 贸易有限公司" size="large" />
            </Form.Item>
            <Form.Item name="industry" label="所属行业">
              <Input placeholder="如：跨境电商 / 传统外贸 / 工贸一体" />
            </Form.Item>
            <Form.Item name="country" label="所在国家">
              <Input placeholder="中国" />
            </Form.Item>
            <Form.Item name="contact" label="联系方式">
              <Input placeholder="选填，如电话/邮箱" />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size="large"
              loading={loading}
              icon={<ArrowRightOutlined />}
            >
              下一步：选择业务场景
            </Button>
          </Form>
        )}

        {/* Step 2: 场景选择 */}
        {current === 1 && (
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <Paragraph type="secondary">
              选择您的主要业务场景，系统将启用对应模块。可后续在设置中调整。
            </Paragraph>
            {SCENARIO_OPTIONS.map((opt) => (
              <Card
                key={opt.value}
                hoverable
                size="small"
                onClick={() => !loading && onSubmitScenario(opt.value as "b2b" | "b2c" | "both")}
                style={{
                  cursor: "pointer",
                  borderColor: status?.scenario === opt.value ? "#1677ff" : undefined,
                }}
              >
                <Space>
                  {status?.scenario === opt.value && (
                    <CheckCircleOutlined style={{ color: "#1677ff" }} />
                  )}
                  <div>
                    <Text strong>{opt.label}</Text>
                    <br />
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {opt.desc}
                    </Text>
                  </div>
                </Space>
              </Card>
            ))}
            {loading && <Text type="secondary">保存中...</Text>}
          </Space>
        )}

        {/* Step 3: AI Key */}
        {current === 2 && (
          <Form form={aiKeyForm} layout="vertical" onFinish={onSubmitAIKeys}>
            <Alert
              type="info"
              showIcon
              message="至少配置一个 AI Key"
              description="数据从本机直连 AI 厂商，我们不经过任何第三方。推荐先用 DeepSeek（注册即送 500 万 token）。"
              style={{ marginBottom: 16 }}
            />
            <Form.Item
              name="deepseek_key"
              label={
                <Space>
                  DeepSeek API Key <Tag color="green">推荐</Tag>
                </Space>
              }
            >
              <Input.Password placeholder="sk-xxxxxxxxxxxx" />
            </Form.Item>
            <Form.Item name="openai_key" label="OpenAI API Key">
              <Input.Password placeholder="sk-proj-xxxxxxxx" />
            </Form.Item>
            <Form.Item name="qwen_key" label="通义千问（Qwen）API Key">
              <Input.Password placeholder="sk-xxxxxxxx" />
            </Form.Item>
            <Form.Item name="anthropic_key" label="Anthropic（Claude）API Key">
              <Input.Password placeholder="sk-ant-xxxxxxxx" />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size="large"
              loading={loading}
              icon={<ArrowRightOutlined />}
            >
              下一步：设置新密码
            </Button>
          </Form>
        )}

        {/* Step 4: 改密码 */}
        {current === 3 && (
          <Form form={pwdForm} layout="vertical" onFinish={onSubmitPassword}>
            <Alert
              type="warning"
              showIcon
              message="请修改默认密码"
              description="默认密码 admin123 不安全，请设置新密码（至少 6 位）。"
              style={{ marginBottom: 16 }}
            />
            <Form.Item
              name="old_password"
              label="原密码"
              rules={[{ required: true, message: "请输入原密码" }]}
            >
              <Input.Password placeholder="admin123" />
            </Form.Item>
            <Form.Item
              name="new_password"
              label="新密码"
              rules={[
                { required: true, message: "请输入新密码" },
                { min: 6, message: "至少 6 位" },
                { max: 64, message: "最多 64 位" },
              ]}
            >
              <Input.Password placeholder="至少 6 位" />
            </Form.Item>
            <Form.Item
              name="confirm"
              label="确认新密码"
              dependencies={["new_password"]}
              rules={[
                { required: true, message: "请确认新密码" },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue("new_password") === value) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error("两次输入不一致"));
                  },
                }),
              ]}
            >
              <Input.Password placeholder="再次输入新密码" />
            </Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              size="large"
              loading={loading}
            >
              完成初始化
            </Button>
          </Form>
        )}
      </Card>
    </div>
  );
}
