// 商品中心 — 列表页。
//
// 功能：搜索 / 来源筛选 / 分类筛选 / 新增 / 编辑 / 删除 / 详情抽屉。
// 数据来源：Chrome 插件采集 + 手动录入。

import { useState, useEffect, useCallback } from "react";
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  InputNumber, Image, Popconfirm, Drawer, Descriptions, message,
  type TableColumnsType,
} from "antd";
import {
  PlusOutlined, SearchOutlined, ReloadOutlined,
  EditOutlined, DeleteOutlined, EyeOutlined,
} from "@ant-design/icons";
import { productApi, type ProductListQuery, type CreateProductInput } from "../../api/product";
import { ApiError } from "../../api/client";
import type { Product, DataSource } from "../../types";

// 来源 → 标签颜色
const SOURCE_COLORS: Record<string, string> = {
  "1688": "orange",
  alibaba: "blue",
  amazon: "gold",
  tiktok: "magenta",
  temu: "red",
  factory: "green",
  manual: "default",
};

// 场景标签
const SCENARIO_TAGS: Record<string, { color: string; label: string }> = {
  b2b: { color: "geekblue", label: "外贸 B2B" },
  b2c: { color: "purple", label: "跨境 B2C" },
};

export default function ProductList() {
  const [data, setData] = useState<Product[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<ProductListQuery>({ page: 1, page_size: 10 });
  const [categories, setCategories] = useState<string[]>([]);

  // 弹窗状态
  const [editModal, setEditModal] = useState<{ open: boolean; product?: Product }>({ open: false });
  const [detailDrawer, setDetailDrawer] = useState<{ open: boolean; product?: Product }>({ open: false });
  const [form] = Form.useForm();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await productApi.list(query);
      setData(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error((e as ApiError).message || "加载失败");
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => { fetchData(); }, [fetchData]);

  useEffect(() => {
    productApi.categories().then(setCategories).catch(() => {});
  }, []);

  // 搜索
  const onSearch = (value: string) => {
    setQuery((q) => ({ ...q, keyword: value, page: 1 }));
  };

  // 新增
  const onAdd = () => {
    form.resetFields();
    form.setFieldsValue({
      purchase_currency: "CNY",
      scenarios: ["b2b", "b2c"],
    });
    setEditModal({ open: true });
  };

  // 编辑
  const onEdit = (product: Product) => {
    let scenArr: string[] = ["b2b", "b2c"];
    try { scenArr = JSON.parse(product.scenarios || '["b2b","b2c"]'); } catch {}
    form.setFieldsValue({
      name: product.name,
      category: product.category,
      description: product.description,
      purchase_price: product.purchase_price ? String(product.purchase_price) : "",
      purchase_currency: product.purchase_currency || "CNY",
      source_url: product.source_url,
      weight_kg: product.weight_kg ? String(product.weight_kg) : "",
      package_spec: product.package_spec,
      scenarios: scenArr,
    });
    setEditModal({ open: true, product });
  };

  // 保存（新增/编辑）
  const onSave = async () => {
    try {
      const raw = await form.validateFields();
      // scenarios 数组 → JSON 字符串（后端要求）
      const values: CreateProductInput = {
        ...raw,
        scenarios: Array.isArray(raw.scenarios) ? JSON.stringify(raw.scenarios) : raw.scenarios,
        purchase_price: raw.purchase_price != null ? String(raw.purchase_price) : undefined,
        weight_kg: raw.weight_kg != null ? String(raw.weight_kg) : undefined,
      };
      if (editModal.product) {
        await productApi.update(editModal.product.id, values);
        message.success("已更新");
      } else {
        await productApi.create(values);
        message.success("已创建");
      }
      setEditModal({ open: false });
      fetchData();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; message?: string };
      if (err.errorFields) return; // 表单校验失败，antd 自行提示
      message.error(err.message || "保存失败");
    }
  };

  // 删除
  const onDelete = async (id: number) => {
    try {
      await productApi.delete(id);
      message.success("已删除");
      fetchData();
    } catch (e) {
      message.error((e as ApiError).message || "删除失败");
    }
  };

  // 解析图片
  const parseImages = (p: Product): string[] => {
    if (!p.image_urls) return [];
    try { return JSON.parse(p.image_urls); } catch { return []; }
  };

  // 解析场景
  const parseScenarios = (p: Product): string[] => {
    if (!p.scenarios) return [];
    try { return JSON.parse(p.scenarios); } catch { return []; }
  };

  const columns: TableColumnsType<Product> = [
    {
      title: "图片",
      dataIndex: "image_urls",
      width: 70,
      render: (_, r) => {
        const imgs = parseImages(r);
        return imgs[0]
          ? <Image src={imgs[0]} width={44} height={44} style={{ objectFit: "cover", borderRadius: 6 }} />
          : <div style={{ width: 44, height: 44, background: "#f5f5f5", borderRadius: 6, display: "flex", alignItems: "center", justifyContent: "center", color: "#ccc", fontSize: 11 }}>无图</div>;
      },
    },
    {
      title: "商品名称",
      dataIndex: "name",
      ellipsis: true,
      render: (name, r) => (
        <div>
          <div style={{ fontWeight: 500 }}>{name}</div>
          {r.category && <div style={{ fontSize: 12, color: "#999" }}>{r.category}</div>}
        </div>
      ),
    },
    {
      title: "采购价",
      dataIndex: "purchase_price",
      width: 110,
      sorter: true,
      render: (price, r) => price ? `¥${price} ${r.purchase_currency || ""}` : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "来源",
      dataIndex: "source",
      width: 90,
      filters: [
        { text: "1688", value: "1688" },
        { text: "阿里巴巴", value: "alibaba" },
        { text: "亚马逊", value: "amazon" },
        { text: "手动录入", value: "manual" },
      ],
      render: (src: DataSource) => <Tag color={SOURCE_COLORS[src] || "default"}>{src}</Tag>,
    },
    {
      title: "场景",
      dataIndex: "scenarios",
      width: 130,
      render: (_, r) => {
        const sc = parseScenarios(r);
        return sc.map((s) => {
          const t = SCENARIO_TAGS[s];
          return t ? <Tag key={s} color={t.color}>{t.label}</Tag> : null;
        });
      },
    },
    {
      title: "AI 评分",
      dataIndex: "ai_score",
      width: 80,
      sorter: true,
      render: (s) => s ? <Tag color={Number(s) >= 8 ? "green" : Number(s) >= 5 ? "orange" : "red"}>{s}</Tag> : <span style={{ color: "#ccc" }}>—</span>,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      width: 150,
      sorter: true,
      render: (t: string) => t ? new Date(t).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }) : "",
    },
    {
      title: "操作",
      width: 140,
      render: (_, r) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => setDetailDrawer({ open: true, product: r })} />
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => onEdit(r)} />
          <Popconfirm title="确定删除？" onConfirm={() => onDelete(r.id)} okText="删除" cancelText="取消" okButtonProps={{ danger: true }}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 工具栏 */}
      <div style={{ marginBottom: 16, display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center" }}>
        <Input.Search
          placeholder="搜索商品名称"
          allowClear
          onSearch={onSearch}
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
            { label: "亚马逊", value: "amazon" },
            { label: "手动录入", value: "manual" },
          ]}
        />
        {categories.length > 0 && (
          <Select
            placeholder="分类筛选"
            allowClear
            style={{ width: 150 }}
            onChange={(v) => setQuery((q) => ({ ...q, category: v, page: 1 }))}
            options={categories.map((c) => ({ label: c, value: c }))}
          />
        )}
        <Button icon={<ReloadOutlined />} onClick={fetchData}>刷新</Button>
        <div style={{ flex: 1 }} />
        <Button type="primary" icon={<PlusOutlined />} onClick={onAdd}>新增商品</Button>
      </div>

      {/* 表格 */}
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
          showTotal: (t) => `共 ${t} 条`,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
        onChange={(_pg, _filters, sorter: any) => {
          if (sorter?.field) {
            setQuery((q) => ({ ...q, sort_by: sorter.field, order: sorter.order === "ascend" ? "asc" : "desc" }));
          }
        }}
      />

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editModal.product ? "编辑商品" : "新增商品"}
        open={editModal.open}
        onOk={onSave}
        onCancel={() => setEditModal({ open: false })}
        okText="保存"
        cancelText="取消"
        width={560}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="商品名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="如：硅胶手机壳定制" />
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Input placeholder="如：手机配件" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="商品描述（选填）" />
          </Form.Item>
          <Space style={{ display: "flex" }} size="middle">
            <Form.Item name="purchase_price" label="采购价" style={{ flex: 1 }}>
              <InputNumber style={{ width: "100%" }} placeholder="0.00" min={0} precision={2} />
            </Form.Item>
            <Form.Item name="purchase_currency" label="币种" style={{ width: 100 }}>
              <Select options={[{ label: "CNY ¥", value: "CNY" }, { label: "USD $", value: "USD" }]} />
            </Form.Item>
          </Space>
          <Form.Item name="weight_kg" label="重量 (kg)">
            <InputNumber style={{ width: "100%" }} placeholder="0.000" min={0} precision={3} />
          </Form.Item>
          <Form.Item name="package_spec" label="包装规格">
            <Input placeholder="如：30×20×5cm" />
          </Form.Item>
          <Form.Item name="source_url" label="来源链接">
            <Input placeholder="https://..." />
          </Form.Item>
          <Form.Item name="scenarios" label="适用场景">
            <Select
              mode="multiple"
              placeholder="选择场景"
              options={[
                { label: "外贸 B2B", value: "b2b" },
                { label: "跨境 B2C", value: "b2c" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情抽屉 */}
      <Drawer
        title="商品详情"
        open={detailDrawer.open}
        onClose={() => setDetailDrawer({ open: false })}
        width={520}
      >
        {detailDrawer.product && (
          <ProductDetail product={detailDrawer.product} images={parseImages(detailDrawer.product)} scenarios={parseScenarios(detailDrawer.product)} />
        )}
      </Drawer>
    </div>
  );
}

// 商品详情组件
function ProductDetail({ product, images, scenarios }: { product: Product; images: string[]; scenarios: string[] }) {
  return (
    <div>
      {images.length > 0 && (
        <Image.PreviewGroup>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
            {images.map((src, i) => (
              <Image key={i} src={src} width={80} height={80} style={{ objectFit: "cover", borderRadius: 6 }} />
            ))}
          </div>
        </Image.PreviewGroup>
      )}
      <Descriptions column={1} bordered size="small">
        <Descriptions.Item label="名称">{product.name}</Descriptions.Item>
        {product.category && <Descriptions.Item label="分类">{product.category}</Descriptions.Item>}
        {product.description && <Descriptions.Item label="描述">{product.description}</Descriptions.Item>}
        {product.purchase_price && (
          <Descriptions.Item label="采购价">¥{product.purchase_price} {product.purchase_currency}</Descriptions.Item>
        )}
        {product.weight_kg && <Descriptions.Item label="重量">{product.weight_kg} kg</Descriptions.Item>}
        {product.package_spec && <Descriptions.Item label="包装">{product.package_spec}</Descriptions.Item>}
        <Descriptions.Item label="来源"><Tag>{product.source}</Tag> {product.source_id && <span style={{ color: "#999" }}>{product.source_id}</span>}</Descriptions.Item>
        {product.source_url && <Descriptions.Item label="链接"><a href={product.source_url} target="_blank" rel="noreferrer">{product.source_url}</a></Descriptions.Item>}
        {scenarios.length > 0 && (
          <Descriptions.Item label="场景">
            {scenarios.map((s) => { const t = SCENARIO_TAGS[s]; return t ? <Tag key={s} color={t.color}>{t.label}</Tag> : null; })}
          </Descriptions.Item>
        )}
        <Descriptions.Item label="创建时间">{new Date(product.created_at).toLocaleString("zh-CN")}</Descriptions.Item>
      </Descriptions>
    </div>
  );
}
