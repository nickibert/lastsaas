import { useState } from 'react';
import { useHighlightRow } from '../../../hooks/useHighlightRow';
import { Link, useSearchParams } from 'react-router-dom';
import { useLocalStorage } from '../../../hooks/useLocalStorage';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Search, Trash2, Pencil, Tag, X, SlidersHorizontal } from 'lucide-react';
import { toast } from 'sonner';
import {
  productsApi, goodsGroupsApi, suppliersApi, productPriceListsApi,
  type Product, type ProductAttribute, type ProductPriceList,
} from '../../../api/procurement';
import { useTenant } from '../../../contexts/TenantContext';
import { Pagination } from '../../../components/Pagination';

const inputCls = 'w-full px-3 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500';
const labelCls = 'block text-xs text-dark-400 mb-1';

type FormTab = 'grunddaten' | 'tags' | 'preislisten';

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
        <input
          type="text"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); add(); } }}
          placeholder="Tag eingeben + Enter"
          className={inputCls}
        />
        <button type="button" onClick={add} className="px-3 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">
          <Plus className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}

function AttributeEditor({ attrs, onChange }: { attrs: ProductAttribute[]; onChange: (a: ProductAttribute[]) => void }) {
  const update = (i: number, field: keyof ProductAttribute, val: string) => {
    const next = attrs.map((a, idx) => idx === i ? { ...a, [field]: val } : a);
    onChange(next);
  };
  const add = () => onChange([...attrs, { key: '', value: '', unit: '' }]);
  const remove = (i: number) => onChange(attrs.filter((_, idx) => idx !== i));

  return (
    <div>
      <label className={labelCls}>Attribute</label>
      <div className="space-y-2">
        {attrs.map((a, i) => (
          <div key={i} className="flex gap-2 items-center">
            <input type="text" placeholder="Merkmal" value={a.key} onChange={e => update(i, 'key', e.target.value)}
              className="flex-1 px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
            <input type="text" placeholder="Wert" value={a.value} onChange={e => update(i, 'value', e.target.value)}
              className="flex-1 px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
            <input type="text" placeholder="Einheit" value={a.unit ?? ''} onChange={e => update(i, 'unit', e.target.value)}
              className="w-24 px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500" />
            <button type="button" onClick={() => remove(i)} className="p-1.5 text-dark-500 hover:text-red-400">
              <X className="w-4 h-4" />
            </button>
          </div>
        ))}
        <button type="button" onClick={add}
          className="flex items-center gap-1.5 text-sm text-dark-400 hover:text-white px-2 py-1 rounded hover:bg-dark-700">
          <Plus className="w-3.5 h-3.5" /> Attribut hinzufügen
        </button>
      </div>
    </div>
  );
}

function PriceListSection({ productId }: { productId: string }) {
  const qc = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<ProductPriceList | null>(null);
  const [form, setForm] = useState<Partial<ProductPriceList>>({ type: 'EK', currency: 'EUR', price: 0 });

  const { data: priceLists = [] } = useQuery({
    queryKey: ['product-price-lists', productId],
    queryFn: () => productPriceListsApi.list(productId),
    throwOnError: false,
  });

  const createMut = useMutation({
    mutationFn: (data: Partial<ProductPriceList>) => productPriceListsApi.create(productId, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['product-price-lists', productId] }); setShowForm(false); resetForm(); toast.success('Preisliste angelegt'); },
    onError: () => toast.error('Fehler'),
  });

  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ProductPriceList> }) => productPriceListsApi.update(productId, id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['product-price-lists', productId] }); setEditing(null); resetForm(); toast.success('Gespeichert'); },
    onError: () => toast.error('Fehler'),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => productPriceListsApi.delete(productId, id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['product-price-lists', productId] }); toast.success('Gelöscht'); },
  });

  const resetForm = () => setForm({ type: 'EK', currency: 'EUR', price: 0 });
  const f = (key: keyof ProductPriceList) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm(prev => ({ ...prev, [key]: key === 'price' ? parseFloat(e.target.value) : e.target.value }));

  const submit = () => {
    if (!form.name || !form.currency) return;
    if (editing) updateMut.mutate({ id: editing.id, data: form });
    else createMut.mutate(form);
  };

  return (
    <div className="space-y-3">
      {(showForm || editing) && (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3 p-3 bg-dark-800/30 rounded-lg">
          <div>
            <label className={labelCls}>Typ</label>
            <select value={form.type ?? 'EK'} onChange={f('type')} className={inputCls}>
              <option value="EK">EK (Einkaufspreis)</option>
              <option value="VK">VK (Verkaufspreis)</option>
            </select>
          </div>
          <div>
            <label className={labelCls}>Name *</label>
            <input type="text" value={form.name ?? ''} onChange={f('name')} className={inputCls} placeholder="z.B. Standard, Kunde A" />
          </div>
          <div>
            <label className={labelCls}>Preis</label>
            <input type="number" step="0.01" value={form.price ?? 0} onChange={f('price')} className={inputCls} />
          </div>
          <div>
            <label className={labelCls}>Währung</label>
            <input type="text" maxLength={3} value={form.currency ?? 'EUR'} onChange={f('currency')} className={inputCls} />
          </div>
          <div>
            <label className={labelCls}>Gültig ab</label>
            <input type="date" value={form.validFrom?.slice(0, 10) ?? ''} onChange={f('validFrom')} className={inputCls} />
          </div>
          <div>
            <label className={labelCls}>Gültig bis</label>
            <input type="date" value={form.validTo?.slice(0, 10) ?? ''} onChange={f('validTo')} className={inputCls} />
          </div>
          <div className="col-span-2 md:col-span-3">
            <label className={labelCls}>Notiz</label>
            <input type="text" value={form.notes ?? ''} onChange={f('notes')} className={inputCls} />
          </div>
          <div className="flex gap-2 col-span-2 md:col-span-3">
            <button onClick={submit} disabled={!form.name || createMut.isPending || updateMut.isPending}
              className="px-3 py-1.5 bg-primary-500 text-white rounded-lg text-sm hover:bg-primary-600 disabled:opacity-50">
              Speichern
            </button>
            <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
              className="px-3 py-1.5 bg-dark-700 text-white rounded-lg text-sm hover:bg-dark-600">
              Abbrechen
            </button>
          </div>
        </div>
      )}

      {priceLists.length > 0 && (
        <table className="w-full text-sm">
          <thead>
            <tr className="text-dark-400 text-xs">
              <th className="text-left py-1 pr-3">Typ</th>
              <th className="text-left py-1 pr-3">Name</th>
              <th className="text-right py-1 pr-3">Preis</th>
              <th className="text-left py-1 pr-3">Währung</th>
              <th className="text-left py-1 pr-3">Gültig ab</th>
              <th className="text-left py-1 pr-3">Gültig bis</th>
              <th className="py-1"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-dark-700/30">
            {priceLists.map(pl => (
              <tr key={pl.id} className="text-dark-300">
                <td className="py-1.5 pr-3">
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${pl.type === 'VK' ? 'bg-emerald-500/20 text-emerald-400' : 'bg-amber-500/20 text-amber-400'}`}>
                    {pl.type}
                  </span>
                </td>
                <td className="py-1.5 pr-3 text-white">{pl.name}</td>
                <td className="py-1.5 pr-3 text-right font-mono">{pl.price.toFixed(2)}</td>
                <td className="py-1.5 pr-3 font-mono">{pl.currency}</td>
                <td className="py-1.5 pr-3">{pl.validFrom ? new Date(pl.validFrom).toLocaleDateString('de-DE') : '—'}</td>
                <td className="py-1.5 pr-3">{pl.validTo ? new Date(pl.validTo).toLocaleDateString('de-DE') : '—'}</td>
                <td className="py-1.5 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => { setEditing(pl); setForm({ ...pl }); setShowForm(false); }}
                      className="p-1 text-dark-500 hover:text-primary-400"><Pencil className="w-3.5 h-3.5" /></button>
                    <button onClick={() => { if (confirm('Preisliste löschen?')) deleteMut.mutate(pl.id); }}
                      className="p-1 text-dark-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5" /></button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!showForm && !editing && (
        <button onClick={() => { resetForm(); setEditing(null); setShowForm(true); }}
          className="flex items-center gap-1.5 text-sm text-dark-400 hover:text-white px-2 py-1 rounded hover:bg-dark-700">
          <Plus className="w-3.5 h-3.5" /> Preisliste hinzufügen
        </button>
      )}
    </div>
  );
}

export default function ProductsPage() {
  const qc = useQueryClient();
  const { activeTenant } = useTenant();
  const enabled = !!activeTenant;
  const [searchParams] = useSearchParams();
  const [search, setSearch] = useState(() => searchParams.get('q') ?? '');
  const [supplierFilter, setSupplierFilter] = useState('');
  const [goodsGroupFilter, setGoodsGroupFilter] = useState('');
  const [tagFilter, setTagFilter] = useState('');
  const [attrKeyFilter, setAttrKeyFilter] = useState('');
  const [attrValueFilter, setAttrValueFilter] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useLocalStorage<number>('table_page_size', 25);
  const [showForm, setShowForm] = useState(false);
  const [editing, setEditing] = useState<Product | null>(null);
  const [tab, setTab] = useState<FormTab>('grunddaten');
  const [form, setForm] = useState<Partial<Product>>({
    nameShort: '', nameLong: '', ean: '', wtn: '',
    tags: [], attributes: [],
  });

  const { data, isLoading } = useQuery({
    queryKey: ['products', search, page, limit, supplierFilter, goodsGroupFilter, tagFilter, attrKeyFilter, attrValueFilter],
    queryFn: () => productsApi.list(
      search || undefined, page, limit,
      supplierFilter || undefined,
      goodsGroupFilter || undefined,
      tagFilter || undefined,
      attrKeyFilter || undefined,
      attrValueFilter || undefined,
    ),
    enabled,
    throwOnError: false,
    placeholderData: prev => prev,
  });

  const products = data?.items ?? [];
  useHighlightRow(products.length > 0);
  const total = data?.total ?? 0;
  const pages = data?.pages ?? 1;

  const handleSearch = (q: string) => { setSearch(q); setPage(1); };
  const handleSupplierFilter = (id: string) => { setSupplierFilter(id); setPage(1); };
  const activeFilterCount = [supplierFilter, goodsGroupFilter, tagFilter, attrKeyFilter].filter(Boolean).length;

  const { data: goodsGroupsData } = useQuery({ queryKey: ['goods-groups', 1, 500], queryFn: () => goodsGroupsApi.list(1, 500), enabled, throwOnError: false });
  const goodsGroups = goodsGroupsData?.items ?? [];
  const { data: suppliersData } = useQuery({ queryKey: ['suppliers', 1, 500], queryFn: () => suppliersApi.list(1, 500), enabled, throwOnError: false });
  const suppliers = suppliersData?.items ?? [];

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

  const resetForm = () => setForm({ nameShort: '', nameLong: '', ean: '', wtn: '', tags: [], attributes: [] });

  const startEdit = (p: Product) => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
    setEditing(p);
    setForm({ ...p, tags: p.tags ?? [], attributes: p.attributes ?? [] });
    setTab('grunddaten');
    setShowForm(true);
  };

  const submit = () => {
    if (editing) updateMutation.mutate({ id: editing.id, data: form });
    else createMutation.mutate(form);
  };

  const inp = (key: keyof Product) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setForm(f => ({ ...f, [key]: e.target.value }));

  const tabCls = (t: FormTab) =>
    `px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${tab === t ? 'bg-dark-800 text-white' : 'text-dark-400 hover:text-white'}`;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Produkte</h1>
          <p className="text-dark-400 mt-1">{total.toLocaleString('de-DE')} Produkte</p>
        </div>
        <button
          onClick={() => { resetForm(); setEditing(null); setTab('grunddaten'); setShowForm(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 text-sm"
        >
          <Plus className="w-4 h-4" />
          Produkt anlegen
        </button>
      </div>

      <div className="space-y-2">
        <div className="flex gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-dark-400" />
            <input type="text" placeholder="Name, Eigenname oder EAN suchen..." value={search}
              onChange={e => handleSearch(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-dark-800 border border-dark-700 rounded-lg text-white placeholder-dark-400 text-sm focus:outline-none focus:border-primary-500" />
          </div>
          <button onClick={() => setShowFilters(!showFilters)}
            className={`flex items-center gap-2 px-3 py-2 border rounded-lg text-sm transition-colors ${
              showFilters || activeFilterCount > 0
                ? 'bg-primary-500/20 border-primary-500/50 text-primary-400'
                : 'bg-dark-800 border-dark-700 text-dark-400 hover:text-white'
            }`}>
            <SlidersHorizontal className="w-4 h-4" />
            Filter{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
          </button>
        </div>
        {showFilters && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-3 bg-dark-800/30 border border-dark-700 rounded-xl">
            <div>
              <label className="block text-xs text-dark-400 mb-1">Lieferant</label>
              <select value={supplierFilter} onChange={e => handleSupplierFilter(e.target.value)}
                className="w-full px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500">
                <option value="">Alle</option>
                {suppliers.map(s => <option key={s.id} value={s.id}>{s.company || `${s.firstname} ${s.lastname}`.trim()}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">Warengruppe</label>
              <select value={goodsGroupFilter} onChange={e => { setGoodsGroupFilter(e.target.value); setPage(1); }}
                className="w-full px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500">
                <option value="">Alle</option>
                {goodsGroups.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">Tag</label>
              <input type="text" value={tagFilter} onChange={e => { setTagFilter(e.target.value); setPage(1); }}
                placeholder="z.B. aktiv" className="w-full px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500" />
            </div>
            <div>
              <label className="block text-xs text-dark-400 mb-1">Attribut Schlüssel</label>
              <input type="text" value={attrKeyFilter} onChange={e => { setAttrKeyFilter(e.target.value); setPage(1); }}
                placeholder="z.B. Farbe" className="w-full px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500" />
            </div>
            {attrKeyFilter && (
              <div>
                <label className="block text-xs text-dark-400 mb-1">Attribut Wert</label>
                <input type="text" value={attrValueFilter} onChange={e => { setAttrValueFilter(e.target.value); setPage(1); }}
                  placeholder="z.B. Rot" className="w-full px-3 py-1.5 bg-dark-800 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500" />
              </div>
            )}
            {activeFilterCount > 0 && (
              <div className="flex items-end">
                <button onClick={() => { setSupplierFilter(''); setGoodsGroupFilter(''); setTagFilter(''); setAttrKeyFilter(''); setAttrValueFilter(''); setPage(1); }}
                  className="px-3 py-1.5 text-xs text-dark-400 hover:text-white bg-dark-700 rounded-lg">
                  Filter zurücksetzen
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {showForm && (
        <div className="bg-dark-900/50 border border-dark-700 rounded-xl overflow-hidden">
          {/* Tab bar */}
          <div className="flex gap-1 px-4 pt-3 border-b border-dark-700 bg-dark-900/30">
            <button className={tabCls('grunddaten')} onClick={() => setTab('grunddaten')}>Grunddaten</button>
            <button className={tabCls('tags')} onClick={() => setTab('tags')}>
              <span className="flex items-center gap-1.5"><Tag className="w-3.5 h-3.5" />Tags & Attribute</span>
            </button>
            {editing && (
              <button className={tabCls('preislisten')} onClick={() => setTab('preislisten')}>Preislisten</button>
            )}
          </div>

          <div className="p-6 space-y-4">
            <h2 className="text-base font-semibold text-white">{editing ? 'Produkt bearbeiten' : 'Neues Produkt'}</h2>

            {tab === 'grunddaten' && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className={labelCls}>Kurzname *</label>
                  <input type="text" value={form.nameShort ?? ''} onChange={inp('nameShort')} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Eigenname</label>
                  <input type="text" value={form.ownNameShort ?? ''} onChange={inp('ownNameShort')} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Langname</label>
                  <input type="text" value={form.nameLong ?? ''} onChange={inp('nameLong')} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>EAN</label>
                  <input type="text" value={form.ean ?? ''} onChange={inp('ean')} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>WTN (Zolltarifnummer)</label>
                  <input type="text" value={form.wtn ?? ''} onChange={inp('wtn')} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>Warengruppe</label>
                  <select value={form.goodsGroupId ?? ''} onChange={inp('goodsGroupId')}
                    className={inputCls}>
                    <option value="">— keine —</option>
                    {goodsGroups.map(g => <option key={g.id} value={g.id}>{g.name} – {g.short}</option>)}
                  </select>
                </div>
                <div>
                  <label className={labelCls}>EK-Preis (EUR)</label>
                  <input type="number" step="0.01" value={form.lastEk ?? 0}
                    onChange={e => setForm(f => ({ ...f, lastEk: parseFloat(e.target.value) }))} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>VPE</label>
                  <input type="number" value={form.vpe ?? 0}
                    onChange={e => setForm(f => ({ ...f, vpe: parseInt(e.target.value) }))} className={inputCls} />
                </div>
              </div>
            )}

            {tab === 'tags' && (
              <div className="space-y-6">
                <TagInput
                  tags={form.tags ?? []}
                  onChange={tags => setForm(f => ({ ...f, tags }))}
                />
                <AttributeEditor
                  attrs={form.attributes ?? []}
                  onChange={attributes => setForm(f => ({ ...f, attributes }))}
                />
              </div>
            )}

            {tab === 'preislisten' && editing && (
              <PriceListSection productId={editing.id} />
            )}

            {tab !== 'preislisten' && (
              <div className="flex gap-3 pt-2">
                <button onClick={submit} disabled={!form.nameShort || createMutation.isPending || updateMutation.isPending}
                  className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50 text-sm">
                  {(createMutation.isPending || updateMutation.isPending) ? 'Speichert...' : 'Speichern'}
                </button>
                <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
                  className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">Abbrechen</button>
              </div>
            )}
            {tab === 'preislisten' && (
              <div className="flex gap-3 pt-2 border-t border-dark-700">
                <button onClick={() => { setShowForm(false); setEditing(null); resetForm(); }}
                  className="px-4 py-2 bg-dark-700 text-white rounded-lg hover:bg-dark-600 text-sm">Schließen</button>
              </div>
            )}
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-dark-400">Lädt...</div>
      ) : (
        <div className="bg-dark-900/50 border border-dark-800 rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead className="border-b border-dark-800">
              <tr>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Kurzname</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Eigenname</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Langname</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">EAN</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Warengruppe</th>
                <th className="text-left px-4 py-3 text-dark-400 font-medium">Tags</th>
                <th className="text-right px-4 py-3 text-dark-400 font-medium">EK (EUR)</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-dark-800/50">
              {products.length === 0 ? (
                <tr><td colSpan={8} className="px-4 py-8 text-center text-dark-400">Keine Produkte gefunden</td></tr>
              ) : products.map(p => (
                <tr key={p.id} data-highlight-id={p.id} className="hover:bg-dark-800/30">
                  <td className="px-4 py-3 text-white font-mono">{p.nameShort}</td>
                  <td className="px-4 py-3 text-dark-300">{p.ownNameShort || '—'}</td>
                  <td className="px-4 py-3 text-dark-400">{p.nameLong || '—'}</td>
                  <td className="px-4 py-3 text-dark-400 font-mono">{p.ean || '—'}</td>
                  <td className="px-4 py-3 text-sm">
                    {p.goodsGroupId
                      ? <Link to={`/procurement/goods-groups?highlight=${p.goodsGroupId}`} className="text-dark-300 hover:text-primary-400 transition-colors">
                          {goodsGroups.find(g => g.id === p.goodsGroupId)?.name ?? '—'}
                        </Link>
                      : <span className="text-dark-500">—</span>}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {(p.tags ?? []).map(t => (
                        <span key={t} className="px-1.5 py-0.5 bg-primary-500/15 text-primary-400 rounded text-xs">{t}</span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right text-dark-300 font-mono">
                    {p.lastEk > 0 ? p.lastEk.toFixed(2) : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex items-center justify-end gap-1">
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
          <Pagination page={page} pages={pages} total={total} limit={limit} onPage={setPage} onLimit={l => { setLimit(l); setPage(1); }} />
        </div>
      )}
    </div>
  );
}
