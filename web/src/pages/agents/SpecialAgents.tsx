// 专用 Agent — B2B（邮件/询盘/报价）+ B2C（上架/评论）。
//
// 每个 Agent 一个 Tab：
//   1. 邮件分析：粘贴邮件 → AI 提取意图 + 生成回复
//   2. 询盘分析：输入 inquiry_id → AI 给出跟进策略
//   3. 报价建议：inquiry_id + product_id → AI 报价区间
//   4. 上架优化：product_id + platform → SEO 标题 + 五点描述
//   5. 评论分析：粘贴评论 → 情感分析 + 改进建议
import { useState } from "react";
import {
  Card,
  Tabs,
  Form,
  Input,
  InputNumber,
  Button,
  Select,
  message,
  Typography,
  Tag,
  List,
  Statistic,
  Row,
  Col,
  Spin,
  Alert,
  Divider,
  Space,
} from "antd";
import {
  MailOutlined,
  FileSearchOutlined,
  DollarOutlined,
  ShoppingOutlined,
  CommentOutlined,
  CopyOutlined,
} from "@ant-design/icons";
import { agentApi } from "../../api/agent";

const { Text, Paragraph, Title } = Typography;
const { TextArea } = Input;

export default function SpecialAgents() {
  return (
    <div>
      <Title level={4}>
        <RobotOutlined /> 专用 Agent 工作台
      </Title>
      <Paragraph type="secondary">
        每个 Agent 解决一个具体的业务问题。输入数据 → AI
        输出结构化建议，可直接复制使用。
      </Paragraph>

      <Tabs
        items={[
          {
            key: "email",
            label: (
              <span>
                <MailOutlined /> 邮件分析
              </span>
            ),
            children: <EmailAgent />,
          },
          {
            key: "inquiry",
            label: (
              <span>
                <FileSearchOutlined /> 询盘分析
              </span>
            ),
            children: <InquiryAgent />,
          },
          {
            key: "quotation",
            label: (
              <span>
                <DollarOutlined /> 报价建议
              </span>
            ),
            children: <QuotationAgent />,
          },
          {
            key: "listing",
            label: (
              <span>
                <ShoppingOutlined /> 上架优化
              </span>
            ),
            children: <ListingAgent />,
          },
          {
            key: "reviews",
            label: (
              <span>
                <CommentOutlined /> 评论分析
              </span>
            ),
            children: <ReviewsAgent />,
          },
        ]}
      />
    </div>
  );
}

import { RobotOutlined } from "@ant-design/icons";

// ============================================================================
// 邮件分析 Agent
// ============================================================================
function EmailAgent() {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const onRun = async () => {
    const v = await form.validateFields();
    setLoading(true);
    setResult(null);
    try {
      const resp = await agentApi.analyzeEmail(v.subject, v.content);
      setResult(resp.analysis);
      message.success("分析完成");
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="粘贴客户邮件，AI 帮你分析">
        <Form form={form} layout="vertical">
          <Form.Item name="subject" label="邮件主题" rules={[{ required: true }]}>
            <Input placeholder="Inquiry about iPhone cases" />
          </Form.Item>
          <Form.Item name="content" label="邮件正文" rules={[{ required: true }]}>
            <TextArea
              rows={8}
              placeholder="Dear Sir, We are interested in your products..."
            />
          </Form.Item>
          <Button type="primary" loading={loading} onClick={onRun}>
            分析邮件
          </Button>
        </Form>
      </Card>

      {loading && (
        <Card style={{ marginTop: 16 }}>
          <div style={{ textAlign: "center", padding: 40 }}>
            <Spin size="large" tip="AI 正在分析..." />
          </div>
        </Card>
      )}

      {result && (
        <Card
          title="分析结果"
          style={{ marginTop: 16 }}
          extra={
            <Button
              size="small"
              icon={<CopyOutlined />}
              onClick={() => {
                navigator.clipboard.writeText(result.suggested_reply || "");
                message.success("回复模板已复制");
              }}
            >
              复制建议回复
            </Button>
          }
        >
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="意图"
                valueRender={() => <Tag color="blue">{result.intent}</Tag>}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="紧急度"
                valueRender={() => (
                  <Tag
                    color={
                      result.urgency === "high"
                        ? "red"
                        : result.urgency === "medium"
                        ? "orange"
                        : "green"
                    }
                  >
                    {result.urgency}
                  </Tag>
                )}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="情感"
                valueRender={() => <Tag>{result.sentiment}</Tag>}
              />
            </Col>
            <Col span={6}>
              <Statistic title="国家/语言" value={`${result.country || "-"} / ${result.language || "-"}`} />
            </Col>
          </Row>
          <Divider />
          <Paragraph strong>{result.summary}</Paragraph>

          {result.key_points?.length > 0 && (
            <>
              <Text strong>关键信息：</Text>
              <List
                size="small"
                dataSource={result.key_points}
                renderItem={(item: string) => (
                  <List.Item>
                    <Text>{item}</Text>
                  </List.Item>
                )}
              />
            </>
          )}

          {result.products_mentioned?.length > 0 && (
            <div style={{ marginTop: 12 }}>
              <Text strong>提及产品：</Text>
              <Space wrap style={{ marginLeft: 8 }}>
                {result.products_mentioned.map((p: string) => (
                  <Tag color="cyan" key={p}>
                    {p}
                  </Tag>
                ))}
              </Space>
            </div>
          )}

          {result.suggested_reply && (
            <Card
              size="small"
              type="inner"
              title="✉️ 建议回复（可直接复制）"
              style={{ marginTop: 16 }}
            >
              <Paragraph>
                <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>
                  {result.suggested_reply}
                </pre>
              </Paragraph>
            </Card>
          )}
        </Card>
      )}
    </div>
  );
}

// ============================================================================
// 询盘分析 Agent
// ============================================================================
function InquiryAgent() {
  const [inquiryId, setInquiryId] = useState<number>(1);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const onRun = async () => {
    setLoading(true);
    setResult(null);
    try {
      const resp = await agentApi.analyzeInquiry(inquiryId);
      setResult(resp.analysis);
      message.success("分析完成");
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="询盘跟进策略生成">
        <Space>
          <Text>询盘 ID：</Text>
          <InputNumber
            min={1}
            value={inquiryId}
            onChange={(v) => setInquiryId(v || 1)}
            style={{ width: 100 }}
          />
          <Button type="primary" loading={loading} onClick={onRun}>
            分析询盘
          </Button>
        </Space>
      </Card>

      {result && (
        <Card title="跟进策略" style={{ marginTop: 16 }}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="预算水平"
                valueRender={() => <Tag color="gold">{result.budget_level}</Tag>}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="决策阶段"
                valueRender={() => <Tag color="purple">{result.decision_stage}</Tag>}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="成交概率"
                value={result.win_probability ? `${result.win_probability}%` : "-"}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="预估量"
                value={result.estimated_quantity || "-"}
              />
            </Col>
          </Row>
          <Divider />
          <Paragraph strong>{result.summary}</Paragraph>
          <Card size="small" type="inner" title="跟进策略">
            <Paragraph>{result.suggested_strategy}</Paragraph>
          </Card>
          {result.reply_template && (
            <Card
              size="small"
              type="inner"
              title="回复模板"
              style={{ marginTop: 12 }}
              extra={
                <Button
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => {
                    navigator.clipboard.writeText(result.reply_template);
                    message.success("已复制");
                  }}
                >
                  复制
                </Button>
              }
            >
              <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>
                {result.reply_template}
              </pre>
            </Card>
          )}
          {result.risks?.length > 0 && (
            <Alert
              style={{ marginTop: 12 }}
              type="warning"
              showIcon
              message="风险提示"
              description={
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  {result.risks.map((r: string) => (
                    <li key={r}>{r}</li>
                  ))}
                </ul>
              }
            />
          )}
        </Card>
      )}
    </div>
  );
}

// ============================================================================
// 报价建议 Agent
// ============================================================================
function QuotationAgent() {
  const [inquiryId, setInquiryId] = useState(1);
  const [productId, setProductId] = useState(1);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const onRun = async () => {
    setLoading(true);
    setResult(null);
    try {
      const resp = await agentApi.adviseQuotation(inquiryId, productId);
      setResult(resp.advice);
      message.success("报价建议生成完成");
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="AI 报价建议">
        <Space>
          <Text>询盘 ID：</Text>
          <InputNumber min={1} value={inquiryId} onChange={(v) => setInquiryId(v || 1)} style={{ width: 80 }} />
          <Text>商品 ID：</Text>
          <InputNumber min={1} value={productId} onChange={(v) => setProductId(v || 1)} style={{ width: 80 }} />
          <Button type="primary" loading={loading} onClick={onRun}>
            生成报价
          </Button>
        </Space>
      </Card>

      {result && (
        <Card title="报价建议" style={{ marginTop: 16 }}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="推荐报价"
                value={result.recommended_price || "-"}
                prefix="$"
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="价格区间"
                valueRender={() => (
                  <Text>
                    ${result.price_range_low} ~ ${result.price_range_high}
                  </Text>
                )}
              />
            </Col>
            <Col span={6}>
              <Statistic title="建议 MOQ" value={result.moq_advice || "-"} />
            </Col>
            <Col span={6}>
              <Statistic
                title="预估利润率"
                value={result.profit_margin ? `${result.profit_margin}%` : "-"}
              />
            </Col>
          </Row>
          <Divider />
          <Paragraph strong>{result.summary}</Paragraph>
          <Card size="small" type="inner" title="竞争分析">
            <Paragraph>{result.competitive_analysis}</Paragraph>
          </Card>
          <Card size="small" type="inner" title="谈判底线" style={{ marginTop: 12 }}>
            <Paragraph>{result.negotiation_bottom_line}</Paragraph>
          </Card>
          {result.tactics?.length > 0 && (
            <Card size="small" type="inner" title="报价战术" style={{ marginTop: 12 }}>
              <List
                size="small"
                dataSource={result.tactics}
                renderItem={(t: string, i: number) => (
                  <List.Item>
                    {i + 1}. {t}
                  </List.Item>
                )}
              />
            </Card>
          )}
        </Card>
      )}
    </div>
  );
}

// ============================================================================
// 上架优化 Agent
// ============================================================================
function ListingAgent() {
  const [productId, setProductId] = useState(1);
  const [platform, setPlatform] = useState("amazon");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const onRun = async () => {
    setLoading(true);
    setResult(null);
    try {
      const resp = await agentApi.optimizeListing(productId, platform);
      setResult(resp.optimization);
      message.success("上架文案已生成");
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="AI 上架文案生成（SEO 优化）">
        <Space>
          <Text>商品 ID：</Text>
          <InputNumber min={1} value={productId} onChange={(v) => setProductId(v || 1)} style={{ width: 80 }} />
          <Text>平台：</Text>
          <Select
            value={platform}
            onChange={setPlatform}
            style={{ width: 120 }}
            options={[
              { value: "amazon", label: "Amazon" },
              { value: "shopify", label: "Shopify" },
              { value: "tiktok", label: "TikTok Shop" },
              { value: "temu", label: "Temu" },
            ]}
          />
          <Button type="primary" loading={loading} onClick={onRun}>
            生成文案
          </Button>
        </Space>
      </Card>

      {result && (
        <Card
          title="上架文案"
          style={{ marginTop: 16 }}
          extra={
            <Button
              size="small"
              icon={<CopyOutlined />}
              onClick={() => {
                const all = `Title: ${result.title}\n\nBullet Points:\n${(result.bullet_points || []).map((p: string, i: number) => `${i + 1}. ${p}`).join("\n")}\n\nDescription:\n${result.description}`;
                navigator.clipboard.writeText(all);
                message.success("全部文案已复制");
              }}
            >
              复制全部
            </Button>
          }
        >
          <Card size="small" type="inner" title="标题（SEO 优化）">
            <Paragraph copyable>{result.title}</Paragraph>
          </Card>
          {result.bullet_points?.length > 0 && (
            <Card size="small" type="inner" title="五点描述" style={{ marginTop: 12 }}>
              <List
                size="small"
                dataSource={result.bullet_points}
                renderItem={(p: string, i: number) => (
                  <List.Item>
                    <Text>
                      <Text strong>{i + 1}.</Text> {p}
                    </Text>
                  </List.Item>
                )}
              />
            </Card>
          )}
          <Card size="small" type="inner" title="长描述" style={{ marginTop: 12 }}>
            <Paragraph>
              <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>
                {result.description}
              </pre>
            </Paragraph>
          </Card>
          <Row gutter={16} style={{ marginTop: 12 }}>
            <Col span={8}>
              <Statistic
                title="预估月销"
                value={result.estimated_sales || "-"}
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="竞品价格"
                value={result.competitor_price ? `$${result.competitor_price}` : "-"}
              />
            </Col>
            <Col span={8}>
              <Statistic title="推荐类目" value={result.category_suggest || "-"} />
            </Col>
          </Row>
          <div style={{ marginTop: 12 }}>
            <Text strong>搜索关键词：</Text>
            <Space wrap style={{ marginLeft: 8 }}>
              {(result.search_terms || []).map((k: string) => (
                <Tag color="blue" key={k}>
                  {k}
                </Tag>
              ))}
              {(result.backend_keywords || []).map((k: string) => (
                <Tag color="cyan" key={k}>
                  {k}
                </Tag>
              ))}
            </Space>
          </div>
        </Card>
      )}
    </div>
  );
}

// ============================================================================
// 评论分析 Agent
// ============================================================================
function ReviewsAgent() {
  const [reviews, setReviews] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);

  const onRun = async () => {
    if (!reviews.trim()) {
      message.warning("请粘贴评论");
      return;
    }
    setLoading(true);
    setResult(null);
    try {
      const resp = await agentApi.analyzeReviews(reviews);
      setResult(resp.analysis);
      message.success("评论分析完成");
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Card title="批量评论情感分析">
        <Paragraph type="secondary">
          粘贴多条评论，用 <Text code>---</Text> 分隔。AI
          会提取共性问题、给出产品改进建议、生成回复模板。
        </Paragraph>
        <TextArea
          rows={8}
          value={reviews}
          onChange={(e) => setReviews(e.target.value)}
          placeholder={`Great product! Fast delivery.\n---\nQuality is bad, broke after 2 days.\n---\nLove it, will buy again.`}
        />
        <Button
          type="primary"
          loading={loading}
          onClick={onRun}
          style={{ marginTop: 12 }}
        >
          分析评论
        </Button>
      </Card>

      {result && (
        <Card title="评论分析结果" style={{ marginTop: 16 }}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="好评"
                value={result.sentiment_distribution?.positive || 0}
                valueStyle={{ color: "#3f8600" }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="中评"
                value={result.sentiment_distribution?.neutral || 0}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="差评"
                value={result.sentiment_distribution?.negative || 0}
                valueStyle={{ color: "#cf1322" }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="综合评分"
                value={result.overall_score || "-"}
                suffix="/ 10"
              />
            </Col>
          </Row>
          <Divider />
          <Paragraph strong>{result.summary}</Paragraph>

          {result.risk_alerts?.length > 0 && (
            <Alert
              type="error"
              showIcon
              message="风险预警"
              description={
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  {result.risk_alerts.map((r: string) => (
                    <li key={r}>{r}</li>
                  ))}
                </ul>
              }
              style={{ marginBottom: 12 }}
            />
          )}

          <Row gutter={16}>
            <Col span={12}>
              <Card size="small" type="inner" title="👍 好评点">
                <List
                  size="small"
                  dataSource={result.top_praises || []}
                  renderItem={(p: string) => <List.Item>{p}</List.Item>}
                />
              </Card>
            </Col>
            <Col span={12}>
              <Card size="small" type="inner" title="👎 差评问题">
                <List
                  size="small"
                  dataSource={result.top_issues || []}
                  renderItem={(p: string) => (
                    <List.Item>
                      <Text type="danger">{p}</Text>
                    </List.Item>
                  )}
                />
              </Card>
            </Col>
          </Row>

              {result.product_improvements?.length > 0 && (
                <Card
                  size="small"
                  type="inner"
                  title="🔧 产品改进建议"
                  style={{ marginTop: 12 }}
                >
                  <List
                    size="small"
                    dataSource={result.product_improvements}
                    renderItem={(p: string, i: number) => (
                      <List.Item>
                        {i + 1}. {p}
                      </List.Item>
                    )}
                  />
                </Card>
              )}

          {result.reply_template_4_negative && (
            <Card
              size="small"
              type="inner"
              title="✉️ 差评回复模板"
              style={{ marginTop: 12 }}
              extra={
                <Button
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => {
                    navigator.clipboard.writeText(result.reply_template_4_negative);
                    message.success("已复制");
                  }}
                >
                  复制
                </Button>
              }
            >
              <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>
                {result.reply_template_4_negative}
              </pre>
            </Card>
          )}
        </Card>
      )}
    </div>
  );
}
