// RAG 知识库 — 文档上传/粘贴 + 语义检索 + 检索增强对话（Week 8）。
//
// 三大功能（Tab 切换）：
//   1. 文件管理：上传文件(txt/md/csv/docx) / 粘贴文本 → 自动分片+向量化
//   2. 语义检索：输入问题 → 返回最相关的知识片段 + 相似度分数
//   3. 智能问答(RAG)：检索知识库 → AI 基于企业资料回答 + 标注来源
import { useEffect, useState } from "react";
import {
  Card,
  Button,
  List,
  Tag,
  Typography,
  message,
  Tabs,
  Upload,
  Modal,
  Form,
  Input,
  Space,
  Spin,
  Empty,
  Row,
  Col,
  Statistic,
  Popconfirm,
  Alert,
  Divider,
  Tooltip,
} from "antd";
import type { UploadRequestOption } from "rc-upload/lib/interface";
import {
  UploadOutlined,
  FileTextOutlined,
  SearchOutlined,
  RobotOutlined,
  DeleteOutlined,
  ReloadOutlined,
  InboxOutlined,
  CopyOutlined,
  DatabaseOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import {
  knowledgeApi,
  type KnowledgeFile,
  type SearchResult,
  type RAGChatAnswer,
  type KnowledgeStats,
} from "../../api/knowledge";

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

// 文件大小可读化
function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

// 状态 Tag 颜色
function statusTag(status: string) {
  switch (status) {
    case "ready":
      return <Tag icon={<CheckCircleOutlined />} color="success">可检索</Tag>;
    case "processing":
      return <Tag icon={<LoadingOutlined />} color="processing">处理中</Tag>;
    case "failed":
      return <Tag icon={<CloseCircleOutlined />} color="error">失败</Tag>;
    default:
      return <Tag>{status}</Tag>;
  }
}

export default function Knowledge() {
  const [activeTab, setActiveTab] = useState("files");
  const [files, setFiles] = useState<KnowledgeFile[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<KnowledgeStats | null>(null);
  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasting, setPasting] = useState(false);
  const [reembedding, setReembedding] = useState<number | null>(null);
  const [form] = Form.useForm();

  // 检索状态
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);

  // RAG 对话状态
  const [chatQuery, setChatQuery] = useState("");
  const [chatAnswer, setChatAnswer] = useState<RAGChatAnswer | null>(null);
  const [chatting, setChatting] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const [listResp, statsResp] = await Promise.all([
        knowledgeApi.list({ page, page_size: 20 }),
        knowledgeApi.stats(),
      ]);
      setFiles(listResp.list || []);
      setTotal(listResp.total);
      setStats(statsResp);
    } catch (e) {
      const err = e as { message?: string };
      message.error(err.message || "加载失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  // 自定义上传（不走 antd 自动上传，手动控制）
  const handleUpload = async (req: UploadRequestOption) => {
    const file = req.file as File;
    try {
      message.loading({ content: `正在处理 ${file.name}...`, key: "upload", duration: 0 });
      const result = await knowledgeApi.upload(file);
      message.success({
        content: `「${result.file.title}」处理完成，生成 ${result.chunk_count} 个知识片段`,
        key: "upload",
      });
      load();
    } catch (e) {
      const err = e as { message?: string };
      message.error({ content: `上传失败: ${err.message}`, key: "upload" });
    }
  };

  // 粘贴文本
  const handlePaste = async () => {
    try {
      const values = await form.validateFields();
      setPasting(true);
      const result = await knowledgeApi.paste(values.title || "粘贴文本", values.content);
      message.success(`「${result.file.title}」处理完成，生成 ${result.chunk_count} 个知识片段`);
      setPasteOpen(false);
      form.resetFields();
      load();
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; message?: string };
      if (err.errorFields) return; // 表单校验失败
      message.error(`处理失败: ${err.message}`);
    } finally {
      setPasting(false);
    }
  };

  // 删除文件
  const handleDelete = async (id: number) => {
    try {
      await knowledgeApi.deleteFile(id);
      message.success("已删除");
      load();
    } catch (e) {
      const err = e as { message?: string };
      message.error(`删除失败: ${err.message}`);
    }
  };

  // 重新向量化（Embedding 失败后一键重试）
  const handleReembed = async (id: number) => {
    setReembedding(id);
    try {
      const resp = await knowledgeApi.reembed(id);
      message.success(`重新向量化完成，生成 ${resp.chunk_count} 个知识片段`);
    } catch (e) {
      const err = e as { message?: string };
      message.warning(`重试失败: ${err.message}`);
    } finally {
      setReembedding(null);
      load(); // 无论成败都刷新列表（失败原因会更新到 parse_error）
    }
  };

  // 语义检索
  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setSearching(true);
    setSearchResults([]);
    try {
      const resp = await knowledgeApi.search(searchQuery, undefined, 5);
      setSearchResults(resp.results || []);
      if (resp.count === 0) {
        message.info("知识库中未找到相关内容");
      }
    } catch (e) {
      const err = e as { message?: string };
      message.error(`检索失败: ${err.message}`);
    } finally {
      setSearching(false);
    }
  };

  // RAG 对话
  const handleChat = async () => {
    if (!chatQuery.trim()) return;
    setChatting(true);
    setChatAnswer(null);
    try {
      const answer = await knowledgeApi.chat(chatQuery);
      setChatAnswer(answer);
    } catch (e) {
      const err = e as { message?: string };
      message.error(`问答失败: ${err.message}`);
    } finally {
      setChatting(false);
    }
  };

  // 复制到剪贴板
  const copyText = (text: string) => {
    navigator.clipboard.writeText(text).then(() => {
      message.success("已复制");
    });
  };

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="知识文件"
              value={stats?.file_count ?? 0}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="知识片段" value={stats?.chunk_count ?? 0} prefix={<FileTextOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="可检索"
              value={stats?.ready_count ?? 0}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="处理失败"
              value={stats?.failed_count ?? 0}
              valueStyle={{ color: stats?.failed_count ? "#ff4d4f" : undefined }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="RAG 知识库：上传企业文档（产品手册/SOP/合同模板等），AI 回答将基于您的资料。支持 txt/md/csv/docx 格式。"
      />

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "files",
            label: (
              <span>
                <FileTextOutlined /> 文件管理
              </span>
            ),
            children: (
              <div>
                <Space style={{ marginBottom: 16 }}>
                  <Upload customRequest={handleUpload} showUploadList={false} accept=".txt,.md,.csv,.docx,.log,.markdown">
                    <Button type="primary" icon={<UploadOutlined />}>
                      上传文件
                    </Button>
                  </Upload>
                  <Button icon={<InboxOutlined />} onClick={() => setPasteOpen(true)}>
                    粘贴文本
                  </Button>
                  <Button onClick={load}>刷新</Button>
                </Space>

                <Spin spinning={loading}>
                  {files.length === 0 && !loading ? (
                    <Empty description="暂无知识文件，点击上方上传或粘贴" />
                  ) : (
                    <List
                      dataSource={files}
                      pagination={{
                        current: page,
                        total,
                        pageSize: 20,
                        onChange: setPage,
                        showTotal: (t) => `共 ${t} 个文件`,
                        size: "small",
                      }}
                      renderItem={(item) => (
                        <List.Item
                          actions={[
                            ...(item.status === "failed" &&
                            item.file_type !== "paste"
                              ? [
                                  <Button
                                    key="reembed"
                                    size="small"
                                    icon={<ReloadOutlined />}
                                    loading={reembedding === item.id}
                                    onClick={() => handleReembed(item.id)}
                                  >
                                    重试向量化
                                  </Button>,
                                ]
                              : []),
                            <Popconfirm
                              key="del"
                              title="确认删除？"
                              description="将同时删除该文件的所有知识片段"
                              onConfirm={() => handleDelete(item.id)}
                            >
                              <Button danger size="small" icon={<DeleteOutlined />}>
                                删除
                              </Button>
                            </Popconfirm>,
                          ]}
                        >
                          <List.Item.Meta
                            avatar={<FileTextOutlined style={{ fontSize: 24, color: "#1677ff" }} />}
                            title={
                              <Space>
                                <Text strong>{item.title}</Text>
                                {statusTag(item.status)}
                                <Tag>{item.file_type}</Tag>
                                <Text type="secondary">{humanSize(item.file_size)}</Text>
                              </Space>
                            }
                            description={
                              <Space direction="vertical" size={0}>
                                <Text type="secondary">
                                  {item.chunk_count > 0 ? `${item.chunk_count} 个知识片段` : "未解析"}
                                  {" · "}
                                  {new Date(item.created_at).toLocaleString("zh-CN")}
                                </Text>
                                {item.parse_error && (
                                  <Text type="danger" style={{ fontSize: 12 }}>
                                    {item.parse_error}
                                  </Text>
                                )}
                              </Space>
                            }
                          />
                        </List.Item>
                      )}
                    />
                  )}
                </Spin>
              </div>
            ),
          },
          {
            key: "search",
            label: (
              <span>
                <SearchOutlined /> 语义检索
              </span>
            ),
            children: (
              <div>
                <Space.Compact style={{ width: "100%", marginBottom: 16 }}>
                  <Input
                    placeholder="输入问题或关键词，语义检索知识库..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    onPressEnter={handleSearch}
                    size="large"
                  />
                  <Button type="primary" size="large" icon={<SearchOutlined />} onClick={handleSearch} loading={searching}>
                    检索
                  </Button>
                </Space.Compact>

                <Spin spinning={searching}>
                  {searchResults.length > 0 ? (
                    <div>
                      <Divider orientation="left">检索结果（{searchResults.length} 条）</Divider>
                      {searchResults.map((r, i) => (
                        <Card
                          key={i}
                          size="small"
                          style={{ marginBottom: 12 }}
                          title={
                            <Space>
                              <Tag color="blue">{r.source_file}</Tag>
                              <Tag>相似度 {(r.score * 100).toFixed(1)}%</Tag>
                            </Space>
                          }
                          extra={
                            <Tooltip title="复制内容">
                              <Button size="small" type="text" icon={<CopyOutlined />} onClick={() => copyText(r.content)} />
                            </Tooltip>
                          }
                        >
                          <Paragraph style={{ margin: 0, whiteSpace: "pre-wrap" }}>{r.content}</Paragraph>
                        </Card>
                      ))}
                    </div>
                  ) : (
                    !searching && (
                      <Empty description="输入问题后点击检索，AI 将从知识库中找到最相关的内容" />
                    )
                  )}
                </Spin>
              </div>
            ),
          },
          {
            key: "chat",
            label: (
              <span>
                <RobotOutlined /> 智能问答
              </span>
            ),
            children: (
              <div>
                <Space.Compact style={{ width: "100%", marginBottom: 16 }}>
                  <Input
                    placeholder="向 AI 提问，回答基于您的企业知识库..."
                    value={chatQuery}
                    onChange={(e) => setChatQuery(e.target.value)}
                    onPressEnter={handleChat}
                    size="large"
                  />
                  <Button type="primary" size="large" icon={<RobotOutlined />} onClick={handleChat} loading={chatting}>
                    提问
                  </Button>
                </Space.Compact>

                <Spin spinning={chatting} tip="AI 正在检索知识库并生成回答...">
                  {chatAnswer ? (
                    <Card
                      title={
                        <Space>
                          <RobotOutlined />
                          <span>AI 回答</span>
                          {chatAnswer.has_context ? (
                            <Tag color="green">基于知识库</Tag>
                          ) : (
                            <Tag color="orange">通用知识</Tag>
                          )}
                          {chatAnswer.provider && <Tag>{chatAnswer.provider}</Tag>}
                        </Space>
                      }
                      extra={
                        <Button size="small" icon={<CopyOutlined />} onClick={() => copyText(chatAnswer.answer)}>
                          复制
                        </Button>
                      }
                    >
                      <Paragraph style={{ whiteSpace: "pre-wrap", marginBottom: chatAnswer.sources.length ? 16 : 0 }}>
                        {chatAnswer.answer}
                      </Paragraph>
                      {chatAnswer.sources.length > 0 && (
                        <>
                          <Divider orientation="left">
                            <Text type="secondary">引用来源（{chatAnswer.sources.length}）</Text>
                          </Divider>
                          {chatAnswer.sources.map((src, i) => (
                            <Card
                              key={i}
                              size="small"
                              type="inner"
                              style={{ marginBottom: 8 }}
                              title={
                                <Space>
                                  <Tag color="blue">{src.source_file}</Tag>
                                  <Tag>相似度 {(src.score * 100).toFixed(1)}%</Tag>
                                </Space>
                              }
                            >
                              <Paragraph style={{ margin: 0, color: "#666", fontSize: 13, whiteSpace: "pre-wrap" }}>
                                {src.content.length > 200
                                  ? src.content.slice(0, 200) + "..."
                                  : src.content}
                              </Paragraph>
                            </Card>
                          ))}
                        </>
                      )}
                    </Card>
                  ) : (
                    !chatting && (
                      <Empty description="提问后，AI 会先检索知识库再生成回答，并标注信息来源" />
                    )
                  )}
                </Spin>
              </div>
            ),
          },
        ]}
      />

      {/* 粘贴文本 Modal */}
      <Modal
        title="粘贴文本到知识库"
        open={pasteOpen}
        onCancel={() => setPasteOpen(false)}
        onOk={handlePaste}
        confirmLoading={pasting}
        okText="提交并向量化"
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="title" label="标题">
            <Input placeholder="给这段知识起个名字（可选）" />
          </Form.Item>
          <Form.Item
            name="content"
            label="文本内容"
            rules={[{ required: true, message: "请输入文本内容" }]}
          >
            <TextArea
              rows={10}
              placeholder="粘贴产品说明、SOP、FAQ、合同条款等文本..."
              showCount
              maxLength={20000}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
