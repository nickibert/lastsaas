import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, ArrowDownToLine, ArrowUpFromLine, BarChart3 } from 'lucide-react';
import { toast } from 'sonner';
import {
  stockApi, productsApi, ordersApi, customersApi,
  type StockMovement, type StockLevel, type Product,
} from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';
import { useLocalStorage } from '../../../hooks/useLocalStorage';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

type Tab = 'receipt' | 'issue' | 'levels';

const typeLabel: Record<string, { label: string; color: string; icon: typeof ArrowDownToLine }> = {
  receipt:    { label: 'Wareneingang', color: 'text-emerald-400 bg-emerald-500/20', icon: ArrowDownToLine },
  issue:      { label: 'Warenausgang', color: 'text-red-400 bg-red-500/20', icon: ArrowUpFromLine },
  adjustment: { label: 'Korrektur', color: 'text-amber-400 bg-amber-500/20', icon: BarChart3 },
};

const emptyForm = (type: 'receipt' | 'issue'): Partial<StockMovement> => ({
  type, quantity: 1, unit: 'Stk', movedAt: new Date().toISOString().split('T')[0],
});

export default function InventoryPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [tab, setTab] = useState<Tab>('levels');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<StockMovement>>(emptyForm('receipt'));
  const [productSearch, setProductSearch] = useState('');

  const { data: levels, isLoading: loadingLevels } = useQuery({
    queryKey: ['stock-levels', page, limit],
    queryFn: () => stockApi.listLevels({ page, limit }),
    enabled: enabled && tab === 'levels',
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const { data: movements, isLoading: loadingMovements } = useQuery({
    queryKey: ['stock-movements', tab, page, limit],
    queryFn: () => stockApi.listMovements({ type: tab as 'receipt' | 'issue', page, limit }),
    enabled: enabled && (tab === 'receipt' || tab === 'issue'),
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const { data: productsData } = useQuery({
    queryKey: ['products', productSearch, 1, 50],
    queryFn: () => productsApi.list(productSearch || undefined, 1, 50),
    enabled,
    throwOnError: false,
  });
  const products: Product[] = productsData?.items ?? [];

  const { data: ordersData } = useQuery({
    queryKey: ['orders', 1, 100],
    queryFn: () => ordersApi.list(undefined, 1, 100),
    enabled,
    throwOnError: false,
  });
  const orders = ordersData?.items ?? [];

  const { data: customersData } = useQuery({
    queryKey: ['customers', 1, 100],
    queryFn: () => customersApi.list(undefined, 1, 100),
    enabled,
    throwOnError: false,
  });
  const customers = customersData?.items ?? [];

  const createMut = useMutation({
    mutationFn: (data: Partial<StockMovement>) => {
      const norm = { ...data };
      if (typeof norm.movedAt === 'string' && !norm.movedAt.includes('T')) {
        norm.movedAt = norm.movedAt + 'T00:00:00Z';
      }
      return stockApi.createMovement(norm);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stock-movements'] });
      qc.invalidateQueries({ queryKey: ['stock-levels'] });
      setShowForm(false);
      setForm(emptyForm(form.type as 'receipt' | 'issue'));
      toast.success(`${form.type === 'receipt' ? 'Wareneingang' : 'Warenausgang'} gebucht`);
    },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const deleteMut = useMutation({
    mutationFn: stockApi.deleteMovement,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stock-movements'] });
      qc.invalidateQueries({ queryKey: ['stock-levels'] });
      toast.success('Buchung storniert');
    },
  });

  const productName = (id: string) => products.find(p => p.id === id)?.nameShort ?? id.slice(-6);
  const orderLabel = (id: string) => {
    const o = orders.find(o => o.id === id);
    return o ? (o.internalNumber || o.orderNumber || o.id.slice(-6)) : id.slice(-6);
  };
  const customerName = (id: string) => customers.find(c => c.id === id)?.company ?? id.slice(-6);

  const tabCls = (t: Tab) =>
    `px-4 py-2 text-sm font-medium rounded-lg transition-colors ${tab === t ? 'bg-primary-500/20 text-primary-400' : 'text-dark-400 hover:text-white hover:bg-dark-800/50'}`;

  const openForm = (type: 'receipt' | 'issue') => {
    setForm(emptyForm(type));
    setShowForm(true);
  };

  const items = movements?.items ?? [];
  const total = (tab === 'levels' ? levels?.total : movements?.total) ?? 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Lager</h1>
          <p className="text-dark-400 mt-1">Wareneingang, Warenausgang und Lagerbestand</p>
        </div>
      </div>

      <div className="flex gap-2">
        <button className={tabCls('levels')} onClick={() => { setTab('levels'); setPage(1); }}>
          <span className="flex items-center gap-1.5"><BarChart3 className="w-4 h-4" />Lagerbestand</span>
        </button>
        <button className={tabCls('receipt')} onClick={() => { setTab('receipt'); setPage(1); }}>
          <span className="flex items-center gap-1.5"><ArrowDownToLine className="w-4 h-4" />Wareneingang</span>
        </button>
        <button className={tabCls('issue')} onClick={() => { setTab('issue'); setPage(1); }}>
          <span className="flex items-center gap-1.5"><ArrowUpFromLine className="w-4 h-4" />Warenausgang</span>
        </button>
        {(tab === 'receipt' || tab === 'issue') && (
          <button onClick={() => openForm(tab)}
            className="ml-auto flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm">
            <Plus className="w-4 h-4" />
            {tab === 'receipt' ? 'Wareneingang' : 'Warenausgang'} buchen
          </button>
        )}
      </div>

      {showForm && (tab === 'receipt' || tab === 'issue') && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-base font-semibold text-white">
            {form.type === 'receipt' ? 'Wareneingang buchen' : 'Warenausgang buchen'}
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="md:col-span-2">
              <label className={labelCls}>Produkt *</label>
              <div className="space-y-1">
                <input type="text" placeholder="Produkt suchen..." value={productSearch}
                  onChange={e => setProductSearch(e.target.value)}
                  className={inputCls} />
                <select value={form.productId ?? ''} onChange={e => setForm(f => ({ ...f, productId: e.target.value }))}
                  className={inputCls} size={products.length > 0 && productSearch ? Math.min(5, products.length) : 1}>
                  <option value="">— Produkt wählen —</option>
                  {products.map(p => <option key={p.id} value={p.id}>{p.nameShort}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label className={labelCls}>Menge *</label>
              <input type="number" min="0" step="0.001" value={form.quantity ?? 1}
                onChange={e => setForm(f => ({ ...f, quantity: parseFloat(e.target.value) }))}
                className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Einheit</label>
              <input type="text" value={form.unit ?? 'Stk'} onChange={e => setForm(f => ({ ...f, unit: e.target.value }))}
                className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Datum</label>
              <input type="date" value={typeof form.movedAt === 'string' ? form.movedAt.slice(0, 10) : ''}
                onChange={e => setForm(f => ({ ...f, movedAt: e.target.value }))}
                className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Lagerort</label>
              <input type="text" value={form.location ?? ''} onChange={e => setForm(f => ({ ...f, location: e.target.value }))}
                className={inputCls} />
            </div>
            {form.type === 'receipt' && (
              <div>
                <label className={labelCls}>Bestellung (optional)</label>
                <select value={form.orderId ?? ''} onChange={e => setForm(f => ({ ...f, orderId: e.target.value || undefined }))}
                  className={inputCls}>
                  <option value="">— keine Bestellung —</option>
                  {orders.map(o => <option key={o.id} value={o.id}>{o.internalNumber || o.orderNumber || o.id.slice(-6)}</option>)}
                </select>
              </div>
            )}
            {form.type === 'issue' && (
              <div>
                <label className={labelCls}>Kunde (optional)</label>
                <select value={form.customerId ?? ''} onChange={e => setForm(f => ({ ...f, customerId: e.target.value || undefined }))}
                  className={inputCls}>
                  <option value="">— kein Kunde —</option>
                  {customers.map(c => <option key={c.id} value={c.id}>{c.company}</option>)}
                </select>
              </div>
            )}
            <div className="md:col-span-2">
              <label className={labelCls}>Notiz</label>
              <input type="text" value={form.notes ?? ''} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))}
                className={inputCls} />
            </div>
          </div>
          <div className="flex gap-3 pt-2">
            <button onClick={() => { if (form.productId && form.quantity) createMut.mutate(form); }}
              disabled={!form.productId || !form.quantity || createMut.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
              {createMut.isPending ? 'Bucht...' : 'Buchen'}
            </button>
            <button onClick={() => setShowForm(false)} className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {/* Lagerbestand */}
      {tab === 'levels' && (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          {loadingLevels ? <div className="p-4 text-dark-400">Lädt...</div> : (
            <table className="w-full text-sm">
              <thead className="border-b border-dark-800">
                <tr>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Produkt</th>
                  <th className="text-right px-4 py-3 text-dark-400 font-medium">Bestand</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Einheit</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Lagerort</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Zuletzt geändert</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-800/50">
                {(levels?.items ?? []).length === 0 ? (
                  <tr><td colSpan={5} className="px-4 py-8 text-center text-dark-400">Keine Lagerbestände vorhanden</td></tr>
                ) : (levels?.items ?? []).map((l: StockLevel) => (
                  <tr key={l.id} className="hover:bg-dark-800/30">
                    <td className="px-4 py-3">
                      <Link to={`/procurement/products?highlight=${l.productId}&q=${encodeURIComponent(productName(l.productId))}`} className="text-white hover:text-primary-400 transition-colors">{productName(l.productId)}</Link>
                    </td>
                    <td className={`px-4 py-3 text-right font-mono font-semibold ${l.quantity < 0 ? 'text-red-400' : l.quantity === 0 ? 'text-dark-400' : 'text-emerald-400'}`}>
                      {l.quantity.toLocaleString('de-DE', { maximumFractionDigits: 3 })}
                    </td>
                    <td className="px-4 py-3 text-dark-300">{l.unit || 'Stk'}</td>
                    <td className="px-4 py-3 text-dark-400">{l.location || '—'}</td>
                    <td className="px-4 py-3 text-dark-500 text-xs">
                      {new Date(l.updatedAt).toLocaleDateString('de-DE')}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <Pagination page={page} pages={levels?.pages ?? 1} total={total} limit={limit}
            onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}

      {/* Bewegungen */}
      {(tab === 'receipt' || tab === 'issue') && (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          {loadingMovements ? <div className="p-4 text-dark-400">Lädt...</div> : (
            <table className="w-full text-sm">
              <thead className="border-b border-dark-800">
                <tr>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Datum</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Produkt</th>
                  <th className="text-right px-4 py-3 text-dark-400 font-medium">Menge</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Einheit</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Bezug</th>
                  <th className="text-left px-4 py-3 text-dark-400 font-medium">Notiz</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-dark-800/50">
                {items.length === 0 ? (
                  <tr><td colSpan={8} className="px-4 py-8 text-center text-dark-400">Keine Buchungen vorhanden</td></tr>
                ) : items.map((m: StockMovement) => {
                  const tl = typeLabel[m.type] ?? typeLabel.receipt;
                  return (
                    <tr key={m.id} className="hover:bg-dark-800/30">
                      <td className="px-4 py-3 text-dark-300">
                        {new Date(m.movedAt).toLocaleDateString('de-DE')}
                      </td>
                      <td className="px-4 py-3">
                        <Link to={`/procurement/products?highlight=${m.productId}&q=${encodeURIComponent(productName(m.productId))}`} className="text-white hover:text-primary-400 transition-colors">{productName(m.productId)}</Link>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${tl.color}`}>{tl.label}</span>
                      </td>
                      <td className={`px-4 py-3 text-right font-mono font-semibold ${m.type === 'issue' ? 'text-red-400' : 'text-emerald-400'}`}>
                        {m.type === 'issue' ? '-' : '+'}{m.quantity.toLocaleString('de-DE', { maximumFractionDigits: 3 })}
                      </td>
                      <td className="px-4 py-3 text-dark-300">{m.unit || 'Stk'}</td>
                      <td className="px-4 py-3 text-dark-400">
                        {m.orderId
                          ? <><span className="text-dark-500">Bestellung: </span><Link to={`/procurement/orders/${m.orderId}`} className="text-dark-300 hover:text-primary-400 transition-colors">{orderLabel(m.orderId)}</Link></>
                          : m.customerId
                          ? <><span className="text-dark-500">Kunde: </span><Link to={`/procurement/customers?highlight=${m.customerId}`} className="text-dark-300 hover:text-primary-400 transition-colors">{customerName(m.customerId)}</Link></>
                          : '—'}
                      </td>
                      <td className="px-4 py-3 text-dark-500 max-w-[200px] truncate">{m.notes || '—'}</td>
                      <td className="px-4 py-3 text-right">
                        <button onClick={() => { if (confirm('Buchung stornieren?')) deleteMut.mutate(m.id); }}
                          className="p-1 text-dark-500 hover:text-red-400" title="Stornieren">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
          <Pagination page={page} pages={movements?.pages ?? 1} total={total} limit={limit}
            onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
