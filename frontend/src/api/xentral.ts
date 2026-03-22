import api from './client';

const BASE = '/procurement/integrations/xentral';

// -------------------------------------------------------------------
// Types
// -------------------------------------------------------------------

export interface XentralInstanceInfo {
  companyName: string;
  version: string;
  edition: string;
  email: string;
  phone: string;
  website: string;
  language: string;
  currency: string;
  timezone: string;
  taxId: string;
  vatId: string;
  street: string;
  zip: string;
  city: string;
  country: string;
  fetchedAt?: string;
}

export interface XentralConfig {
  id?: string;
  tenantId?: string;
  baseUrl: string;
  apiTokenMask?: string; // server-side masked token preview
  enabled: boolean;
  instanceInfo?: XentralInstanceInfo; // populated by connection test
  syncIntervalH: number; // 0 = manual only
  syncProducts: boolean;
  syncCustomers: boolean;
  syncSuppliers: boolean;
  syncOrders: boolean;
  lastSyncProducts?: string;
  lastSyncCustomers?: string;
  lastSyncSuppliers?: string;
  lastSyncOrders?: string;
  updatedAt?: string;
}

export interface XentralConfigSavePayload extends XentralConfig {
  apiToken?: string; // only send when changing; empty = keep existing
}

export interface XentralSyncLog {
  id: string;
  tenantId: string;
  entity: 'products' | 'customers' | 'suppliers' | 'orders';
  status: 'running' | 'success' | 'partial' | 'error' | 'skipped';
  created: number;
  updated: number;
  skipped: number;
  errors?: string[];
  startedAt: string;
  finishedAt?: string;
}

export interface XentralMapping {
  id: string;
  entity: string;
  localId: string;
  xentralId: string;
  xentralNr?: string;
  createdAt: string;
  updatedAt: string;
}

// -------------------------------------------------------------------
// API
// -------------------------------------------------------------------

export const xentralApi = {
  getConfig: () =>
    api.get<XentralConfig>(BASE).then(r => r.data),

  saveConfig: (data: XentralConfigSavePayload) =>
    api.put(BASE, data),

  testConnection: () =>
    api.post<{ ok: boolean; error?: string; instanceInfo?: XentralInstanceInfo }>(`${BASE}/test`).then(r => r.data),

  importAccount: () =>
    api.post<{ created: boolean; supplierId: string }>(`${BASE}/import-account`).then(r => r.data),

  syncEntity: (entity: 'products' | 'customers' | 'suppliers' | 'orders') =>
    api.post<XentralSyncLog>(`${BASE}/sync/${entity}`).then(r => r.data),

  syncAll: () =>
    api.post<XentralSyncLog[]>(`${BASE}/sync`).then(r => r.data),

  getLogs: () =>
    api.get<XentralSyncLog[]>(`${BASE}/logs`).then(r => r.data),

  getMappings: (entity?: string) =>
    api.get<XentralMapping[]>(`${BASE}/mappings`, { params: entity ? { entity } : {} }).then(r => r.data),
};
