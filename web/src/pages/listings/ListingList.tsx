// B2C 上架商品管理 — 跨平台上架列表（Amazon/Shopify/TikTok/Temu）。
//
// 数据来源：店铺平台 API 同步（V2 接入）；当前版本：手动登记 + 状态流转。
// 注意（gotcha #20）：selling_price 是 decimal 的字符串序列化，表单边界需
// String()/Number() 转换；后端 UpdateListing 是全量 Save，行内状态流转
// 必须展开整行对象后 PUT，否则字段被零值覆盖。
import { useEffect, useState } from "react";
import {
  Card,
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  message,
  Typography,
  Popconfirm,
  Tooltip,
} from "antd";
import { TagsOutlined, PlusOutlined, LinkOutlined } from "@ant-design/icons";
import { b2cApi, type Listing, type Store } from "../../api/b2c";
import dayjs from "dayjs";

const { Title } = Typography;

const STATUS_OPTIONS = [
  { value: "draft", label: "草稿" },
  { value: "active", label: "在售" },
  { value: "paused", label: "暂停" },
  { value: "closed", label: "已关闭" },
];

const STATUS_TAG: Record<string, { color: string; label: string }> = {
  draft: { color: "default", label: "草稿" },
  active: { color: "success", label: "在售" },
  paused: { color: "warning", label: "暂停" },
  closed: { color: "error", label: "已关闭" },
};

const CURRENCY_OPTIONS = ["USD", "CNY", "EUR", "GBP", "JPY", "AUD", "CAD"].map(
  (c) => ({ value: c, label: c })
);

export default function ListingList() {
  const [listings, setListings] = useState<Listing[]>([]);
  const [stores, setStores] = useState<Store[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [filterStore, setFilterStore] = useState<number | undefined>();
  const [filterStatus, setFilterStatus] = useState<string>("");
  const [createOpen, setCreateOpen] = useState(false);
  const [editListing, setEditListing] = useState<Listing | null>(null);
  const [form] = Form.useForm();

  // 店铺映射（store_id → name/平台），私有化部署前 100 家足够
  const storeMap = new Map(stores.map((s) => [s.id, s]));

  const load = async () => {
    setLoading(true);
    try {
      const [listResp, storeResp] = await Promise.all([
        b2cApi.listListings({
          page,
          page_size: 15,
          store_id: filterStore,
          status: filterStatus || undefined,
        }),
        b2cApi.listStores({ page: 1, page_size: 100 }),
      ]);
      setListings(listResp.list || []);
      setTotal(listResp.total || 0);
      setStores(storeResp.list || []);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [page, filterStore, filterStatus]);

  // 创建（gotcha #20：InputNumber 的 number → 后端要字符串）
  const onCreate = async () => {
    let raw: Record<string, unknown>;
    try {
      raw = await form.validateFields();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown };
      if (err.errorFields) return;
      throw e;
    }
    try {
      await b2cApi.createListing({
        ...(raw as Partial<Listing>),
        selling_price: String(raw.selling_price ?? "0"),
      });
      message.success("上架商品已登记");
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  // 编辑（gotcha #20：decimal 字符串 → InputNumber number 回填）
  const onUpdate = async () => {
    let raw: Record<string, unknown>;
    try {
      raw = await form.validateFields();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown };
      if (err.errorFields) return;
      throw e;
    }
    if (!editListing) return;
    try {
      await b2cApi.updateListing(editListing.id, {
        ...(raw as Partial<Listing>),
        selling_price: String(raw.selling_price ?? "0"),
      });
      message.success("已更新");
      setEditListing(null);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  // 行内状态流转：后端全量 Save，必须展开整行（含 store_id/platform_sku/title 等）
  const onStatusChange = async (record: Listing, status: string) => {
    try {
      await b2cApi.updateListing(record.id, { ...record, status });
      message.success("状态已流转");
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const onDelete = async (id: number) => {
    try {
      await b2cApi.deleteListing(id);
      message.success("已删除");
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const storeOptions = stores.map((s) => ({
    value: s.id,
    label: `${s.name}（${s.platform?.toUpperCase() || "?"}）`,
  }));

  const columns = [
    {
      title: "平台 SKU",
      dataIndex: "platform_sku",
      width: 150,
      render: (sku: string, r: Listing) => (
        <Space direction="vertical" size={0}>
          <span>{sku}</span>
          {r.platform_asin ? (
            <span style={{ fontSize: 12, color: "#999" }}>{r.platform_asin}</span>
          ) : null}
        </Space>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      ellipsis: { showTitle: false },
      render: (t: string, r: Listing) => (
        <Tooltip title={t}>
          {r.listing_url ? (
            <a href={r.listing_url} target="_blank" rel="noreferrer">
              <LinkOutlined /> {t}
            </a>
          ) : (
            t
          )}
        </Tooltip>
      ),
    },
    {
      title: "店铺",
      dataIndex: "store_id",
      width: 160,
      render: (id: number) => {
        const s = storeMap.get(id);
        return s ? `${s.name}` : `#${id}`;
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 120,
      render: (s: string, r: Listing) => (
        <Select
            size="small"
            value={s}
            options={STATUS_OPTIONS}
            style={{ width: 96 }}
            onChange={(v) => onStatusChange(r, v as string)}
            popupMatchSelectWidth={false}
          >
            {STATUS_OPTIONS.map((o) => (
              <Select.Option key={o.value} value={o.value}>
                <Tag color={STATUS_TAG[o.value]?.color} style={{ marginRight: 0 }}>
                  {o.label}
                </Tag>
              </Select.Option>
            ))}
          </Select>
        ),
    },
    {
      title: "售价",
      dataIndex: "selling_price",
      width: 110,
      render: (p: string | number, r: Listing) => `${p} ${r.currency}`,
    },
    {
      title: "库存",
      dataIndex: "stock",
      width: 80,
      render: (v?: number) =>
        v === null || v === undefined ? "-" : v <= 10 ? <Tag color="red">{v}</Tag> : v,
    },
    {
      title: "发布时间",
      dataIndex: "published_at",
      width: 110,
      render: (t?: string) => (t ? dayjs(t).format("YY-MM-DD") : "-"),
    },
    {
      title: "操作",
      width: 140,
      render: (_: unknown, r: Listing) => (
        <Space>
          <Button
            size="small"
            onClick={() => {
              setEditListing(r);
              form.setFieldsValue({
                store_id: r.store_id,
                platform_sku: r.platform_sku,
                platform_asin: r.platform_asin,
                title: r.title,
                status: r.status,
                selling_price: Number(r.selling_price),
                currency: r.currency,
                stock: r.stock,
                listing_url: r.listing_url,
              });
            }}
          >
            编辑
          </Button>
          <Popconfirm title="确定删除该上架记录？" onConfirm={() => onDelete(r.id)}>
            <Button size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const modalForm = (
    <Form form={form} layout="vertical">
      <Form.Item name="store_id" label="所属店铺" rules={[{ required: true, message: "请选择店铺" }]}>
        <Select options={storeOptions} placeholder="选择店铺" showSearch optionFilterProp="label" />
      </Form.Item>
      <Form.Item
        name="platform_sku"
        label="平台 SKU"
        rules={[{ required: true, message: "请输入 SKU" }]}
      >
        <Input placeholder="例如：TM-BOTTLE-500ML" maxLength={100} />
      </Form.Item>
      <Form.Item name="platform_asin" label="ASIN（Amazon）">
        <Input placeholder="B0XXXXXXXX" maxLength={20} />
      </Form.Item>
      <Form.Item name="title" label="标题" rules={[{ required: true, message: "请输入标题" }]}>
        <Input placeholder="Listing 标题" maxLength={500} />
      </Form.Item>
      <Space size="middle" style={{ display: "flex" }}>
        <Form.Item name="selling_price" label="售价" initialValue={0} style={{ width: 160 }}>
          <InputNumber min={0} precision={2} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="currency" label="币种" initialValue="USD" style={{ width: 120 }}>
          <Select options={CURRENCY_OPTIONS} />
        </Form.Item>
        <Form.Item name="stock" label="库存" style={{ width: 120 }}>
          <InputNumber min={0} precision={0} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Form.Item name="status" label="状态" initialValue="draft">
        <Select options={STATUS_OPTIONS} />
      </Form.Item>
      <Form.Item name="listing_url" label="Listing 链接">
        <Input placeholder="https://www.amazon.com/dp/B0XXX" maxLength={2000} />
      </Form.Item>
    </Form>
  );

  return (
    <div>
      <Title level={4}>
        <TagsOutlined /> 上架商品（B2C）
      </Title>
      <Card
        extra={
          <Space>
            <Select
              allowClear
              placeholder="按店铺筛选"
              style={{ width: 180 }}
              value={filterStore}
              onChange={(v) => {
                setPage(1);
                setFilterStore(v);
              }}
              options={storeOptions}
              showSearch
              optionFilterProp="label"
            />
            <Select
              allowClear
              placeholder="按状态筛选"
              style={{ width: 130 }}
              value={filterStatus || undefined}
              onChange={(v) => {
                setPage(1);
                setFilterStatus(v || "");
              }}
              options={STATUS_OPTIONS}
            />
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                form.resetFields();
                setCreateOpen(true);
              }}
            >
              登记上架
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={listings}
          loading={loading}
          pagination={{
            current: page,
            total,
            pageSize: 15,
            onChange: setPage,
            showTotal: (t) => `共 ${t} 条`,
          }}
        />
      </Card>

      <Modal
        title="登记上架商品"
        open={createOpen}
        onOk={onCreate}
        onCancel={() => setCreateOpen(false)}
        destroyOnClose
      >
        {modalForm}
      </Modal>

      <Modal
        title="编辑上架商品"
        open={!!editListing}
        onOk={onUpdate}
        onCancel={() => {
          setEditListing(null);
          form.resetFields();
        }}
        destroyOnClose
      >
        {modalForm}
      </Modal>
    </div>
  );
}
