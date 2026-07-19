// 供应商管理 — 列表 + 风险评估 + AI 评分 + 详情抽屉。
//
// 供应商来自 Chrome 插件 1688/阿里巴巴采集时的自动入库。
// 老板/采购员可在此评估风险（low/medium/high）+ 手动评分（0-10）。

import { useState, useEffect, useCallback } from "react";
import {
  Table, Button, Input, Select, Space, Tag, Drawer, Descriptions,
  Popconfirm, message, Rate, Radio, type TableColumnsType,
} from "antd";
import {
  SearchOutlined, ReloadOutlined, EditOutlined, DeleteOutlined, EyeOutlined,
} from "@ant-design/icons";
import { supplierApi, type SupplierListQuery } from "../../api/supplier";
import { ApiError } from "../../api/client";
import type { Supplier } from "../../types";

const SOURCE_COLORS: Record<string, string> = {
  "1688": "orange",
  alibaba: "blue",
  amazon: "gold",
  factory: "green",
  manual: "default",
};

const RISK_TAGS: Record<string, { color: string; label: string }> = {
  low: { color: "green", label: "低风险" },
  medium: { color: "orange", label: "中风险" },
  high: { color: "red", label: "高风险" },
};

export default function SupplierList() {
  const [data, setData] = useState<Supplier[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<SupplierListQuery>({ page: 1, page_size: 10 });
  const [detail, setDetail] = useState<{ open: boolean; data?: any }>({ open: false });
  const [riskModal, setRiskModal] = useState<{ open: boolean; supplier?: Supplier }>({ open: false });
  const [riskLevel, setRiskLevel] = useState("medium");
  const [aiScore, setAiScore] = useState(0);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await supplierApi.list(query);
      setData(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error((e as ApiError).message || "加载失败");
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const onEditRisk = (s: Supplier) => {
    setRiskLevel(s.risk_level || "medium");
    setAiScore(Number(s.ai_score) || 0);
    setRiskModal({ open: true, supplier: s });
  };

  const onSaveRisk = async () => {
    if (!riskModal.supplier) return;
    try {
      await supplierApi.updateRisk(riskModal.supplier.id, {
        risk_level: riskLevel,
        ai_score: String(aiScore),
      });
      message.success("评分已更新");
      setRiskModal({ open: false });
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "保存失败");
    }
  };

  const onDelete = async (id: number) => {
    try {
      await supplierApi.delete(id);
      message.success("已删除");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "删除失败");
    }
  };

  const onView = async (s: Supplier) => {
    try {
      const d = await supplierApi.get(s.id);
      setDetail({ open: true, data: d });
    } catch {
      setDetail({ open: true, data: s });
    }
  };

  const columns: TableColumnsType<Supplier> = [
    {
      title: "供应商名称",
      dataIndex: "name",
      ellipsis: true,
      render: (name, r) => (
        <div>
          <div style={{ fontWeight: 500 }}>{name}</div>
          {r.location && <div style={{ fontSize: 12, color: "#999" }}>📍 {r.location}</div>}
        </div>
      ),
    },
    {
      title: "来源",
      dataIndex: "source",
      width: 90,
      render: (src) => <Tag color={SOURCE_COLORS[src] || "default"}>{src}</Tag>,
    },
    {
      title: "商品数",
      dataIndex: "product_count",
      width: 80,
      sorter: true,
      render: (n) => n ? <Tag color="blue">{n}</Tag> : <span style={{ color: "#ccc" }}>0</span>,
    },
    {
      title: "AI 评分",
      dataIndex: "ai_score",
      width: 130,
      sorter: true,
      render: (s) => {
        const score = Number(s) || 0;
        return <Rate disabled count={10} value={score} style={{ fontSize: 12 }} />;
      },
    },
    {
      title: "风险等级",
      dataIndex: "risk_level",
      width: 100,
      filters: [
        { text: "低风险", value: "low" },
        { text: "中风险", value: "medium" },
        { text: "高风险", value: "high" },
      ],
      render: (r) => {
        const t = RISK_TAGS[r] || RISK_TAGS.medium;
        return <Tag color={t.color}>{t.label}</Tag>;
      },
    },
    {
      title: "最后活跃",
      dataIndex: "last_active_at",
      width: 140,
      render: (t) => t ? new Date(t).toLocaleDateString("zh-CN") : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "操作",
      width: 140,
      render: (_, r) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => onView(r)} />
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => onEditRisk(r)} />
          <Popconfirm title="确定删除？" onConfirm={() => onDelete(r.id)} okText="删除" cancelText="取消" okButtonProps={{ danger: true }}>
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
          placeholder="搜索供应商名称"
          allowClear
          onSearch={(v) => setQuery((q) => ({ ...q, keyword: v, page: 1 }))}
          style={{ width: 240 }}
          prefix={<SearchOutlined />}
        />
        <Select
          placeholder="来源筛选"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, source: v, page: 1 }))}
          options={[
            { label: "1688", value: "1688" },
            { label: "阿里巴巴", value: "alibaba" },
            { label: "工厂", value: "factory" },
          ]}
        />
        <Select
          placeholder="风险等级"
          allowClear
          style={{ width: 130 }}
          onChange={(v) => setQuery((q) => ({ ...q, risk_level: v, page: 1 }))}
          options={[
            { label: "低风险", value: "low" },
            { label: "中风险", value: "medium" },
            { label: "高风险", value: "high" },
          ]}
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
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
          showTotal: (t) => `共 ${t} 家供应商`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
      />

      {/* 详情抽屉 */}
      <Drawer
        title="供应商详情"
        open={detail.open}
        onClose={() => setDetail({ open: false })}
        width={480}
      >
        {detail.data && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="名称">{detail.data.name}</Descriptions.Item>
            <Descriptions.Item label="来源">
              <Tag>{detail.data.source}</Tag> {detail.data.source_id}
            </Descriptions.Item>
            {detail.data.location && <Descriptions.Item label="所在地">📍 {detail.data.location}</Descriptions.Item>}
            <Descriptions.Item label="关联商品数">{detail.data.product_count ?? 0}</Descriptions.Item>
            <Descriptions.Item label="AI 评分">
              <Rate disabled count={10} value={Number(detail.data.ai_score) || 0} style={{ fontSize: 12 }} />
            </Descriptions.Item>
            <Descriptions.Item label="风险等级">
              <Tag color={(RISK_TAGS[detail.data.risk_level] || RISK_TAGS.medium).color}>
                {(RISK_TAGS[detail.data.risk_level] || RISK_TAGS.medium).label}
              </Tag>
            </Descriptions.Item>
            {detail.data.contact && (
              <Descriptions.Item label="联系方式">{detail.data.contact}</Descriptions.Item>
            )}
            <Descriptions.Item label="入库时间">{new Date(detail.data.created_at).toLocaleString("zh-CN")}</Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>

      {/* 评分 Modal */}
      {riskModal.open && (
        <Drawer
          title={`评估：${riskModal.supplier?.name}`}
          open={riskModal.open}
          onClose={() => setRiskModal({ open: false })}
          width={420}
          extra={<Button type="primary" onClick={onSaveRisk}>保存</Button>}
        >
          <div style={{ marginBottom: 24 }}>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>风险等级</div>
            <Radio.Group value={riskLevel} onChange={(e) => setRiskLevel(e.target.value)}>
              <Radio.Button value="low">🟢 低风险</Radio.Button>
              <Radio.Button value="medium">🟡 中风险</Radio.Button>
              <Radio.Button value="high">🔴 高风险</Radio.Button>
            </Radio.Group>
          </div>
          <div>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>AI 评分（0-10）</div>
            <Rate count={10} value={aiScore} onChange={setAiScore} />
            <span style={{ marginLeft: 12 }}>{aiScore} 分</span>
          </div>
          <div style={{ marginTop: 32, padding: 12, background: "#f6f8fa", borderRadius: 6, fontSize: 13, color: "#666" }}>
            💡 提示：风险评估会影响老板驾驶舱的高风险供应商预警。AI 评分可作为后续 Agent 自动决策的参考。
          </div>
        </Drawer>
      )}
    </div>
  );
}
