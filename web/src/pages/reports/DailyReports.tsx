// 老板日报 — AI 生成 + 飞书推送 + 历史归档。
//
// 核心功能：
//   1. 「生成今日日报」按钮 → AI 聚合数据 → 叙述 + Kill/Scale 建议
//   2. 报告卡片：日期 / AI 叙述 / 机会 / Kill/Scale Tag / 推送状态
//   3. 「推送到飞书」按钮（需先在设置里配 webhook）
//   4. 历史日报列表
import { useEffect, useState } from "react";
import {
  Card,
  Button,
  List,
  Tag,
  Typography,
  message,
  Modal,
  Form,
  Input,
  Space,
  Spin,
  Empty,
  Row,
  Col,
  Divider,
  Tooltip,
  Alert,
} from "antd";
import {
  FileTextOutlined,
  ThunderboltOutlined,
  SendOutlined,
  SettingOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  RiseOutlined,
  FallOutlined,
} from "@ant-design/icons";
import { dailyReportApi, type DailyReport, type FeishuConfig } from "../../api/dailyReport";
import dayjs from "dayjs";

const { Text, Paragraph, Title } = Typography;

export default function DailyReports() {
  const [reports, setReports] = useState<DailyReport[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [delivering, setDelivering] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [feishuConfigured, setFeishuConfigured] = useState(false);
  const [configOpen, setConfigOpen] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [listResp, cfg] = await Promise.all([
        dailyReportApi.list({ page, page_size: 10 }),
        dailyReportApi.getFeishuConfig(),
      ]);
      setReports(listResp.list || []);
      setTotal(listResp.total || 0);
      setFeishuConfigured(!!cfg.webhook_url);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [page]);

  const onGenerate = async () => {
    setGenerating(true);
    try {
      await dailyReportApi.generate(false);
      message.success("日报已生成");
      load();
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setGenerating(false);
    }
  };

  const onDeliver = async (id: number) => {
    setDelivering(id);
    try {
      await dailyReportApi.deliverToFeishu(id);
      message.success("已推送到飞书");
      load();
    } catch (e: any) {
      message.error("推送失败: " + e.message);
    } finally {
      setDelivering(null);
    }
  };

  const onSaveConfig = async () => {
    const v: FeishuConfig = await form.validateFields();
    try {
      await dailyReportApi.updateFeishuConfig(v);
      message.success("配置已保存");
      setConfigOpen(false);
      setFeishuConfigured(!!v.webhook_url);
    } catch (e: any) {
      message.error(e.message);
    }
  };

  return (
    <div>
      <Title level={4}>
        <FileTextOutlined /> 老板日报
      </Title>

      <Alert
        type="info"
        showIcon
        message="每天 18:00 自动生成日报"
        description={
          feishuConfigured ? (
            <span>
              ✅ 飞书推送已启用，日报会自动推送到飞书群。
              <Button
                type="link"
                size="small"
                icon={<SettingOutlined />}
                onClick={() => {
                  form.resetFields();
                  setConfigOpen(true);
                }}
              >
                修改配置
              </Button>
            </span>
          ) : (
            <span>
              ⚠️ 未配置飞书 webhook，日报只存本地。
              <Button
                type="link"
                size="small"
                icon={<SettingOutlined />}
                onClick={() => {
                  form.resetFields();
                  setConfigOpen(true);
                }}
              >
                配置飞书推送
              </Button>
            </span>
          )
        }
        style={{ marginBottom: 16 }}
        action={
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            loading={generating}
            onClick={onGenerate}
          >
            立即生成今日日报
          </Button>
        }
      />

      {loading && reports.length === 0 ? (
        <Card>
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spin />
          </div>
        </Card>
      ) : reports.length === 0 ? (
        <Card>
          <Empty description="还没有日报，点击上方按钮生成" />
        </Card>
      ) : (
        <List
          loading={loading}
          dataSource={reports}
          pagination={{
            current: page,
            total,
            pageSize: 10,
            onChange: setPage,
          }}
          renderItem={(report) => (
            <ReportCard
              report={report}
              onDeliver={onDeliver}
              delivering={delivering === report.id}
              feishuConfigured={feishuConfigured}
            />
          )}
        />
      )}

      {/* 飞书 webhook 配置 Modal */}
      <Modal
        title="配置飞书 webhook 推送"
        open={configOpen}
        onOk={onSaveConfig}
        onCancel={() => setConfigOpen(false)}
      >
        <Paragraph type="secondary">
          在飞书群 → 设置 → 群机器人 → 添加自定义机器人，复制 webhook URL
          填入下面。
        </Paragraph>
        <Form form={form} layout="vertical">
          <Form.Item
            name="webhook_url"
            label="Webhook URL"
            rules={[{ required: true, message: "请输入 webhook URL" }]}
          >
            <Input placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" />
          </Form.Item>
          <Form.Item
            name="secret"
            label="签名校验 Secret（可选）"
            extra="如启用签名校验，填这里。留空表示不校验。"
          >
            <Input.Password placeholder="sec-xxx" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// 报告卡片
function ReportCard({
  report,
  onDeliver,
  delivering,
  feishuConfigured,
}: {
  report: DailyReport;
  onDeliver: (id: number) => void;
  delivering: boolean;
  feishuConfigured: boolean;
}) {
  // 解析 opportunities / kill_scale JSON
  let opportunities: string[] = [];
  let killScale: Array<{ action: string; target: string; reason: string }> = [];
  try {
    opportunities = JSON.parse(report.opportunities || "[]");
    killScale = JSON.parse(report.kill_scale || "[]");
  } catch {}

  return (
    <Card
      style={{ marginBottom: 12 }}
      title={
        <Space>
          <Text strong>{dayjs(report.report_date).format("YYYY-MM-DD")}</Text>
          {report.delivered_to_feishu ? (
            <Tooltip title="已推送到飞书">
              <Tag color="success" icon={<CheckCircleOutlined />}>
                已推送飞书
              </Tag>
            </Tooltip>
          ) : (
            <Tag icon={<CloseCircleOutlined />}>未推送</Tag>
          )}
        </Space>
      }
      extra={
        <Button
          type="primary"
          size="small"
          icon={<SendOutlined />}
          loading={delivering}
          disabled={!feishuConfigured}
          onClick={() => onDeliver(report.id)}
        >
          {report.delivered_to_feishu ? "重新推送" : "推送到飞书"}
        </Button>
      }
    >
      {/* AI 叙述 */}
      <Paragraph style={{ fontSize: 15, lineHeight: 1.8 }}>
        {report.ai_narrative || "（无 AI 叙述）"}
      </Paragraph>

      {(opportunities.length > 0 || killScale.length > 0) && (
        <Row gutter={16}>
          {opportunities.length > 0 && (
            <Col span={12}>
              <Card size="small" type="inner" title="💡 发现机会">
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  {opportunities.map((o, i) => (
                    <li key={i}>
                      <Text>{o}</Text>
                    </li>
                  ))}
                </ul>
              </Card>
            </Col>
          )}
          {killScale.length > 0 && (
            <Col span={12}>
              <Card size="small" type="inner" title="🎯 Kill/Scale 建议">
                {killScale.map((ks, i) => {
                  const isScale = ks.action === "Scale";
                  return (
                    <div key={i} style={{ marginBottom: 8 }}>
                      <Tag
                        color={isScale ? "success" : "error"}
                        icon={isScale ? <RiseOutlined /> : <FallOutlined />}
                      >
                        {ks.action}
                      </Tag>
                      <Text strong>{ks.target}</Text>
                      <div style={{ marginLeft: 8, marginTop: 4 }}>
                        <Text type="secondary">{ks.reason}</Text>
                      </div>
                    </div>
                  );
                })}
              </Card>
            </Col>
          )}
        </Row>
      )}

      <Divider style={{ margin: "12px 0" }} />
      <Text type="secondary" style={{ fontSize: 12 }}>
        生成于 {dayjs(report.created_at).format("YYYY-MM-DD HH:mm")}
      </Text>
    </Card>
  );
}
