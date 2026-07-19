// 供应商 API。
import client from "./client";
import type { PageData, Supplier } from "../types";

export interface SupplierListQuery {
  page?: number;
  page_size?: number;
  keyword?: string;
  source?: string;
  risk_level?: string;
}

export interface SupplierDetail extends Supplier {
  product_count: number;
}

export interface SupplierOverview {
  total: number;
  risk_high: number;
  risk_medium: number;
  risk_low: number;
  with_ai_score: number;
}

export interface UpdateRiskInput {
  risk_level: string;
  ai_score?: string;
}

export const supplierApi = {
  list: (q: SupplierListQuery) =>
    client.get<unknown, PageData<Supplier>>("/suppliers", { params: q }),

  overview: () =>
    client.get<unknown, SupplierOverview>("/suppliers/overview"),

  get: (id: number) =>
    client.get<unknown, SupplierDetail>(`/suppliers/${id}`),

  products: (id: number, page = 1, pageSize = 20) =>
    client.get<unknown, PageData<Supplier>>(`/suppliers/${id}/products`, {
      params: { page, page_size: pageSize },
    }),

  updateRisk: (id: number, input: UpdateRiskInput) =>
    client.put<unknown, { updated: boolean }>(`/suppliers/${id}/risk`, input),

  delete: (id: number) =>
    client.delete<unknown, { deleted: boolean }>(`/suppliers/${id}`),
};
