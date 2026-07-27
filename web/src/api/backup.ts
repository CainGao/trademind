// 数据备份 API：生成/列表/下载/删除（管理员）。
import client from "./client";

export interface BackupInfo {
  filename: string;
  size: number;
  created_at: string;
}

export const backupApi = {
  /** 生成一份完整备份（DB 快照 + 附件） */
  create() {
    return client.post<unknown, BackupInfo>("/system/backup");
  },

  /** 备份列表（按时间倒序） */
  list() {
    return client.get<unknown, BackupInfo[]>("/system/backups");
  },

  /** 下载备份 zip（需 JWT 鉴权，用 blob 方式触发浏览器下载） */
  async download(filename: string) {
    const resp = await client.get(`/system/backups/${encodeURIComponent(filename)}`, {
      responseType: "blob",
    });
    // 创建临时下载链接
    const url = window.URL.createObjectURL(resp as unknown as Blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
  },

  /** 删除一份备份 */
  delete(filename: string) {
    return client.delete<unknown, { deleted: boolean }>(
      `/system/backups/${encodeURIComponent(filename)}`
    );
  },
};
