// B2B 客户/询盘/报价 API。
import client from "./client";
import type { PageData } from "../types";

// ===== 客户 =====
export interface Customer {
  id: number;
  company_name: string;
  country?: string;
  contact_person?: string;
  email?: string;
  phone?: string;
  wechat?: string;
  demand?: string;
  stage: string;
  deal_probability?: number;
  last_contact_at?: string;
  created_at: string;
}

export interface CreateCustomerInput {
  company_name: string;
  country?: string;
  contact_person?: string;
  email?: string;
  phone?: string;
  wechat?: string;
  demand?: string;
  stage?: string;
}

export interface UpdateCustomerInput extends Partial<CreateCustomerInput> {}

export interface CustomerListQuery {
  page?: number;
  page_size?: number;
  keyword?: string;
  country?: string;
  stage?: string;
}

export const customerApi = {
  list: (q: CustomerListQuery) =>
    client.get<unknown, PageData<Customer>>("/customers", { params: q }),
  get: (id: number) =>
    client.get<unknown, Customer>(`/customers/${id}`),
  create: (input: CreateCustomerInput) =>
    client.post<unknown, Customer>("/customers", input),
  update: (id: number, input: UpdateCustomerInput) =>
    client.put<unknown, Customer>(`/customers/${id}`, input),
  delete: (id: number) =>
    client.delete<unknown, { deleted: boolean }>(`/customers/${id}`),
};

// ===== 询盘 =====
export interface Inquiry {
  id: number;
  customer_id?: number;
  source: string;
  product_desc: string;
  quantity?: number;
  target_price?: number;
  destination?: string;
  status: string;
  ai_analysis?: string;
  created_at: string;
}

export interface CreateInquiryInput {
  customer_id?: number;
  source: string;
  product_desc: string;
  quantity?: number;
  target_price?: string;
  destination?: string;
}

export const inquiryApi = {
  list: (q: { page?: number; page_size?: number; source?: string; status?: string }) =>
    client.get<unknown, PageData<Inquiry>>("/inquiries", { params: q }),
  get: (id: number) => client.get<unknown, Inquiry>(`/inquiries/${id}`),
  create: (input: CreateInquiryInput) =>
    client.post<unknown, Inquiry>("/inquiries", input),
  delete: (id: number) =>
    client.delete<unknown, { deleted: boolean }>(`/inquiries/${id}`),
};

// ===== 报价单 =====
export interface Quotation {
  id: number;
  quotation_no: string;
  inquiry_id?: number;
  customer_id?: number;
  currency: string;
  total_amount: number;
  status: string;
  valid_until?: string;
  items?: string;
  created_at: string;
}

export interface CreateQuotationInput {
  inquiry_id?: number;
  customer_id?: number;
  currency?: string;
  total_amount: string;
  items?: string;
  valid_days?: number;
}

export const quotationApi = {
  list: (q: { page?: number; page_size?: number; status?: string }) =>
    client.get<unknown, PageData<Quotation>>("/quotations", { params: q }),
  get: (id: number) => client.get<unknown, Quotation>(`/quotations/${id}`),
  create: (input: CreateQuotationInput) =>
    client.post<unknown, Quotation>("/quotations", input),
  updateStatus: (id: number, status: string) =>
    client.put<unknown, { updated: boolean }>(`/quotations/${id}/status`, { status }),
  delete: (id: number) =>
    client.delete<unknown, { deleted: boolean }>(`/quotations/${id}`),
};

// ===== AI =====
export interface ChatMessage {
  role: "system" | "user" | "assistant";
  content: string;
}

export interface ChatResponse {
  provider: string;
  model: string;
  content: string;
  usage?: { prompt_tokens: number; completion_tokens: number; total_tokens: number };
}

export interface ProviderInfo {
  name: string;
  configured: boolean;
  is_default: boolean;
}

export const aiApi = {
  providers: () =>
    client.get<unknown, { providers: ProviderInfo[] }>("/ai/providers"),
  chat: (messages: ChatMessage[], provider?: string) =>
    client.post<unknown, ChatResponse>("/ai/chat", { messages, provider }),
  analyzeProduct: (productId: number, provider?: string) =>
    client.post<unknown, any>(`/agent/analyze-product?product_id=${productId}${provider ? `&provider=${provider}` : ""}`),
};
