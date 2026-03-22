import { useState } from 'react';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, Link } from 'react-router-dom';
import { Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { ordersApi, suppliersApi, type Order, type DeliveryStatus, type PaymentStatus, type ReceiptStatus } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';
import { DataTable, type BulkAction, type ColumnDef, type SortDir } from '../../../components/DataTable';

const deliveryBadge: Record<DeliveryStatus, string> = {
  pending: 'bg-dark-700 text-dark-300',
  ordered: 'bg-indigo-500/20 text-indigo-400',
  shipped: 'bg-blue-500/20 text-blue-400',
  arrived: 'bg-teal-500/20 text-teal-400',
  partial: 'bg-amber-500/20 text-amber-400',
};
const deliveryLabel: Record<DeliveryStatus, string> = {
  pending: 'Ausstehend',
  ordered: 'Bestellt',
  shipped: 'Verschifft',
  arrived: 'Angekommen',
  partial: 'Teillieferung',
};
const paymentBadge: Record<PaymentStatus, string> = {
  unpaid: 'bg-red-500/20 text-red-400',
  deposit_paid: 'bg-amber-500/20 text-amber-400',
  fully_paid: 'bg-emerald-500/20 text-emerald-400',
  overdue: 'bg-red-600/30 text-red-300',
};
const paymentLabel: Record<PaymentStatus, string> = {
  unpaid: 'Unbezahlt',
  deposit_paid: 'Anzahlung',
  fully_paid: 'Bezahlt',
  overdue: 'Überfällig',
};
const receiptBadge: Record<ReceiptStatus, string> = {
  pending: 'bg-dark-700 text-dark-300',
  partial: 'bg-amber-500/20 text-amber-400',
  received: 'bg-teal-500/20 text-teal-400',
  distributed: 'bg-emerald-500/20 text-emerald-400',
};
const receiptLabel: Record<ReceiptStatus, string> = {
  pending: 'Ausstehend',
  partial: 'Teileingang',
  received: 'Wareneingang',
  distributed: 'Eingelagert',
};

export default function OrdersPage() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [showForm, setShowForm] = useState(false);
  const [sortKey, setSortKey] = useState('orderDate');
  const [sortDir, setSortDir] = useState<SortDir>('desc');
  const handleSort = (key: string, dir: SortDir) => { setSortKey(key); setSortDir(dir); };
  const [form, setForm] = useState<Partial<Order>>({
    orderNumber: '',
    orderDate: new Date().toISOString().slice(0, 10),
    orderContents: '',
    orderSumUsd: 0,
  });

  const { data, isLoading } = useQuery({
    queryKey: ['orders', search, page, limit],
    queryFn: () => ordersApi.list(search || undefined, page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const orders = data?.items ?? [];
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const { data: suppliersData } = useQuery({
    queryKey: ['suppliers', 1, 500],
    queryFn: () => suppliersApi.list(1, 500),
    enabled,
    throwOnError: false,
  });

  const suppliers = suppliersData?.items ?? [];

  const createMutation = useMutation({
    mutationFn: (data: Partial<Order>) => ordersApi.create(data),
    onSuccess: (newOrder) => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Bestellung angelegt – Positionen und Kosten können jetzt hinzugefügt werden');
      navigate('/procurement/orders/' + newOrder.id);
    },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => ordersApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Bestellung gelöscht');
    },
  });

  const supplierName = (id?: string) => {
    if (!id) return '—';
    return suppliers.find(s => s.id === id)?.company ?? '—';
  };

  const orderColumns: ColumnDef<Order>[] = [
    {
      key: 'orderNumber',
      header: 'Bestellnr.',
      sortable: true,
      render: row => (
        <div>
          <div className="font-mono text-sm text-white">{row.internalNumber || row.orderNumber || '—'}</div>
          {row.internalNumber && row.orderNumber && (
            <div className="font-mono text-xs text-dark-500">{row.orderNumber}</div>
          )}
        </div>
      ),
    },
    {
      key: 'orderDate',
      header: 'Datum',
      sortable: true,
      render: row => <span className="text-dark-300 text-sm">{new Date(row.orderDate).toLocaleDateString('de-DE')}</span>,
    },
    {
      key: 'supplier',
      header: 'Lieferant',
      render: row => row.supplierId
        ? <Link to={`/procurement/suppliers?highlight=${row.supplierId}`} className="text-dark-300 hover:text-primary-400 transition-colors text-sm" onClick={e => e.stopPropagation()}>{supplierName(row.supplierId)}</Link>
        : <span className="text-dark-500">—</span>,
    },
    {
      key: 'currency',
      header: 'Währung',
      width: 'w-20',
      render: row => <span className="text-dark-300 text-sm font-mono">{row.currency || '—'}</span>,
    },
    {
      key: 'orderSumUsd',
      header: 'Betrag',
      render: row => <span className="text-dark-300 text-sm font-mono">{row.orderSumUsd.toLocaleString('de-DE', { minimumFractionDigits: 2 })}</span>,
    },
    {
      key: 'deliveryStatus',
      header: 'Lieferung',
      render: row => row.deliveryStatus
        ? <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${deliveryBadge[row.deliveryStatus]}`}>{deliveryLabel[row.deliveryStatus]}</span>
        : <span className="text-dark-500">—</span>,
    },
    {
      key: 'paymentStatus',
      header: 'Zahlung',
      render: row => row.paymentStatus
        ? <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${paymentBadge[row.paymentStatus]}`}>{paymentLabel[row.paymentStatus]}</span>
        : <span className="text-dark-500">—</span>,
    },
    {
      key: 'products',
      header: 'Positionen',
      render: row => <span className="text-dark-300 text-sm">{row.products?.length ?? 0}</span>,
    },
    {
      key: 'updatedAt',
      header: 'Geändert',
      defaultVisible: false,
      render: row => <span className="text-dark-400 text-sm">{new Date(row.updatedAt).toLocaleDateString('de-DE')}</span>,
    },
  ];

  const orderBulkActions: BulkAction<Order>[] = [
    {
      label: 'Löschen',
      icon: <Trash2 className="w-3.5 h-3.5" />,
      variant: 'danger',
      onClick: (rows) => {
        const label = rows.length === 1
          ? (rows[0].internalNumber || rows[0].orderNumber || rows[0].id)
          : `${rows.length} Bestellungen`;
        if (!confirm(`${label} löschen?`)) return;
        rows.forEach(r => deleteMutation.mutate(r.id));
      },
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Bestellungen</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Bestellungen</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Bestellung anlegen
        </button>
      </div>


      {/* Create form */}
      {showForm && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Neue Bestellung</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Lieferanten-Bestellnr. <span className="text-dark-500">(optional)</span></label>
              <input
                type="text"
                value={form.orderNumber ?? ''}
                placeholder="Wird automatisch vergeben"
                onChange={e => setForm(f => ({ ...f, orderNumber: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-600 focus:outline-none focus:border-primary-500"
              />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Datum</label>
              <input
                type="date"
                value={form.orderDate}
                onChange={e => setForm(f => ({ ...f, orderDate: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Lieferant</label>
              <select
                value={form.supplierId ?? ''}
                onChange={e => setForm(f => ({ ...f, supplierId: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              >
                <option value="">— auswählen —</option>
                {suppliers.map(s => (
                  <option key={s.id} value={s.id}>{s.company}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Bestellsumme (USD)</label>
              <input
                type="number"
                step="0.01"
                value={form.orderSumUsd}
                onChange={e => setForm(f => ({ ...f, orderSumUsd: parseFloat(e.target.value) }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm text-dark-400 mb-1">Inhalt / Beschreibung</label>
              <input
                type="text"
                value={form.orderContents}
                onChange={e => setForm(f => ({ ...f, orderContents: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => createMutation.mutate(form)}
              disabled={createMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
            >
              {createMutation.isPending ? 'Wird angelegt...' : 'Anlegen'}
            </button>
            <button
              onClick={() => setShowForm(false)}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600"
            >
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {/* Orders table */}
      <DataTable
        tableKey="orders"
        columns={orderColumns}
        data={orders}
        getRowId={r => r.id}
        searchValue={search}
        onSearchChange={v => { setSearch(v); setPage(1); }}
        searchPlaceholder="Bestellnummer oder Inhalt suchen..."
        bulkActions={orderBulkActions}
        onRowClick={row => navigate(`/procurement/orders/${row.id}`)}
        emptyMessage="Keine Bestellungen gefunden"
        isLoading={isLoading}
        sortKey={sortKey}
        sortDir={sortDir}
        onSort={handleSort}
      />
      <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
    </div>
  );
}
