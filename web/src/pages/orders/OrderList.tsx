// B2C 订单管理 — 跨平台订单聚合。
//
// 数据来源：店铺平台 API 同步（V2 接入 Amazon SP-API / Shopify Admin API）。
// 当前版本：手动补录 + 状态流转。
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
  Statistic,
  Row,
  Col,
  message,
  Typography,
} from "antd";
import {
  ShoppingOutlined,
  DollarOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
} from "@ant-design/icons";
import { b2cApi, type Order, type OrderOverview } from "../../api/b2c";
import dayjs from "dayjs";

const { Title, Text } = Typography;

const STATUS_COLOR: Record<string, string> = {
  pending: "default",
  paid: "processing",
  shipped: "warning",
  delivered: "success",
  cancelled: "error",
  refunded: "error",
};

const STATUS_LABEL: Record<string, string> = {
  pending: "待支付",
  paid: "已支付",
  shipped: "已发货",
  delivered: "已送达",
  cancelled: "已取消",
  refunded: "已退款",
};

export default function OrderList() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [overview, setOverview] = useState<OrderOverview | null>(null);
  const [filterStatus, setFilterStatus] = useState<string>("");
  const [createOpen, setCreateOpen] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const [listResp, ov] = await Promise.all([
        b2cApi.listOrders({
          page,
          page_size: 15,
          status: filterStatus || undefined,
        }),
        b2cApi.orderOverview(),
      ]);
      setOrders(listResp.list || []);
      setTotal(listResp.total || 0);
      setOverview(ov);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, [page, filterStatus]);

  const onCreate = async () => {
    const v = await form.validateFields();
    try {
      await b2cApi.createOrder({
        ...v,
        ordered_at: new Date().toISOString(),
      });
      message.success("订单创建成功");
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const onUpdateStatus = async (id: number, status: string) => {
    try {
      await b2cApi.updateOrderStatus(id, status);
      message.success("状态已更新");
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const columns = [
    { title: "订单号", dataIndex: "platform_order_no", width: 180 },
    {
      title: "店铺",
      dataIndex: "store_name",
      width: 120,
      render: (n: string, r: Order) => (
        <Space direction="vertical" size={0}>
          <Text strong>{n || "-"}</Text>
          <Tag>{r.platform}</Tag>
        </Space>
      ),
    },
    {
      title: "买家",
      width: 150,
      render: (_: any, r: Order) => (
        <Space direction="vertical" size={0}>
          <Text>{r.buyer_name || "-"}</Text>
          <Text type="secondary">{r.buyer_country || "-"}</Text>
        </Space>
      ),
    },
    {
      title: "金额",
      dataIndex: "amount",
      width: 110,
      render: (a: number, r: Order) => (
        <Text strong>
          {r.currency} {a?.toFixed(2)}
        </Text>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 100,
      render: (s: string) => (
        <Tag color={STATUS_COLOR[s]}>{STATUS_LABEL[s] || s}</Tag>
      ),
    },
    {
      title: "下单时间",
      dataIndex: "ordered_at",
      width: 140,
      render: (t: string) => dayjs(t).format("MM-DD HH:mm"),
    },
    {
      title: "操作",
      width: 140,
      render: (_: any, r: Order) => (
        <Select
          size="small"
          placeholder="更新状态"
          style={{ width: 120 }}
          onChange={(v) => onUpdateStatus(r.id, v)}
          options={[
            { value: "paid", label: "标记已支付" },
            { value: "shipped", label: "标记已发货" },
            { value: "delivered", label: "标记已送达" },
            { value: "cancelled", label: "取消订单" },
            { value: "refunded", label: "退款" },
          ]}
        />
      ),
    },
  ];

  return (
    <div>
      <Title level={4}>
        <ShoppingOutlined /> 订单管理（B2C）
      </Title>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总订单"
              value={overview?.total_orders || 0}
              prefix={<ShoppingOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总收入"
              value={overview?.total_revenue || 0}
              precision={2}
              prefix={<DollarOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待支付"
              value={overview?.pending_count || 0}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已送达"
              value={overview?.delivered_count || 0}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        extra={
          <Space>
            <Select
              placeholder="状态筛选"
              allowClear
              style={{ width: 120 }}
              onChange={(v) => {
                setFilterStatus(v || "");
                setPage(1);
              }}
              options={Object.entries(STATUS_LABEL).map(([k, v]) => ({
                value: k,
                label: v,
              }))}
            />
            <Button type="primary" onClick={() => setCreateOpen(true)}>
              手动补录
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={orders}
          loading={loading}
          pagination={{
            current: page,
            total,
            pageSize: 15,
            onChange: setPage,
          }}
        />
      </Card>

      <Modal
        title="手动补录订单"
        open={createOpen}
        onOk={onCreate}
        onCancel={() => setCreateOpen(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="platform_order_no"
            label="平台订单号"
            rules={[{ required: true }]}
          >
            <Input placeholder="AMZ-12345" />
          </Form.Item>
          <Form.Item name="store_id" label="店铺 ID" rules={[{ required: true }]}>
            <InputNumber style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="amount" label="金额" rules={[{ required: true }]}>
            <InputNumber style={{ width: "100%" }} precision={2} />
          </Form.Item>
          <Form.Item name="currency" label="币种" initialValue="USD">
            <Select
              options={[
                { value: "USD", label: "USD" },
                { value: "EUR", label: "EUR" },
                { value: "GBP", label: "GBP" },
                { value: "JPY", label: "JPY" },
              ]}
            />
          </Form.Item>
          <Form.Item name="buyer_name" label="买家姓名">
            <Input />
          </Form.Item>
          <Form.Item name="buyer_country" label="买家国家">
            <Input placeholder="US" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

