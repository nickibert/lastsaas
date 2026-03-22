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
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface CustomerAddress {
  street?: string;
  city?: string;
  zip?: string;
  country?: string;
}

export interface Customer {
  id: string;
  company: string;
  firstname?: string;
  lastname?: string;
  email?: string;
  phone?: string;
  address?: CustomerAddress;
  misc?: string;
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

export type StockMovementType = 'receipt' | 'issue' | 'adjustment';

export interface StockMovement {
  id: string;
  productId: string;
  orderId?: string;
  customerId?: string;
  type: StockMovementType;
  quantity: number;
  unit?: string;
  location?: string;
  notes?: string;
  processedBy: string;
  movedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface StockLevel {
  id: string;
  productId: string;
  quantity: number;
  unit?: string;
  location?: string;
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
  supplierIds?: string[];
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
  tags?: string[];
  attributes?: ProductAttribute[];
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
  iso2?: string;
  iso3?: string;
  isoNumeric?: string;
  currency?: string;
  currencyCode?: string;
  currencySymbol?: string;
  phoneCode?: string;
  region?: string;
  capital?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ProductAttribute {
  key: string;
  value: string;
  unit?: string;
}

export interface ProductPriceList {
  id: string;
  productId: string;
  type: 'EK' | 'VK';
  name: string;
  price: number;
  currency: string;
  validFrom?: string;
  validTo?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CalendarTask {
  id: string;
  orderId: string;
  orderNumber: string;
  text: string;
  dueDate: string;
  doneAt?: string;
}

export interface CalendarArrival {
  orderId: string;
  orderNumber: string;
  estimatedDate: string;
  actualDate?: string;
  supplierName?: string;
}

export type CalendarEntryType = 'reminder' | 'milestone' | 'appointment' | 'deadline';

export interface CalendarEntry {
  id: string;
  title: string;
  notes?: string;
  date: string;
  entryType: CalendarEntryType;
  createdAt: string;
  updatedAt: string;
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
  // Sea freight (USD)
  seaFreightUsd: number;
  emergencyBunkerSurchargeUsd: number;
  peakSeasonSurchargeUsd: number;
  suezCanalAddonUsd: number;
  dangerPayUsd: number;
  dollarRate: number;
  // Domestic / port costs (EUR)
  freightageEur: number;
  preFreightageEur: number;
  thcEur: number;
  ispsEur: number;
  blDocFeeEur: number;
  followUpFeesEur: number;
  // Customs (EUR)
  customsClearanceEur: number;
  customsEur: number;
  customsPercent: number;
  ztn: string;
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

export type DeliveryStatus = 'pending' | 'ordered' | 'shipped' | 'arrived' | 'partial';
export type PaymentStatus = 'unpaid' | 'deposit_paid' | 'fully_paid' | 'overdue';
export type ReceiptStatus = 'pending' | 'partial' | 'received' | 'distributed';

export interface Order {
  id: string;
  supplierId?: string;
  customerId?: string;
  orderNumber?: string;
  internalNumber?: string;
  orderDate: string;
  orderContents: string;
  misc: string;
  orderSumUsd: number;
  transportInsurancePermille: number;
  discount: number;
  preDollarRate: number;
  invoiceFreightCarrierEur: number;
  deliveryStatus?: DeliveryStatus;
  paymentStatus?: PaymentStatus;
  receiptStatus?: ReceiptStatus;
  tags?: string[];
  products: OrderProduct[];
  freights?: OrderFreight[];
  createdAt: string;
  updatedAt: string;
}

export type TaskStatus = 'open' | 'in_progress' | 'done';
export type TaskPriority = 'low' | 'medium' | 'high';

export interface OrderTask {
  id: string;
  orderId: string;
  title?: string;
  text: string;
  status?: TaskStatus;
  priority?: TaskPriority;
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
  tags?: string[];
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
  list: (q?: string, page = 1, limit = 25, supplierId?: string, goodsGroupId?: string, tag?: string, attrKey?: string, attrValue?: string) =>
    api.get<ProductsPage>(`${BASE}/products`, { params: { q, page, limit, supplierId, goodsGroupId, tag, attrKey, attrValue } }).then(r => r.data),
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
  seed: () => api.post<{ inserted?: number; existing?: number; message?: string }>(`${BASE}/countries/seed`).then(r => r.data),
};

// Product price lists
export const productPriceListsApi = {
  list: (productId: string) => api.get<ProductPriceList[]>(`${BASE}/products/${productId}/price-lists`).then(r => r.data),
  create: (productId: string, data: Partial<ProductPriceList>) => api.post<ProductPriceList>(`${BASE}/products/${productId}/price-lists`, data).then(r => r.data),
  update: (productId: string, id: string, data: Partial<ProductPriceList>) => api.put(`${BASE}/products/${productId}/price-lists/${id}`, data),
  delete: (productId: string, id: string) => api.delete(`${BASE}/products/${productId}/price-lists/${id}`),
};

// Calendar
export const calendarApi = {
  get: (from?: string, to?: string) =>
    api.get<{ tasks: CalendarTask[]; arrivals: CalendarArrival[]; entries: CalendarEntry[] }>(`${BASE}/calendar`, { params: { from, to } }).then(r => r.data),
};

// Calendar entries (manual)
export const calendarEntriesApi = {
  create: (data: Partial<CalendarEntry>) => api.post<CalendarEntry>(`${BASE}/calendar/entries`, data).then(r => r.data),
  update: (id: string, data: Partial<CalendarEntry>) => api.put(`${BASE}/calendar/entries/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/calendar/entries/${id}`),
};

// Orders
const normDate = (v: string | undefined) => v ? (v.includes('T') ? v : v + 'T00:00:00Z') : undefined;

const normalizeOrderDates = (data: Partial<Order>): Partial<Order> => {
  const p = { ...data };
  if (p.orderDate && !p.orderDate.includes('T')) p.orderDate = p.orderDate + 'T00:00:00Z';
  if (p.freights) {
    p.freights = p.freights.map(f => ({
      ...f,
      shippingDate: normDate(f.shippingDate),
      estimatedArrival: normDate(f.estimatedArrival),
      arrival: normDate(f.arrival),
      avisShipperDate: normDate(f.avisShipperDate),
      docOfOrigin: normDate(f.docOfOrigin),
    }));
  }
  return p;
};

export const ordersApi = {
  list: (q?: string, page = 1, limit = 25) => api.get<PagedResult<Order>>(`${BASE}/orders`, { params: { q, page, limit } }).then(r => r.data),
  applyEK: (id: string) => api.post<{ updated: number }>(`${BASE}/orders/${id}/apply-ek`).then(r => r.data),
  create: (data: Partial<Order>) => api.post<Order>(`${BASE}/orders`, normalizeOrderDates(data)).then(r => r.data),
  get: (id: string) => api.get<Order>(`${BASE}/orders/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Order>) => api.put(`${BASE}/orders/${id}`, normalizeOrderDates(data)),
  patchFreightDate: (id: string, index: number, field: string, date: string) =>
    api.patch(`${BASE}/orders/${id}/freight-date`, { index, field, date: date === '' ? '' : (date.includes('T') ? date : date + 'T00:00:00Z') }),
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

export interface TaskTemplate {
  id: string;
  text: string;
  daysAfter: number;
  phase: 1 | 2; // 1=order placed, 2=shipped
  createdAt: string;
  updatedAt: string;
}

// Task templates
export const taskTemplatesApi = {
  list: () => api.get<TaskTemplate[]>(`${BASE}/task-templates`).then(r => r.data),
  create: (data: Partial<TaskTemplate>) => api.post<TaskTemplate>(`${BASE}/task-templates`, data).then(r => r.data),
  update: (id: string, data: Partial<TaskTemplate>) => api.put(`${BASE}/task-templates/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/task-templates/${id}`),
};

// Customers
export const customersApi = {
  list: (q?: string, page = 1, limit = 25) => api.get<PagedResult<Customer>>(`${BASE}/customers`, { params: { q, page, limit } }).then(r => r.data),
  create: (data: Partial<Customer>) => api.post<Customer>(`${BASE}/customers`, data).then(r => r.data),
  get: (id: string) => api.get<Customer>(`${BASE}/customers/${id}`).then(r => r.data),
  update: (id: string, data: Partial<Customer>) => api.put(`${BASE}/customers/${id}`, data),
  delete: (id: string) => api.delete(`${BASE}/customers/${id}`),
};

// Stock movements & levels
export const stockApi = {
  listMovements: (params?: { productId?: string; type?: StockMovementType; page?: number; limit?: number }) =>
    api.get<PagedResult<StockMovement>>(`${BASE}/stock/movements`, { params }).then(r => r.data),
  createMovement: (data: Partial<StockMovement>) => api.post<StockMovement>(`${BASE}/stock/movements`, data).then(r => r.data),
  deleteMovement: (id: string) => api.delete(`${BASE}/stock/movements/${id}`),
  listLevels: (params?: { productId?: string; nonZero?: boolean; page?: number; limit?: number }) =>
    api.get<PagedResult<StockLevel>>(`${BASE}/stock/levels`, { params }).then(r => r.data),
};
