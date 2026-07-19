// 共享类型定义（与后端 models 对齐）。

// ===== 通用响应（规范 §3.2）=====
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

export interface PageData<T> {
  list: T[];
  total: number;
  page: number;
  size: number;
}

// ===== 用户与认证 =====
export type UserRole = "admin" | "boss" | "sourcing" | "sales" | "operator" | "staff";

export interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  role: UserRole;
  avatar?: string;
}

export interface LoginInput {
  username: string;
  password: string;
}

export interface RegisterInput {
  username: string;
  password: string;
  nickname?: string;
  role?: UserRole;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: UserInfo;
}

// ===== 商品 =====
export type DataSource = "1688" | "alibaba" | "factory" | "manual" | "tiktok" | "temu" | "amazon";

export interface Product {
  id: number;
  name: string;
  category?: string;
  description?: string;
  purchase_price?: number;
  purchase_currency?: string;
  supplier_id?: number;
  source?: DataSource;
  source_id?: string;
  source_url?: string;
  image_urls?: string;
  weight_kg?: number;
  volume_cbm?: number;
  package_spec?: string;
  b2b_moq?: number;
  b2b_fob_price?: number;
  b2c_platform?: string;
  b2c_selling_price?: number;
  b2c_fba_stock?: number;
  ai_score?: number;
  scenarios?: string;
  created_at: string;
  updated_at: string;
}

// ===== 供应商 =====
export interface Supplier {
  id: number;
  name: string;
  source?: DataSource;
  source_id?: string;
  location?: string;
  contact?: string;
  product_count?: number;
  ai_score?: number;
  risk_level?: "low" | "medium" | "high";
  last_active_at?: string;
  created_at: string;
}

// ===== 审计日志 =====
export interface AuditLog {
  id: number;
  user_id: number;
  action: string;
  resource: string;
  resource_id?: number;
  detail?: string;
  ip?: string;
  created_at: string;
}
