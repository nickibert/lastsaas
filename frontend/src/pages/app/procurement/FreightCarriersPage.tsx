import { useState } from 'react';
import { useHighlightRow } from '../../../hooks/useHighlightRow';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil } from 'lucide-react';
import { toast } from 'sonner';
import { freightCarriersApi, type FreightCarrier } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

export default function FreightCarriersPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [editing, setEditing] = useState<FreightCarrier | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<FreightCarrier>>({ name: '', description: '' });
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);

  const { data, isLoading } = useQuery({
    queryKey: ['freight-carriers', page, limit],
    queryFn: () => freightCarriersApi.list(page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const carriers = data?.items ?? [];
  useHighlightRow(carriers.length > 0);
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const createMutation = useMutation({
    mutationFn: freightCarriersApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['freight-carriers'] }); setShowForm(false); reset(); toast.success('Frachtführer angelegt'); },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<FreightCarrier> }) => freightCarriersApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['freight-carriers'] }); setEditing(null); reset(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });

  const deleteMutation = useMutation({
    mutationFn: freightCarriersApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['freight-carriers'] }); toast.success('Gelöscht'); },
  });

  const reset = () => setForm({ name: '', description: '' });

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Frachtführer</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Frachtführer</p>
        </div>
        <button
          onClick={() => { reset(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Frachtführer anlegen
        </button>
      </div>

      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Bearbeiten' : 'Neuer Frachtführer'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-dark-400 mb-1">Name *</label>
              <input type="text" value={form.name ?? ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-sm text-dark-400 mb-1">Beschreibung</label>
              <input type="text" value={form.description ?? ''} onChange={e => setForm(f => ({ ...f, description: e.target.value }))}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.name || createMutation.isPending || updateMutation.isPending}
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
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Beschreibung</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {carriers.length === 0 ? (
                <tr><td colSpan={3} className="px-4 py-8 text-center text-dark-400">Keine Frachtführer</td></tr>
              ) : carriers.map(c => (
                <tr key={c.id} data-highlight-id={c.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white text-sm font-medium">{c.name}</td>
                  <td className="px-4 py-3 text-dark-300 text-sm">{c.description || '—'}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button onClick={() => { window.scrollTo({ top: 0, behavior: 'smooth' }); setEditing(c); setForm({ name: c.name, description: c.description }); setShowForm(false); }}
                        className="p-1 text-dark-400 hover:text-primary-400"><Pencil className="w-4 h-4" /></button>
                      <button onClick={() => { if (confirm(`${c.name} löschen?`)) deleteMutation.mutate(c.id); }}
                        className="p-1 text-dark-400 hover:text-red-400"><Trash2 className="w-4 h-4" /></button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
