// 商品中心 API。
import client from "./client";
import type { PageData, Product } from "../types";

export interface ProductListQuery {
  page?: number;
  page_size?: number;
  keyword?: string;
  category?: string;
  source?: string;
  sort_by?: string;
  order?: string;
}

export interface CreateProductInput {
  name: string;
  category?: string;
  description?: string;
  purchase_price?: string;
  purchase_currency?: string;
  source_url?: string;
  image_urls?: string;
  weight_kg?: string;
  package_spec?: string;
  scenarios?: string;
}

export interface UpdateProductInput {
  name?: string;
  category?: string;
  description?: string;
  purchase_price?: string;
  purchase_currency?: string;
  source_url?: string;
  image_urls?: string;
  weight_kg?: string;
  package_spec?: string;
  scenarios?: string;
}

export const productApi = {
  list: (query: ProductListQuery) =>
    client.get<unknown, PageData<Product>>("/products", { params: query }),

  get: (id: number) =>
    client.get<unknown, Product>(`/products/${id}`),

  create: (input: CreateProductInput) =>
    client.post<unknown, Product>("/products", input),

  update: (id: number, input: UpdateProductInput) =>
    client.put<unknown, Product>(`/products/${id}`, input),

  delete: (id: number) =>
    client.delete<unknown, { deleted: boolean }>(`/products/${id}`),

  categories: () =>
    client.get<unknown, string[]>("/products/categories"),
};
