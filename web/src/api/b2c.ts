// B2C 跨境电商 API：店铺/上架/订单。
import client from "./client";

// ===== 店铺 =====
export interface Store {
  id: number;
  name: string;
  platform: string; // amazon|shopify|tiktok|temu
  region?: string;
  store_id?: string;
  status: string;
  synced_at?: string;
  created_at: string;
}

export interface CreateStoreInput {
  name: string;
  platform: string;
  region?: string;
  store_id?: string;
  status?: string;
}

// ===== 上架 =====
// 注意：selling_price 是 shopspring/decimal 的字符串序列化（gotcha #20），
// 表单 InputNumber 得到 number，保存边界需 String() / Number() 转换。
export interface Listing {
  id: number;
  store_id: number;
  product_id?: number;
  platform_sku: string;
  platform_asin?: string;
  title: string;
  status: string;
  listing_url?: string;
  selling_price: string | number;
  currency: string;
  stock?: number;
  published_at?: string;
  created_at: string;
}

// ===== 订单 =====
export interface Order {
  id: number;
  store_id: number;
  store_name?: string;
  platform?: string;
  listing_id?: number;
  platform_order_no: string;
  status: string;
  amount: number;
  currency: string;
  buyer_name?: string;
  buyer_country?: string;
  items?: string;
  tracking_no?: string;
  ordered_at: string;
  shipped_at?: string;
  delivered_at?: string;
  created_at: string;
}

export interface OrderOverview {
  total_orders: number;
  total_revenue: number;
  pending_count: number;
  shipped_count: number;
  delivered_count: number;
}

// ===== API =====
export const b2cApi = {
  // 店铺
  listStores(params: { page?: number; page_size?: number; platform?: string }) {
    return client.get<unknown, { list: Store[]; total: number }>(
      "/b2c/stores",
      { params }
    );
  },
  createStore(data: CreateStoreInput) {
    return client.post<unknown, Store>("/b2c/stores", data);
  },
  updateStore(id: number, data: Partial<CreateStoreInput>) {
    return client.put<unknown, Store>(`/b2c/stores/${id}`, data);
  },
  deleteStore(id: number) {
    return client.delete<unknown, { deleted: boolean }>(`/b2c/stores/${id}`);
  },

  // 上架
  listListings(params: {
    page?: number;
    page_size?: number;
    store_id?: number;
    status?: string;
  }) {
    return client.get<unknown, { list: Listing[]; total: number }>(
      "/b2c/listings",
      { params }
    );
  },
  createListing(data: Partial<Listing>) {
    return client.post<unknown, Listing>("/b2c/listings", data);
  },
  updateListing(id: number, data: Partial<Listing>) {
    return client.put<unknown, Listing>(`/b2c/listings/${id}`, data);
  },
  deleteListing(id: number) {
    return client.delete<unknown, { deleted: boolean }>(`/b2c/listings/${id}`);
  },

  // 订单
  listOrders(params: {
    page?: number;
    page_size?: number;
    store_id?: number;
    status?: string;
    country?: string;
  }) {
    return client.get<unknown, { list: Order[]; total: number }>(
      "/b2c/orders",
      { params }
    );
  },
  createOrder(data: Partial<Order>) {
    return client.post<unknown, Order>("/b2c/orders", data);
  },
  updateOrderStatus(id: number, status: string) {
    return client.put<unknown, { updated: boolean }>(
      `/b2c/orders/${id}/status`,
      { status }
    );
  },
  orderOverview() {
    return client.get<unknown, OrderOverview>("/b2c/overview");
  },
};
