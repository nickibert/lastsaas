import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Plus, Trash2, Check, X, Calculator, Pencil, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import {
  ordersApi, suppliersApi, productsApi, containersApi, harboursApi, freightCarriersApi,
  type Order, type OrderProduct, type OrderFreight, type OrderTask, type OrderPayment, type Product,
} from '../../../api/procurement';
import { ProductSearch } from '../../../components/ProductSearch';
import { useTenant } from '../../../contexts/TenantContext';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

// Frankfurter API (EZB data, no key required)
// Returns USD per 1 EUR (= dollarRate in our model)
async function fetchEurUsdRate(date?: string): Promise<number> {
  const endpoint = date ? `https://api.frankfurter.app/${date}` : 'https://api.frankfurter.app/latest';
  const res = await fetch(`${endpoint}?from=EUR&to=USD`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const data = await res.json();
  return data.rates.USD as number;
}

function FetchRateButton({ date, label = 'Vordollarrate', onRate }: {
  date?: string;
  label?: string;
  onRate: (rate: number) => void;
}) {
  const [loading, setLoading] = useState(false);
  const handle = async () => {
    setLoading(true);
    try {
      const rate = await fetchEurUsdRate(date);
      onRate(rate);
      toast.success(`${label}: ${rate.toFixed(4)} (EZB${date ? ' ' + date : ''})`);
    } catch {
      toast.error('Kurs konnte nicht abgerufen werden (Frankfurter/EZB)');
    } finally {
      setLoading(false);
    }
  };
  return (
    <button
      type="button"
      onClick={handle}
      disabled={loading}
      title={`Aktuellen Kurs von EZB abrufen${date ? ' für ' + date : ''}`}
      className="flex-shrink-0 self-end mb-0.5 p-2 rounded-lg text-dark-400 hover:text-primary-400 hover:bg-dark-800 disabled:opacity-40 transition-colors"
    >
      <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
    </button>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-dark-900/50 border border-dark-800 rounded-xl p-6 space-y-4">
      <h2 className="text-base font-semibold text-white border-b border-dark-700 pb-2">{title}</h2>
      {children}
    </div>
  );
}

function DateInput({ label, value, onChange }: { label: string; value?: string; onChange: (v: string) => void }) {
  // Back-end returns ISO datetime, HTML date input needs YYYY-MM-DD
  const dateOnly = value ? value.slice(0, 10) : '';
  return (
    <div>
      <label className={labelCls}>{label}</label>
      <input type="date" value={dateOnly} onChange={e => onChange(e.target.value)}
        className={inputCls} />
    </div>
  );
}

function NumInput({ label, value, onChange, step = '0.01' }: { label: string; value: number; onChange: (v: number) => void; step?: string }) {
  return (
    <div>
      <label className={labelCls}>{label}</label>
      <input type="number" step={step} value={value ?? 0}
        onChange={e => onChange(parseFloat(e.target.value) || 0)}
        className={inputCls} />
    </div>
  );
}

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant && !!id;

  const { data: order, isLoading: orderLoading } = useQuery({
    queryKey: ['order', id],
    queryFn: () => ordersApi.get(id!),
    enabled,
    throwOnError: false,
  });

  const { data: suppliersData } = useQuery({ queryKey: ['suppliers', 1, 500], queryFn: () => suppliersApi.list(1, 500), enabled, throwOnError: false });
  const suppliers = suppliersData?.items ?? [];
  // productNames caches id→name for display in order product rows
  const [productNames, setProductNames] = useState<Record<string, string>>({});
  const setProductName = (id: string, p: Product) => setProductNames(prev => ({ ...prev, [id]: p.nameShort }));
  const { data: containersData } = useQuery({ queryKey: ['containers', 1, 500], queryFn: () => containersApi.list(1, 500), enabled, throwOnError: false });
  const containers = containersData?.items ?? [];
  const { data: harboursData } = useQuery({ queryKey: ['harbours', 1, 500], queryFn: () => harboursApi.list(1, 500), enabled, throwOnError: false });
  const harbours = harboursData?.items ?? [];
  const { data: freightCarriersData } = useQuery({ queryKey: ['freight-carriers', 1, 500], queryFn: () => freightCarriersApi.list(1, 500), enabled, throwOnError: false });
  const freightCarriers = freightCarriersData?.items ?? [];
  const { data: tasks = [] } = useQuery({ queryKey: ['order-tasks', id], queryFn: () => ordersApi.listTasks(id!), enabled, throwOnError: false });
  const { data: payments = [] } = useQuery({ queryKey: ['order-payments', id], queryFn: () => ordersApi.listPayments(id!), enabled, throwOnError: false });

  // Local draft state — initialized from fetched order
  const [draft, setDraft] = useState<Partial<Order>>({});
  const [draftInit, setDraftInit] = useState(false);

  if (order && !draftInit) {
    setDraft(order);
    setDraftInit(true);
  }

  // Load product names for all productIds in the order when it first loads
  useEffect(() => {
    if (!order) return;
    const ids = ((order.products ?? []) as OrderProduct[])
      .map(p => p.productId)
      .filter((id): id is string => !!id);
    if (ids.length === 0) return;
    ids.forEach(pid => {
      productsApi.get(pid).then(p => setProductName(pid, p)).catch(() => {});
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [order?.id]);

  const updateMutation = useMutation({
    mutationFn: (data: Partial<Order>) => ordersApi.update(id!, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['order', id] }); qc.invalidateQueries({ queryKey: ['orders'] }); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  // Tasks
  const [newTaskText, setNewTaskText] = useState('');
  const createTaskMutation = useMutation({
    mutationFn: (text: string) => ordersApi.createTask(id!, { text }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['order-tasks', id] }); setNewTaskText(''); toast.success('Aufgabe angelegt'); },
    onError: () => toast.error('Fehler'),
  });
  const toggleTaskMutation = useMutation({
    mutationFn: (task: OrderTask) => ordersApi.updateTask(id!, task.id, { doneAt: task.doneAt ? undefined : new Date().toISOString() }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['order-tasks', id] }),
  });
  const deleteTaskMutation = useMutation({
    mutationFn: (taskId: string) => ordersApi.deleteTask(id!, taskId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['order-tasks', id] }),
  });

  // Payments
  const [newPayment, setNewPayment] = useState<Partial<OrderPayment>>({ nr: 0, paymentAmountEur: 0, paymentDollarRate: 0, paymentFees: 0 });
  const [showPaymentForm, setShowPaymentForm] = useState(false);
  const [editingPayment, setEditingPayment] = useState<OrderPayment | null>(null);
  const createPaymentMutation = useMutation({
    mutationFn: (data: Partial<OrderPayment>) => {
      const payload = { ...data };
      if (payload.paymentDate && !payload.paymentDate.includes('T')) payload.paymentDate = payload.paymentDate + 'T00:00:00Z';
      return ordersApi.createPayment(id!, payload);
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['order-payments', id] }); setShowPaymentForm(false); setNewPayment({ nr: 0, paymentAmountEur: 0, paymentDollarRate: 0, paymentFees: 0 }); toast.success('Zahlung angelegt'); },
    onError: () => toast.error('Fehler'),
  });
  const updatePaymentMutation = useMutation({
    mutationFn: (data: OrderPayment) => {
      const payload = { ...data };
      if (payload.paymentDate && !payload.paymentDate.includes('T')) payload.paymentDate = payload.paymentDate + 'T00:00:00Z';
      return ordersApi.updatePayment(id!, data.id, payload);
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['order-payments', id] }); setEditingPayment(null); toast.success('Zahlung gespeichert'); },
    onError: () => toast.error('Fehler'),
  });
  const deletePaymentMutation = useMutation({
    mutationFn: (paymentId: string) => ordersApi.deletePayment(id!, paymentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['order-payments', id] }),
  });

  const set = (key: keyof Order, value: unknown) => setDraft(d => ({ ...d, [key]: value }));

  // Products within order
  const orderProducts: OrderProduct[] = (draft.products as OrderProduct[]) ?? [];

  const addProduct = () => {
    const p: OrderProduct = { productId: '', quantity: 1, unitPriceUsd: 0, totalPriceUsd: 0, lengthMm: 0, widthMm: 0, heightMm: 0, volumeM3: 0, weightKg: 0, credited: false, inventoryChecked: false };
    setDraft(d => ({ ...d, products: [...(d.products ?? []), p] }));
  };

  const updateProduct = (idx: number, key: keyof OrderProduct, value: unknown) => {
    setDraft(d => {
      const arr = [...((d.products ?? []) as OrderProduct[])];
      arr[idx] = { ...arr[idx], [key]: value };
      if (key === 'quantity' || key === 'unitPriceUsd') {
        arr[idx].totalPriceUsd = arr[idx].quantity * arr[idx].unitPriceUsd;
      }
      if (key === 'lengthMm' || key === 'widthMm' || key === 'heightMm') {
        arr[idx].volumeM3 = (arr[idx].lengthMm * arr[idx].widthMm * arr[idx].heightMm) / 1e9;
      }
      return { ...d, products: arr };
    });
  };

  const removeProduct = (idx: number) => {
    setDraft(d => ({ ...d, products: ((d.products ?? []) as OrderProduct[]).filter((_, i) => i !== idx) }));
  };

  const applyEKMutation = useMutation({
    mutationFn: () => ordersApi.applyEK(id!),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['products'] });
      toast.success(`EK-Preise für ${res.updated} Produkt(e) übernommen`);
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : 'Fehler';
      toast.error(msg);
    },
  });

  // Freight helper
  const freight: Partial<OrderFreight> = (draft.freight as OrderFreight) ?? {};
  const setFreight = (key: keyof OrderFreight, value: unknown) => setDraft(d => ({ ...d, freight: { ...((d.freight ?? {}) as OrderFreight), [key]: value } }));

  // Warenbezugskosten-Kalkulation (live, based on current draft values)
  // Matches legacy get_wbk() formula exactly.
  const calcEK = (() => {
    const orderSumUSD = draft.orderSumUsd ?? 0;
    const discount = draft.discount ?? 0;

    // Dollar rate: arithmetic average of payment rates; fallback to preDollarRate
    const preDollarRate = draft.preDollarRate ?? 0;
    const ratesWithValue = payments.filter(p => p.paymentDollarRate > 0);
    const dollarRateAvg = ratesWithValue.length > 0
      ? ratesWithValue.reduce((s, p) => s + p.paymentDollarRate, 0) / ratesWithValue.length
      : preDollarRate;

    const totalPaymentFees = payments.reduce((s, p) => s + p.paymentFees, 0);

    const warenwertUSD = orderSumUSD - discount;
    const warenwertEUR = dollarRateAvg > 0 ? warenwertUSD / dollarRateAvg : 0;

    // Freight costs
    const f = draft.freight as Partial<OrderFreight> | undefined;
    const fRate = (f?.dollarRate ?? 0) > 0 ? (f?.dollarRate ?? 0) : dollarRateAvg;
    const totalSeaUSD = (f?.seaFreightUsd ?? 0) + (f?.emergencyBunkerSurchargeUsd ?? 0) +
      (f?.peakSeasonSurchargeUsd ?? 0) + (f?.suezCanalAddonUsd ?? 0) + (f?.dangerPayUsd ?? 0);
    const seaFreightEUR = fRate > 0 ? totalSeaUSD / fRate : 0;
    const freightageEUR = (f?.freightageEur ?? 0) > 0 ? (f?.freightageEur ?? 0) : (f?.preFreightageEur ?? 0);
    const portFeesEUR = (f?.thcEur ?? 0) + (f?.ispsEur ?? 0) + (f?.blDocFeeEur ?? 0) + (f?.followUpFeesEur ?? 0);
    const customsEUR = (f?.customsClearanceEur ?? 0) + (f?.customsEur ?? 0);
    const totalFreightEUR = freightageEUR + seaFreightEUR + portFeesEUR + customsEUR;

    const permille = draft.transportInsurancePermille ?? 0;

    const products = (draft.products ?? []) as OrderProduct[];
    const totalVolumeM3 = products.reduce((s, p) => s + p.volumeM3 * p.quantity, 0);
    const totalQuantity = products.reduce((s, p) => s + p.quantity, 0);
    const hasAllVolumes = products.every(p => p.volumeM3 > 0);

    const rows = products.map(p => {
      const prodVolume = p.volumeM3 * p.quantity;
      const volumeFactor = (totalVolumeM3 > 0 && hasAllVolumes)
        ? prodVolume / totalVolumeM3
        : (totalQuantity > 0 ? p.quantity / totalQuantity : 0);
      const priceFactor = orderSumUSD > 0 ? p.unitPriceUsd / orderSumUSD : 0;

      const freightShare = totalFreightEUR * volumeFactor;
      const feesShare = totalPaymentFees * priceFactor;
      const wbk = (freightShare + feesShare) * (1000 + permille) / 1000;

      const productDiscount = (orderSumUSD + discount) > 0
        ? discount / (orderSumUSD + discount) * p.unitPriceUsd
        : 0;
      const unitEkEUR = dollarRateAvg > 0
        ? wbk / p.quantity + (p.unitPriceUsd - productDiscount) / dollarRateAvg
        : 0;

      return { productId: p.productId, quantity: p.quantity, unitPriceUsd: p.unitPriceUsd, volumeM3: p.volumeM3, volumeFactor, freightShare, feesShare, wbk, unitEkEUR };
    });

    const totalWBK = rows.reduce((s, r) => s + r.wbk, 0);
    const totalEUR = warenwertEUR + totalWBK;

    return { dollarRateAvg, warenwertUSD, warenwertEUR, totalSeaUSD, seaFreightEUR, freightageEUR, portFeesEUR, customsEUR, totalFreightEUR, totalPaymentFees, permille, totalWBK, totalEUR, totalVolumeM3, rows };
  })();

  if (orderLoading) return <div className="text-dark-400 p-8">Lädt...</div>;
  if (!order) return <div className="text-dark-400 p-8">Bestellung nicht gefunden</div>;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/procurement/orders" className="text-dark-400 hover:text-white transition-colors">
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-white font-mono">{order.internalNumber || order.orderNumber || '—'}</h1>
            {order.internalNumber && order.orderNumber && (
              <p className="text-dark-500 text-xs font-mono mt-0.5">Lieferant: {order.orderNumber}</p>
            )}
            <p className="text-dark-400 text-sm mt-0.5">Bestelldetails</p>
          </div>
        </div>
        <button
          onClick={() => updateMutation.mutate(draft)}
          disabled={updateMutation.isPending}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {updateMutation.isPending ? 'Speichert...' : 'Speichern'}
        </button>
      </div>

      {/* Section 1: Grunddaten */}
      <Section title="Grunddaten">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className={labelCls}>Interne Nr. (automatisch)</label>
            <input type="text" value={draft.internalNumber ?? ''} readOnly disabled
              className={`${inputCls} opacity-50 cursor-not-allowed`} placeholder="Wird beim Speichern vergeben" />
          </div>
          <div>
            <label className={labelCls}>Lieferanten-Bestellnr.</label>
            <input type="text" value={draft.orderNumber ?? ''} onChange={e => set('orderNumber', e.target.value || undefined)} className={inputCls} />
          </div>
          <DateInput label="Bestelldatum" value={draft.orderDate} onChange={v => set('orderDate', v)} />
          <div>
            <label className={labelCls}>Lieferant</label>
            <select value={draft.supplierId ?? ''} onChange={e => set('supplierId', e.target.value || undefined)} className={inputCls}>
              <option value="">— auswählen —</option>
              {suppliers.map(s => <option key={s.id} value={s.id}>{s.company}</option>)}
            </select>
          </div>
          <div>
            <label className={labelCls}>Lieferstatus</label>
            <select value={draft.deliveryStatus ?? 'pending'} onChange={e => set('deliveryStatus', e.target.value)} className={inputCls}>
              <option value="pending">Ausstehend</option>
              <option value="shipped">Verschifft</option>
              <option value="delivered">Geliefert</option>
            </select>
          </div>
          <div>
            <label className={labelCls}>Zahlstatus</label>
            <select value={draft.paymentStatus ?? 'unpaid'} onChange={e => set('paymentStatus', e.target.value)} className={inputCls}>
              <option value="unpaid">Unbezahlt</option>
              <option value="partial">Teilbezahlt</option>
              <option value="paid">Bezahlt</option>
            </select>
          </div>
          <div>
            <label className={labelCls}>Wareneingangsstatus</label>
            <select value={draft.receiptStatus ?? 'pending'} onChange={e => set('receiptStatus', e.target.value)} className={inputCls}>
              <option value="pending">Ausstehend</option>
              <option value="partial">Teileingang</option>
              <option value="received">Wareneingang</option>
              <option value="distributed">Eingelagert</option>
            </select>
          </div>
          <div className="md:col-span-2">
            <label className={labelCls}>Inhalt / Beschreibung</label>
            <input type="text" value={draft.orderContents ?? ''} onChange={e => set('orderContents', e.target.value)} className={inputCls} />
          </div>
          <div className="md:col-span-1">
            <label className={labelCls}>Notizen</label>
            <input type="text" value={draft.misc ?? ''} onChange={e => set('misc', e.target.value)} className={inputCls} />
          </div>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <NumInput label="Bestellsumme USD" value={draft.orderSumUsd ?? 0} onChange={v => set('orderSumUsd', v)} />
          <NumInput label="Transportversicherung ‰" value={draft.transportInsurancePermille ?? 0} onChange={v => set('transportInsurancePermille', v)} />
          <NumInput label="Rabatt" value={draft.discount ?? 0} onChange={v => set('discount', v)} />
          <div className="flex items-end gap-1">
            <NumInput label="Vordollarrate" value={draft.preDollarRate ?? 0} onChange={v => set('preDollarRate', v)} />
            <FetchRateButton label="Vordollarrate" onRate={v => set('preDollarRate', v)} />
          </div>
          <NumInput label="Frachtführer-Rechnung EUR" value={draft.invoiceFreightCarrierEur ?? 0} onChange={v => set('invoiceFreightCarrierEur', v)} />
        </div>
      </Section>

      {/* Section 2: Positionen */}
      <Section title="Positionen (Produkte)">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b border-dark-700">
              <tr>
                <th className="text-left py-2 pr-3 text-dark-400 font-medium">Produkt</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-20">Menge</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-28">EK-Preis USD</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-28">Gesamt USD</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-20">L (mm)</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-20">B (mm)</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-20">H (mm)</th>
                <th className="text-right py-2 pr-3 text-dark-400 font-medium w-24">Gewicht (kg)</th>
                <th className="text-center py-2 pr-3 text-dark-400 font-medium w-16">Gepr.</th>
                <th className="w-8"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {orderProducts.length === 0 && (
                <tr><td colSpan={10} className="py-4 text-center text-dark-500">Keine Positionen</td></tr>
              )}
              {orderProducts.map((op, idx) => (
                <tr key={idx} className="hover:bg-dark-800/20">
                  <td className="py-2 pr-3 min-w-[200px]">
                    <ProductSearch
                      value={op.productId}
                      currentName={op.productId ? (productNames[op.productId] ?? op.productId) : undefined}
                      onChange={(id, p) => { updateProduct(idx, 'productId', id); setProductName(id, p); }}
                      supplierId={draft.supplierId}
                    />
                  </td>
                  <td className="py-2 pr-3">
                    <input type="number" step="1" min="1" value={op.quantity}
                      onChange={e => updateProduct(idx, 'quantity', parseInt(e.target.value) || 1)}
                      className="w-full px-2 py-1 bg-dark-800 border border-dark-700 rounded text-white text-sm text-right focus:outline-none focus:border-primary-500" />
                  </td>
                  <td className="py-2 pr-3">
                    <input type="number" step="0.01" value={op.unitPriceUsd}
                      onChange={e => updateProduct(idx, 'unitPriceUsd', parseFloat(e.target.value) || 0)}
                      className="w-full px-2 py-1 bg-dark-800 border border-dark-700 rounded text-white text-sm text-right focus:outline-none focus:border-primary-500" />
                  </td>
                  <td className="py-2 pr-3 text-right text-dark-300 font-mono text-sm">
                    {op.totalPriceUsd.toLocaleString('de-DE', { minimumFractionDigits: 2 })}
                  </td>
                  {(['lengthMm', 'widthMm', 'heightMm'] as const).map(dim => (
                    <td key={dim} className="py-2 pr-3">
                      <input type="number" step="1" value={op[dim]}
                        onChange={e => updateProduct(idx, dim, parseInt(e.target.value) || 0)}
                        className="w-full px-2 py-1 bg-dark-800 border border-dark-700 rounded text-white text-sm text-right focus:outline-none focus:border-primary-500" />
                    </td>
                  ))}
                  <td className="py-2 pr-3">
                    <input type="number" step="0.01" value={op.weightKg}
                      onChange={e => updateProduct(idx, 'weightKg', parseFloat(e.target.value) || 0)}
                      className="w-full px-2 py-1 bg-dark-800 border border-dark-700 rounded text-white text-sm text-right focus:outline-none focus:border-primary-500" />
                  </td>
                  <td className="py-2 pr-3 text-center">
                    <input type="checkbox" checked={op.inventoryChecked}
                      onChange={e => updateProduct(idx, 'inventoryChecked', e.target.checked)}
                      className="w-4 h-4 rounded border-dark-600 bg-dark-800 text-primary-500" />
                  </td>
                  <td className="py-2">
                    <button onClick={() => removeProduct(idx)} className="p-1 text-dark-500 hover:text-red-400">
                      <X className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <button onClick={addProduct}
          className="flex items-center gap-2 px-3 py-1.5 text-sm bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 hover:text-white">
          <Plus className="w-4 h-4" />
          Position hinzufügen
        </button>
        {orderProducts.length > 0 && (
          <p className="text-sm text-dark-400 text-right">
            Gesamt: <span className="text-white font-mono font-medium">
              {orderProducts.reduce((s, p) => s + p.totalPriceUsd, 0).toLocaleString('de-DE', { minimumFractionDigits: 2 })} USD
            </span>
          </p>
        )}
      </Section>

      {/* Section 3: Versand */}
      <Section title="Versand / Fracht">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className={labelCls}>Container</label>
            <select value={freight.containerId ?? ''} onChange={e => setFreight('containerId', e.target.value)} className={inputCls}>
              <option value="">— auswählen —</option>
              {containers.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelCls}>Container-Nr.</label>
            <input type="text" value={freight.containerNr ?? ''} onChange={e => setFreight('containerNr', e.target.value)} className={inputCls} />
          </div>
          <div>
            <label className={labelCls}>Frachtführer</label>
            <select value={freight.freightCarrierId ?? ''} onChange={e => setFreight('freightCarrierId', e.target.value)} className={inputCls}>
              <option value="">— auswählen —</option>
              {freightCarriers.map(fc => <option key={fc.id} value={fc.id}>{fc.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelCls}>Hafen ab</label>
            <select value={freight.harbourIdFrom ?? ''} onChange={e => setFreight('harbourIdFrom', e.target.value)} className={inputCls}>
              <option value="">— auswählen —</option>
              {harbours.map(h => <option key={h.id} value={h.id}>{h.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelCls}>Hafen an</label>
            <select value={freight.harbourIdTo ?? ''} onChange={e => setFreight('harbourIdTo', e.target.value)} className={inputCls}>
              <option value="">— auswählen —</option>
              {harbours.map(h => <option key={h.id} value={h.id}>{h.name}</option>)}
            </select>
          </div>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <DateInput label="Versanddatum" value={freight.shippingDate} onChange={v => setFreight('shippingDate', v)} />
          <DateInput label="Voraussichtliche Ankunft" value={freight.estimatedArrival} onChange={v => setFreight('estimatedArrival', v)} />
          <DateInput label="Ankunft" value={freight.arrival} onChange={v => setFreight('arrival', v)} />
          <DateInput label="Avis Spediteur" value={freight.avisShipperDate} onChange={v => setFreight('avisShipperDate', v)} />
          <DateInput label="Ursprungszeugnis" value={freight.docOfOrigin} onChange={v => setFreight('docOfOrigin', v)} />
        </div>
        <div className="flex gap-6">
          {([
            ['docOfOriginChecked', 'UZ geprüft'],
            ['docOfOriginSigned', 'UZ unterschrieben'],
            ['docOfOriginShipped', 'UZ versandt'],
          ] as const).map(([key, label]) => (
            <label key={key} className="flex items-center gap-2 text-sm text-dark-300 cursor-pointer">
              <input type="checkbox" checked={!!(freight as Record<string, unknown>)[key]}
                onChange={e => setFreight(key as keyof OrderFreight, e.target.checked)}
                className="w-4 h-4 rounded border-dark-600 bg-dark-800 text-primary-500" />
              {label}
            </label>
          ))}
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <NumInput label="Seefracht USD" value={freight.seaFreightUsd ?? 0} onChange={v => setFreight('seaFreightUsd', v)} />
          <NumInput label="EBS USD" value={freight.emergencyBunkerSurchargeUsd ?? 0} onChange={v => setFreight('emergencyBunkerSurchargeUsd', v)} />
          <NumInput label="PSS USD" value={freight.peakSeasonSurchargeUsd ?? 0} onChange={v => setFreight('peakSeasonSurchargeUsd', v)} />
          <NumInput label="Suez-Kanal-Zuschlag USD" value={freight.suezCanalAddonUsd ?? 0} onChange={v => setFreight('suezCanalAddonUsd', v)} />
          <NumInput label="Gefahrgutzuschlag USD" value={freight.dangerPayUsd ?? 0} onChange={v => setFreight('dangerPayUsd', v)} />
          <NumInput label="Dollarkurs (Fracht)" value={freight.dollarRate ?? 0} onChange={v => setFreight('dollarRate', v)} step="0.00001" />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <NumInput label="Frachtsumme EUR" value={freight.freightageEur ?? 0} onChange={v => setFreight('freightageEur', v)} />
          <NumInput label="Vorauszahlung Fracht EUR" value={freight.preFreightageEur ?? 0} onChange={v => setFreight('preFreightageEur', v)} />
          <NumInput label="THC EUR" value={freight.thcEur ?? 0} onChange={v => setFreight('thcEur', v)} />
          <NumInput label="ISPS EUR" value={freight.ispsEur ?? 0} onChange={v => setFreight('ispsEur', v)} />
          <NumInput label="Konnossement-Gebühr EUR" value={freight.blDocFeeEur ?? 0} onChange={v => setFreight('blDocFeeEur', v)} />
          <NumInput label="Nachfolgegebühren EUR" value={freight.followUpFeesEur ?? 0} onChange={v => setFreight('followUpFeesEur', v)} />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <NumInput label="Zollabfertigung EUR" value={freight.customsClearanceEur ?? 0} onChange={v => setFreight('customsClearanceEur', v)} />
          <NumInput label="Zoll EUR" value={freight.customsEur ?? 0} onChange={v => setFreight('customsEur', v)} />
          <NumInput label="Zollsatz %" value={freight.customsPercent ?? 0} onChange={v => setFreight('customsPercent', v)} step="0.001" />
          <div>
            <label className={labelCls}>ZTN / Referenz</label>
            <input type="text" value={freight.ztn ?? ''} onChange={e => setFreight('ztn', e.target.value)} className={inputCls} />
          </div>
        </div>
      </Section>

      {/* Section 4: Warenbezugskosten */}
      <Section title="Warenbezugskosten & EK-Kalkulation">
        {calcEK.dollarRateAvg <= 0 ? (
          <p className="text-amber-400 text-sm">Bitte Dollarkurs (Vordollarrate) in den Grunddaten eintragen.</p>
        ) : (
          <>
            {/* Cost breakdown */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-1 text-sm">
                <h3 className="text-xs font-medium text-dark-400 uppercase tracking-wide mb-2">Kostenzusammenfassung</h3>
                <div className="flex justify-between text-dark-300">
                  <span>Warenwert (brutto)</span>
                  <span className="font-mono">{(draft.orderSumUsd ?? 0).toFixed(2)} USD</span>
                </div>
                {(draft.discount ?? 0) > 0 && (
                  <div className="flex justify-between text-dark-400">
                    <span>− Rabatt</span>
                    <span className="font-mono">−{(draft.discount ?? 0).toFixed(2)} USD</span>
                  </div>
                )}
                <div className="flex justify-between text-dark-300">
                  <span>Warenwert (netto) ÷ {calcEK.dollarRateAvg.toFixed(4)}</span>
                  <span className="font-mono">{calcEK.warenwertEUR.toFixed(2)} EUR</span>
                </div>
                <div className="border-t border-dark-700 my-1" />
                <div className="flex justify-between text-dark-300">
                  <span>Seefracht ({calcEK.totalSeaUSD.toFixed(2)} USD) ÷ {((draft.freight as Partial<OrderFreight>)?.dollarRate ?? 0) > 0 ? ((draft.freight as Partial<OrderFreight>)?.dollarRate ?? 0).toFixed(4) : calcEK.dollarRateAvg.toFixed(4)}</span>
                  <span className="font-mono">{calcEK.seaFreightEUR.toFixed(2)} EUR</span>
                </div>
                <div className="flex justify-between text-dark-300">
                  <span>Inlandsfrachtkosten</span>
                  <span className="font-mono">{calcEK.freightageEUR.toFixed(2)} EUR</span>
                </div>
                {calcEK.portFeesEUR > 0 && (
                  <div className="flex justify-between text-dark-300">
                    <span>Hafengebühren (THC, ISPS, BL, etc.)</span>
                    <span className="font-mono">{calcEK.portFeesEUR.toFixed(2)} EUR</span>
                  </div>
                )}
                {calcEK.customsEUR > 0 && (
                  <div className="flex justify-between text-dark-300">
                    <span>Zoll + Zollabfertigung</span>
                    <span className="font-mono">{calcEK.customsEUR.toFixed(2)} EUR</span>
                  </div>
                )}
                {calcEK.totalPaymentFees > 0 && (
                  <div className="flex justify-between text-dark-300">
                    <span>Zahlungsgebühren</span>
                    <span className="font-mono">{calcEK.totalPaymentFees.toFixed(2)} EUR</span>
                  </div>
                )}
                {calcEK.permille > 0 && (
                  <div className="flex justify-between text-dark-300">
                    <span>Transportversicherung ({calcEK.permille}‰ auf WBK)</span>
                    <span className="font-mono">{(calcEK.totalWBK - (calcEK.totalFreightEUR + calcEK.totalPaymentFees)).toFixed(2)} EUR</span>
                  </div>
                )}
                <div className="border-t border-dark-700 my-1" />
                <div className="flex justify-between text-white font-medium">
                  <span>Gesamt Warenbezugskosten</span>
                  <span className="font-mono">{calcEK.totalEUR.toFixed(2)} EUR</span>
                </div>
                <div className="flex justify-between text-dark-400">
                  <span>davon WBK (Fracht + Gebühren + Versicherung)</span>
                  <span className="font-mono">{calcEK.totalWBK.toFixed(2)} EUR</span>
                </div>
                <div className="flex justify-between text-dark-400">
                  <span>Gesamtvolumen (Bestellung)</span>
                  <span className="font-mono">{calcEK.totalVolumeM3.toFixed(4)} m³</span>
                </div>
              </div>

              {/* Per-product EK */}
              {calcEK.rows.length > 0 && (
                <div>
                  <h3 className="text-xs font-medium text-dark-400 uppercase tracking-wide mb-2">EK je Produkt (inkl. Fracht)</h3>
                  <div className="overflow-x-auto">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="border-b border-dark-700">
                          <th className="text-left py-1.5 pr-2 text-dark-400 font-medium">Produkt</th>
                          <th className="text-right py-1.5 pr-2 text-dark-400 font-medium">Vol. m³</th>
                          <th className="text-right py-1.5 pr-2 text-dark-400 font-medium">Anteil</th>
                          <th className="text-right py-1.5 pr-2 text-dark-400 font-medium">Fracht/Stk.</th>
                          <th className="text-right py-1.5 text-dark-400 font-medium">EK EUR/Stk.</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-dark-800/50">
                        {calcEK.rows.map((row, i) => (
                          <tr key={i} className="hover:bg-dark-800/20">
                            <td className="py-1.5 pr-2 text-dark-300 font-mono">
                              {productNames[row.productId] ?? row.productId.slice(-6)}
                            </td>
                            <td className="py-1.5 pr-2 text-right text-dark-400 font-mono">
                              {row.volumeM3.toFixed(4)}
                            </td>
                            <td className="py-1.5 pr-2 text-right text-dark-400 font-mono">
                              {(row.volumeFactor * 100).toFixed(1)}%
                            </td>
                            <td className="py-1.5 pr-2 text-right text-dark-400 font-mono">
                              {(row.wbk / row.quantity).toFixed(4)}
                            </td>
                            <td className="py-1.5 text-right font-medium font-mono text-primary-400">
                              {row.unitEkEUR.toFixed(4)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {calcEK.totalVolumeM3 === 0 && (
                    <p className="text-amber-400 text-xs mt-2">Hinweis: Kein Volumen erfasst — Frachtkosten werden nicht auf Produkte verteilt.</p>
                  )}
                </div>
              )}
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => {
                  if (calcEK.dollarRateAvg <= 0) { toast.error('Dollarkurs muss > 0 sein'); return; }
                  if (calcEK.rows.length === 0) { toast.error('Keine Produkte in der Bestellung'); return; }
                  applyEKMutation.mutate();
                }}
                disabled={applyEKMutation.isPending}
                className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-500 disabled:opacity-50 text-sm"
              >
                <Calculator className="w-4 h-4" />
                {applyEKMutation.isPending ? 'Übernehme...' : 'EK-Preise in Produkte übernehmen'}
              </button>
            </div>
          </>
        )}
      </Section>

      {/* Section 6: Zahlungen */}
      <Section title="Zahlungen">
        <div className="space-y-2">
          {payments.length === 0 && !showPaymentForm && (
            <p className="text-dark-500 text-sm">Keine Zahlungen</p>
          )}
          {payments.map(p => (
            editingPayment?.id === p.id ? (
              <div key={p.id} className="grid grid-cols-2 md:grid-cols-4 gap-3 p-3 bg-dark-800/30 rounded-lg">
                <div>
                  <label className={labelCls}>Typ</label>
                  <select value={editingPayment.nr ?? 0} onChange={e => setEditingPayment(ep => ep && ({ ...ep, nr: parseInt(e.target.value) }))} className={inputCls}>
                    <option value={0}>Anzahlung</option>
                    <option value={1}>Restzahlung</option>
                    <option value={2}>Zahlung 2</option>
                  </select>
                </div>
                <NumInput label="Betrag EUR" value={editingPayment.paymentAmountEur ?? 0} onChange={v => setEditingPayment(ep => ep && ({ ...ep, paymentAmountEur: v }))} />
                <div>
                  <label className={labelCls}>Datum</label>
                  <input type="date" value={editingPayment.paymentDate?.slice(0, 10) ?? ''}
                    onChange={e => setEditingPayment(ep => ep && ({ ...ep, paymentDate: e.target.value }))} className={inputCls} />
                </div>
                <div className="flex items-end gap-1">
                  <NumInput label="Dollarkurs" value={editingPayment.paymentDollarRate ?? 0} onChange={v => setEditingPayment(ep => ep && ({ ...ep, paymentDollarRate: v }))} />
                  <FetchRateButton
                    date={editingPayment.paymentDate?.slice(0, 10)}
                    label="Dollarkurs"
                    onRate={v => setEditingPayment(ep => ep && ({ ...ep, paymentDollarRate: v }))}
                  />
                </div>
                <NumInput label="Bankgebühren" value={editingPayment.paymentFees ?? 0} onChange={v => setEditingPayment(ep => ep && ({ ...ep, paymentFees: v }))} />
                <div className="flex items-end gap-2 col-span-2">
                  <button onClick={() => updatePaymentMutation.mutate(editingPayment)}
                    disabled={updatePaymentMutation.isPending}
                    className="px-3 py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
                    Speichern
                  </button>
                  <button onClick={() => setEditingPayment(null)}
                    className="px-3 py-2 bg-dark-700 text-white rounded-lg text-sm hover:bg-dark-600">Abbrechen</button>
                </div>
              </div>
            ) : (
              <div key={p.id} className="flex items-center gap-4 px-3 py-2 bg-dark-800/50 rounded-lg text-sm">
                <span className="text-dark-400 w-20">{p.nr === 0 ? 'Anzahlung' : `Zahlung ${p.nr}`}</span>
                <span className="text-white font-mono w-28 text-right">{p.paymentAmountEur.toLocaleString('de-DE', { minimumFractionDigits: 2 })} EUR</span>
                {p.paymentDate && <span className="text-dark-400">{new Date(p.paymentDate).toLocaleDateString('de-DE')}</span>}
                <span className="text-dark-400">Kurs: {p.paymentDollarRate}</span>
                <span className="text-dark-400">Gebühren: {p.paymentFees}</span>
                <div className="ml-auto flex items-center gap-1">
                  <button onClick={() => setEditingPayment(p)} className="p-1 text-dark-500 hover:text-primary-400"><Pencil className="w-3.5 h-3.5" /></button>
                  <button onClick={() => { if (confirm('Zahlung löschen?')) deletePaymentMutation.mutate(p.id); }}
                    className="p-1 text-dark-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5" /></button>
                </div>
              </div>
            )
          ))}
          {showPaymentForm && (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-3 bg-dark-800/30 rounded-lg">
              <div>
                <label className={labelCls}>Typ</label>
                <select value={newPayment.nr ?? 0} onChange={e => setNewPayment(p => ({ ...p, nr: parseInt(e.target.value) }))} className={inputCls}>
                  <option value={0}>Anzahlung</option>
                  <option value={1}>Restzahlung</option>
                  <option value={2}>Zahlung 2</option>
                </select>
              </div>
              <NumInput label="Betrag EUR" value={newPayment.paymentAmountEur ?? 0} onChange={v => setNewPayment(p => ({ ...p, paymentAmountEur: v }))} />
              <div>
                <label className={labelCls}>Datum</label>
                <input type="date" value={newPayment.paymentDate?.slice(0, 10) ?? ''}
                  onChange={e => setNewPayment(p => ({ ...p, paymentDate: e.target.value }))} className={inputCls} />
              </div>
              <div className="flex items-end gap-1">
                <NumInput label="Dollarkurs" value={newPayment.paymentDollarRate ?? 0} onChange={v => setNewPayment(p => ({ ...p, paymentDollarRate: v }))} />
                <FetchRateButton
                  date={newPayment.paymentDate?.slice(0, 10)}
                  label="Dollarkurs"
                  onRate={v => setNewPayment(p => ({ ...p, paymentDollarRate: v }))}
                />
              </div>
              <NumInput label="Bankgebühren" value={newPayment.paymentFees ?? 0} onChange={v => setNewPayment(p => ({ ...p, paymentFees: v }))} />
              <div className="flex items-end gap-2 col-span-2">
                <button onClick={() => createPaymentMutation.mutate(newPayment)}
                  disabled={createPaymentMutation.isPending}
                  className="px-3 py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
                  Speichern
                </button>
                <button onClick={() => setShowPaymentForm(false)}
                  className="px-3 py-2 bg-dark-700 text-white rounded-lg text-sm hover:bg-dark-600">Abbrechen</button>
              </div>
            </div>
          )}
        </div>
        {!showPaymentForm && (
          <button onClick={() => setShowPaymentForm(true)}
            className="flex items-center gap-2 px-3 py-1.5 text-sm bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 hover:text-white">
            <Plus className="w-4 h-4" />
            Zahlung hinzufügen
          </button>
        )}
      </Section>

      {/* Section 7: Aufgaben */}
      <Section title="Aufgaben">
        <div className="space-y-2">
          {tasks.length === 0 && (
            <p className="text-dark-500 text-sm">Keine Aufgaben</p>
          )}
          {tasks.map(task => {
            const isDone = task.status === 'done' || !!task.doneAt;
            const priorityCls = task.priority === 'high' ? 'text-red-400' : task.priority === 'medium' ? 'text-amber-400' : 'text-dark-500';
            const statusCls = task.status === 'done' ? 'bg-emerald-500/20 text-emerald-400' : task.status === 'in_progress' ? 'bg-blue-500/20 text-blue-400' : 'bg-dark-700 text-dark-400';
            const statusLabel = task.status === 'done' ? 'Erledigt' : task.status === 'in_progress' ? 'In Arbeit' : 'Offen';
            return (
              <div key={task.id} className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors ${isDone ? 'bg-dark-800/20' : 'bg-dark-800/50'}`}>
                <button onClick={() => toggleTaskMutation.mutate(task)}
                  className={`w-5 h-5 rounded border flex items-center justify-center flex-shrink-0 transition-colors ${isDone ? 'bg-primary-500 border-primary-500 text-white' : 'border-dark-600 hover:border-primary-400'}`}>
                  {isDone && <Check className="w-3 h-3" />}
                </button>
                <span className={`flex-1 ${isDone ? 'line-through text-dark-500' : 'text-dark-200'}`}>
                  {task.title || task.text}
                </span>
                <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${statusCls}`}>{statusLabel}</span>
                {task.priority && task.priority !== 'low' && (
                  <span className={`text-xs font-medium ${priorityCls}`}>{task.priority === 'high' ? '↑ Hoch' : '→ Mittel'}</span>
                )}
                {task.dueDate && (
                  <span className="text-dark-500 text-xs">{new Date(task.dueDate).toLocaleDateString('de-DE')}</span>
                )}
                <button onClick={() => deleteTaskMutation.mutate(task.id)}
                  className="p-1 text-dark-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5" /></button>
              </div>
            );
          })}
        </div>
        <div className="flex gap-2">
          <input type="text" value={newTaskText} onChange={e => setNewTaskText(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter' && newTaskText.trim()) createTaskMutation.mutate(newTaskText.trim()); }}
            placeholder="Neue Aufgabe..."
            className="flex-1 px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500 placeholder-dark-500" />
          <button onClick={() => { if (newTaskText.trim()) createTaskMutation.mutate(newTaskText.trim()); }}
            disabled={!newTaskText.trim() || createTaskMutation.isPending}
            className="px-3 py-2 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
            <Plus className="w-4 h-4" />
          </button>
        </div>
      </Section>

      {/* Footer save button */}
      <div className="flex justify-end pb-4">
        <button
          onClick={() => updateMutation.mutate(draft)}
          disabled={updateMutation.isPending}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {updateMutation.isPending ? 'Speichert...' : 'Speichern'}
        </button>
      </div>
    </div>
  );
}
