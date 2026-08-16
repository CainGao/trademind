// 工作台（Dashboard）— 登录后首页：真实数据指标卡 + 今日日报 + 快捷入口。
//
// 2026-08-16 重写：替换 Week 1 的静态占位（指标全 "—" + 过时的产品路线列表）。
// 数据源：
//   - 商品池 / 供应商 / AI 执行次数 / 待跟进询盘 → 4 个并行 API 的 total
//   - 今日日报 → dailyReportApi.today()，无则提示可手动生成
// 单项 API 失败不阻塞整页（Promise.allSettled），对应指标显示 0。

import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button, Card, Col, Row, Statistic, Typography } from "antd";
import {
  ShoppingOutlined,
  ShopOutlined,
  RobotOutlined,
  BellOutlined,
  FileTextOutlined,
  ArrowRightOutlined,
  TeamOutlined,
  DashboardOutlined,
  BookOutlined,
} from "@ant-design/icons";
import { productApi } from "../../api/product";
import { supplierApi } from "../../api/supplier";
import { agentApi } from "../../api/agent";
import { inquiryApi } from "../../api/b2b";
import { dailyReportApi, type DailyReport } from "../../api/dailyReport";

const { Title, Paragraph, Text } = Typography;

interface Overview {
  productTotal: number;
  supplierTotal: number;
  agentRuns: number;
  pendingInquiries: number;
}

const QUICK_ENTRIES = [
  { path: "/products", icon: <ShoppingOutlined />, label: "商品中心", desc: "选品池 · AI 分析" },
  { path: "/customers", icon: <TeamOutlined />, label: "客户管理", desc: "B2B 客户跟进" },
  { path: "/agents", icon: <RobotOutlined />, label: "Agent 任务", desc: "选品 · 采购 · 定时" },
  { path: "/daily-reports", icon: <FileTextOutlined />, label: "老板日报", desc: "每日经营总结" },
  { path: "/cockpit", icon: <DashboardOutlined />, label: "老板驾驶舱", desc: "团队行为资产" },
  { path: "/knowledge", icon: <BookOutlined />, label: "RAG 知识库", desc: "企业知识沉淀" },
];

export default function DashboardPage() {
  const navigate = useNavigate();
  const [overview, setOverview] = useState<Overview>({
    productTotal: 0,
    supplierTotal: 0,
    agentRuns: 0,
    pendingInquiries: 0,
  });
  const [loading, setLoading] = useState(true);
  const [report, setReport] = useState<DailyReport | null>(null);
  const [reportLoading, setReportLoading] = useState(true);

  useEffect(() => {
    const tasks = [
      productApi.list({ page: 1, page_size: 1 }),
      supplierApi.overview(),
      agentApi.listRuns({ page: 1, page_size: 1 }),
      inquiryApi.list({ page: 1, page_size: 1, status: "new" }),
    ];
    Promise.allSettled(tasks).then((results) => {
      const next: Overview = { productTotal: 0, supplierTotal: 0, agentRuns: 0, pendingInquiries: 0 };
      if (results[0].status === "fulfilled") next.productTotal = results[0].value.total ?? 0;
      if (results[1].status === "fulfilled") next.supplierTotal = results[1].value.total ?? 0;
      if (results[2].status === "fulfilled") next.agentRuns = results[2].value.total ?? 0;
      if (results[3].status === "fulfilled") next.pendingInquiries = results[3].value.total ?? 0;
      setOverview(next);
      setLoading(false);
    });

    dailyReportApi
      .today()
      .then((r) => setReport(r ?? null))
      .catch(() => setReport(null))
      .finally(() => setReportLoading(false));
  }, []);

  const narrative =
    report?.ai_narrative && report.ai_narrative.length > 200
      ? report.ai_narrative.slice(0, 200) + "…"
      : report?.ai_narrative;

  return (
    <div>
      <Title level={4}>工作台</Title>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="商品池"
              loading={loading}
              value={overview.productTotal}
              prefix={<ShoppingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="供应商"
              loading={loading}
              value={overview.supplierTotal}
              prefix={<ShopOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="AI 执行次数"
              loading={loading}
              value={overview.agentRuns}
              prefix={<RobotOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待跟进询盘"
              loading={loading}
              value={overview.pendingInquiries}
              prefix={<BellOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        style={{ marginTop: 24 }}
        title="今日日报"
        extra={
          <Button type="link" onClick={() => navigate("/daily-reports")}>
            查看全部 <ArrowRightOutlined />
          </Button>
        }
        loading={reportLoading}
      >
        {report ? (
          <div>
            <Paragraph style={{ whiteSpace: "pre-wrap", marginBottom: 8 }}>{narrative}</Paragraph>
            <Text type="secondary">报告日期：{report.report_date}</Text>
          </div>
        ) : (
          <Text type="secondary">
            今日日报尚未生成。可前往「老板日报」页手动生成，或等待每天 18:00 自动生成。
          </Text>
        )}
      </Card>

      <Card style={{ marginTop: 24 }} title="快捷入口">
        <Row gutter={[16, 16]}>
          {QUICK_ENTRIES.map((entry) => (
            <Col span={8} key={entry.path}>
              <Card
                hoverable
                size="small"
                onClick={() => navigate(entry.path)}
                styles={{ body: { display: "flex", alignItems: "center", gap: 12 } }}
              >
                <span style={{ fontSize: 24, color: "#1677ff" }}>{entry.icon}</span>
                <div>
                  <div style={{ fontWeight: 600 }}>{entry.label}</div>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {entry.desc}
                  </Text>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>
    </div>
  );
}
