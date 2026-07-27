// 系统设置页：数据备份/恢复管理（管理员）。
//
// 备份策略：
//   - 使用 SQLite VACUUM INTO 生成一致性快照，无需停服
//   - 备份含数据库 + 知识库附件，打包为 zip
//   - 恢复为 CLI 操作（./trademind --restore <file>），需停服
//
// 「数据 100% 本地」是产品核心承诺，可备份是企业私有化底线能力。

import { useState, useCallback } from "react";
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
} from "antd";
import {
  CloudUploadOutlined,
  DownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import { backupApi, type BackupInfo } from "../../api/backup";

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
