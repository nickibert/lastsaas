import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Trash2, Pencil, ChevronLeft, ChevronRight } from 'lucide-react';
import { toast } from 'sonner';
import { productsApi, goodsGroupsApi, type Product } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

export default function ProductsPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [form, setForm] = useState<Partial<Product>>({ nameShort: '', nameLong: '', ean: '', wtn: '' });

  const { data, isLoading } = useQuery({
    queryKey: ['products', search, page],
    queryFn: () => productsApi.list(search || undefined, page),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const products = data?.items ?? [];
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  // Reset to page 1 on new search
  const handleSearch = (q: string) => { setSearch(q); setPage(1); };

  const { data: goodsGroups = [] } = useQuery({ queryKey: ['goods-groups'], queryFn: goodsGroupsApi.list, enabled, throwOnError: false });

  const createMutation = useMutation({
    mutationFn: productsApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); setShowForm(false); resetForm(); toast.success('Produkt angelegt'); },
    onError: () => toast.error('Fehler'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Product> }) => productsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); setEditing(null); resetForm(); toast.success('Gespeichert'); },
  });

  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['products'] }); toast.success('Gelöscht'); },
  });

  const resetForm = () => setForm({ nameShort: '', nameLong: '', ean: '', wtn: '' });

  const startEdit = (p: Product) => { setEditing(p); setForm(p); setShowForm(true); };

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Produkte</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Produkte</p>
        </div>
        <button
          onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Produkt anlegen
        </button>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
        <input
          type="text"
          placeholder="Name, Eigenname oder EAN suchen..."
          value={search}
          onChange={e => handleSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-400 focus:outline-none focus:border-primary-500"
        />
      </div>

      {showForm && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Produkt bearbeiten' : 'Neues Produkt'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Kurzname *</label>
              <input type="text" value={form.nameShort ?? ''} onChange={e => setForm(f => ({ ...f, nameShort: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Eigenname</label>
              <input type="text" value={form.ownNameShort ?? ''} onChange={e => setForm(f => ({ ...f, ownNameShort: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Langname</label>
              <input type="text" value={form.nameLong ?? ''} onChange={e => setForm(f => ({ ...f, nameLong: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">EAN</label>
              <input type="text" value={form.ean ?? ''} onChange={e => setForm(f => ({ ...f, ean: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">WTN (Zolltarifnummer)</label>
              <input type="text" value={form.wtn ?? ''} onChange={e => setForm(f => ({ ...f, wtn: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Warengruppe</label>
              <select value={form.goodsGroupId ?? ''} onChange={e => setForm(f => ({ ...f, goodsGroupId: e.target.value || undefined }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500">
                <option value="">— keine —</option>
                {goodsGroups.map(g => <option key={g.id} value={g.id}>{g.name} – {g.short}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">EK-Preis (EUR)</label>
              <input type="number" step="0.01" value={form.lastEk ?? 0} onChange={e => setForm(f => ({ ...f, lastEk: parseFloat(e.target.value) }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">VPE</label>
              <input type="number" value={form.vpe ?? 0} onChange={e => setForm(f => ({ ...f, vpe: parseInt(e.target.value) }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.nameShort || createMutation.isPending || updateMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50">
              {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600">Abbrechen</button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-dark-400">Lädt...</div>
      ) : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Kurzname</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Eigenname</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Langname</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">EAN</th>
                <th className="text-right px-4 py-3 text-sm font-medium text-dark-400">EK (EUR)</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {products.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-dark-400">Keine Produkte gefunden</td></tr>
              ) : products.map(p => (
                <tr key={p.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white text-sm font-mono">{p.nameShort}</td>
                  <td className="px-4 py-3 text-dark-300 text-sm">{p.ownNameShort || '—'}</td>
                  <td className="px-4 py-3 text-dark-400 text-sm">{p.nameLong || '—'}</td>
                  <td className="px-4 py-3 text-dark-400 text-sm font-mono">{p.ean || '—'}</td>
                  <td className="px-4 py-3 text-right text-dark-300 text-sm font-mono">
                    {p.lastEk > 0 ? p.lastEk.toFixed(2) : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button onClick={() => startEdit(p)} className="p-1 text-dark-400 hover:text-primary-400">
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button onClick={() => { if (confirm(`${p.nameShort} löschen?`)) deleteMutation.mutate(p.id); }}
                        className="p-1 text-dark-400 hover:text-red-400">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {/* Pagination */}
          {pages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t border-dark-800">
              <span className="text-sm text-dark-400">
                Seite {page} von {pages} ({total.toLocaleString('de-DE')} Einträge)
              </span>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="p-1.5 rounded-lg text-dark-400 hover:text-white hover:bg-dark-700 disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  <ChevronLeft className="w-4 h-4" />
                </button>
                {/* Page number buttons — show up to 7 around current page */}
                {Array.from({ length: Math.min(7, pages) }, (_, i) => {
                  const start = Math.max(1, Math.min(page - 3, pages - 6));
                  return start + i;
                }).map(p => (
                  <button
                    key={p}
                    onClick={() => setPage(p)}
                    className={`w-8 h-8 rounded-lg text-sm transition-colors ${
                      p === page
                        ? 'bg-primary-500 text-white'
                        : 'text-dark-400 hover:text-white hover:bg-dark-700'
                    }`}
                  >
                    {p}
                  </button>
                ))}
                <button
                  onClick={() => setPage(p => Math.min(pages, p + 1))}
                  disabled={page === pages}
                  className="p-1.5 rounded-lg text-dark-400 hover:text-white hover:bg-dark-700 disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  <ChevronRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
