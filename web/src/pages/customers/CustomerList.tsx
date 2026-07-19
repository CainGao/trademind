// 客户管理 — B2B 客户列表 + 销售阶段看板。
//
// 客户阶段：lead（线索）→ quoting（报价中）→ negotiating（谈判中）→ won（成交）/ lost（流失）

import { useState, useEffect, useCallback } from "react";
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  Popconfirm, Drawer, Descriptions, message, type TableColumnsType,
} from "antd";
import {
  PlusOutlined, SearchOutlined, ReloadOutlined, EditOutlined,
  DeleteOutlined, EyeOutlined,
} from "@ant-design/icons";
import { customerApi, type CustomerListQuery, type Customer } from "../../api/b2b";
import { ApiError } from "../../api/client";

const STAGE_TAGS: Record<string, { color: string; label: string }> = {
  lead: { color: "default", label: "线索" },
  quoting: { color: "blue", label: "报价中" },
  negotiating: { color: "orange", label: "谈判中" },
  won: { color: "green", label: "已成交" },
  lost: { color: "red", label: "已流失" },
};

export default function CustomerList() {
  const [data, setData] = useState<Customer[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<CustomerListQuery>({ page: 1, page_size: 10 });
  const [editModal, setEditModal] = useState<{ open: boolean; customer?: Customer }>({ open: false });
  const [detail, setDetail] = useState<{ open: boolean; customer?: Customer }>({ open: false });
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await customerApi.list(query);
      setData(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error((e as ApiError).message || "加载失败");
    } finally { setLoading(false); }
  }, [query]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const onAdd = () => {
    form.resetFields();
    form.setFieldsValue({ stage: "lead" });
    setEditModal({ open: true });
  };

  const onEdit = (c: Customer) => {
    form.setFieldsValue(c);
    setEditModal({ open: true, customer: c });
  };

  const onSave = async () => {
    try {
      const values = await form.validateFields();
      if (editModal.customer) {
        await customerApi.update(editModal.customer.id, values);
        message.success("已更新");
      } else {
        await customerApi.create(values);
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
      await customerApi.delete(id);
      message.success("已删除");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "删除失败");
    }
  };

  const columns: TableColumnsType<Customer> = [
    {
      title: "公司名称",
      dataIndex: "company_name",
      render: (name, r) => (
        <div>
          <div style={{ fontWeight: 500 }}>{name}</div>
          {r.contact_person && <div style={{ fontSize: 12, color: "#999" }}>👤 {r.contact_person}</div>}
        </div>
      ),
    },
    {
      title: "国家",
      dataIndex: "country",
      width: 100,
      render: (c) => c || <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "联系方式",
      width: 180,
      render: (_, r) => (
        <div style={{ fontSize: 12 }}>
          {r.email && <div>✉️ {r.email}</div>}
          {r.wechat && <div>💬 {r.wechat}</div>}
        </div>
      ),
    },
    {
      title: "阶段",
      dataIndex: "stage",
      width: 110,
      filters: Object.entries(STAGE_TAGS).map(([v, t]) => ({ text: t.label, value: v })),
      render: (s) => {
        const t = STAGE_TAGS[s] || STAGE_TAGS.lead;
        return <Tag color={t.color}>{t.label}</Tag>;
      },
    },
    {
      title: "成交概率",
      dataIndex: "deal_probability",
      width: 100,
      sorter: true,
      render: (p) => p ? <Tag color={Number(p) >= 0.7 ? "green" : Number(p) >= 0.4 ? "orange" : "red"}>{Math.round(Number(p) * 100)}%</Tag> : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "最近联系",
      dataIndex: "last_contact_at",
      width: 120,
      render: (t) => t ? new Date(t).toLocaleDateString("zh-CN") : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "操作",
      width: 130,
      render: (_, r) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={async () => {
            try { const d = await customerApi.get(r.id); setDetail({ open: true, customer: d }); }
            catch { setDetail({ open: true, customer: r }); }
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
        <Input.Search
          placeholder="公司名/联系人/邮箱"
          allowClear
          onSearch={(v) => setQuery((q) => ({ ...q, keyword: v, page: 1 }))}
          style={{ width: 260 }}
          prefix={<SearchOutlined />}
        />
        <Select
          placeholder="阶段"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, stage: v, page: 1 }))}
          options={Object.entries(STAGE_TAGS).map(([v, t]) => ({ label: t.label, value: v }))}
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
        <div style={{ flex: 1 }} />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>新增客户</Button>
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
          showTotal: (t) => `共 ${t} 个客户`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
      />

      {/* 新增/编辑 Modal */}
      <Modal
        title={editModal.customer ? "编辑客户" : "新增客户"}
        open={editModal.open}
        onOk={onSave}
        onCancel={() => setEditModal({ open: false })}
        okText="保存"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="company_name" label="公司名称" rules={[{ required: true }]}>
            <Input placeholder="如：ABC Trading Co." />
          </Form.Item>
          <Form.Item name="country" label="国家">
            <Input placeholder="如：United States" />
          </Form.Item>
          <Form.Item name="contact_person" label="联系人">
            <Input placeholder="John Smith" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input placeholder="john@abc.com" />
          </Form.Item>
          <Form.Item name="phone" label="电话">
            <Input placeholder="+1 555-1234" />
          </Form.Item>
          <Form.Item name="wechat" label="微信">
            <Input placeholder="WeChat ID" />
          </Form.Item>
          <Form.Item name="demand" label="需求描述">
            <Input.TextArea rows={2} placeholder="客户在找什么产品？数量？目标价？" />
          </Form.Item>
          <Form.Item name="stage" label="阶段">
            <Select options={Object.entries(STAGE_TAGS).map(([v, t]) => ({ label: t.label, value: v }))} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer title="客户详情" open={detail.open} onClose={() => setDetail({ open: false })} width={480}>
        {detail.customer && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="公司">{detail.customer.company_name}</Descriptions.Item>
            {detail.customer.country && <Descriptions.Item label="国家">{detail.customer.country}</Descriptions.Item>}
            {detail.customer.contact_person && <Descriptions.Item label="联系人">{detail.customer.contact_person}</Descriptions.Item>}
            {detail.customer.email && <Descriptions.Item label="邮箱">{detail.customer.email}</Descriptions.Item>}
            {detail.customer.phone && <Descriptions.Item label="电话">{detail.customer.phone}</Descriptions.Item>}
            {detail.customer.wechat && <Descriptions.Item label="微信">{detail.customer.wechat}</Descriptions.Item>}
            <Descriptions.Item label="阶段">
              <Tag color={(STAGE_TAGS[detail.customer.stage] || STAGE_TAGS.lead).color}>
                {(STAGE_TAGS[detail.customer.stage] || STAGE_TAGS.lead).label}
              </Tag>
            </Descriptions.Item>
            {detail.customer.demand && <Descriptions.Item label="需求">{detail.customer.demand}</Descriptions.Item>}
            <Descriptions.Item label="建档时间">{new Date(detail.customer.created_at).toLocaleString("zh-CN")}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
}
