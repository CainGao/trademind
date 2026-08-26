// 系统设置页：安全设置（修改密码）+ 数据备份/恢复管理（管理员）。
//
// 备份策略：
//   - 使用 SQLite VACUUM INTO 生成一致性快照，无需停服
//   - 备份含数据库 + 知识库附件，打包为 zip
//   - 恢复为 CLI 操作（./trademind --restore <file>），需停服
//
// 「数据 100% 本地」是产品核心承诺，可备份是企业私有化底线能力。
// 修改密码（gotcha #88）：admin 仍用默认密码时顶部横幅会引导到本页。

import { useState, useCallback, useEffect } from "react";
import {
  Card,
  Table,
  Button,
  Space,
  Popconfirm,
  message,
  Typography,
  Alert,
  Tag,
  Tooltip,
  Form,
  Input,
} from "antd";
import {
  CloudUploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  LockOutlined,
} from "@ant-design/icons";
import { backupApi, type BackupInfo } from "../../api/backup";
import { setupApi } from "../../api/setup";
import { useAuthStore } from "../../store/auth";

const { Title, Text, Paragraph } = Typography;

// 格式化文件大小
function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

// 格式化时间
function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface PwdFormValues {
  old_password: string;
  new_password: string;
  confirm_password: string;
}

// PasswordCard 修改管理员密码卡片（admin only）。
function PasswordCard() {
  const [form] = Form.useForm<PwdFormValues>();
  const [changing, setChanging] = useState(false);
  const { mustChangePassword, clearMustChangePassword } = useAuthStore();

  const handleChangePassword = async (values: PwdFormValues) => {
    setChanging(true);
    try {
      await setupApi.changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      });
      message.success("密码修改成功");
      form.resetFields();
      // 清除「仍在使用默认密码」提醒横幅
      clearMustChangePassword();
    } catch (e) {
      const err = e as { message?: string };
      message.error(`修改失败: ${err.message || "未知错误"}`);
    } finally {
      setChanging(false);
    }
  };

  const onFinish = (values: PwdFormValues) => {
    void handleChangePassword(values);
  };

  return (
    <Card
      title={
        <Space>
          <LockOutlined />
          <span>安全设置</span>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      {mustChangePassword && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message="检测到当前管理员仍在使用默认密码，请立即修改"
        />
      )}
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 420 }}>
        <Form.Item
          name="old_password"
          label="当前密码"
          rules={[{ required: true, message: "请输入当前密码" }]}
        >
          <Input.Password placeholder="当前密码" autoComplete="current-password" />
        </Form.Item>
        <Form.Item
          name="new_password"
          label="新密码"
          rules={[
            { required: true, message: "请输入新密码" },
            { min: 6, max: 64, message: "长度 6-64 位" },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || value !== getFieldValue("old_password")) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error("新密码不能与当前密码相同"));
              },
            }),
          ]}
        >
          <Input.Password placeholder="新密码（6-64 位）" autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="confirm_password"
          label="确认新密码"
          dependencies={["new_password"]}
          rules={[
            { required: true, message: "请再次输入新密码" },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || value === getFieldValue("new_password")) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error("两次输入的密码不一致"));
              },
            }),
          ]}
        >
          <Input.Password placeholder="再次输入新密码" autoComplete="new-password" />
        </Form.Item>
        <Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
          常见弱密码（如 admin123/123456）已被系统黑名单拦截，修改后不可改回。
        </Text>
        <Button type="primary" htmlType="submit" loading={changing}>
          修改密码
        </Button>
      </Form>
    </Card>
  );
}

export default function SettingsPage() {
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);

  const fetchBackups = useCallback(async () => {
    setLoading(true);
    try {
      const list = await backupApi.list();
      setBackups(list || []);
    } catch (e) {
      const err = e as { message?: string };
      message.error(`加载备份列表失败: ${err.message || "未知错误"}`);
    } finally {
      setLoading(false);
    }
  }, []);

  // 进入页面自动加载备份列表
  //（修复存量 bug：此前无 useEffect，列表从不自动加载，只能手动点「刷新」，
  //  老板首次进入看到「暂无备份」会误以为自动备份没有在工作）
  useEffect(() => {
    void fetchBackups();
  }, [fetchBackups]);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const info = await backupApi.create();
      message.success(`备份成功: ${info.filename} (${formatSize(info.size)})`);
      await fetchBackups();
    } catch (e) {
      const err = e as { message?: string };
      message.error(`备份失败: ${err.message || "未知错误"}`);
    } finally {
      setCreating(false);
    }
  };

  const handleDownload = async (filename: string) => {
    const hide = message.loading("正在下载...", 0);
    try {
      await backupApi.download(filename);
      hide();
      message.success("下载已开始");
    } catch (e) {
      hide();
      const err = e as { message?: string };
      message.error(`下载失败: ${err.message || "未知错误"}`);
    }
  };

  const handleDelete = async (filename: string) => {
    try {
      await backupApi.delete(filename);
      message.success("已删除");
      await fetchBackups();
    } catch (e) {
      const err = e as { message?: string };
      message.error(`删除失败: ${err.message || "未知错误"}`);
    }
  };

  const columns = [
    {
      title: "文件名",
      dataIndex: "filename",
      key: "filename",
      render: (name: string) => <Text code copyable>{name}</Text>,
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 120,
      render: (size: number) => formatSize(size),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
      render: (t: string) => formatTime(t),
    },
    {
      title: "操作",
      key: "action",
      width: 180,
      render: (_: unknown, record: BackupInfo) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => handleDownload(record.filename)}
          >
            下载
          </Button>
          <Popconfirm
            title="确认删除此备份？"
            description="删除后不可恢复"
            onConfirm={() => handleDelete(record.filename)}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
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
        <SafetyCertificateOutlined /> 系统设置
      </Title>

      {/* 修改密码卡片（gotcha #88 默认密码治理） */}
      <PasswordCard />

      {/* 数据备份卡片 */}
      <Card
        title={
          <Space>
            <CloudUploadOutlined />
            <span>数据备份与恢复</span>
            <Tag color="blue">数据 100% 本地</Tag>
          </Space>
        }
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchBackups}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<CloudUploadOutlined />}
              onClick={handleCreate}
              loading={creating}
            >
              立即备份
            </Button>
          </Space>
        }
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="备份包含完整数据库快照（VACUUM INTO）和知识库附件"
          description={
            <div>
              <Paragraph style={{ marginBottom: 4 }}>
                <Text strong>恢复方法：</Text>
                停止服务后执行命令：
                <Text code copyable>
                  ./trademind --restore &lt;备份文件路径&gt;
                </Text>
              </Paragraph>
              <Text type="secondary">
                恢复前会自动将当前数据库备份为 .bak 文件，确保数据安全。
              </Text>
            </div>
          }
        />

        <Table
          columns={columns}
          dataSource={backups}
          rowKey="filename"
          loading={loading}
          size="small"
          pagination={false}
          locale={{ emptyText: "暂无备份，点击「立即备份」创建第一份" }}
          style={{ marginTop: 8 }}
        />

        {backups.length > 0 && (
          <div style={{ marginTop: 16 }}>
            <Tooltip title="备份文件存储于 runtime/backups/ 目录">
              <Text type="secondary">
                共 {backups.length} 份备份，占用磁盘{" "}
                {formatSize(backups.reduce((sum, b) => sum + b.size, 0))}
              </Text>
            </Tooltip>
          </div>
        )}
      </Card>
    </div>
  );
}
