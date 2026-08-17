// 询盘管理 — B2B 询盘列表 + 状态流转 + AI 分析入口。
//
// 询盘状态：new（新询盘）→ quoting（报价中）→ quoted（已报价）→ won（成交）/ lost（流失）

import { useState, useEffect, useCallback, useMemo } from "react";
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  Popconfirm, Drawer, Descriptions, message, InputNumber, Tooltip, Card,
  type TableColumnsType,
} from "antd";
import {
  PlusOutlined, ReloadOutlined, EditOutlined,
  DeleteOutlined, EyeOutlined, RobotOutlined, CopyOutlined,
} from "@ant-design/icons";
import { inquiryApi, customerApi, type Inquiry, type Customer } from "../../api/b2b";
import { agentApi } from "../../api/agent";
import { ApiError } from "../../api/client";

const STATUS_TAGS: Record<string, { color: string; label: string }> = {
  new: { color: "blue", label: "新询盘" },
  quoting: { color: "orange", label: "报价中" },
  quoted: { color: "cyan", label: "已报价" },
  won: { color: "green", label: "已成交" },
  lost: { color: "red", label: "已流失" },
};

const SOURCE_TAGS: Record<string, { color: string; label: string }> = {
  alibaba: { color: "orange", label: "Alibaba" },
  exhibition: { color: "purple", label: "展会" },
  email: { color: "blue", label: "邮件" },
  website: { color: "geekblue", label: "官网" },
};

export default function InquiryList() {
  const [data, setData] = useState<Inquiry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<{ page: number; page_size: number; source?: string; status?: string }>({ page: 1, page_size: 10 });
  const [editModal, setEditModal] = useState<{ open: boolean; inquiry?: Inquiry }>({ open: false });
  const [detail, setDetail] = useState<{ open: boolean; inquiry?: Inquiry }>({ open: false });
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [analyzing, setAnalyzing] = useState(false);
  const [analysis, setAnalysis] = useState<Record<string, unknown> | null>(null);
  const [form] = Form.useForm();

  const customerMap = useMemo(() => {
    const m = new Map<number, string>();
    customers.forEach((c) => m.set(c.id, c.company_name));
    return m;
  }, [customers]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await inquiryApi.list(query);
      setData(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error((e as ApiError).message || "加载失败");
    } finally { setLoading(false); }
  }, [query]);

  // 客户列表（创建表单选择 + 列表显示公司名，私有化部署取前 100 足够）
  useEffect(() => {
    customerApi.list({ page: 1, page_size: 100 })
      .then((res) => setCustomers(res.list || []))
      .catch(() => { /* 客户加载失败不阻塞询盘列表 */ });
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const onAdd = () => {
    form.resetFields();
    form.setFieldsValue({ source: "alibaba" });
    setEditModal({ open: true });
  };

  const onEdit = (i: Inquiry) => {
    form.setFieldsValue(i);
    setEditModal({ open: true, inquiry: i });
  };

  const onSave = async () => {
    try {
      const raw = await form.validateFields();
      if (editModal.inquiry) {
        await inquiryApi.update(editModal.inquiry.id, raw);
        message.success("已更新");
      } else {
        await inquiryApi.create(raw);
        message.success("已创建");
      }
      setEditModal({ open: false });
      fetchData();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; message?: string };
      if (err.errorFields) return;
      message.error(err.message || "保存失败");
    }
  };

  const onDelete = async (id: number) => {
    try {
      await inquiryApi.delete(id);
      message.success("已删除");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "删除失败");
    }
  };

  // 状态快速流转（列表行内下拉）
  const onStatusChange = async (id: number, status: string) => {
    try {
      await inquiryApi.update(id, { status });
      message.success("状态已更新");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "状态更新失败");
    }
  };

  // AI 分析询盘（Week 6 询盘分析 Agent，结果回写 ai_analysis）
  const onAnalyze = async (inquiry: Inquiry) => {
    setAnalyzing(true);
    setAnalysis(null);
    try {
      const res = await agentApi.analyzeInquiry(inquiry.id);
      setAnalysis(res.analysis || null);
      message.success("AI 分析完成");
      // 重新拉详情（后端已回写 ai_analysis）
      try {
        const d = await inquiryApi.get(inquiry.id);
        setDetail({ open: true, inquiry: d });
      } catch { /* 保持当前数据 */ }
    } catch (e) {
      message.error((e as ApiError).message || "AI 分析失败（检查 AI Key 配置）");
    } finally {
      setAnalyzing(false);
    }
  };

  // ai_analysis 是 JSON 字符串（预算/决策阶段/成交概率/跟进策略）
  const parsedAnalysis = useMemo(() => {
    const raw = detail.inquiry?.ai_analysis;
    if (!raw) return null;
    try { return JSON.parse(raw) as Record<string, unknown>; } catch { return null; }
  }, [detail.inquiry]);

  const columns: TableColumnsType<Inquiry> = [
    {
      title: "询价产品",
      dataIndex: "product_desc",
      render: (desc, r) => (
        <div>
          <div style={{ fontWeight: 500, maxWidth: 320, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            {desc}
          </div>
          <div style={{ fontSize: 12, color: "#999" }}>
            {r.destination && <>📍 {r.destination}　</>}
            {r.quantity && <>📦 {r.quantity.toLocaleString()}</>}
          </div>
        </div>
      ),
    },
    {
      title: "客户",
      dataIndex: "customer_id",
      width: 160,
      render: (cid) => (cid && customerMap.get(cid)) || <span style={{ color: "#ccc" }}>未关联</span>,
    },
    {
      title: "来源",
      dataIndex: "source",
      width: 100,
      render: (s) => {
        const t = SOURCE_TAGS[s];
        return t ? <Tag color={t.color}>{t.label}</Tag> : <Tag>{s || "—"}</Tag>;
      },
    },
    {
      title: "目标价",
      dataIndex: "target_price",
      width: 100,
      render: (p) => p ? `$${p}` : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 130,
      render: (s, r) => (
        <Select
          size="small"
          value={s}
          style={{ width: 105 }}
          onChange={(v) => onStatusChange(r.id, v)}
          options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
        />
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 110,
      render: (t) => t ? new Date(t).toLocaleDateString("zh-CN") : "—",
    },
    {
      title: "操作",
      width: 130,
      render: (_, r) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={async () => {
            try { const d = await inquiryApi.get(r.id); setDetail({ open: true, inquiry: d }); }
            catch { setDetail({ open: true, inquiry: r }); }
          }} />
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => onEdit(r)} />
          <Popconfirm title="确定删除？" onConfirm={() => onDelete(r.id)} okText="删除" okButtonProps={{ danger: true }}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <Select
          placeholder="来源"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, source: v, page: 1 }))}
          options={Object.entries(SOURCE_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
        />
        <Select
          placeholder="状态"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, status: v, page: 1 }))}
          options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
        <div style={{ flex: 1 }} />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>录入询盘</Button>
      </div>

      <Table
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        size="middle"
        scroll={{ x: 900 }}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条询盘`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
      />

      {/* 新增/编辑 Modal */}
      <Modal
        title={editModal.inquiry ? "编辑询盘" : "录入询盘"}
        open={editModal.open}
        onOk={onSave}
        onCancel={() => setEditModal({ open: false })}
        okText="保存"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="customer_id" label="关联客户">
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="选择客户（可不选）"
              options={customers.map((c) => ({ label: c.company_name, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="source" label="来源" rules={[{ required: true }]}>
            <Select
              placeholder="询盘来源"
              showSearch
              options={[
                ...Object.entries(SOURCE_TAGS).map(([v, t]) => ({ label: t.label, value: v })),
                { label: "其他（手动输入）", value: "other" },
              ]}
            />
          </Form.Item>
          <Form.Item name="product_desc" label="询价产品描述" rules={[{ required: true, message: "请填写产品描述" }]}>
            <Input.TextArea rows={3} placeholder="客户在找什么产品？规格？材质？如：stainless steel water bottle 500ml, MOQ 3000" />
          </Form.Item>
          <Space size="large">
            <Form.Item name="quantity" label="数量">
              <InputNumber min={0} placeholder="5000" style={{ width: 140 }} />
            </Form.Item>
            <Form.Item name="target_price" label="目标价 (USD)">
              <Input placeholder="2.35" style={{ width: 140 }} />
            </Form.Item>
          </Space>
          <Form.Item name="destination" label="目的港">
            <Input placeholder="如：Hamburg / LA / Dubai" />
          </Form.Item>
          {editModal.inquiry && (
            <Form.Item name="status" label="状态">
              <Select options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))} />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={`询盘 #${detail.inquiry?.id ?? ""}`}
        open={detail.open}
        onClose={() => { setDetail({ open: false }); setAnalysis(null); }}
        width={520}
      >
        {detail.inquiry && (
          <>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="产品描述">{detail.inquiry.product_desc}</Descriptions.Item>
              <Descriptions.Item label="客户">
                {(detail.inquiry.customer_id && customerMap.get(detail.inquiry.customer_id)) || "未关联"}
              </Descriptions.Item>
              <Descriptions.Item label="来源">
                <Tag color={(SOURCE_TAGS[detail.inquiry.source] || {}).color}>{(SOURCE_TAGS[detail.inquiry.source] || {}).label || detail.inquiry.source}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="数量">{detail.inquiry.quantity?.toLocaleString() || "—"}</Descriptions.Item>
              <Descriptions.Item label="目标价">{detail.inquiry.target_price ? `$${detail.inquiry.target_price}` : "—"}</Descriptions.Item>
              <Descriptions.Item label="目的港">{detail.inquiry.destination || "—"}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Select
                  size="small"
                  value={detail.inquiry.status}
                  style={{ width: 105 }}
                  onChange={(v) => onStatusChange(detail.inquiry!.id, v)}
                  options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
                />
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{new Date(detail.inquiry.created_at).toLocaleString("zh-CN")}</Descriptions.Item>
            </Descriptions>

            <Button
              type="primary"
              ghost
              icon={<RobotOutlined />}
              loading={analyzing}
              style={{ marginTop: 16 }}
              onClick={() => onAnalyze(detail.inquiry!)}
            >
              {analyzing ? "AI 分析中…" : "🤖 AI 分析询盘"}
            </Button>

            {(analysis || parsedAnalysis) && (
              <Card title="AI 分析结果" style={{ marginTop: 16 }} size="small"
                extra={
                  <Tooltip title="复制分析结果">
                    <Button size="small" icon={<CopyOutlined />} onClick={() => {
                      navigator.clipboard.writeText(JSON.stringify(analysis || parsedAnalysis, null, 2));
                      message.success("已复制");
                    }} />
                  </Tooltip>
                }>
                <pre style={{ fontSize: 12, maxHeight: 360, overflow: "auto", margin: 0 }}>
                  {JSON.stringify(analysis || parsedAnalysis, null, 2)}
                </pre>
              </Card>
            )}
          </>
        )}
      </Drawer>
    </div>
  );
}
