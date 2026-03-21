import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil } from 'lucide-react';
import { toast } from 'sonner';
import { countriesApi, type Country } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';

export default function CountriesPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [editing, setEditing] = useState<Country | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<Country>>({ name: '' });

  const { data: countries = [], isLoading } = useQuery({
    queryKey: ['countries'],
    queryFn: countriesApi.list,
    enabled,
    throwOnError: false,
  });

  const createMutation = useMutation({
    mutationFn: countriesApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['countries'] }); setShowForm(false); reset(); toast.success('Land angelegt'); },
    onError: () => toast.error('Fehler beim Anlegen'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Country> }) => countriesApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['countries'] }); setEditing(null); reset(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });

  const deleteMutation = useMutation({
    mutationFn: countriesApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['countries'] }); toast.success('Gelöscht'); },
  });

  const reset = () => setForm({ name: '' });

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Länder</h1>
          <p className="text-dark-400 mt-1">{countries.length} Länder</p>
        </div>
        <button
          onClick={() => { reset(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Land anlegen
        </button>
      </div>

      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Bearbeiten' : 'Neues Land'}</h2>
          <div className="max-w-sm">
            <label className="block text-sm text-dark-400 mb-1">Name *</label>
            <input type="text" value={form.name ?? ''} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
              className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500" />
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
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {countries.length === 0 ? (
                <tr><td colSpan={2} className="px-4 py-8 text-center text-dark-400">Keine Länder</td></tr>
              ) : countries.map(c => (
                <tr key={c.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white text-sm">{c.name}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button onClick={() => { setEditing(c); setForm({ name: c.name }); setShowForm(false); }}
                        className="p-1 text-dark-400 hover:text-primary-400"><Pencil className="w-4 h-4" /></button>
                      <button onClick={() => { if (confirm(`${c.name} löschen?`)) deleteMutation.mutate(c.id); }}
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
