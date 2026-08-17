// 报价单 — B2B 报价单列表 + 状态流转（draft→sent→accepted/rejected/expired）+ 明细查看。
//
// 单号自动生成（QUO-2026-0001），本页负责创建、状态推进、明细展示。

import { useState, useEffect, useCallback, useMemo } from "react";
import {
  Table, Button, Select, Space, Modal, Form,
  Popconfirm, Drawer, Descriptions, message, InputNumber, Input,
  Typography, type TableColumnsType,
} from "antd";
import {
  PlusOutlined, ReloadOutlined,
  DeleteOutlined, EyeOutlined, CopyOutlined,
} from "@ant-design/icons";
import { quotationApi, customerApi, type Quotation, type Customer } from "../../api/b2b";
import { ApiError } from "../../api/client";

const { Text } = Typography;

const STATUS_TAGS: Record<string, { color: string; label: string }> = {
  draft: { color: "default", label: "草稿" },
  sent: { color: "blue", label: "已发送" },
  accepted: { color: "green", label: "已接受" },
  rejected: { color: "red", label: "已拒绝" },
  expired: { color: "gray", label: "已过期" },
};

const CURRENCIES = ["USD", "CNY", "EUR", "GBP", "JPY", "AUD", "CAD"];

interface QuotationItem {
  name?: string;
  sku?: string;
  qty?: number;
  price?: string;
  subtotal?: string;
}

export default function QuotationList() {
  const [data, setData] = useState<Quotation[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<{ page: number; page_size: number; status?: string }>({ page: 1, page_size: 10 });
  const [createOpen, setCreateOpen] = useState(false);
  const [detail, setDetail] = useState<{ open: boolean; quotation?: Quotation }>({ open: false });
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [form] = Form.useForm();

  const customerMap = useMemo(() => {
    const m = new Map<number, string>();
    customers.forEach((c) => m.set(c.id, c.company_name));
    return m;
  }, [customers]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await quotationApi.list(query);
      setData(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error((e as ApiError).message || "加载失败");
    } finally { setLoading(false); }
  }, [query]);

  useEffect(() => {
    customerApi.list({ page: 1, page_size: 100 })
      .then((res) => setCustomers(res.list || []))
      .catch(() => { /* 客户加载失败不阻塞报价单列表 */ });
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const onAdd = () => {
    form.resetFields();
    form.setFieldsValue({ currency: "USD", valid_days: 30 });
    setCreateOpen(true);
  };

  const onCreate = async () => {
    try {
      const raw = await form.validateFields();
      await quotationApi.create(raw);
      message.success("报价单已创建");
      setCreateOpen(false);
      fetchData();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; message?: string };
      if (err.errorFields) return;
      message.error(err.message || "创建失败");
    }
  };

  const onDelete = async (id: number) => {
    try {
      await quotationApi.delete(id);
      message.success("已删除");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "删除失败");
    }
  };

  const onStatusChange = async (id: number, status: string) => {
    try {
      await quotationApi.updateStatus(id, status);
      message.success("状态已更新");
      fetchData();
      // 如果详情抽屉打开，同步刷新
      if (detail.open && detail.quotation?.id === id) {
        try { const d = await quotationApi.get(id); setDetail({ open: true, quotation: d }); } catch { /* ignore */ }
      }
    } catch (e) {
      message.error((e as ApiError).message || "状态更新失败");
    }
  };

  // items 是 JSON 字符串（商品/数量/单价/小计）
  const parsedItems = useMemo<QuotationItem[] | null>(() => {
    const raw = detail.quotation?.items;
    if (!raw) return null;
    try {
      const v = JSON.parse(raw);
      return Array.isArray(v) ? (v as QuotationItem[]) : null;
    } catch { return null; }
  }, [detail.quotation]);

  const columns: TableColumnsType<Quotation> = [
    {
      title: "报价单号",
      dataIndex: "quotation_no",
      width: 150,
      render: (no) => <span style={{ fontWeight: 500 }}>{no}</span>,
    },
    {
      title: "客户",
      dataIndex: "customer_id",
      width: 170,
      render: (cid) => (cid && customerMap.get(cid)) || <span style={{ color: "#ccc" }}>未关联</span>,
    },
    {
      title: "金额",
      dataIndex: "total_amount",
      width: 140,
      align: "right",
      render: (amt, r) => (
        <Text strong>{r.currency} {Number(amt).toLocaleString("en-US", { minimumFractionDigits: 2 })}</Text>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 120,
      render: (s, r) => (
        <Select
          size="small"
          value={s}
          style={{ width: 100 }}
          onChange={(v) => onStatusChange(r.id, v)}
          options={Object.entries(STATUS_TAGS).map(([v, tt]) => ({ label: tt.label, value: v }))}
        />
      ),
    },
    {
      title: "有效期至",
      dataIndex: "valid_until",
      width: 110,
      render: (t) => {
        if (!t) return <span style={{ color: "#ccc" }}>长期</span>;
        const expired = new Date(t) < new Date();
        return <span style={{ color: expired ? "#ff4d4f" : undefined }}>{new Date(t).toLocaleDateString("zh-CN")}{expired ? "（已过期）" : ""}</span>;
      },
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 110,
      render: (t) => t ? new Date(t).toLocaleDateString("zh-CN") : "—",
    },
    {
      title: "操作",
      width: 100,
      render: (_, r) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={async () => {
            try { const d = await quotationApi.get(r.id); setDetail({ open: true, quotation: d }); }
            catch { setDetail({ open: true, quotation: r }); }
          }} />
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
          placeholder="状态"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, status: v, page: 1 }))}
          options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
        <div style={{ flex: 1 }} />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>新建报价单</Button>
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
          showTotal: (t) => `共 ${t} 份报价单`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
      />

      {/* 创建 Modal */}
      <Modal
        title="新建报价单"
        open={createOpen}
        onOk={onCreate}
        onCancel={() => setCreateOpen(false)}
        okText="创建"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="customer_id" label="客户">
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="选择客户（可不选）"
              options={customers.map((c) => ({ label: c.company_name, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="inquiry_id" label="关联询盘 ID">
            <InputNumber min={1} placeholder="如：12（可从询盘管理页查看 ID）" style={{ width: "100%" }} />
          </Form.Item>
          <Space size="large">
            <Form.Item name="currency" label="币种" rules={[{ required: true }]}>
              <Select style={{ width: 100 }} options={CURRENCIES.map((c) => ({ label: c, value: c }))} />
            </Form.Item>
            <Form.Item name="total_amount" label="总金额" rules={[{ required: true, message: "请填写总金额" }]}>
              <Input placeholder="12500.00" style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="valid_days" label="有效期（天）" tooltip="0 表示不设置有效期">
              <InputNumber min={0} max={3650} placeholder="30" style={{ width: 110 }} />
            </Form.Item>
          </Space>
          <Form.Item
            name="items"
            label="报价明细（JSON 数组，可留空）"
            tooltip='格式：[{"name":"保温杯500ml","qty":3000,"price":"2.35","subtotal":"7050.00"}]'
          >
            <Input.TextArea
              rows={4}
              placeholder={'[{"name":"保温杯500ml","qty":3000,"price":"2.35","subtotal":"7050.00"}]'}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title={`报价单 ${detail.quotation?.quotation_no ?? ""}`}
        open={detail.open}
        onClose={() => setDetail({ open: false })}
        width={520}
      >
        {detail.quotation && (
          <>
            <Descriptions column={1} bordered size="small">
              <Descriptions.Item label="单号">
                <Space>
                  <Text strong>{detail.quotation.quotation_no}</Text>
                  <Button size="small" type="text" icon={<CopyOutlined />} onClick={() => {
                    navigator.clipboard.writeText(detail.quotation!.quotation_no);
                    message.success("已复制单号");
                  }} />
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="客户">
                {(detail.quotation.customer_id && customerMap.get(detail.quotation.customer_id)) || "未关联"}
              </Descriptions.Item>
              {detail.quotation.inquiry_id && (
                <Descriptions.Item label="关联询盘">#{detail.quotation.inquiry_id}</Descriptions.Item>
              )}
              <Descriptions.Item label="金额">
                <Text strong>
                  {detail.quotation.currency} {Number(detail.quotation.total_amount).toLocaleString("en-US", { minimumFractionDigits: 2 })}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Select
                  size="small"
                  value={detail.quotation.status}
                  style={{ width: 100 }}
                  onChange={(v) => onStatusChange(detail.quotation!.id, v)}
                  options={Object.entries(STATUS_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
                />
              </Descriptions.Item>
              <Descriptions.Item label="有效期至">
                {detail.quotation.valid_until ? new Date(detail.quotation.valid_until).toLocaleDateString("zh-CN") : "长期有效"}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间">{new Date(detail.quotation.created_at).toLocaleString("zh-CN")}</Descriptions.Item>
            </Descriptions>

            {(parsedItems || detail.quotation.items) && (
              <Table
                style={{ marginTop: 16 }}
                size="small"
                rowKey={(_, i) => String(i)}
                pagination={false}
                dataSource={parsedItems || []}
                columns={[
                  { title: "商品", dataIndex: "name", render: (n, r: QuotationItem) => n || r.sku || "—" },
                  { title: "数量", dataIndex: "qty", width: 80, align: "right", render: (q) => q?.toLocaleString() ?? "—" },
                  { title: "单价", dataIndex: "price", width: 90, align: "right", render: (p) => p != null ? String(p) : "—" },
                  { title: "小计", dataIndex: "subtotal", width: 110, align: "right", render: (s) => s != null ? String(s) : "—" },
                ]}
                locale={{ emptyText: "明细格式非 JSON，原始内容见下" }}
              />
            )}
            {detail.quotation.items && !parsedItems && (
              <pre style={{ fontSize: 12, marginTop: 8, whiteSpace: "pre-wrap" }}>{detail.quotation.items}</pre>
            )}
          </>
        )}
      </Drawer>
    </div>
  );
}
