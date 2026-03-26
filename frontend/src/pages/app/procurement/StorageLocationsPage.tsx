import { useState } from 'react';
import { useHighlightRow } from '../../../hooks/useHighlightRow';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil, Search, X } from 'lucide-react';
import { toast } from 'sonner';
import { storageLocationsApi, warehousesApi, type StorageLocation } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

const emptyForm = (): Partial<StorageLocation> => ({ name: '', warehouseId: '', aisle: '', rack: '', level: '', active: true });

export default function StorageLocationsPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<StorageLocation | null>(null);
  const [form, setForm] = useState<Partial<StorageLocation>>(emptyForm());
  const [search, setSearch] = useState('');
  const [warehouseFilter, setWarehouseFilter] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);

  const { data, isLoading } = useQuery({
    queryKey: ['storage-locations', search, warehouseFilter, page, limit],
    queryFn: () => storageLocationsApi.list(search || undefined, warehouseFilter || undefined, undefined, page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  // Load all warehouses for the dropdown (max 500)
  const { data: warehouseData } = useQuery({
    queryKey: ['warehouses', '', '', 1, 500],
    queryFn: () => warehousesApi.list(undefined, undefined, undefined, 1, 500),
    enabled,
    throwOnError: false,
  });
  const allWarehouses = warehouseData?.items ?? [];
  const warehouseMap = Object.fromEntries(allWarehouses.map(w => [w.id, w.name]));

  const locations = data?.items ?? [];
  useHighlightRow(locations.length > 0);
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const createMutation = useMutation({
    mutationFn: storageLocationsApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['storage-locations'] }); setShowForm(false); resetForm(); toast.success('Lagerplatz angelegt'); },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<StorageLocation> }) => storageLocationsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['storage-locations'] }); setEditing(null); resetForm(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler beim Speichern'),
  });

  const deleteMutation = useMutation({
    mutationFn: storageLocationsApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['storage-locations'] }); toast.success('Gelöscht'); },
    onError: () => toast.error('Fehler beim Löschen'),
  });

  const resetForm = () => setForm(emptyForm());

  const startEdit = (loc: StorageLocation) => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
    setEditing(loc);
    setForm({ name: loc.name, warehouseId: loc.warehouseId, aisle: loc.aisle ?? '', rack: loc.rack ?? '', level: loc.level ?? '', active: loc.active });
  };

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  const handleSearch = (v: string) => { setSearch(v); setPage(1); };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Lagerplätze</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Lagerplätze</p>
        </div>
        <button
          onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Lagerplatz anlegen
        </button>
      </div>

      {/* Search + warehouse filter */}
      <div className="flex gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
          <input
            type="text"
            value={search}
            onChange={e => handleSearch(e.target.value)}
            placeholder="Name, Gang oder Regal suchen..."
            className="w-full pl-9 pr-8 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder:text-dark-500 focus:outline-none focus:border-primary-500"
          />
          {search && (
            <button onClick={() => handleSearch('')} className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-400 hover:text-white">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>
        <select
          value={warehouseFilter}
          onChange={e => { setWarehouseFilter(e.target.value); setPage(1); }}
          className="px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500 min-w-44"
        >
          <option value="">Alle Lager</option>
          {allWarehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
        </select>
      </div>

      {/* Create / Edit form */}
      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Lagerplatz bearbeiten' : 'Neuer Lagerplatz'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input type="text" value={form.name ?? ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Lager *</label>
              <select value={form.warehouseId ?? ''} onChange={e => setForm(f => ({ ...f, warehouseId: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500">
                <option value="">Lager auswählen...</option>
                {allWarehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Gang</label>
              <input type="text" value={form.aisle ?? ''} onChange={e => setForm(f => ({ ...f, aisle: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Regal</label>
              <input type="text" value={form.rack ?? ''} onChange={e => setForm(f => ({ ...f, rack: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Ebene</label>
              <input type="text" value={form.level ?? ''} onChange={e => setForm(f => ({ ...f, level: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div className="flex items-center gap-2 self-end pb-1">
              <input type="checkbox" id="sl-active" checked={form.active ?? true} onChange={e => setForm(f => ({ ...f, active: e.target.checked }))}
                className="w-4 h-4 accent-primary-500" />
              <label htmlFor="sl-active" className="text-sm text-dark-300">Aktiv</label>
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.name || !form.warehouseId || createMutation.isPending || updateMutation.isPending}
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
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Lager</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Gang</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Regal</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Ebene</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Status</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {locations.length === 0 ? (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-dark-400">Keine Lagerplätze gefunden</td></tr>
              ) : (
                locations.map(loc => (
                  <tr key={loc.id} data-highlight-id={loc.id} className="hover:bg-dark-800/30">
                    <td className="px-4 py-3 text-white text-sm font-medium">{loc.name}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{warehouseMap[loc.warehouseId] ?? '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{loc.aisle || '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{loc.rack || '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{loc.level || '—'}</td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-0.5 rounded text-xs ${loc.active ? 'bg-green-500/20 text-green-400' : 'bg-dark-700 text-dark-400'}`}>
                        {loc.active ? 'Aktiv' : 'Inaktiv'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button onClick={() => startEdit(loc)} className="p-1 text-dark-400 hover:text-primary-400">
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button onClick={() => { if (confirm(`${loc.name} löschen?`)) deleteMutation.mutate(loc.id); }}
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
