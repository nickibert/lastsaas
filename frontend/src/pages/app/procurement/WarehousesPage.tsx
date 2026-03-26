import { useState } from 'react';
import { useHighlightRow } from '../../../hooks/useHighlightRow';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil, Search, X, MapPin } from 'lucide-react';
import { toast } from 'sonner';
import { warehousesApi, type Warehouse, type CustomerAddress } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

const emptyAddress = (): CustomerAddress => ({ street: '', city: '', zip: '', country: '' });
const emptyForm = (): Partial<Warehouse> => ({ name: '', shortName: '', description: '', active: true, address: emptyAddress() });

export default function WarehousesPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Warehouse | null>(null);
  const [form, setForm] = useState<Partial<Warehouse>>(emptyForm());
  const [search, setSearch] = useState('');
  const [countryFilter, setCountryFilter] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);

  const { data, isLoading } = useQuery({
    queryKey: ['warehouses', search, countryFilter, page, limit],
    queryFn: () => warehousesApi.list(search || undefined, countryFilter || undefined, undefined, page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const warehouses = data?.items ?? [];
  useHighlightRow(warehouses.length > 0);
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const createMutation = useMutation({
    mutationFn: warehousesApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['warehouses'] }); setShowForm(false); resetForm(); toast.success('Lager angelegt'); },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Warehouse> }) => warehousesApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['warehouses'] }); setEditing(null); resetForm(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const deleteMutation = useMutation({
    mutationFn: warehousesApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['warehouses'] }); toast.success('Gelöscht'); },
    onError: () => toast.error('Fehler beim Löschen'),
  });

  const resetForm = () => { setForm(emptyForm()); };

  const startEdit = (w: Warehouse) => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
    setEditing(w);
    setForm({
      name: w.name,
      shortName: w.shortName ?? '',
      description: w.description ?? '',
      active: w.active,
      address: { street: w.address?.street ?? '', city: w.address?.city ?? '', zip: w.address?.zip ?? '', country: w.address?.country ?? '' },
    });
  };

  const setAddr = (field: keyof CustomerAddress, value: string) =>
    setForm(f => ({ ...f, address: { ...emptyAddress(), ...f.address, [field]: value } }));

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  const handleSearch = (v: string) => { setSearch(v); setPage(1); };
  const handleCountry = (v: string) => { setCountryFilter(v); setPage(1); };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Lager</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Lager</p>
        </div>
        <button
          onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Lager anlegen
        </button>
      </div>

      {/* Search + filters */}
      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
          <input
            type="text"
            value={search}
            onChange={e => handleSearch(e.target.value)}
            placeholder="Name oder Beschreibung suchen..."
            className="w-full pl-9 pr-8 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder:text-dark-500 focus:outline-none focus:border-primary-500"
          />
          {search && (
            <button onClick={() => handleSearch('')} className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-400 hover:text-white">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
        <div className="relative">
          <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
          <input
            type="text"
            value={countryFilter}
            onChange={e => handleCountry(e.target.value)}
            placeholder="Land filtern..."
            className="pl-9 pr-8 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder:text-dark-500 focus:outline-none focus:border-primary-500 w-44"
          />
          {countryFilter && (
            <button onClick={() => handleCountry('')} className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-400 hover:text-white">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Create / Edit form */}
      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Lager bearbeiten' : 'Neues Lager'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input type="text" value={form.name ?? ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Kurzname</label>
              <input type="text" value={form.shortName ?? ''} onChange={e => setForm(f => ({ ...f, shortName: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm text-dark-400 mb-1">Beschreibung</label>
              <input type="text" value={form.description ?? ''} onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
          </div>

          <div>
            <p className="text-sm font-medium text-dark-300 mb-2">Adresse <span className="text-dark-500 font-normal">(für steuerliche Zuordnung)</span></p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <label className="block text-sm text-dark-400 mb-1">Straße</label>
                <input type="text" value={form.address?.street ?? ''} onChange={e => setAddr('street', e.target.value)}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
              </div>
              <div>
                <label className="block text-sm text-dark-400 mb-1">PLZ</label>
                <input type="text" value={form.address?.zip ?? ''} onChange={e => setAddr('zip', e.target.value)}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
              </div>
              <div>
                <label className="block text-sm text-dark-400 mb-1">Stadt</label>
                <input type="text" value={form.address?.city ?? ''} onChange={e => setAddr('city', e.target.value)}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
              </div>
              <div>
                <label className="block text-sm text-dark-400 mb-1">Land <span className="text-amber-400">*</span></label>
                <input type="text" value={form.address?.country ?? ''} onChange={e => setAddr('country', e.target.value)}
                  placeholder="z.B. DE, AT, CH"
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
              </div>
              <div className="flex items-center gap-2 self-end pb-1">
                <input type="checkbox" id="wh-active" checked={form.active ?? true} onChange={e => setForm(f => ({ ...f, active: e.target.checked }))}
                  className="w-4 h-4 accent-primary-500" />
                <label htmlFor="wh-active" className="text-sm text-dark-300">Aktiv</label>
              </div>
            </div>
          </div>

          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.name || createMutation.isPending || updateMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50">
              {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600">
              Abbrechen
            </button>
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
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Name</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Kurzname</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Adresse</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Land</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Status</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {warehouses.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-dark-400">Keine Lager gefunden</td></tr>
              ) : (
                warehouses.map(w => (
                  <tr key={w.id} data-highlight-id={w.id} className="hover:bg-dark-800/30">
                    <td className="px-4 py-3 text-white text-sm font-medium">{w.name}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{w.shortName || '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">
                      {[w.address?.street, w.address?.zip, w.address?.city].filter(Boolean).join(', ') || '—'}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {w.address?.country
                        ? <span className="px-2 py-0.5 bg-dark-700 rounded text-dark-200 text-xs font-mono">{w.address.country}</span>
                        : <span className="text-amber-400 text-xs">Kein Land</span>}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-0.5 rounded text-xs ${w.active ? 'bg-green-500/20 text-green-400' : 'bg-dark-700 text-dark-400'}`}>
                        {w.active ? 'Aktiv' : 'Inaktiv'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button onClick={() => startEdit(w)} className="p-1 text-dark-400 hover:text-primary-400">
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button onClick={() => { if (confirm(`${w.name} löschen?`)) deleteMutation.mutate(w.id); }}
                          className="p-1 text-dark-400 hover:text-red-400">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
          <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
