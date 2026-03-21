import api from './client';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Supplier {
  id: string;
  company: string;
  firstname: string;
  lastname: string;
  email: string;
  skype: string;
  origin: string;
  misc: string;
  createdAt: string;
  updatedAt: string;
}

export interface SupplierCode {
  id: string;
  supplierId: string;
  short: string;
  createdAt: string;
}

export interface GoodsGroup {
  id: string;
  name: string;
  short: string;
}

export interface Product {
  id: string;
  nameShort: string;
  ownNameShort: string;
  nameLong: string;
  description: string;
  ean: string;
  wtn: string;
  aco: string;
  goodsGroupId?: string;
  supplierCodeId?: string;
  supplierId?: string;
  widthMm: number;
  heightMm: number;
  lengthMm: number;
  weightKg: number;
  vpe: number;
  lastEk: number;
  lastEkDate?: string;
  userDef01: string;
  userDef02: string;
  userDef03: string;
  userDef04: string;
  userDef05: string;
  virtual: boolean;
  checked: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface FreightCarrier {
  id: string;
  name: string;
  description: string;
}

export interface Harbour {
  id: string;
  name: string;
  description: string;
}

export interface Container {
  id: string;
  name: string;
  description: string;
  volumeM3: number;
  heightM: number;
  lengthM: number;
  widthM: number;
}

export interface Country {
  id: string;
  name: string;
}

export interface OrderProduct {
  productId: string;
  quantity: number;
  unitPriceUsd: number;
  totalPriceUsd: number;
  lengthMm: number;
  widthMm: number;
  heightMm: number;
  volumeM3: number;
  weightKg: number;
  credited: boolean;
  inventoryChecked: boolean;
}

export interface OrderFreight {
  containerId: string;
  harbourIdFrom: string;
  harbourIdTo: string;
  freightCarrierId: string;
  containerNr: string;
  shippingDate?: string;
  estimatedArrival?: string;
  arrival?: string;
  avisShipperDate?: string;
  docOfOrigin?: string;
  docOfOriginChecked: boolean;
  docOfOriginSigned: boolean;
  docOfOriginShipped: boolean;
  freightageEur: number;
  preFreightageEur: number;
  seaFreightUsd: number;
  emergencyBunkerSurchargeUsd: number;
  peakSeasonSurchargeUsd: number;
  suezCanalAddonUsd: number;
}

export interface OrderPayment {
  id: string;
  orderId: string;
  nr: number;
  paymentAmountEur: number;
  paymentDate?: string;
  paymentDollarRate: number;
  paymentFees: number;
  createdAt: string;
  updatedAt: string;
}

export interface Order {
  id: string;
  supplierId?: string;
  orderNumber: string;
  orderDate: string;
  orderContents: string;
  misc: string;
  orderSumUsd: number;
  transportInsurance: number;
  discount: number;
  preDollarRate: number;
  invoiceFreightCarrierEur: number;
  products: OrderProduct[];
  freight?: OrderFreight;
  createdAt: string;
  updatedAt: string;
}

export interface OrderTask {
  id: string;
  orderId: string;
  text: string;
  dueDate?: string;
  doneAt?: string;
  assigneeId?: string;
}

export interface Offer {
  id: string;
  productId?: string;
  supplierId?: string;
  isStockOffer: boolean;
  nameShort: string;
  description: string;
  priceUsd: number;
  quantity: number;
  validUntil?: string;
  misc: string;
  createdAt: string;
}

// ---------------------------------------------------------------------------
// API calls
// ---------------------------------------------------------------------------

const BASE = '/procurement';

// Suppliers
export const suppliersApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<Supplier>>(`${BASE}/suppliers`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<Supplier>) => api.post<Supplier>(`${BASE}/suppliers`, data).then(r => r.data),
  get: (id: string) => api.get<Supplier>(`${BASE}/suppliers/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Supplier>) => api.put(`${BASE}/suppliers/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/suppliers/${id}`),
};

// Supplier codes
export const supplierCodesApi = {
  list: (supplierId?: string) => api.get<SupplierCode[]>(`${BASE}/supplier-codes`, { params: { supplierId } }).then(r => r.data),
  create: (data: Partial<SupplierCode>) => api.post<SupplierCode>(`${BASE}/supplier-codes`, data).then(r => r.data),
  update: (id: string, data: Partial<SupplierCode>) => api.put(`${BASE}/supplier-codes/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/supplier-codes/${id}`),
};

// Goods groups
export const goodsGroupsApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<GoodsGroup>>(`${BASE}/goods-groups`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<GoodsGroup>) => api.post<GoodsGroup>(`${BASE}/goods-groups`, data).then(r => r.data),
  update: (id: string, data: Partial<GoodsGroup>) => api.put(`${BASE}/goods-groups/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/goods-groups/${id}`),
};

export interface PagedResult<T> {
  items: T[];
  total: number;
  page: number;
  pages: number;
}

export interface ProductsPage {
  items: Product[];
  total: number;
  page: number;
  pages: number;
}

// Products
export const productsApi = {
  list: (q?: string, page = 1) => api.get<ProductsPage>(`${BASE}/products`, { params: { q, page } }).then(r => r.data),
  create: (data: Partial<Product>) => api.post<Product>(`${BASE}/products`, data).then(r => r.data),
  get: (id: string) => api.get<Product>(`${BASE}/products/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Product>) => api.put(`${BASE}/products/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/products/${id}`),
};

// Freight carriers
export const freightCarriersApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<FreightCarrier>>(`${BASE}/freight-carriers`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<FreightCarrier>) => api.post<FreightCarrier>(`${BASE}/freight-carriers`, data).then(r => r.data),
  update: (id: string, data: Partial<FreightCarrier>) => api.put(`${BASE}/freight-carriers/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/freight-carriers/${id}`),
};

// Harbours
export const harboursApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<Harbour>>(`${BASE}/harbours`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<Harbour>) => api.post<Harbour>(`${BASE}/harbours`, data).then(r => r.data),
  update: (id: string, data: Partial<Harbour>) => api.put(`${BASE}/harbours/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/harbours/${id}`),
};

// Containers
export const containersApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<Container>>(`${BASE}/containers`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<Container>) => api.post<Container>(`${BASE}/containers`, data).then(r => r.data),
  update: (id: string, data: Partial<Container>) => api.put(`${BASE}/containers/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/containers/${id}`),
};

// Countries
export const countriesApi = {
  list: (page = 1, limit = 25) => api.get<PagedResult<Country>>(`${BASE}/countries`, { params: { page, limit } }).then(r => r.data),
  create: (data: Partial<Country>) => api.post<Country>(`${BASE}/countries`, data).then(r => r.data),
  update: (id: string, data: Partial<Country>) => api.put(`${BASE}/countries/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/countries/${id}`),
};

// Orders
export const ordersApi = {
  list: (q?: string, page = 1, limit = 25) => api.get<PagedResult<Order>>(`${BASE}/orders`, { params: { q, page, limit } }).then(r => r.data),
  create: (data: Partial<Order>) => {
    // Go time.Time requires RFC3339; HTML date inputs return "YYYY-MM-DD"
    const payload = { ...data };
    if (payload.orderDate && !payload.orderDate.includes('T')) {
      payload.orderDate = payload.orderDate + 'T00:00:00Z';
    }
    return api.post<Order>(`${BASE}/orders`, payload).then(r => r.data);
  },
  get: (id: string) => api.get<Order>(`${BASE}/orders/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Order>) => {
    const payload = { ...data };
    if (payload.orderDate && !payload.orderDate.includes('T')) {
      payload.orderDate = payload.orderDate + 'T00:00:00Z';
    }
    return api.put(`${BASE}/orders/${id}`, payload);
  },
  delete: (id: string) => api.delete(`${BASE}/orders/${id}`),
  listTasks: (id: string) => api.get<OrderTask[]>(`${BASE}/orders/${id}/tasks`).then(r => r.data),
  createTask: (id: string, data: Partial<OrderTask>) => api.post<OrderTask>(`${BASE}/orders/${id}/tasks`, data).then(r => r.data),
  updateTask: (id: string, taskId: string, data: Partial<OrderTask>) => api.put(`${BASE}/orders/${id}/tasks/${taskId}`, data),
  deleteTask: (id: string, taskId: string) => api.delete(`${BASE}/orders/${id}/tasks/${taskId}`),
  listPayments: (id: string) => api.get<OrderPayment[]>(`${BASE}/orders/${id}/payments`).then(r => r.data),
  createPayment: (id: string, data: Partial<OrderPayment>) => api.post<OrderPayment>(`${BASE}/orders/${id}/payments`, data).then(r => r.data),
  updatePayment: (id: string, paymentId: string, data: Partial<OrderPayment>) => api.put(`${BASE}/orders/${id}/payments/${paymentId}`, data),
  deletePayment: (id: string, paymentId: string) => api.delete(`${BASE}/orders/${id}/payments/${paymentId}`),
};

// Offers
export const offersApi = {
  list: (stock?: boolean, page = 1, limit = 25) => api.get<PagedResult<Offer>>(`${BASE}/offers`, { params: { stock, page, limit } }).then(r => r.data),
  create: (data: Partial<Offer>) => api.post<Offer>(`${BASE}/offers`, data).then(r => r.data),
  get: (id: string) => api.get<Offer>(`${BASE}/offers/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Offer>) => api.put(`${BASE}/offers/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/offers/${id}`),
};
