// 审计日志页（管理员）：查看登录/登录失败等敏感操作记录。
// 后端从 Week 1 起持续写入 audit_logs，本页是其唯一查看入口（合规要求）。
import { useEffect, useState, useCallback } from "react";
import { Table, Card, Select, DatePicker, Button, Space, Tag, Typography, message } from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { auditApi, type AuditLogItem } from "../../api/audit";
import dayjs, { type Dayjs } from "dayjs";

const { Text } = Typography;

const PAGE_SIZE = 20;

// action → 展示文案 + 颜色（与后端写入侧对齐：login/login_failed，后续扩展 create/update/delete/export）
const actionMeta: Record<string, { label: string; color: string }> = {
  login: { label: "登录成功", color: "green" },
  login_failed: { label: "登录失败", color: "red" },
  create: { label: "创建", color: "blue" },
  update: { label: "更新", color: "orange" },
  delete: { label: "删除", color: "red" },
  export: { label: "导出", color: "purple" },
};

function actionTag(action: string) {
  const meta = actionMeta[action];
  if (meta) return <Tag color={meta.color}>{meta.label}</Tag>;
  return <Tag>{action}</Tag>;
}

export default function AuditLogs() {
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<AuditLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [action, setAction] = useState<string | undefined>(undefined);
  const [range, setRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);

  const fetchData = useCallback(
    async (p: number) => {
      setLoading(true);
      try {
        const params: Record<string, unknown> = { page: p, page_size: PAGE_SIZE };
        if (action) params.action = action;
        if (range && range[0]) params.start_date = range[0].format("YYYY-MM-DD");
        if (range && range[1]) params.end_date = range[1].format("YYYY-MM-DD");
        const res = await auditApi.list(params);
        setRows(res.list ?? []);
        setTotal(res.total ?? 0);
        setPage(p);
      } catch (e: unknown) {
        const err = e as { message?: string };
        message.error(err.message || "加载审计日志失败");
      } finally {
        setLoading(false);
      }
    },
    [action, range]
  );

  useEffect(() => {
    fetchData(1);
  }, [fetchData]);

  const columns: ColumnsType<AuditLogItem> = [
    {
      title: "时间",
      dataIndex: "created_at",
      width: 170,
      render: (v: string) => <Text type="secondary">{dayjs(v).format("YYYY-MM-DD HH:mm:ss")}</Text>,
    },
    {
      title: "用户",
      dataIndex: "username",
      width: 130,
      render: (v: string, r: AuditLogItem) => (
        <span>
          {v}
          {r.user_id ? <Text type="secondary"> #{r.user_id}</Text> : null}
        </span>
      ),
    },
    {
      title: "操作",
      dataIndex: "action",
      width: 110,
      render: (v: string) => actionTag(v),
    },
    {
      title: "对象",
      dataIndex: "resource",
      width: 110,
      render: (v: string, r: AuditLogItem) => (
        <span>
          {v}
          {r.resource_id ? <Text type="secondary"> #{r.resource_id}</Text> : null}
        </span>
      ),
    },
    {
      title: "详情",
      dataIndex: "detail",
      ellipsis: true,
    },
    {
      title: "IP",
      dataIndex: "ip",
      width: 130,
      render: (v: string) => <Text code>{v || "—"}</Text>,
    },
  ];

  return (
    <Card
      title="审计日志"
      extra={
        <Space>
          <Select
            allowClear
            placeholder="操作类型"
            style={{ width: 140 }}
            value={action}
            onChange={setAction}
            options={Object.entries(actionMeta).map(([value, m]) => ({ value, label: m.label }))}
          />
          <DatePicker.RangePicker
            value={range as [Dayjs, Dayjs] | null}
            onChange={(vals) => setRange(vals as [Dayjs | null, Dayjs | null] | null)}
          />
          <Button icon={<SearchOutlined />} type="primary" onClick={() => fetchData(1)}>
            查询
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => fetchData(page)} />
        </Space>
      }
    >
      <Table<AuditLogItem>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={rows}
        pagination={{
          current: page,
          pageSize: PAGE_SIZE,
          total,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p) => fetchData(p),
        }}
      />
    </Card>
  );
}
