// 工作台（Dashboard）— 员工首页，快捷入口 + 今日概览占位。
// 后续接真实 API。

import { Card, Col, Row, Statistic, Typography } from "antd";
import { ShoppingOutlined, ShopOutlined, RobotOutlined, BellOutlined } from "@ant-design/icons";

const { Title } = Typography;

export default function DashboardPage() {
  return (
    <div>
      <Title level={4}>工作台</Title>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="商品池" value={"—"} prefix={<ShoppingOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="供应商" value={"—"} prefix={<ShopOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="AI 分析次数" value={"—"} prefix={<RobotOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="待办提醒" value={"—"} prefix={<BellOutlined />} />
          </Card>
        </Col>
      </Row>
      <Card style={{ marginTop: 24 }} title="产品路线">
        <ul style={{ paddingLeft: 20, color: "#666" }}>
          <li>✅ Week 1 D1-D2: 项目骨架 + 用户认证 + JWT</li>
          <li>✅ Week 1 D3-D4: React 前端脚手架（当前）</li>
          <li>⏳ Week 1 D5-D7: go:embed 前端 + 首次启动向导</li>
          <li>⏳ Week 2: Chrome 插件 + 商品中心</li>
          <li>⏳ Week 3: 供应商 + 行为采集</li>
          <li>⏳ Week 4: AI 网关 + 客户模块</li>
          <li>⏳ Week 5-6: Agent 体系（选品/采购/销售/上架）</li>
          <li>⏳ Week 7: 老板驾驶舱</li>
          <li>⏳ Week 8: RAG 知识库 + 打包发布</li>
        </ul>
      </Card>
    </div>
  );
}
