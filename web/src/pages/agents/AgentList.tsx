// Agent 任务中心 — 选品/采购 Agent + 运行历史 + 定时配置。
//
// 功能：
//   1. 手动触发选品/采购 Agent，展示结构化报告
//   2. 运行历史列表（成功/失败/耗时/Token）
//   3. 单次运行详情（输入 prompt + 输出 JSON）
//   4. 定时配置（cron 表达式，默认每天 9:00 / 10:00）
import { useEffect, useState } from "react";
import {
  Card,
  Row,
  Col,
  Button,
  Table,
  Tag,
  Modal,
  Tabs,
  Form,
  Input,
  message,
  Descriptions,
  List,
  Typography,
  Spin,
  Empty,
} from "antd";
import {
  RobotOutlined,
  PlayCircleOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
  SettingOutlined,
  ThunderboltOutlined,
  WarningOutlined,
  CheckCircleOutlined,
} from "@ant-design/icons";
import { agentApi, type AgentRun, type AgentScheduleItem } from "../../api/agent";
import dayjs from "dayjs";

const { Text, Paragraph, Title } = Typography;

// Agent 类型中文映射
const AGENT_NAMES: Record<string, string> = {
  selection: "选品 Agent",
  sourcing: "采购 Agent",
  analysis: "商品分析 Agent",
};

const STATUS_COLOR: Record<string, string> = {
  running: "processing",
  done: "success",
  failed: "error",
};

export default function AgentList() {
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [running, setRunning] = useState<string>(""); // 正在跑的 agent type
  const [detailRun, setDetailRun] = useState<AgentRun | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [schedule, setSchedule] = useState<AgentScheduleItem[]>([]);
  const [report, setReport] = useState<any>(null); // 最近一次报告
  const [activeTab, setActiveTab] = useState("run");

  // 加载历史 + 定时
  const loadData = async () => {
    setLoading(true);
    try {
      const [runsResp, schedResp] = await Promise.all([
        agentApi.listRuns({ page, page_size: 15 }),
        agentApi.getSchedule(),
      ]);
      setRuns(runsResp.list || []);
      setTotal(runsResp.total || 0);
      setSchedule(schedResp || []);
    } catch (e: any) {
      message.error(e.message || "加载失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [page]);

  // 触发选品 Agent
  const runSelection = async () => {
    setRunning("selection");
    setReport(null);
    try {
      const resp = await agentApi.runSelection(14, "");
      setReport({ type: "selection", data: resp.report, run: resp.run });
      message.success("选品 Agent 执行完成");
      loadData();
    } catch (e: any) {
      message.error("选品 Agent 失败: " + e.message);
    } finally {
      setRunning("");
    }
  };

  // 触发采购 Agent
  const runSourcing = async () => {
    setRunning("sourcing");
    setReport(null);
    try {
      const resp = await agentApi.runSourcing("");
      setReport({ type: "sourcing", data: resp.report, run: resp.run });
      message.success("采购 Agent 执行完成");
      loadData();
    } catch (e: any) {
      message.error("采购 Agent 失败: " + e.message);
    } finally {
      setRunning("");
    }
  };

  // 查看详情
  const showDetail = async (id: number) => {
    try {
      const run = await agentApi.getRun(id);
      setDetailRun(run);
      setDetailOpen(true);
    } catch (e: any) {
      message.error(e.message);
    }
  };

  // 更新定时
  const onUpdateSchedule = async (agent_type: string, cron: string) => {
    try {
      const updated = await agentApi.updateSchedule(agent_type, cron);
      setSchedule(updated || []);
      message.success(`${AGENT_NAMES[agent_type]} 定时已更新: ${cron}`);
    } catch (e: any) {
      message.error(e.message);
    }
  };

  // 历史表格列
  const columns = [
    {
      title: "ID",
      dataIndex: "id",
      width: 70,
    },
    {
      title: "Agent",
      dataIndex: "agent_type",
      width: 120,
      render: (t: string) => (
        <Tag color="blue" icon={<RobotOutlined />}>
          {AGENT_NAMES[t] || t}
        </Tag>
      ),
    },
    {
      title: "触发方式",
      dataIndex: "triggered_by",
      width: 90,
      render: (t: string) =>
        t === "cron" ? (
          <Tag color="purple">定时</Tag>
        ) : t === "user" ? (
          <Tag color="cyan">手动</Tag>
        ) : (
          <Tag>事件</Tag>
        ),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 90,
      render: (s: string) => {
        const icon =
          s === "done" ? (
            <CheckCircleOutlined />
          ) : s === "failed" ? (
            <WarningOutlined />
          ) : (
            <Spin size="small" />
          );
        return (
          <Tag color={STATUS_COLOR[s] || "default"} icon={icon}>
            {s === "done" ? "成功" : s === "failed" ? "失败" : "运行中"}
          </Tag>
        );
      },
    },
    {
      title: "Token",
      dataIndex: "tokens_used",
      width: 90,
      render: (t?: number) => (t ? t.toLocaleString() : "-"),
    },
    {
      title: "开始时间",
      dataIndex: "started_at",
      width: 160,
      render: (t: string) => dayjs(t).format("MM-DD HH:mm:ss"),
    },
    {
      title: "耗时",
      width: 90,
      render: (_: any, r: AgentRun) => {
        if (!r.finished_at) return "-";
        const sec = dayjs(r.finished_at).diff(dayjs(r.started_at), "second");
        return `${sec}s`;
      },
    },
    {
      title: "操作",
      width: 90,
      render: (_: any, r: AgentRun) => (
        <Button type="link" size="small" onClick={() => showDetail(r.id)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <div>
      <Title level={4}>
        <RobotOutlined /> AI Agent 任务中心
      </Title>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "run",
            label: (
              <span>
                <ThunderboltOutlined /> 执行 Agent
              </span>
            ),
            children: (
              <div>
                <Row gutter={16}>
                  <Col span={12}>
                    <Card
                      title={
                        <span>
                          <ThunderboltOutlined /> 选品 Agent
                        </span>
                      }
                      bordered
                    >
                      <p>
                        基于近 14 天员工行为（搜索/浏览/收藏）+ 现有商品库，AI
                        输出选品建议。
                      </p>
                      <Button
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        loading={running === "selection"}
                        onClick={runSelection}
                      >
                        立即执行
                      </Button>
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card
                      title={
                        <span>
                          <ThunderboltOutlined /> 采购 Agent
                        </span>
                      }
                      bordered
                    >
                      <p>
                        基于商品库 + 供应商数据，AI 输出采购计划、风险提示、成本优化建议。
                      </p>
                      <Button
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        loading={running === "sourcing"}
                        onClick={runSourcing}
                      >
                        立即执行
                      </Button>
                    </Card>
                  </Col>
                </Row>

                {/* 报告展示 */}
                {report && (
                  <Card
                    title={`${
                      AGENT_NAMES[report.type] || report.type
                    } 执行报告`}
                    style={{ marginTop: 16 }}
                    extra={
                      report.run?.status === "done" ? (
                        <Tag color="success" icon={<CheckCircleOutlined />}>
                          成功
                        </Tag>
                      ) : (
                        <Tag color="error" icon={<WarningOutlined />}>
                          失败
                        </Tag>
                      )
                    }
                  >
                    {report.data?.summary && (
                      <Paragraph strong>{report.data.summary}</Paragraph>
                    )}
                    <ReportDetails report={report.data} type={report.type} />
                  </Card>
                )}

                {running && !report && (
                  <Card style={{ marginTop: 16 }}>
                    <div style={{ textAlign: "center", padding: 40 }}>
                      <Spin size="large" tip="AI 正在分析数据..." />
                    </div>
                  </Card>
                )}
              </div>
            ),
          },
          {
            key: "history",
            label: (
              <span>
                <HistoryOutlined /> 运行历史
              </span>
            ),
            children: (
              <Table
                rowKey="id"
                columns={columns}
                dataSource={runs}
                loading={loading}
                pagination={{
                  current: page,
                  total,
                  pageSize: 15,
                  onChange: setPage,
                  showTotal: (t) => `共 ${t} 条`,
                }}
              />
            ),
          },
          {
            key: "schedule",
            label: (
              <span>
                <SettingOutlined /> 定时配置
              </span>
            ),
            children: (
              <ScheduleTab
                schedule={schedule}
                onUpdate={onUpdateSchedule}
              />
            ),
          },
        ]}
      />

      {/* 单次运行详情 */}
      <Modal
        title={`Agent 运行详情 #${detailRun?.id || ""}`}
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={800}
      >
        {detailRun && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="Agent" span={2}>
              <Tag color="blue">{AGENT_NAMES[detailRun.agent_type] || detailRun.agent_type}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={STATUS_COLOR[detailRun.status]}>
                {detailRun.status}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="触发方式">
              {detailRun.triggered_by}
            </Descriptions.Item>
            <Descriptions.Item label="开始时间">
              {dayjs(detailRun.started_at).format("YYYY-MM-DD HH:mm:ss")}
            </Descriptions.Item>
            <Descriptions.Item label="结束时间">
              {detailRun.finished_at
                ? dayjs(detailRun.finished_at).format("YYYY-MM-DD HH:mm:ss")
                : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="Token" span={2}>
              {detailRun.tokens_used?.toLocaleString() || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="输入（Prompt）" span={2}>
              <Paragraph>
                <pre
                  style={{
                    maxHeight: 200,
                    overflow: "auto",
                    background: "#f5f5f5",
                    padding: 8,
                    fontSize: 12,
                  }}
                >
                  {detailRun.input}
                </pre>
              </Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="输出（AI 响应）" span={2}>
              <Paragraph>
                <pre
                  style={{
                    maxHeight: 300,
                    overflow: "auto",
                    background: "#f5f5f5",
                    padding: 8,
                    fontSize: 12,
                  }}
                >
                  {detailRun.output}
                </pre>
              </Paragraph>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}

// 报告详情子组件
function ReportDetails({ report, type }: { report: any; type: string }) {
  if (!report) return <Empty description="无报告数据" />;
  const sections =
    type === "selection"
      ? [
          { key: "hot_categories", title: "🔥 热门类目推荐", color: "#f50" },
          { key: "avoid_categories", title: "⚠️ 避坑类目", color: "#fa8c16" },
          { key: "high_demand_products", title: "📈 高需求商品方向", color: "#52c41a" },
          { key: "market_trends", title: "🌊 市场趋势", color: "#1677ff" },
          { key: "next_actions", title: "🎯 下一步行动", color: "#722ed1" },
        ]
      : [
          { key: "urgent_purchase", title: "🚨 紧急采购", color: "#f50" },
          { key: "supplier_risks", title: "⚠️ 供应商风险", color: "#fa8c16" },
          { key: "cost_optimization", title: "💰 成本优化", color: "#52c41a" },
          { key: "negotiation_tips", title: "🤝 谈判建议", color: "#1677ff" },
          { key: "alternatives", title: "🔄 替代方案", color: "#722ed1" },
        ];

  return (
    <div>
      {sections.map((sec) => {
        const items = report[sec.key] || [];
        if (!items.length) return null;
        return (
          <Card
            key={sec.key}
            size="small"
            title={<span style={{ color: sec.color }}>{sec.title}</span>}
            style={{ marginBottom: 12 }}
          >
            <List
              size="small"
              dataSource={items}
              renderItem={(item: string, idx: number) => (
                <List.Item>
                  <Text>
                    {idx + 1}. {item}
                  </Text>
                </List.Item>
              )}
            />
          </Card>
        );
      })}
    </div>
  );
}

// 定时配置子组件
function ScheduleTab({
  schedule,
  onUpdate,
}: {
  schedule: AgentScheduleItem[];
  onUpdate: (agent_type: string, cron: string) => void;
}) {
  const [form] = Form.useForm();

  return (
    <Card title={<span><ClockCircleOutlined /> Agent 定时调度</span>}>
      <Paragraph type="secondary">
        使用标准 cron 表达式。默认：选品 Agent 每天 9:00、采购 Agent 每天
        10:00。修改后立即生效（重启调度器）。
      </Paragraph>
      <Table
        rowKey="agent_type"
        dataSource={schedule}
        pagination={false}
        columns={[
          {
            title: "Agent",
            dataIndex: "agent_type",
            render: (t: string) => (
              <Tag color="blue">{AGENT_NAMES[t] || t}</Tag>
            ),
          },
          { title: "Cron 表达式", dataIndex: "cron" },
          {
            title: "说明",
            render: (_: any, r: AgentScheduleItem) =>
              explainCron(r.cron),
          },
          {
            title: "状态",
            dataIndex: "enabled",
            render: (e: boolean) =>
              e ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>,
          },
          {
            title: "操作",
            render: (_: any, r: AgentScheduleItem) => (
              <Button
                size="small"
                onClick={() => {
                  Modal.confirm({
                    title: `修改 ${AGENT_NAMES[r.agent_type]} 定时`,
                    content: (
                      <Form form={form} layout="vertical">
                        <Form.Item
                          name="cron"
                          label="Cron 表达式"
                          initialValue={r.cron}
                          rules={[{ required: true }]}
                        >
                          <Input placeholder="0 9 * * *" />
                        </Form.Item>
                      </Form>
                    ),
                    onOk: async () => {
                      const v = await form.validateFields();
                      onUpdate(r.agent_type, v.cron);
                    },
                  });
                }}
              >
                修改
              </Button>
            ),
          },
        ]}
      />
    </Card>
  );
}

// cron 表达式简单说明
function explainCron(cron: string): string {
  const p = cron.split(/\s+/);
  if (p.length !== 5) return cron;
  const [min, hour, day, month, week] = p;
  if (day === "*" && month === "*" && week === "*") {
    if (min === "0" && hour !== "*") return `每天 ${hour}:00`;
    return `每天 ${hour}:${min.padStart(2, "0")}`;
  }
  return cron;
}
