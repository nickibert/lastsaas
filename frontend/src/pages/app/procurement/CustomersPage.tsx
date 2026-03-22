import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Trash2, Pencil, X } from 'lucide-react';
import { toast } from 'sonner';
import { customersApi, type Customer } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';
import { useLocalStorage } from '../../../hooks/useLocalStorage';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

function TagInput({ tags, onChange }: { tags: string[]; onChange: (t: string[]) => void }) {
  const [input, setInput] = useState('');
  const add = () => {
    const v = input.trim().toLowerCase();
    if (v && !tags.includes(v)) onChange([...tags, v]);
    setInput('');
  };
  return (
    <div>
      <label className={labelCls}>Tags</label>
      <div className="flex flex-wrap gap-1.5 mb-2">
        {tags.map(t => (
          <span key={t} className="flex items-center gap-1 px-2 py-0.5 bg-primary-500/20 text-primary-300 rounded-full text-xs">
            {t}
            <button type="button" onClick={() => onChange(tags.filter(x => x !== t))} className="hover:text-white">
              <X className="w-3 h-3" />
            </button>
          </span>
        ))}
      </div>
      <div className="flex gap-2">
        <input type="text" value={input} onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); add(); } }}
          placeholder="Tag eingeben + Enter" className={inputCls} />
        <button type="button" onClick={add} className="px-3 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">
          <Plus className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

const emptyForm = (): Partial<Customer> => ({ company: '', tags: [] });

export default function CustomersPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Customer | null>(null);
  const [form, setForm] = useState<Partial<Customer>>(emptyForm());

  const { data, isLoading } = useQuery({
    queryKey: ['customers', search, page, limit],
    queryFn: () => customersApi.list(search || undefined, page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });
  const customers = data?.items ?? [];
  const total = data?.total ?? 0;

  const createMut = useMutation({
    mutationFn: customersApi.create,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['customers'] }); setShowForm(false); setForm(emptyForm()); toast.success('Kunde angelegt'); },
    onError: () => toast.error('Fehler beim Speichern'),
  });
  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Customer> }) => customersApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['customers'] }); setEditing(null); setShowForm(false); setForm(emptyForm()); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });
  const deleteMut = useMutation({
    mutationFn: customersApi.delete,
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['customers'] }); toast.success('Gelöscht'); },
  });

  const inp = (key: keyof Customer) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm(f => ({ ...f, [key]: e.target.value }));
  const addrInp = (key: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm(f => ({ ...f, address: { ...f.address, [key]: e.target.value } }));

  const submit = () => {
    if (!form.company) return;
    if (editing) updateMut.mutate({ id: editing.id, data: form });
    else createMut.mutate(form);
  };

  const startEdit = (c: Customer) => {
    setEditing(c);
    setForm({ ...c, tags: c.tags ?? [] });
    setShowForm(true);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Kunden</h1>
          <p className="text-dark-400 mt-1">{total} Kunden</p>
        </div>
        <button onClick={() => { setForm(emptyForm()); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm">
          <Plus className="w-4 h-4" /> Kunde anlegen
        </button>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
        <input type="text" placeholder="Name, Firma oder E-Mail..." value={search}
          onChange={e => { setSearch(e.target.value); setPage(1); }}
          className="w-full pl-10 pr-4 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-400 text-sm focus:outline-none focus:border-primary-500" />
      </div>

      {showForm && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-base font-semibold text-white">{editing ? 'Kunde bearbeiten' : 'Neuer Kunde'}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="md:col-span-2">
              <label className={labelCls}>Firma *</label>
              <input type="text" value={form.company ?? ''} onChange={inp('company')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Vorname</label>
              <input type="text" value={form.firstname ?? ''} onChange={inp('firstname')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Nachname</label>
              <input type="text" value={form.lastname ?? ''} onChange={inp('lastname')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>E-Mail</label>
              <input type="email" value={form.email ?? ''} onChange={inp('email')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Telefon</label>
              <input type="text" value={form.phone ?? ''} onChange={inp('phone')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Straße</label>
              <input type="text" value={form.address?.street ?? ''} onChange={addrInp('street')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>PLZ</label>
              <input type="text" value={form.address?.zip ?? ''} onChange={addrInp('zip')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Stadt</label>
              <input type="text" value={form.address?.city ?? ''} onChange={addrInp('city')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Land</label>
              <input type="text" value={form.address?.country ?? ''} onChange={addrInp('country')} className={inputCls} />
            </div>
            <div className="md:col-span-2">
              <label className={labelCls}>Notizen</label>
              <textarea value={form.misc ?? ''} onChange={inp('misc')} rows={2}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500 resize-none" />
            </div>
            <div className="md:col-span-2">
              <TagInput tags={form.tags ?? []} onChange={tags => setForm(f => ({ ...f, tags }))} />
            </div>
          </div>
          <div className="flex gap-3 pt-2">
            <button onClick={submit} disabled={!form.company || createMut.isPending || updateMut.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
              {createMut.isPending || updateMut.isPending ? 'Speichert...' : 'Speichern'}
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); setForm(emptyForm()); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">Abbrechen</button>
          </div>
        </div>
      )}

      {isLoading ? <div className="text-dark-400">Lädt...</div> : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Firma</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Kontakt</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">E-Mail</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Ort</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Tags</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {customers.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-dark-400">Keine Kunden gefunden</td></tr>
              ) : customers.map(c => (
                <tr key={c.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white font-medium">{c.company}</td>
                  <td className="px-4 py-3 text-dark-300">
                    {[c.firstname, c.lastname].filter(Boolean).join(' ') || '—'}
                  </td>
                  <td className="px-4 py-3 text-dark-400">{c.email || '—'}</td>
                  <td className="px-4 py-3 text-dark-400">
                    {[c.address?.city, c.address?.country].filter(Boolean).join(', ') || '—'}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {(c.tags ?? []).map(t => (
                        <span key={t} className="px-1.5 py-0.5 bg-primary-500/15 text-primary-400 rounded text-xs">{t}</span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button onClick={() => startEdit(c)} className="p-1 text-dark-400 hover:text-primary-400">
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button onClick={() => { if (confirm(`${c.company} löschen?`)) deleteMut.mutate(c.id); }}
                        className="p-1 text-dark-400 hover:text-red-400">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination page={page} pages={data?.pages ?? 1} total={total} limit={limit}
            onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
