import { useState } from 'react';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2, Pencil, Download } from 'lucide-react';
import { toast } from 'sonner';
import { countriesApi, type Country } from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

const REGIONS = ['Africa', 'Americas', 'Asia', 'Europe', 'Oceania'];

export default function CountriesPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [editing, setEditing] = useState<Country | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Partial<Country>>({ name: '', currency: 'EUR', currencyCode: 'EUR' });
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [filterRegion, setFilterRegion] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['countries', page, limit],
    queryFn: () => countriesApi.list(page, limit),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const countries = data?.items ?? [];
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const filtered = filterRegion ? countries.filter(c => c.region === filterRegion) : countries;

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

  const seedMutation = useMutation({
    mutationFn: countriesApi.seed,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['countries'] });
      if (res.message === 'already seeded') {
        toast.info(`Bereits ${res.existing} Länder vorhanden`);
      } else {
        toast.success(`${res.inserted} Länder importiert`);
      }
    },
    onError: () => toast.error('Fehler beim Importieren'),
  });

  const reset = () => setForm({ name: '', currency: 'EUR', currencyCode: 'EUR' });

  const startEdit = (c: Country) => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
    setEditing(c);
    setForm({ ...c });
    setShowForm(false);
  };

  const submit = () => {
    if (!form.name) return;
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  const f = (key: keyof Country) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm(prev => ({ ...prev, [key]: e.target.value }));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Länder</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Länder</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => { if (confirm('Alle ISO-Länder der Welt importieren (nur wenn noch keine vorhanden)?')) seedMutation.mutate(); }}
            disabled={seedMutation.isPending}
            title="Alle Länder der Welt nach ISO 3166-1 importieren"
            className="flex items-center gap-2 px-4 py-2 bg-dark-700 text-dark-300 rounded-lg hover:bg-dark-600 hover:text-white disabled:opacity-50 text-sm"
          >
            <Download className="w-4 h-4" />
            {seedMutation.isPending ? 'Importiere...' : 'ISO-Länder importieren'}
          </button>
          <button
            onClick={() => { reset(); setEditing(null); setShowForm(true); }}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm"
          >
            <Plus className="w-4 h-4" />
            Land anlegen
          </button>
        </div>
      </div>

      {(showForm || editing) && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">{editing ? 'Bearbeiten' : 'Neues Land'}</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            <div className="lg:col-span-2">
              <label className={labelCls}>Name *</label>
              <input type="text" value={form.name ?? ''} onChange={f('name')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>ISO 2 (z.B. DE)</label>
              <input type="text" maxLength={2} value={form.iso2 ?? ''} onChange={f('iso2')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>ISO 3 (z.B. DEU)</label>
              <input type="text" maxLength={3} value={form.iso3 ?? ''} onChange={f('iso3')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>ISO Numerisch</label>
              <input type="text" maxLength={3} value={form.isoNumeric ?? ''} onChange={f('isoNumeric')} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>Währung</label>
              <input type="text" value={form.currency ?? ''} onChange={f('currency')} className={inputCls} placeholder="Euro" />
            </div>
            <div>
              <label className={labelCls}>Währungscode (ISO 4217)</label>
              <input type="text" maxLength={3} value={form.currencyCode ?? ''} onChange={f('currencyCode')} className={inputCls} placeholder="EUR" />
            </div>
            <div>
              <label className={labelCls}>Währungssymbol</label>
              <input type="text" value={form.currencySymbol ?? ''} onChange={f('currencySymbol')} className={inputCls} placeholder="€" />
            </div>
            <div>
              <label className={labelCls}>Vorwahl</label>
              <input type="text" value={form.phoneCode ?? ''} onChange={f('phoneCode')} className={inputCls} placeholder="+49" />
            </div>
            <div>
              <label className={labelCls}>Region</label>
              <select value={form.region ?? ''} onChange={f('region')} className={inputCls}>
                <option value="">— wählen —</option>
                {REGIONS.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
            </div>
            <div>
              <label className={labelCls}>Hauptstadt</label>
              <input type="text" value={form.capital ?? ''} onChange={f('capital')} className={inputCls} />
            </div>
          </div>
          <div className="flex gap-3">
            <button onClick={submit} disabled={!form.name || createMutation.isPending || updateMutation.isPending}
              className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
              {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); reset(); }}
              className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">Abbrechen</button>
          </div>
        </div>
      )}

      {/* Region filter */}
      <div className="flex items-center gap-2">
        <span className="text-sm text-dark-400">Region:</span>
        {['', ...REGIONS].map(r => (
          <button key={r} onClick={() => setFilterRegion(r)}
            className={`px-3 py-1 rounded-full text-xs font-medium transition-colors ${filterRegion === r ? 'bg-primary-500 text-white' : 'bg-dark-800 text-dark-400 hover:text-white'}`}>
            {r || 'Alle'}
          </button>
        ))}
      </div>

      {isLoading ? <div className="text-dark-400">Lädt...</div> : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Name</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">ISO2</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">ISO3</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Währung</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Code</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Vorwahl</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Region</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Hauptstadt</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {filtered.length === 0 ? (
                <tr><td colSpan={9} className="px-4 py-8 text-center text-dark-400">Keine Länder</td></tr>
              ) : filtered.map(c => (
                <tr key={c.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-2.5 text-white font-medium">{c.name}</td>
                  <td className="px-4 py-2.5 text-dark-300 font-mono">{c.iso2 ?? '—'}</td>
                  <td className="px-4 py-2.5 text-dark-300 font-mono">{c.iso3 ?? '—'}</td>
                  <td className="px-4 py-2.5 text-dark-300">{c.currency ?? '—'}</td>
                  <td className="px-4 py-2.5 text-dark-300 font-mono">{c.currencyCode ?? '—'}{c.currencySymbol ? ` ${c.currencySymbol}` : ''}</td>
                  <td className="px-4 py-2.5 text-dark-300">{c.phoneCode ?? '—'}</td>
                  <td className="px-4 py-2.5">
                    {c.region && (
                      <span className="px-2 py-0.5 rounded-full text-xs bg-dark-700 text-dark-300">{c.region}</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-dark-300">{c.capital ?? '—'}</td>
                  <td className="px-4 py-2.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <button onClick={() => startEdit(c)} className="p-1 text-dark-500 hover:text-primary-400"><Pencil className="w-3.5 h-3.5" /></button>
                      <button onClick={() => { if (confirm(`${c.name} löschen?`)) deleteMutation.mutate(c.id); }}
                        className="p-1 text-dark-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5" /></button>
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
