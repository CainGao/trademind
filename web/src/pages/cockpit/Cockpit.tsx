// 老板驾驶舱 — 全量数据看板。
//
// 核心：把 Chrome 插件采集的"员工行为资产"可视化成老板看得懂的指标。
// 包含：四指标卡 + 14 天行为趋势折线图 + Top 搜索词 + 供应商风险分布。
//
// 老板登录后默认进这里，一眼看清团队在忙什么、在看什么商品、风险在哪。

import { useEffect, useState, useRef } from "react";
import { Card, Row, Col, Statistic, Spin, Empty, Tag, List, Typography } from "antd";
import {
  EyeOutlined, SearchOutlined, ShoppingOutlined, TeamOutlined,
  RiseOutlined, AlertOutlined, TrophyOutlined,
} from "@ant-design/icons";
import * as echarts from "echarts";
import { statsApi, type Dashboard } from "../../api/stats";
import { ApiError } from "../../api/client";

const { Text } = Typography;

export default function Cockpit() {
  const [data, setData] = useState<Dashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const trendRef = useRef<HTMLDivElement>(null);
  const riskRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    statsApi.dashboard(14)
      .then(setData)
      .catch((e: ApiError) => console.error(e.message))
      .finally(() => setLoading(false));
  }, []);

  // 趋势折线图
  useEffect(() => {
    if (!data?.daily_trend?.length || !trendRef.current) return;
    const chart = echarts.init(trendRef.current);
    const trend = data.daily_trend;
    chart.setOption({
      tooltip: { trigger: "axis" },
      legend: { data: ["浏览", "搜索", "采集", "收藏"], bottom: 0 },
      grid: { left: 40, right: 20, top: 20, bottom: 40 },
      xAxis: { type: "category", data: trend.map((t) => t.date.slice(5)) },
      yAxis: { type: "value" },
      series: [
        { name: "浏览", type: "line", smooth: true, data: trend.map((t) => Number(t.browse) || 0), itemStyle: { color: "#1677ff" } },
        { name: "搜索", type: "line", smooth: true, data: trend.map((t) => Number(t.search) || 0), itemStyle: { color: "#52c41a" } },
        { name: "采集", type: "line", smooth: true, data: trend.map((t) => Number(t.collect) || 0), itemStyle: { color: "#faad14" } },
        { name: "收藏", type: "line", smooth: true, data: trend.map((t) => Number(t.favorite) || 0), itemStyle: { color: "#eb2f96" } },
      ],
    });
    const onResize = () => chart.resize();
    window.addEventListener("resize", onResize);
    return () => { window.removeEventListener("resize", onResize); chart.dispose(); };
  }, [data]);

  // 风险分布饼图
  useEffect(() => {
    if (!data?.supplier_overview || !riskRef.current) return;
    const ov = data.supplier_overview;
    if (ov.total === 0) return;
    const chart = echarts.init(riskRef.current);
    chart.setOption({
      tooltip: { trigger: "item", formatter: "{b}: {c} ({d}%)" },
      legend: { bottom: 0 },
      series: [{
        type: "pie",
        radius: ["45%", "70%"],
        center: ["50%", "45%"],
        label: { show: false },
        data: [
          { value: ov.risk_high, name: "高风险", itemStyle: { color: "#ff4d4f" } },
          { value: ov.risk_medium, name: "中风险", itemStyle: { color: "#faad14" } },
          { value: ov.risk_low, name: "低风险", itemStyle: { color: "#52c41a" } },
        ],
      }],
    });
    const onResize = () => chart.resize();
    window.addEventListener("resize", onResize);
    return () => { window.removeEventListener("resize", onResize); chart.dispose(); };
  }, [data]);

  if (loading) return <div style={{ textAlign: "center", padding: 80 }}><Spin size="large" /></div>;
  if (!data) return <Empty description="暂无数据" />;

  const bo = data.behavior_overview;

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          📊 老板驾驶舱
        </Typography.Title>
        <Text type="secondary">近 14 天团队行为资产概览 · 每日自动更新</Text>
      </div>

      {/* 顶部四指标卡 */}
      <Row gutter={[16, 16]}>
        <Col xs={12} sm={12} md={6}>
          <Card>
            <Statistic
              title="商品总数"
              value={data.product_total}
              prefix={<ShoppingOutlined style={{ color: "#1677ff" }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card>
            <Statistic
              title="供应商"
              value={data.supplier_overview.total}
              prefix={<TeamOutlined style={{ color: "#52c41a" }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card>
            <Statistic
              title="近 7 天行为"
              value={bo.last_7_days}
              prefix={<RiseOutlined style={{ color: "#faad14" }} />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={6}>
          <Card>
            <Statistic
              title="高风险供应商"
              value={data.supplier_overview.risk_high}
              valueStyle={{ color: data.supplier_overview.risk_high > 0 ? "#ff4d4f" : undefined }}
              prefix={<AlertOutlined style={{ color: "#ff4d4f" }} />}
            />
          </Card>
        </Col>
      </Row>

      {/* 行为分类统计 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={12} md={8}>
          <Card size="small">
            <Statistic title="浏览记录" value={bo.browse_cnt} prefix={<EyeOutlined />} />
          </Card>
        </Col>
        <Col xs={12} md={8}>
          <Card size="small">
            <Statistic title="搜索关键词" value={bo.search_cnt} prefix={<SearchOutlined />} />
          </Card>
        </Col>
        <Col xs={12} md={8}>
          <Card size="small">
            <Statistic title="采集入库" value={bo.collect_cnt} prefix={<ShoppingOutlined />} />
          </Card>
        </Col>
      </Row>

      {/* 趋势图 + 风险分布 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={16}>
          <Card title="📈 行为趋势（14 天）" size="small">
            <div ref={trendRef} style={{ height: 300 }} />
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="⚠️ 供应商风险分布" size="small">
            <div ref={riskRef} style={{ height: 300 }} />
          </Card>
        </Col>
      </Row>

      {/* Top 搜索词 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="🔍 热门搜索词 Top 10" size="small">
            {data.top_keywords?.length ? (
              <List
                size="small"
                dataSource={data.top_keywords}
                renderItem={(item, idx) => (
                  <List.Item>
                    <span style={{ width: 24, fontWeight: 700, color: idx < 3 ? "#faad14" : "#999" }}>
                      {idx < 3 ? <TrophyOutlined /> : null} {idx + 1}
                    </span>
                    <span style={{ flex: 1 }}>{item.keyword}</span>
                    <Tag>{item.cnt} 次</Tag>
                  </List.Item>
                )}
              />
            ) : <Empty description="暂无搜索记录" image={Empty.PRESENTED_IMAGE_SIMPLE} />}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="📊 行为类型分布" size="small">
            {data.stats_by_type?.length ? (
              <List
                size="small"
                dataSource={data.stats_by_type}
                renderItem={(item) => (
                  <List.Item>
                    <Tag color={
                      item.event_type === "browse" ? "blue" :
                      item.event_type === "search" ? "green" :
                      item.event_type === "collect" ? "orange" :
                      item.event_type === "favorite" ? "magenta" : "default"
                    }>
                      {item.event_type}
                    </Tag>
                    <span style={{ flex: 1 }} />
                    <strong>{item.cnt}</strong>
                  </List.Item>
                )}
              />
            ) : <Empty description="暂无数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />}
          </Card>
        </Col>
      </Row>

      <div style={{ marginTop: 24, padding: 12, background: "#f6f8fa", borderRadius: 6, fontSize: 13, color: "#999", textAlign: "center" }}>
        💡 数据来自 Chrome 插件采集的团队行为资产。员工浏览/搜索/采集的商品都会沉淀为企业知识库。
      </div>
    </div>
  );
}
