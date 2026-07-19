// B2C 店铺管理 — 多平台授权（Amazon/Shopify/TikTok/Temu）。
//
// 当前版本：本地登记 + CRUD。OAuth 授权流程在 V2 接入平台 API 时实现。
import { useEffect, useState } from "react";
import {
  Card,
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  message,
  Typography,
  Popconfirm,
} from "antd";
import { ShopOutlined, PlusOutlined } from "@ant-design/icons";
import { b2cApi, type Store, type CreateStoreInput } from "../../api/b2c";
import dayjs from "dayjs";

const { Title } = Typography;

const PLATFORM_COLOR: Record<string, string> = {
  amazon: "#ff9900",
  shopify: "#96bf48",
  tiktok: "#000000",
  temu: "#fb6d14",
};

export default function StoreList() {
  const [stores, setStores] = useState<Store[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [editStore, setEditStore] = useState<Store | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const resp = await b2cApi.listStores({ page, page_size: 20 });
      setStores(resp.list || []);
      setTotal(resp.total || 0);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [page]);

  const onCreate = async () => {
    const v: CreateStoreInput = await form.validateFields();
    try {
      await b2cApi.createStore(v);
      message.success("店铺已添加");
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const onUpdate = async () => {
    const v = await form.validateFields();
    if (!editStore) return;
    try {
      await b2cApi.updateStore(editStore.id, v);
      message.success("已更新");
      setEditStore(null);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const onDelete = async (id: number) => {
    try {
      await b2cApi.deleteStore(id);
      message.success("已删除");
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const columns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "店铺名", dataIndex: "name" },
    {
      title: "平台",
      dataIndex: "platform",
      width: 110,
      render: (p: string) => (
        <Tag color={PLATFORM_COLOR[p] || "default"}>{p?.toUpperCase()}</Tag>
      ),
    },
    { title: "区域", dataIndex: "region", width: 80 },
    { title: "平台店铺 ID", dataIndex: "store_id", width: 140 },
    {
      title: "状态",
      dataIndex: "status",
      width: 100,
      render: (s: string) =>
        s === "active" ? (
          <Tag color="success">已授权</Tag>
        ) : s === "expired" ? (
          <Tag color="warning">已过期</Tag>
        ) : (
          <Tag color="error">已撤销</Tag>
        ),
    },
    {
      title: "最近同步",
      dataIndex: "synced_at",
      width: 140,
      render: (t?: string) => (t ? dayjs(t).format("MM-DD HH:mm") : "-"),
    },
    {
      title: "操作",
      width: 140,
      render: (_: any, r: Store) => (
        <Space>
          <Button
            size="small"
            onClick={() => {
              setEditStore(r);
              form.setFieldsValue(r);
            }}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除？"
            onConfirm={() => onDelete(r.id)}
          >
            <Button size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Title level={4}>
        <ShopOutlined /> 店铺管理（B2C）
      </Title>
      <Card
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              form.resetFields();
              setCreateOpen(true);
            }}
          >
            添加店铺
          </Button>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={stores}
          loading={loading}
          pagination={{
            current: page,
            total,
            pageSize: 20,
            onChange: setPage,
          }}
        />
      </Card>

      <Modal
        title="添加店铺"
        open={createOpen}
        onOk={onCreate}
        onCancel={() => setCreateOpen(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="店铺名" rules={[{ required: true }]}>
            <Input placeholder="例如：Aisen US Store" />
          </Form.Item>
          <Form.Item
            name="platform"
            label="平台"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                { value: "amazon", label: "Amazon" },
                { value: "shopify", label: "Shopify" },
                { value: "tiktok", label: "TikTok Shop" },
                { value: "temu", label: "Temu" },
              ]}
            />
          </Form.Item>
          <Form.Item name="region" label="区域">
            <Input placeholder="us / uk / de / jp" />
          </Form.Item>
          <Form.Item name="store_id" label="平台店铺 ID">
            <Input placeholder="AMZ-A2B3XYZ..." />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select
              options={[
                { value: "active", label: "已授权" },
                { value: "expired", label: "已过期" },
                { value: "revoked", label: "已撤销" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="编辑店铺"
        open={!!editStore}
        onOk={onUpdate}
        onCancel={() => {
          setEditStore(null);
          form.resetFields();
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="店铺名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="platform" label="平台" rules={[{ required: true }]}>
            <Select
              options={[
                { value: "amazon", label: "Amazon" },
                { value: "shopify", label: "Shopify" },
                { value: "tiktok", label: "TikTok Shop" },
                { value: "temu", label: "Temu" },
              ]}
            />
          </Form.Item>
          <Form.Item name="region" label="区域">
            <Input />
          </Form.Item>
          <Form.Item name="store_id" label="平台店铺 ID">
            <Input />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              options={[
                { value: "active", label: "已授权" },
                { value: "expired", label: "已过期" },
                { value: "revoked", label: "已撤销" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
