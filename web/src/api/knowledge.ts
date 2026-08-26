// 知识库 API：上传/粘贴/检索/RAG 对话（Week 8）。
import client from "./client";

export interface KnowledgeFile {
  id: number;
  title: string;
  filename: string;
  file_type: string;
  file_size: number;
  chunk_count: number;
  status: "processing" | "ready" | "failed";
  parse_error?: string;
  created_at: string;
}

export interface SearchResult {
  file_id: number;
  source_file: string;
  content: string;
  score: number;
}

export interface RAGChatAnswer {
  answer: string;
  sources: SearchResult[];
  provider?: string;
  has_context: boolean;
}

export interface KnowledgeStats {
  file_count: number;
  chunk_count: number;
  ready_count: number;
  failed_count: number;
}

export interface UploadResult {
  file: KnowledgeFile;
  chunk_count: number;
  warning?: string;
}

export const knowledgeApi = {
  /** 上传文件（multipart） */
  upload(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    return client.post<unknown, UploadResult>("/knowledge/upload", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },

  /** 粘贴文本 */
  paste(title: string, content: string) {
    return client.post<unknown, UploadResult>("/knowledge/paste", {
      title,
      content,
    });
  },

  /** 文件列表 */
  list(params: { page?: number; page_size?: number; keyword?: string }) {
    return client.get<unknown, { list: KnowledgeFile[]; total: number }>(
      "/knowledge/files",
      { params }
    );
  },

  /** 文件详情 */
  getFile(id: number) {
    return client.get<unknown, { file: KnowledgeFile; chunk_count: number }>(
      `/knowledge/files/${id}`
    );
  },

  /** 删除文件 */
  deleteFile(id: number) {
    return client.delete<unknown, { deleted: boolean }>(
      `/knowledge/files/${id}`
    );
  },

  /** 重新向量化（Embedding 失败后换有效 Key 一键重试） */
  reembed(id: number) {
    return client.post<unknown, UploadResult>(
      `/knowledge/files/${id}/reembed`
    );
  },

  /** 语义检索 */
  search(query: string, fileId?: number, topK?: number) {
    return client.post<unknown, { results: SearchResult[]; count: number }>(
      "/knowledge/search",
      { query, file_id: fileId, top_k: topK }
    );
  },

  /** RAG 对话 */
  chat(query: string, history?: string) {
    return client.post<unknown, RAGChatAnswer>("/knowledge/chat", {
      query,
      history,
    });
  },

  /** 知识库统计 */
  stats() {
    return client.get<unknown, KnowledgeStats>("/knowledge/stats");
  },
};
