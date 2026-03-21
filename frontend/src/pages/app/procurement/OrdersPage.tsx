import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Plus, Search, Trash2, Eye } from 'lucide-react';
import { toast } from 'sonner';
import { ordersApi, suppliersApi, type Order } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

export default function OrdersPage() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [search, setSearch] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<Order>>({
    orderNumber: '',
    orderDate: new Date().toISOString().slice(0, 10),
    orderContents: '',
    orderSumUsd: 0,
  });

  const { data: orders = [], isLoading } = useQuery({
    queryKey: ['orders', search],
    queryFn: () => ordersApi.list(search || undefined),
    enabled,
    throwOnError: false,
  });

  const { data: suppliers = [] } = useQuery({
    queryKey: ['suppliers'],
    queryFn: suppliersApi.list,
    enabled,
    throwOnError: false,
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<Order>) => ordersApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders'] });
      setShowForm(false);
      setForm({ orderNumber: '', orderDate: new Date().toISOString().slice(0, 10), orderContents: '', orderSumUsd: 0 });
      toast.success('Bestellung angelegt');
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Bestellungen</h1>
          <p className="text-dark-400 mt-1">{orders.length} Bestellungen</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition-colors"
        >
          <Plus className="w-4 h-4" />
          Bestellung anlegen
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
        <input
          type="text"
          placeholder="Bestellnummer oder Inhalt suchen..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-400 focus:outline-none focus:border-primary-500"
        />
      </div>

      {/* Create form */}
      {showForm && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Neue Bestellung</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Bestellnummer *</label>
              <input
                type="text"
                value={form.orderNumber}
                onChange={e => setForm(f => ({ ...f, orderNumber: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
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
              disabled={!form.orderNumber || createMutation.isPending}
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
      {isLoading ? (
        <div className="text-dark-400">Lädt...</div>
      ) : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Bestellnummer</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Datum</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Lieferant</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Inhalt</th>
                <th className="text-right px-4 py-3 text-sm font-medium text-dark-400">Summe (USD)</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {orders.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-dark-400">
                    Keine Bestellungen gefunden
                  </td>
                </tr>
              ) : (
                orders.map(order => (
                  <tr key={order.id} className="hover:bg-dark-800/30 transition-colors">
                    <td className="px-4 py-3 text-white font-mono text-sm">{order.orderNumber}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">
                      {new Date(order.orderDate).toLocaleDateString('de-DE')}
                    </td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{supplierName(order.supplierId)}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{order.orderContents || '—'}</td>
                    <td className="px-4 py-3 text-right text-dark-300 text-sm font-mono">
                      {order.orderSumUsd.toLocaleString('de-DE', { minimumFractionDigits: 2 })}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button onClick={() => navigate(`/procurement/orders/${order.id}`)}
                          className="p-1 text-dark-400 hover:text-primary-400 transition-colors" title="Details">
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => {
                            if (confirm(`Bestellung ${order.orderNumber} löschen?`)) {
                              deleteMutation.mutate(order.id);
                            }
                          }}
                          className="p-1 text-dark-400 hover:text-red-400 transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
