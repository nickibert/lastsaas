import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil } from 'lucide-react';
import { toast } from 'sonner';
import { suppliersApi, type Supplier } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

export default function SuppliersPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Supplier | null>(null);
  const [form, setForm] = useState<Partial<Supplier>>({ company: '', firstname: '', lastname: '', email: '', origin: '' });
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(25);

  const { data, isLoading } = useQuery({
    queryKey: ['suppliers', page, limit],
    queryFn: () => suppliersApi.list(page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const suppliers = data?.items ?? [];
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const createMutation = useMutation({
    mutationFn: suppliersApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['suppliers'] }); setShowForm(false); resetForm(); toast.success('Lieferant angelegt'); },
    onError: () => toast.error('Fehler'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Supplier> }) => suppliersApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['suppliers'] }); setEditing(null); resetForm(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });

  const deleteMutation = useMutation({
    mutationFn: suppliersApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['suppliers'] }); toast.success('Gelöscht'); },
  });

  const resetForm = () => setForm({ company: '', firstname: '', lastname: '', email: '', origin: '' });

  const startEdit = (s: Supplier) => {
    setEditing(s);
    setForm({ company: s.company, firstname: s.firstname, lastname: s.lastname, email: s.email, origin: s.origin, skype: s.skype, misc: s.misc });
  };

  const submit = () => {
    if (editing) {
      updateMutation.mutate({ id: editing.id, data: form });
    } else {
      createMutation.mutate(form);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Lieferanten</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Lieferanten</p>
        </div>
        <button
          onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600"
        >
          <Plus className="w-4 h-4" />
          Lieferant anlegen
        </button>
      </div>

      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Lieferant bearbeiten' : 'Neuer Lieferant'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              { label: 'Firma *', key: 'company' },
              { label: 'Vorname', key: 'firstname' },
              { label: 'Nachname', key: 'lastname' },
              { label: 'E-Mail', key: 'email' },
              { label: 'Herkunftsland', key: 'origin' },
              { label: 'Skype', key: 'skype' },
            ].map(({ label, key }) => (
              <div key={key}>
                <label className="block text-sm text-dark-400 mb-1">{label}</label>
                <input
                  type="text"
                  value={(form as any)[key] ?? ''}
                  onChange={e => setForm(f => ({ ...f, [key]: e.target.value }))}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
                />
              </div>
            ))}
            <div className="md:col-span-2">
              <label className="block text-sm text-dark-400 mb-1">Notizen</label>
              <textarea
                value={form.misc ?? ''}
                onChange={e => setForm(f => ({ ...f, misc: e.target.value }))}
                rows={3}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
          </div>
          <div className="flex gap-3">
            <button
              onClick={submit}
              disabled={!form.company || createMutation.isPending || updateMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
            >
              {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
            </button>
            <button
              onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600"
            >
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
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Firma</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Kontakt</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">E-Mail</th>
                <th className="text-left px-4 py-3 text-sm font-medium text-dark-400">Herkunft</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {suppliers.length === 0 ? (
                <tr><td colSpan={5} className="px-4 py-8 text-center text-dark-400">Keine Lieferanten</td></tr>
              ) : (
                suppliers.map(s => (
                  <tr key={s.id} className="hover:bg-dark-800/30">
                    <td className="px-4 py-3 text-white text-sm font-medium">{s.company}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{[s.firstname, s.lastname].filter(Boolean).join(' ') || '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{s.email || '—'}</td>
                    <td className="px-4 py-3 text-dark-300 text-sm">{s.origin || '—'}</td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button onClick={() => startEdit(s)} className="p-1 text-dark-400 hover:text-primary-400">
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => { if (confirm(`${s.company} löschen?`)) deleteMutation.mutate(s.id); }}
                          className="p-1 text-dark-400 hover:text-red-400"
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
          <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
