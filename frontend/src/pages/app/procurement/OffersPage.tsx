import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil } from 'lucide-react';
import { toast } from 'sonner';
import { offersApi, suppliersApi, productsApi, type Offer } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

export default function OffersPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [editing, setEditing] = useState<Offer | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<Offer>>({ nameShort: '', description: '', priceUsd: 0, quantity: 0, isStockOffer: false, misc: '' });

  const { data: offers = [], isLoading } = useQuery({
    queryKey: ['offers'],
    queryFn: () => offersApi.list(),
    enabled,
    throwOnError: false,
  });

  const { data: suppliers = [] } = useQuery({
    queryKey: ['suppliers'],
    queryFn: suppliersApi.list,
    enabled,
    throwOnError: false,
  });

  const { data: productsData } = useQuery({
    queryKey: ['products'],
    queryFn: () => productsApi.list(),
    enabled,
    throwOnError: false,
  });
  const products = productsData?.items ?? [];

  const createMutation = useMutation({
    mutationFn: offersApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['offers'] }); setShowForm(false); reset(); toast.success('Angebot angelegt'); },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Offer> }) => offersApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['offers'] }); setEditing(null); reset(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });

  const deleteMutation = useMutation({
    mutationFn: offersApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['offers'] }); toast.success('Gelöscht'); },
  });

  const reset = () => setForm({ nameShort: '', description: '', priceUsd: 0, quantity: 0, isStockOffer: false, misc: '' });

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  const supplierName = (id?: string) => suppliers.find(s => s.id === id)?.company ?? '—';
  const productName = (id?: string) => products.find(p => p.id === id)?.nameShort ?? '—';

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Angebote</h1>
          <p className="text-dark-400 mt-1">{offers.length} Angebote</p>
        </div>
        <button
          onClick={() => { reset(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Angebot anlegen
        </button>
      </div>

      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Bearbeiten' : 'Neues Angebot'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input type="text" value={form.nameShort ?? ''} onChange={e => setForm(f => ({ ...f, nameShort: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Lieferant</label>
              <select value={form.supplierId ?? ''} onChange={e => setForm(f => ({ ...f, supplierId: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500">
                <option value="">— auswählen —</option>
                {suppliers.map(s => <option key={s.id} value={s.id}>{s.company}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Produkt</label>
              <select value={form.productId ?? ''} onChange={e => setForm(f => ({ ...f, productId: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500">
                <option value="">— auswählen —</option>
                {products.map(p => <option key={p.id} value={p.id}>{p.nameShort}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Preis (USD)</label>
              <input type="number" step="0.01" value={form.priceUsd ?? 0} onChange={e => setForm(f => ({ ...f, priceUsd: parseFloat(e.target.value) || 0 }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Menge</label>
              <input type="number" value={form.quantity ?? 0} onChange={e => setForm(f => ({ ...f, quantity: parseInt(e.target.value) || 0 }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Gültig bis</label>
              <input type="date" value={form.validUntil ?? ''} onChange={e => setForm(f => ({ ...f, validUntil: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm text-dark-400 mb-1">Beschreibung</label>
              <input type="text" value={form.description ?? ''} onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div className="flex items-center gap-2">
              <input type="checkbox" id="isStockOffer" checked={form.isStockOffer ?? false} onChange={e => setForm(f => ({ ...f, isStockOffer: e.target.checked }))}
                className="w-4 h-4 rounded border-dark-700 bg-dark-800 text-primary-500" />
              <label htmlFor="isStockOffer" className="text-sm text-dark-300">Lagerangebot</label>
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.nameShort || createMutation.isPending || updateMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50">
              {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); reset(); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600">Abbrechen</button>
          </div>
        </div>
      )}

      {isLoading ? <div className="text-dark-400">Lädt...</div> : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Name</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Lieferant</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Produkt</th>
                <th className="text-right px-4 py-3 text-sm font-medium text-dark-400">Preis (USD)</th>
                <th className="text-right px-4 py-3 text-sm font-medium text-dark-400">Menge</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {offers.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-dark-400">Keine Angebote</td></tr>
              ) : offers.map(o => (
                <tr key={o.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white text-sm font-medium">{o.nameShort}</td>
                  <td className="px-4 py-3 text-dark-300 text-sm">{supplierName(o.supplierId)}</td>
                  <td className="px-4 py-3 text-dark-300 text-sm">{productName(o.productId)}</td>
                  <td className="px-4 py-3 text-right text-dark-300 text-sm font-mono">
                    {o.priceUsd.toLocaleString('de-DE', { minimumFractionDigits: 2 })}
                  </td>
                  <td className="px-4 py-3 text-right text-dark-300 text-sm">{o.quantity}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button onClick={() => { setEditing(o); setForm({ nameShort: o.nameShort, description: o.description, priceUsd: o.priceUsd, quantity: o.quantity, isStockOffer: o.isStockOffer, supplierId: o.supplierId, productId: o.productId, misc: o.misc }); setShowForm(false); }}
                        className="p-1 text-dark-400 hover:text-primary-400"><Pencil className="w-4 h-4" /></button>
                      <button onClick={() => { if (confirm(`${o.nameShort} löschen?`)) deleteMutation.mutate(o.id); }}
                        className="p-1 text-dark-400 hover:text-red-400"><Trash2 className="w-4 h-4" /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
